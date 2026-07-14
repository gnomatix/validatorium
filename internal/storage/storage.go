package storage

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("not found")
)

// Store defines the boundary storage abstraction for Validatorium.
type Store interface {
	Put(ctx context.Context, key string, val []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	Close() error
}
