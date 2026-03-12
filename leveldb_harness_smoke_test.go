package main

import (
	"errors"
	"os"
	"testing"
)

func TestLevelDBHarnessLifecycle(t *testing.T) {
	var harnessPath string

	t.Run("create write and read", func(t *testing.T) {
		db, dbPath, cleanup := createLevelDBHarness(t)
		t.Cleanup(cleanup)
		harnessPath = dbPath

		if err := db.Put([]byte("name"), []byte("ldb editor"), nil); err != nil {
			t.Fatalf("put fixture value: %v", err)
		}

		value, err := db.Get([]byte("name"), nil)
		if err != nil {
			t.Fatalf("get fixture value: %v", err)
		}
		if string(value) != "ldb editor" {
			t.Fatalf("unexpected value: got %q", string(value))
		}
	})

	if _, err := os.Stat(harnessPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("harness path should be deleted after cleanup: %q", harnessPath)
	}
}
