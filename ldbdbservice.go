package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"unicode/utf8"

	"github.com/syndtr/goleveldb/leveldb"
)

// LevelDBService provides LevelDB database operations for the frontend.
type LevelDBService struct {
	mu sync.RWMutex
	db *leveldb.DB

	// dbLocked controls whether write operations are allowed.
	dbLocked bool
}

var errDatabaseLocked = errors.New("database is locked")

// OpenDatabaseResult is the result of opening a LevelDB database.
type OpenDatabaseResult struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
}

// OpenDatabase opens a LevelDB database at the given path.
// Returns Ok=true on success, Ok=false with Error set when the path is not a valid LevelDB database.
func (s *LevelDBService) OpenDatabase(path string) OpenDatabaseResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		_ = s.db.Close()
		s.db = nil
	}

	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return OpenDatabaseResult{Ok: false, Error: err.Error()}
	}

	s.db = db
	s.dbLocked = false
	return OpenDatabaseResult{Ok: true}
}

// GetKeys returns all keys in the currently open database.
// Keys are returned in lexicographic order.
// Non-UTF-8 keys are hex-encoded.
func (s *LevelDBService) GetKeys() ([]string, error) {
	s.mu.RLock()
	db := s.db
	s.mu.RUnlock()

	if db == nil {
		return nil, nil
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
func (s *LevelDBService) GetValue(keyDisplay string) (string, error) {
	s.mu.RLock()
	db := s.db
	s.mu.RUnlock()

	if db == nil {
		return "", nil
	}

	key, err := displayStringToBytes(keyDisplay)
	if err != nil {
		return "", err
	}

	value, err := db.Get(key, nil)
	if err != nil {
		return "", err
	}

	return bytesToDisplayString(value), nil
}

// PutValue creates or updates the value for a key.
// Key and value use the same display format as GetKeys/GetValue (UTF-8 or "0x" prefixed hex).
func (s *LevelDBService) PutValue(keyDisplay string, valueDisplay string) error {
	db, err := s.writableDatabase()
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

// DeleteKey deletes the given key.
// keyDisplay uses the same format as returned by GetKeys.
func (s *LevelDBService) DeleteKey(keyDisplay string) error {
	db, err := s.writableDatabase()
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
func (s *LevelDBService) RenameKey(oldKeyDisplay string, newKeyDisplay string) error {
	db, err := s.writableDatabase()
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
func (s *LevelDBService) SetDatabaseLocked(locked bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return errors.New("database is not open")
	}

	s.dbLocked = locked
	return nil
}

// CloseDatabase closes the currently open database.
func (s *LevelDBService) CloseDatabase() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return nil
	}

	err := s.db.Close()
	s.db = nil
	s.dbLocked = false
	return err
}

func (s *LevelDBService) writableDatabase() (*leveldb.DB, error) {
	s.mu.RLock()
	db := s.db
	locked := s.dbLocked
	s.mu.RUnlock()

	if db == nil {
		return nil, errors.New("database is not open")
	}
	if locked {
		return nil, errDatabaseLocked
	}
	return db, nil
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
