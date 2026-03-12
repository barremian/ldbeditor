package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
)

func TestLevelDBServiceWithHarnessDatabase(t *testing.T) {
	sourceDB, dbPath, cleanup := createLevelDBHarness(t)
	t.Cleanup(cleanup)

	if err := sourceDB.Put([]byte("alpha"), []byte("one"), nil); err != nil {
		t.Fatalf("seed alpha: %v", err)
	}
	if err := sourceDB.Put([]byte{0xff, 0x00}, []byte{0xff, 0x02}, nil); err != nil {
		t.Fatalf("seed binary key: %v", err)
	}
	if err := sourceDB.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	service := &LevelDBService{}
	openResult := service.OpenDatabase(dbPath, false)
	if !openResult.Ok {
		t.Fatalf("open seeded db: %s", openResult.Error)
	}
	if openResult.CanonicalPath == "" {
		t.Fatal("open should return canonical path")
	}
	canonicalPath := openResult.CanonicalPath
	t.Cleanup(func() {
		if err := service.CloseDatabase(canonicalPath); err != nil {
			t.Errorf("close service db: %v", err)
		}
	})

	keys, err := service.GetKeys(canonicalPath)
	if err != nil {
		t.Fatalf("get keys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d (%v)", len(keys), keys)
	}

	value, err := service.GetValue(canonicalPath, "alpha")
	if err != nil {
		t.Fatalf("get value for alpha: %v", err)
	}
	if value != "one" {
		t.Fatalf("unexpected alpha value: got %q", value)
	}

	hexValue, err := service.GetValue(canonicalPath, "0xff00")
	if err != nil {
		t.Fatalf("get value for binary key: %v", err)
	}
	if hexValue != "0xff02" {
		t.Fatalf("unexpected binary value: got %q", hexValue)
	}

	t.Run("put creates and updates utf8 values", func(t *testing.T) {
		if err := service.PutValue(canonicalPath, "beta", "two"); err != nil {
			t.Fatalf("create beta: %v", err)
		}
		got, err := service.GetValue(canonicalPath, "beta")
		if err != nil {
			t.Fatalf("get beta after create: %v", err)
		}
		if got != "two" {
			t.Fatalf("unexpected beta value after create: %q", got)
		}

		if err := service.PutValue(canonicalPath, "alpha", "uno"); err != nil {
			t.Fatalf("update alpha: %v", err)
		}
		got, err = service.GetValue(canonicalPath, "alpha")
		if err != nil {
			t.Fatalf("get alpha after update: %v", err)
		}
		if got != "uno" {
			t.Fatalf("unexpected alpha value after update: %q", got)
		}
	})

	t.Run("put supports binary key and value", func(t *testing.T) {
		if err := service.PutValue(canonicalPath, "0x0102", "0xffee"); err != nil {
			t.Fatalf("create binary entry: %v", err)
		}
		got, err := service.GetValue(canonicalPath, "0x0102")
		if err != nil {
			t.Fatalf("get binary entry: %v", err)
		}
		if got != "0xffee" {
			t.Fatalf("unexpected binary value: %q", got)
		}
	})

	t.Run("guarded put succeeds without conflict", func(t *testing.T) {
		result, err := service.PutValueIfUnchanged(canonicalPath, "alpha", "uno", "eins", false)
		if err != nil {
			t.Fatalf("guarded put should succeed: %v", err)
		}
		if !result.Ok || result.Conflict {
			t.Fatalf("expected successful guarded put, got: %+v", result)
		}

		got, err := service.GetValue(canonicalPath, "alpha")
		if err != nil {
			t.Fatalf("get alpha after guarded put: %v", err)
		}
		if got != "eins" {
			t.Fatalf("unexpected alpha value after guarded put: %q", got)
		}
	})

	t.Run("guarded put detects stale expected value and supports force overwrite", func(t *testing.T) {
		if err := service.PutValue(canonicalPath, "alpha", "remote-update"); err != nil {
			t.Fatalf("seed external change: %v", err)
		}

		conflictResult, err := service.PutValueIfUnchanged(canonicalPath, "alpha", "eins", "local-update", false)
		if err != nil {
			t.Fatalf("guarded put conflict should not return error: %v", err)
		}
		if conflictResult.Ok || !conflictResult.Conflict {
			t.Fatalf("expected conflict result, got: %+v", conflictResult)
		}
		if !conflictResult.CurrentValueExists {
			t.Fatalf("expected conflicting value to exist, got: %+v", conflictResult)
		}
		if conflictResult.CurrentValue != "remote-update" {
			t.Fatalf("unexpected conflicting current value: %q", conflictResult.CurrentValue)
		}

		got, err := service.GetValue(canonicalPath, "alpha")
		if err != nil {
			t.Fatalf("get alpha after conflict check: %v", err)
		}
		if got != "remote-update" {
			t.Fatalf("conflict path must not write local value, got: %q", got)
		}

		forceResult, err := service.PutValueIfUnchanged(canonicalPath, "alpha", "eins", "local-update", true)
		if err != nil {
			t.Fatalf("forced guarded put should succeed: %v", err)
		}
		if !forceResult.Ok || forceResult.Conflict {
			t.Fatalf("expected forced overwrite success, got: %+v", forceResult)
		}

		got, err = service.GetValue(canonicalPath, "alpha")
		if err != nil {
			t.Fatalf("get alpha after force overwrite: %v", err)
		}
		if got != "local-update" {
			t.Fatalf("unexpected alpha value after force overwrite: %q", got)
		}
	})

	t.Run("guarded put conflicts when key is deleted externally", func(t *testing.T) {
		if err := service.PutValue(canonicalPath, "conflict-delete", "initial"); err != nil {
			t.Fatalf("seed conflict-delete: %v", err)
		}
		if err := service.DeleteKey(canonicalPath, "conflict-delete"); err != nil {
			t.Fatalf("delete conflict-delete: %v", err)
		}

		result, err := service.PutValueIfUnchanged(canonicalPath, "conflict-delete", "initial", "local-new", false)
		if err != nil {
			t.Fatalf("guarded put delete conflict should not return error: %v", err)
		}
		if result.Ok || !result.Conflict {
			t.Fatalf("expected delete conflict, got: %+v", result)
		}
		if result.CurrentValueExists {
			t.Fatalf("expected deleted key to report missing current value, got: %+v", result)
		}
	})

	t.Run("delete removes existing keys and ignores missing", func(t *testing.T) {
		if err := service.DeleteKey(canonicalPath, "beta"); err != nil {
			t.Fatalf("delete beta: %v", err)
		}
		_, err := service.GetValue(canonicalPath, "beta")
		if !errors.Is(err, leveldb.ErrNotFound) {
			t.Fatalf("expected not found after delete, got: %v", err)
		}

		if err := service.DeleteKey(canonicalPath, "does-not-exist"); err != nil {
			t.Fatalf("delete missing key should not fail: %v", err)
		}
	})

	t.Run("rename moves value to new key", func(t *testing.T) {
		if err := service.RenameKey(canonicalPath, "alpha", "gamma"); err != nil {
			t.Fatalf("rename alpha->gamma: %v", err)
		}

		_, err := service.GetValue(canonicalPath, "alpha")
		if !errors.Is(err, leveldb.ErrNotFound) {
			t.Fatalf("expected old key to be missing, got: %v", err)
		}

		got, err := service.GetValue(canonicalPath, "gamma")
		if err != nil {
			t.Fatalf("get renamed key: %v", err)
		}
		if got != "local-update" {
			t.Fatalf("unexpected renamed value: %q", got)
		}
	})

	t.Run("rename fails on missing source and existing target", func(t *testing.T) {
		if err := service.PutValue(canonicalPath, "target", "value"); err != nil {
			t.Fatalf("seed target key: %v", err)
		}

		err := service.RenameKey(canonicalPath, "missing", "fresh")
		if err == nil || !strings.Contains(err.Error(), "source key does not exist") {
			t.Fatalf("expected missing source error, got: %v", err)
		}

		err = service.RenameKey(canonicalPath, "gamma", "target")
		if err == nil || !strings.Contains(err.Error(), "target key already exists") {
			t.Fatalf("expected target exists error, got: %v", err)
		}
	})

	t.Run("invalid hex input is rejected", func(t *testing.T) {
		if err := service.PutValue(canonicalPath, "0xzz", "ok"); err == nil {
			t.Fatal("expected invalid hex key error")
		}
		if err := service.PutValue(canonicalPath, "ok", "0xzz"); err == nil {
			t.Fatal("expected invalid hex value error")
		}
		if err := service.DeleteKey(canonicalPath, "0xzz"); err == nil {
			t.Fatal("expected invalid hex key delete error")
		}
		if err := service.RenameKey(canonicalPath, "0xzz", "next"); err == nil {
			t.Fatal("expected invalid old key rename error")
		}
		if err := service.RenameKey(canonicalPath, "gamma", "0xzz"); err == nil {
			t.Fatal("expected invalid new key rename error")
		}
	})

}

func TestLevelDBServiceWriteMethodsRequireOpenDatabase(t *testing.T) {
	service := &LevelDBService{}
	absPath, err := filepath.Abs("tmp/not-open")
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}

	if err := service.PutValue(absPath, "k", "v"); err == nil {
		t.Fatal("expected put to fail when db is not open")
	}
	if _, err := service.PutValueIfUnchanged(absPath, "k", "v", "v2", false); err == nil {
		t.Fatal("expected guarded put to fail when db is not open")
	}
	if err := service.DeleteKey(absPath, "k"); err == nil {
		t.Fatal("expected delete to fail when db is not open")
	}
	if err := service.RenameKey(absPath, "a", "b"); err == nil {
		t.Fatal("expected rename to fail when db is not open")
	}
}

func TestLevelDBServiceOpenDatabaseDedupesByCanonicalPath(t *testing.T) {
	db, dbPath, cleanup := createLevelDBHarness(t)
	t.Cleanup(cleanup)
	if err := db.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	service := &LevelDBService{}
	openA := service.OpenDatabase(dbPath, false)
	if !openA.Ok {
		t.Fatalf("open A failed: %s", openA.Error)
	}
	openB := service.OpenDatabase(filepath.Join(".", dbPath), false)
	if !openB.Ok {
		t.Fatalf("open B failed: %s", openB.Error)
	}
	if openB.CanonicalPath != openA.CanonicalPath {
		t.Fatalf("expected canonical paths to match: A=%q B=%q", openA.CanonicalPath, openB.CanonicalPath)
	}
	if !openB.AlreadyOpen {
		t.Fatal("second open of same path should return alreadyOpen=true")
	}

	if err := service.CloseDatabase(openA.CanonicalPath); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := service.PutValue(openA.CanonicalPath, "still-open", "yes"); err != nil {
		t.Fatalf("db should remain open while refcount > 0: %v", err)
	}

	if err := service.CloseDatabase(openA.CanonicalPath); err != nil {
		t.Fatalf("second close: %v", err)
	}
	if err := service.PutValue(openA.CanonicalPath, "should-fail", "no"); err == nil {
		t.Fatal("write should fail after final close")
	}
}

func TestLevelDBServiceCloseDatabaseRemainsSafeAfterLeaseRebalance(t *testing.T) {
	db, dbPath, cleanup := createLevelDBHarness(t)
	t.Cleanup(cleanup)
	if err := db.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	service := &LevelDBService{}
	openA := service.OpenDatabase(dbPath, false)
	if !openA.Ok {
		t.Fatalf("open A failed: %s", openA.Error)
	}
	openB := service.OpenDatabase(filepath.Join(".", dbPath), false)
	if !openB.Ok {
		t.Fatalf("open B failed: %s", openB.Error)
	}
	if !openB.AlreadyOpen {
		t.Fatal("second open should reuse existing canonical entry")
	}

	// Simulate frontend lease rebalance when an already-open tab is focused again.
	if err := service.CloseDatabase(openA.CanonicalPath); err != nil {
		t.Fatalf("lease rebalance close failed: %v", err)
	}
	if err := service.PutValue(openA.CanonicalPath, "still-open-after-rebalance", "yes"); err != nil {
		t.Fatalf("db should remain open after lease rebalance: %v", err)
	}

	// Simulate tab close; this should close the final lease.
	if err := service.CloseDatabase(openA.CanonicalPath); err != nil {
		t.Fatalf("tab close failed: %v", err)
	}
	if err := service.PutValue(openA.CanonicalPath, "should-fail", "no"); err == nil {
		t.Fatal("write should fail after releasing final lease")
	}

	// Additional closes should stay safe no-op cleanup operations.
	if err := service.CloseDatabase(openA.CanonicalPath); err != nil {
		t.Fatalf("extra close should be a safe no-op: %v", err)
	}
}

func TestLevelDBServiceSupportsMultipleOpenDatabases(t *testing.T) {
	dbA, pathA, cleanupA := createLevelDBHarness(t)
	t.Cleanup(cleanupA)
	dbB, pathB, cleanupB := createLevelDBHarness(t)
	t.Cleanup(cleanupB)

	if err := dbA.Put([]byte("k"), []byte("a"), nil); err != nil {
		t.Fatalf("seed A: %v", err)
	}
	if err := dbB.Put([]byte("k"), []byte("b"), nil); err != nil {
		t.Fatalf("seed B: %v", err)
	}
	if err := dbA.Close(); err != nil {
		t.Fatalf("close A seed db: %v", err)
	}
	if err := dbB.Close(); err != nil {
		t.Fatalf("close B seed db: %v", err)
	}

	service := &LevelDBService{}
	openA := service.OpenDatabase(pathA, false)
	openB := service.OpenDatabase(pathB, false)
	if !openA.Ok || !openB.Ok {
		t.Fatalf("open failed: A=%+v B=%+v", openA, openB)
	}
	t.Cleanup(func() {
		_ = service.CloseDatabase(openA.CanonicalPath)
		_ = service.CloseDatabase(openB.CanonicalPath)
	})

	valueA, err := service.GetValue(openA.CanonicalPath, "k")
	if err != nil {
		t.Fatalf("read A: %v", err)
	}
	valueB, err := service.GetValue(openB.CanonicalPath, "k")
	if err != nil {
		t.Fatalf("read B: %v", err)
	}
	if valueA != "a" || valueB != "b" {
		t.Fatalf("values should stay isolated: A=%q B=%q", valueA, valueB)
	}

	if err := service.PutValue(openA.CanonicalPath, "x", "1"); err != nil {
		t.Fatalf("A writes should work: %v", err)
	}
	if err := service.PutValue(openB.CanonicalPath, "x", "2"); err != nil {
		t.Fatalf("B writes should still work: %v", err)
	}
}

func TestLevelDBServiceOpenDatabaseIntentionalReadOnlyMode(t *testing.T) {
	db, dbPath, cleanup := createLevelDBHarness(t)
	t.Cleanup(cleanup)
	if err := db.Put([]byte("alpha"), []byte("one"), nil); err != nil {
		t.Fatalf("seed alpha: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	service := &LevelDBService{}
	result := service.OpenDatabase(dbPath, true)
	if !result.Ok {
		t.Fatalf("intentional read-only open failed: %s", result.Error)
	}
	if !result.ReadOnly || !result.ForcedReadOnly || !result.IntentionalReadOnly {
		t.Fatalf("expected intentional read-only open result, got %+v", result)
	}
	if result.ReadOnlyReason == "" {
		t.Fatalf("expected readOnlyReason to be populated, got %+v", result)
	}

	if _, err := service.GetValue(result.CanonicalPath, "alpha"); err != nil {
		t.Fatalf("reads should work in intentional read-only mode: %v", err)
	}
	if err := service.PutValue(result.CanonicalPath, "blocked", "value"); !errors.Is(err, errDatabaseForcedReadOnly) {
		t.Fatalf("writes should be blocked in intentional read-only mode: %v", err)
	}
	if err := service.CloseDatabase(result.CanonicalPath); err != nil {
		t.Fatalf("close intentional read-only handle: %v", err)
	}
}

func TestLevelDBServiceOpenDatabaseFallsBackToReadOnlyWhenExternallyLocked(t *testing.T) {
	db, dbPath, cleanup := createLevelDBHarness(t)
	t.Cleanup(cleanup)
	if err := db.Put([]byte("alpha"), []byte("one"), nil); err != nil {
		t.Fatalf("seed alpha: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	lockedHandle, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		t.Fatalf("open locked handle: %v", err)
	}
	defer func() {
		if closeErr := lockedHandle.Close(); closeErr != nil {
			t.Errorf("close locked handle: %v", closeErr)
		}
	}()

	service := &LevelDBService{}
	result := service.OpenDatabase(dbPath, false)
	if !result.Ok {
		t.Fatalf("read-only fallback open failed: %s", result.Error)
	}
	if !result.ReadOnly {
		t.Fatalf("expected readOnly=true, got %+v", result)
	}
	if !result.ForcedReadOnly {
		t.Fatalf("expected forcedReadOnly=true, got %+v", result)
	}
	if result.ReadOnlyReason == "" {
		t.Fatalf("expected readOnlyReason to be populated, got %+v", result)
	}

	if _, err := service.GetKeys(result.CanonicalPath); err != nil {
		t.Fatalf("reads should work in forced read-only mode: %v", err)
	}
	if err := service.PutValue(result.CanonicalPath, "blocked", "value"); !errors.Is(err, errDatabaseForcedReadOnly) {
		t.Fatalf("writes should be blocked by forced read-only mode: %v", err)
	}
	if err := service.CloseDatabase(result.CanonicalPath); err != nil {
		t.Fatalf("close forced read-only handle: %v", err)
	}
}

func TestLevelDBServiceRefreshDatabaseReloadsForcedReadOnlyHandle(t *testing.T) {
	db, dbPath, cleanup := createLevelDBHarness(t)
	t.Cleanup(cleanup)
	if err := db.Put([]byte("alpha"), []byte("one"), nil); err != nil {
		t.Fatalf("seed alpha: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	lockedHandle, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		t.Fatalf("open locked handle: %v", err)
	}
	defer func() {
		if closeErr := lockedHandle.Close(); closeErr != nil {
			t.Errorf("close locked handle: %v", closeErr)
		}
	}()

	service := &LevelDBService{}
	openResult := service.OpenDatabase(dbPath, false)
	if !openResult.Ok {
		t.Fatalf("read-only fallback open failed: %s", openResult.Error)
	}
	if !openResult.ForcedReadOnly {
		t.Fatalf("expected forced read-only open, got: %+v", openResult)
	}
	t.Cleanup(func() {
		_ = service.CloseDatabase(openResult.CanonicalPath)
	})

	valueBeforeRefresh, err := service.GetValue(openResult.CanonicalPath, "alpha")
	if err != nil {
		t.Fatalf("get value before refresh: %v", err)
	}
	if valueBeforeRefresh != "one" {
		t.Fatalf("unexpected initial value: %q", valueBeforeRefresh)
	}

	if err := lockedHandle.Put([]byte("alpha"), []byte("two"), nil); err != nil {
		t.Fatalf("update alpha via external writer: %v", err)
	}
	if err := lockedHandle.Put([]byte("beta"), []byte("added"), nil); err != nil {
		t.Fatalf("add beta via external writer: %v", err)
	}

	// Existing long-lived read-only handles stay stale until explicitly reopened.
	staleValue, err := service.GetValue(openResult.CanonicalPath, "alpha")
	if err != nil {
		t.Fatalf("get stale value before refresh: %v", err)
	}
	if staleValue != "one" {
		t.Fatalf("expected stale value before refresh, got %q", staleValue)
	}
	staleKeys, err := service.GetKeys(openResult.CanonicalPath)
	if err != nil {
		t.Fatalf("get stale keys before refresh: %v", err)
	}
	for _, key := range staleKeys {
		if key == "beta" {
			t.Fatalf("expected key list to remain stale before refresh, got keys: %v", staleKeys)
		}
	}

	refreshResult, err := service.RefreshDatabase(openResult.CanonicalPath)
	if err != nil {
		t.Fatalf("refresh database: %v", err)
	}
	if !refreshResult.Ok {
		t.Fatalf("refresh returned not ok result: %+v", refreshResult)
	}
	if !refreshResult.ForcedReadOnly {
		t.Fatalf("refresh should remain forced read-only while lock is held: %+v", refreshResult)
	}

	valueAfterRefresh, err := service.GetValue(openResult.CanonicalPath, "alpha")
	if err != nil {
		t.Fatalf("get value after refresh: %v", err)
	}
	if valueAfterRefresh != "two" {
		t.Fatalf("expected refreshed value, got %q", valueAfterRefresh)
	}

	keysAfterRefresh, err := service.GetKeys(openResult.CanonicalPath)
	if err != nil {
		t.Fatalf("get keys after refresh: %v", err)
	}
	foundBeta := false
	for _, key := range keysAfterRefresh {
		if key == "beta" {
			foundBeta = true
			break
		}
	}
	if !foundBeta {
		t.Fatalf("expected refreshed keys to include beta, got %v", keysAfterRefresh)
	}
}

func TestLevelDBServiceRefreshDatabaseFailsWhenReopenCannotSucceed(t *testing.T) {
	db, dbPath, cleanup := createLevelDBHarness(t)
	if err := db.Put([]byte("alpha"), []byte("one"), nil); err != nil {
		t.Fatalf("seed alpha: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	service := &LevelDBService{}
	openResult := service.OpenDatabase(dbPath, false)
	if !openResult.Ok {
		t.Fatalf("open database: %s", openResult.Error)
	}

	cleanup()
	if err := os.RemoveAll(dbPath); err != nil {
		t.Fatalf("remove database path: %v", err)
	}

	_, err := service.RefreshDatabase(openResult.CanonicalPath)
	if err == nil {
		t.Fatal("expected refresh to fail when database path has been removed")
	}

	if _, err := service.GetKeys(openResult.CanonicalPath); err == nil {
		t.Fatal("expected database to be closed after failed refresh reopen")
	}
}

func TestParseLsofProcessName(t *testing.T) {
	t.Run("returns first command when no pid is skipped", func(t *testing.T) {
		output := []byte("p123\ncCode\np456\ncOtherApp\n")
		got := parseLsofProcessName(output, 0)
		if got != "Code" {
			t.Fatalf("expected first command name, got %q", got)
		}
	})

	t.Run("skips own pid and returns next locker", func(t *testing.T) {
		output := []byte("p111\ncOurApp\np222\ncLevelDBBrowser\n")
		got := parseLsofProcessName(output, 111)
		if got != "LevelDBBrowser" {
			t.Fatalf("expected external command name, got %q", got)
		}
	})
}
