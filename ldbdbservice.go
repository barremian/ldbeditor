package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"unicode/utf8"

	"github.com/syndtr/goleveldb/leveldb"
)

// LevelDBService provides LevelDB database operations for the frontend.
type LevelDBService struct {
	mu sync.RWMutex
	dbs map[string]*dbEntry
}

type dbEntry struct {
	db       *leveldb.DB
	dbLocked bool
	refCount int
}

var errDatabaseLocked = errors.New("database is locked")

// OpenDatabaseResult is the result of opening a LevelDB database.
type OpenDatabaseResult struct {
	Ok            bool   `json:"ok"`
	Error         string `json:"error"`
	CanonicalPath string `json:"canonicalPath"`
	AlreadyOpen   bool   `json:"alreadyOpen"`
}

// PutValueIfUnchangedResult is the result of a guarded value write.
type PutValueIfUnchangedResult struct {
	Ok                 bool   `json:"ok"`
	Conflict           bool   `json:"conflict"`
	CurrentValue       string `json:"currentValue"`
	CurrentValueExists bool   `json:"currentValueExists"`
}

// OpenDatabase opens a LevelDB database at the given path.
// Returns Ok=true on success, Ok=false with Error set when the path is not a valid LevelDB database.
func (s *LevelDBService) OpenDatabase(path string) OpenDatabaseResult {
	canonicalPath, err := canonicalizePath(path)
	if err != nil {
		return OpenDatabaseResult{Ok: false, Error: err.Error()}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dbs == nil {
		s.dbs = map[string]*dbEntry{}
	}

	if existing, ok := s.dbs[canonicalPath]; ok {
		existing.refCount++
		return OpenDatabaseResult{
			Ok:            true,
			CanonicalPath: canonicalPath,
			AlreadyOpen:   true,
		}
	}

	db, openErr := leveldb.OpenFile(canonicalPath, nil)
	if openErr != nil {
		return OpenDatabaseResult{Ok: false, Error: openErr.Error()}
	}

	s.dbs[canonicalPath] = &dbEntry{
		db:       db,
		dbLocked: false,
		refCount: 1,
	}
	return OpenDatabaseResult{
		Ok:            true,
		CanonicalPath: canonicalPath,
		AlreadyOpen:   false,
	}
}

// GetKeys returns all keys in the requested open database.
// Keys are returned in lexicographic order.
// Non-UTF-8 keys are hex-encoded.
func (s *LevelDBService) GetKeys(path string) ([]string, error) {
	db, err := s.readableDatabase(path)
	if err != nil {
		return nil, err
	}

	var keys []string
	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		keyBytes := make([]byte, len(iter.Key()))
		copy(keyBytes, iter.Key())
		keys = append(keys, bytesToDisplayString(keyBytes))
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	return keys, nil
}

// GetValue returns the value for the given key.
// The key should be in the same format as returned by GetKeys (UTF-8 or hex with "0x" prefix).
// Non-UTF-8 values are hex-encoded in the response.
func (s *LevelDBService) GetValue(path string, keyDisplay string) (string, error) {
	db, err := s.readableDatabase(path)
	if err != nil {
		return "", err
	}

	key, decodeErr := displayStringToBytes(keyDisplay)
	if decodeErr != nil {
		return "", decodeErr
	}

	value, getErr := db.Get(key, nil)
	if getErr != nil {
		return "", getErr
	}

	return bytesToDisplayString(value), nil
}

// PutValue creates or updates the value for a key.
// Key and value use the same display format as GetKeys/GetValue (UTF-8 or "0x" prefixed hex).
func (s *LevelDBService) PutValue(path string, keyDisplay string, valueDisplay string) error {
	db, err := s.writableDatabase(path)
	if err != nil {
		return err
	}

	key, err := displayStringToBytes(keyDisplay)
	if err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}

	value, err := displayStringToBytes(valueDisplay)
	if err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}

	return db.Put(key, value, nil)
}

// PutValueIfUnchanged updates a key only if its current value matches expectedValueDisplay.
// Set force=true to overwrite regardless of current database value.
func (s *LevelDBService) PutValueIfUnchanged(path string, keyDisplay string, expectedValueDisplay string, newValueDisplay string, force bool) (PutValueIfUnchangedResult, error) {
	db, err := s.writableDatabase(path)
	if err != nil {
		return PutValueIfUnchangedResult{}, err
	}

	key, err := displayStringToBytes(keyDisplay)
	if err != nil {
		return PutValueIfUnchangedResult{}, fmt.Errorf("invalid key: %w", err)
	}

	expectedValue, err := displayStringToBytes(expectedValueDisplay)
	if err != nil {
		return PutValueIfUnchangedResult{}, fmt.Errorf("invalid expected value: %w", err)
	}

	newValue, err := displayStringToBytes(newValueDisplay)
	if err != nil {
		return PutValueIfUnchangedResult{}, fmt.Errorf("invalid value: %w", err)
	}

	if !force {
		currentValue, getErr := db.Get(key, nil)
		if getErr != nil {
			if errors.Is(getErr, leveldb.ErrNotFound) {
				return PutValueIfUnchangedResult{
					Ok:                 false,
					Conflict:           true,
					CurrentValueExists: false,
				}, nil
			}
			return PutValueIfUnchangedResult{}, getErr
		}
		if !bytes.Equal(currentValue, expectedValue) {
			return PutValueIfUnchangedResult{
				Ok:                 false,
				Conflict:           true,
				CurrentValue:       bytesToDisplayString(currentValue),
				CurrentValueExists: true,
			}, nil
		}
	}

	if err := db.Put(key, newValue, nil); err != nil {
		return PutValueIfUnchangedResult{}, err
	}

	return PutValueIfUnchangedResult{
		Ok:       true,
		Conflict: false,
	}, nil
}

// DeleteKey deletes the given key.
// keyDisplay uses the same format as returned by GetKeys.
func (s *LevelDBService) DeleteKey(path string, keyDisplay string) error {
	db, err := s.writableDatabase(path)
	if err != nil {
		return err
	}

	key, err := displayStringToBytes(keyDisplay)
	if err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}

	return db.Delete(key, nil)
}

// RenameKey renames a key while preserving its value.
// oldKeyDisplay and newKeyDisplay use the same key format as GetKeys.
func (s *LevelDBService) RenameKey(path string, oldKeyDisplay string, newKeyDisplay string) error {
	db, err := s.writableDatabase(path)
	if err != nil {
		return err
	}

	oldKey, err := displayStringToBytes(oldKeyDisplay)
	if err != nil {
		return fmt.Errorf("invalid old key: %w", err)
	}

	newKey, err := displayStringToBytes(newKeyDisplay)
	if err != nil {
		return fmt.Errorf("invalid new key: %w", err)
	}

	if string(oldKey) == string(newKey) {
		return nil
	}

	value, err := db.Get(oldKey, nil)
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return errors.New("source key does not exist")
		}
		return err
	}

	targetExists, err := db.Has(newKey, nil)
	if err != nil {
		return err
	}
	if targetExists {
		return errors.New("target key already exists")
	}

	batch := new(leveldb.Batch)
	batch.Put(newKey, value)
	batch.Delete(oldKey)
	return db.Write(batch, nil)
}

// SetDatabaseLocked sets whether write operations are blocked for the open database.
func (s *LevelDBService) SetDatabaseLocked(path string, locked bool) error {
	canonicalPath, err := canonicalizePath(path)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.dbs[canonicalPath]
	if !ok {
		return errors.New("database is not open")
	}

	entry.dbLocked = locked
	return nil
}

// CloseDatabase releases one open reference for a database path and closes the DB when refcount reaches zero.
func (s *LevelDBService) CloseDatabase(path string) error {
	canonicalPath, err := canonicalizePath(path)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.dbs[canonicalPath]
	if !ok {
		return nil
	}

	entry.refCount--
	if entry.refCount > 0 {
		return nil
	}

	closeErr := entry.db.Close()
	delete(s.dbs, canonicalPath)
	return closeErr
}

func (s *LevelDBService) readableDatabase(path string) (*leveldb.DB, error) {
	canonicalPath, err := canonicalizePath(path)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	entry, ok := s.dbs[canonicalPath]
	s.mu.RUnlock()

	if !ok || entry.db == nil {
		return nil, errors.New("database is not open")
	}
	return entry.db, nil
}

func (s *LevelDBService) writableDatabase(path string) (*leveldb.DB, error) {
	canonicalPath, err := canonicalizePath(path)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	entry, ok := s.dbs[canonicalPath]
	s.mu.RUnlock()

	if !ok || entry.db == nil {
		return nil, errors.New("database is not open")
	}
	if entry.dbLocked {
		return nil, errDatabaseLocked
	}
	return entry.db, nil
}

func canonicalizePath(path string) (string, error) {
	if path == "" {
		return "", errors.New("database path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

// bytesToDisplayString converts bytes to a display string. Valid UTF-8 is returned as-is; otherwise hex is used.
func bytesToDisplayString(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	return "0x" + hex.EncodeToString(b)
}

// displayStringToBytes converts a display string back to bytes. Handles "0x" hex prefix.
func displayStringToBytes(s string) ([]byte, error) {
	if len(s) >= 2 && s[:2] == "0x" {
		return hex.DecodeString(s[2:])
	}
	return []byte(s), nil
}
