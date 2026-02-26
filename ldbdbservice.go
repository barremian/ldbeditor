package main

import (
	"encoding/hex"
	"sync"
	"unicode/utf8"

	"github.com/syndtr/goleveldb/leveldb"
)

// LevelDBService provides LevelDB database operations for the frontend.
type LevelDBService struct {
	mu sync.RWMutex
	db *leveldb.DB
}

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

// CloseDatabase closes the currently open database.
func (s *LevelDBService) CloseDatabase() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return nil
	}

	err := s.db.Close()
	s.db = nil
	return err
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
