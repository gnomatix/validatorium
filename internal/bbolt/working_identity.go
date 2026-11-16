package bbolt

import (
	"context"
	"errors"
	"fmt"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"

	bolt "go.etcd.io/bbolt"
)

// PutIdentity stores an immutable version of an identity and points its
// fingerprint at that version. Earlier versions stay addressable by CID.
func (w *WorkingStore) PutIdentity(ctx context.Context, i *model.Identity) (storage.VersionRef, error) {
	if i == nil {
		return storage.VersionRef{}, errors.New("working store: nil identity")
	}
	if err := i.ValidateRecord(); err != nil {
		return storage.VersionRef{}, err
	}
	data, err := i.CanonicalBytes()
	if err != nil {
		return storage.VersionRef{}, err
	}
	id, err := storage.ContentID(data)
	if err != nil {
		return storage.VersionRef{}, err
	}
	err = w.db.Update(func(tx *bolt.Tx) error {
		if err := putImmutable(tx.Bucket(bucketIdentityVers), []byte(id), data); err != nil {
			return err
		}
		return tx.Bucket(bucketIdentities).Put([]byte(i.UUID), []byte(id))
	})
	if err != nil {
		return storage.VersionRef{}, err
	}
	return storage.VersionRef{UUID: i.UUID, CID: id}, nil
}

// GetIdentity returns the current version of the identity with this
// fingerprint, or storage.ErrNotFound.
func (w *WorkingStore) GetIdentity(ctx context.Context, fingerprint string) (*model.Identity, error) {
	var data []byte
	err := w.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketIdentities).Get([]byte(fingerprint))
		if c == nil {
			return storage.ErrNotFound
		}
		v := tx.Bucket(bucketIdentityVers).Get(c)
		if v == nil {
			return fmt.Errorf("working store: identity %s points at missing version %s", fingerprint, c)
		}
		data = append([]byte(nil), v...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return model.IdentityFromBytes(data)
}
