package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

const levelDBHarnessRoot = "tmp/leveldb-harness"
const keepLevelDBHarnessEnv = "KEEP_LEVELDB_HARNESS_DB"

// createLevelDBHarness opens a disposable LevelDB under tmp/ for tests.
// Callers should register the returned cleanup function with t.Cleanup.
// Set KEEP_LEVELDB_HARNESS_DB=1 to preserve the DB files after tests.
func createLevelDBHarness(t *testing.T) (db *leveldb.DB, dbPath string, cleanup func()) {
	t.Helper()

	rootPath := filepath.Join(".", levelDBHarnessRoot)
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		t.Fatalf("create harness root %q: %v", rootPath, err)
	}

	testName := strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(t.Name())
	dbPath = filepath.Join(rootPath, fmt.Sprintf("%s-%d", testName, time.Now().UnixNano()))

	openedDB, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		t.Fatalf("open harness db %q: %v", dbPath, err)
	}

	cleanup = func() {
		if err := openedDB.Close(); err != nil && !errors.Is(err, leveldb.ErrClosed) {
			t.Errorf("close harness db %q: %v", dbPath, err)
		}
		if shouldKeepHarnessDB() {
			t.Logf("keeping harness db at %q (%s=1)", dbPath, keepLevelDBHarnessEnv)
			return
		}
		if err := os.RemoveAll(dbPath); err != nil {
			t.Errorf("remove harness db %q: %v", dbPath, err)
		}
	}

	return openedDB, dbPath, cleanup
}

func shouldKeepHarnessDB() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(keepLevelDBHarnessEnv))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
