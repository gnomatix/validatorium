package bbolt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gnomatix/validatorium/internal/storage"

	bolt "go.etcd.io/bbolt"
)

type Store struct {
	db     *bolt.DB
	bucket []byte
}

var _ storage.Store = (*Store)(nil)

func NewStore(dbPath string, bucketName string) (*Store, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}

	bucket := []byte(bucketName)
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Store{
		db:     db,
		bucket: bucket,
	}, nil
}

func (s *Store) Put(ctx context.Context, key string, val []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		if b == nil {
			return fmt.Errorf("bucket not found")
		}
		return b.Put([]byte(key), val)
	})
}

func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	var val []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		if b == nil {
			return fmt.Errorf("bucket not found")
		}
		v := b.Get([]byte(key))
		if v == nil {
			return storage.ErrNotFound
		}
		// Copy value to prevent reference escape after tx
		val = make([]byte, len(v))
		copy(val, v)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		if b == nil {
			return fmt.Errorf("bucket not found")
		}
		return b.Delete([]byte(key))
	})
}

func (s *Store) Close() error {
	return s.db.Close()
}
