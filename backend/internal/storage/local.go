package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage stores blobs as files under a base directory.
type LocalStorage struct{ dir string }

// NewLocal ensures the base directory exists and returns a filesystem-backed Storage.
func NewLocal(dir string) (*LocalStorage, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	return &LocalStorage{dir: dir}, nil
}

// path resolves a key to a file path. filepath.Base strips any stray separators —
// keys are server-generated UUIDs, but this is defence in depth against traversal.
func (s *LocalStorage) path(key string) string {
	return filepath.Join(s.dir, filepath.Base(key))
}

func (s *LocalStorage) Save(_ context.Context, key string, r io.Reader) (int64, error) {
	f, err := os.Create(s.path(key))
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(f, r)
	// Capture the close error too: for writes it signals unflushed data.
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return n, err
}

func (s *LocalStorage) Open(_ context.Context, key string) (io.ReadCloser, error) {
	return os.Open(s.path(key))
}
