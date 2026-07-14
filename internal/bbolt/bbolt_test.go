package bbolt_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gnomatix/validatorium/internal/bbolt"
	"gnomatix/validatorium/internal/storage"
)

func TestBBoltStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bbolt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := bbolt.NewStore(dbPath, "testbucket")
	if err != nil {
		t.Fatalf("failed to create bbolt store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Retrieve non-existent key
	_, err = store.Get(ctx, "non-existent")
	if err != storage.ErrNotFound {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}

	// Put and get key
	key := "test-key"
	val := []byte("test-value")
	err = store.Put(ctx, key, val)
	if err != nil {
		t.Fatalf("failed to put key: %v", err)
	}

	got, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("failed to get key: %v", err)
	}

	if string(got) != string(val) {
		t.Errorf("expected %q, got %q", val, got)
	}

	// Delete key
	err = store.Delete(ctx, key)
	if err != nil {
		t.Fatalf("failed to delete key: %v", err)
	}

	_, err = store.Get(ctx, key)
	if err != storage.ErrNotFound {
		t.Errorf("expected ErrNotFound after deletion, got: %v", err)
	}
}
