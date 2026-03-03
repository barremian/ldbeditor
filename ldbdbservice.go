package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

// LevelDBService provides LevelDB database operations for the frontend.
type LevelDBService struct {
	mu  sync.RWMutex
	dbs map[string]*dbEntry
}

type dbEntry struct {
	db             *leveldb.DB
	dbLocked       bool
	forcedReadOnly bool
	readOnlyReason string
	lockedByApp    string
	refCount       int
}

type openedDB struct {
	db             *leveldb.DB
	forcedReadOnly bool
	readOnlyReason string
	lockedByApp    string
}

var errDatabaseLocked = errors.New("database is locked")
var errDatabaseForcedReadOnly = errors.New("database is open read-only because it is locked by another application")

// OpenDatabaseResult is the result of opening a LevelDB database.
type OpenDatabaseResult struct {
	Ok             bool   `json:"ok"`
	Error          string `json:"error"`
	CanonicalPath  string `json:"canonicalPath"`
	AlreadyOpen    bool   `json:"alreadyOpen"`
	ReadOnly       bool   `json:"readOnly"`
	ForcedReadOnly bool   `json:"forcedReadOnly"`
	ReadOnlyReason string `json:"readOnlyReason"`
	LockedByApp    string `json:"lockedByApp"`
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
			Ok:             true,
			CanonicalPath:  canonicalPath,
			AlreadyOpen:    true,
			ReadOnly:       existing.dbLocked || existing.forcedReadOnly,
			ForcedReadOnly: existing.forcedReadOnly,
			ReadOnlyReason: existing.readOnlyReason,
			LockedByApp:    existing.lockedByApp,
		}
	}

	opened, openErr := openDatabaseHandle(canonicalPath)
	if openErr != nil {
		return OpenDatabaseResult{Ok: false, Error: openErr.Error()}
	}

	s.dbs[canonicalPath] = &dbEntry{
		db:             opened.db,
		dbLocked:       false,
		forcedReadOnly: opened.forcedReadOnly,
		readOnlyReason: opened.readOnlyReason,
		lockedByApp:    opened.lockedByApp,
		refCount:       1,
	}
	return openResultFromEntry(canonicalPath, false, s.dbs[canonicalPath])
}

// RefreshDatabase closes and reopens an already-open database path so reads reflect external updates.
func (s *LevelDBService) RefreshDatabase(path string) (OpenDatabaseResult, error) {
	canonicalPath, err := canonicalizePath(path)
	if err != nil {
		return OpenDatabaseResult{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.dbs[canonicalPath]
	if !ok || entry.db == nil {
		return OpenDatabaseResult{}, errors.New("database is not open")
	}

	refCount := entry.refCount
	dbLocked := entry.dbLocked
	if closeErr := entry.db.Close(); closeErr != nil {
		return OpenDatabaseResult{}, closeErr
	}

	opened, openErr := openDatabaseHandle(canonicalPath)
	if openErr != nil {
		delete(s.dbs, canonicalPath)
		return OpenDatabaseResult{}, openErr
	}

	s.dbs[canonicalPath] = &dbEntry{
		db:             opened.db,
		dbLocked:       dbLocked,
		forcedReadOnly: opened.forcedReadOnly,
		readOnlyReason: opened.readOnlyReason,
		lockedByApp:    opened.lockedByApp,
		refCount:       refCount,
	}

	return openResultFromEntry(canonicalPath, true, s.dbs[canonicalPath]), nil
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
	if entry.forcedReadOnly {
		return nil, errDatabaseForcedReadOnly
	}
	if entry.dbLocked {
		return nil, errDatabaseLocked
	}
	return entry.db, nil
}

func isLevelDBLockError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, storage.ErrLocked) {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "already locked") ||
		strings.Contains(message, "resource temporarily unavailable")
}

func openDatabaseHandle(canonicalPath string) (*openedDB, error) {
	db, openErr := leveldb.OpenFile(canonicalPath, nil)
	if openErr == nil {
		return &openedDB{db: db}, nil
	}
	if !isLevelDBLockError(openErr) {
		return nil, openErr
	}

	readOnlyDB, readOnlyErr := openLevelDBReadOnlyNoLock(canonicalPath)
	if readOnlyErr != nil {
		return nil, readOnlyErr
	}

	lockedByApp := detectLockingAppName(canonicalPath)
	reason := "Database is locked by another application."
	if lockedByApp != "" {
		reason = fmt.Sprintf("Database is locked by %s.", lockedByApp)
	}

	return &openedDB{
		db:             readOnlyDB,
		forcedReadOnly: true,
		readOnlyReason: reason,
		lockedByApp:    lockedByApp,
	}, nil
}

func openResultFromEntry(canonicalPath string, alreadyOpen bool, entry *dbEntry) OpenDatabaseResult {
	if entry == nil {
		return OpenDatabaseResult{
			Ok:            false,
			Error:         "database is not open",
			CanonicalPath: canonicalPath,
		}
	}
	return OpenDatabaseResult{
		Ok:             true,
		CanonicalPath:  canonicalPath,
		AlreadyOpen:    alreadyOpen,
		ReadOnly:       entry.dbLocked || entry.forcedReadOnly,
		ForcedReadOnly: entry.forcedReadOnly,
		ReadOnlyReason: entry.readOnlyReason,
		LockedByApp:    entry.lockedByApp,
	}
}

func detectLockingAppName(canonicalPath string) string {
	lockFilePath := filepath.Join(canonicalPath, "LOCK")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	// lsof is best-effort and may not be available on every platform/build.
	output, err := exec.CommandContext(ctx, "lsof", "-Fpc", "--", lockFilePath).Output()
	if err != nil {
		return ""
	}
	return parseLsofProcessName(output, os.Getpid())
}

func parseLsofProcessName(output []byte, skipPID int) string {
	lines := strings.Split(string(output), "\n")
	currentPID := 0
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		switch line[0] {
		case 'p':
			pidText := strings.TrimSpace(line[1:])
			currentPID = 0
			if pidText == "" {
				continue
			}
			var parsedPID int
			_, _ = fmt.Sscanf(pidText, "%d", &parsedPID)
			currentPID = parsedPID
		case 'c':
			if currentPID != 0 && currentPID == skipPID {
				continue
			}
			name := strings.TrimSpace(line[1:])
			if name != "" {
				return name
			}
		}
	}
	return ""
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
