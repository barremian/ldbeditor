package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

var errNoLockReadOnlyStorage = errors.New("leveldb/storage: storage is read-only")

// noLockReadOnlyStorage provides read-only file access without taking a LOCK file flock.
// This allows inspection of databases that are exclusively locked by another process.
type noLockReadOnlyStorage struct {
	mu     sync.Mutex
	path   string
	closed bool
}

type noLockReadOnlyLocker struct {
	fs *noLockReadOnlyStorage
}

func (l *noLockReadOnlyLocker) Unlock() {
	if l.fs == nil {
		return
	}
	_ = l.fs.Close()
}

func openNoLockReadOnlyStorage(path string) (storage.Storage, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("leveldb/storage: open %s: not a directory", path)
	}
	return &noLockReadOnlyStorage{path: path}, nil
}

func (fs *noLockReadOnlyStorage) Lock() (storage.Locker, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.closed {
		return nil, storage.ErrClosed
	}
	return &noLockReadOnlyLocker{fs: fs}, nil
}

func (fs *noLockReadOnlyStorage) Log(string) {}

func (fs *noLockReadOnlyStorage) SetMeta(storage.FileDesc) error {
	return errNoLockReadOnlyStorage
}

func (fs *noLockReadOnlyStorage) GetMeta() (storage.FileDesc, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.closed {
		return storage.FileDesc{}, storage.ErrClosed
	}

	tryCurrent := func(name string) (storage.FileDesc, error) {
		content, err := os.ReadFile(filepath.Join(fs.path, name))
		if err != nil {
			return storage.FileDesc{}, err
		}
		if len(content) < 1 || content[len(content)-1] != '\n' {
			return storage.FileDesc{}, &storage.ErrCorrupted{
				Fd:  storage.FileDesc{Type: storage.TypeManifest},
				Err: errors.New("leveldb/storage: corrupted or incomplete CURRENT file"),
			}
		}
		fd, ok := parseStorageFileDesc(strings.TrimSuffix(string(content), "\n"))
		if !ok {
			return storage.FileDesc{}, &storage.ErrCorrupted{
				Fd:  storage.FileDesc{Type: storage.TypeManifest},
				Err: errors.New("leveldb/storage: corrupted or incomplete CURRENT file"),
			}
		}
		if _, err := os.Stat(filepath.Join(fs.path, fd.String())); err != nil {
			return storage.FileDesc{}, err
		}
		return fd, nil
	}

	if fd, err := tryCurrent("CURRENT"); err == nil {
		return fd, nil
	}
	return tryCurrent("CURRENT.bak")
}

func (fs *noLockReadOnlyStorage) List(ft storage.FileType) ([]storage.FileDesc, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.closed {
		return nil, storage.ErrClosed
	}

	entries, err := os.ReadDir(fs.path)
	if err != nil {
		return nil, err
	}
	out := make([]storage.FileDesc, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fd, ok := parseStorageFileDesc(entry.Name())
		if !ok {
			continue
		}
		if fd.Type&ft != 0 {
			out = append(out, fd)
		}
	}
	return out, nil
}

func (fs *noLockReadOnlyStorage) Open(fd storage.FileDesc) (storage.Reader, error) {
	if !storage.FileDescOk(fd) {
		return nil, storage.ErrInvalidFile
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.closed {
		return nil, storage.ErrClosed
	}

	reader, err := os.Open(filepath.Join(fs.path, fd.String()))
	if err != nil && fd.Type == storage.TypeTable && os.IsNotExist(err) {
		// Backward compatibility for old table extension.
		reader, err = os.Open(filepath.Join(fs.path, fmt.Sprintf("%06d.sst", fd.Num)))
	}
	if err != nil {
		return nil, err
	}
	return reader, nil
}

func (fs *noLockReadOnlyStorage) Create(storage.FileDesc) (storage.Writer, error) {
	return nil, errNoLockReadOnlyStorage
}

func (fs *noLockReadOnlyStorage) Remove(storage.FileDesc) error {
	return errNoLockReadOnlyStorage
}

func (fs *noLockReadOnlyStorage) Rename(storage.FileDesc, storage.FileDesc) error {
	return errNoLockReadOnlyStorage
}

func (fs *noLockReadOnlyStorage) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.closed {
		return storage.ErrClosed
	}
	fs.closed = true
	return nil
}

func parseStorageFileDesc(name string) (storage.FileDesc, bool) {
	if strings.HasPrefix(name, "MANIFEST-") {
		numText := strings.TrimPrefix(name, "MANIFEST-")
		num, err := strconv.ParseInt(numText, 10, 64)
		if err == nil && num >= 0 {
			return storage.FileDesc{Type: storage.TypeManifest, Num: num}, true
		}
		return storage.FileDesc{}, false
	}

	dot := strings.LastIndexByte(name, '.')
	if dot <= 0 || dot == len(name)-1 {
		return storage.FileDesc{}, false
	}
	num, err := strconv.ParseInt(name[:dot], 10, 64)
	if err != nil || num < 0 {
		return storage.FileDesc{}, false
	}

	switch name[dot+1:] {
	case "log":
		return storage.FileDesc{Type: storage.TypeJournal, Num: num}, true
	case "ldb", "sst":
		return storage.FileDesc{Type: storage.TypeTable, Num: num}, true
	case "tmp":
		return storage.FileDesc{Type: storage.TypeTemp, Num: num}, true
	default:
		return storage.FileDesc{}, false
	}
}

func openLevelDBReadOnlyNoLock(path string) (*leveldb.DB, error) {
	stor, err := openNoLockReadOnlyStorage(path)
	if err != nil {
		return nil, err
	}
	db, err := leveldb.Open(stor, &opt.Options{ReadOnly: true})
	if err != nil {
		_ = stor.Close()
		return nil, err
	}
	return db, nil
}
