// Package storage abstracts binary document storage. A local-filesystem backend is
// used in dev; Phase 12 swaps in Azure Blob behind the same interface.
package storage

import (
	"context"
	"io"
)

// Storage persists and retrieves document blobs by an opaque, server-generated key.
type Storage interface {
	Save(ctx context.Context, key string, r io.Reader) (int64, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
}
