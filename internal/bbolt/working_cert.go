package bbolt

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"

	bolt "go.etcd.io/bbolt"
)

// PutCertification stores an immutable version of a self-certification,
// points its UUID at that version, and indexes it by subject. Earlier
// versions stay addressable by CID.
func (w *WorkingStore) PutCertification(ctx context.Context, c *model.SelfCertification) (storage.VersionRef, error) {
	if c == nil {
		return storage.VersionRef{}, errors.New("working store: nil certification")
	}
	if err := c.ValidateRecord(); err != nil {
		return storage.VersionRef{}, err
	}
	data, err := c.CanonicalBytes()
	if err != nil {
		return storage.VersionRef{}, err
	}
	id, err := storage.ContentID(data)
	if err != nil {
		return storage.VersionRef{}, err
	}
	err = w.db.Update(func(tx *bolt.Tx) error {
		if err := putImmutable(tx.Bucket(bucketCertVersions), []byte(id), data); err != nil {
			return err
		}
		if err := tx.Bucket(bucketCerts).Put([]byte(c.UUID), []byte(id)); err != nil {
			return err
		}
		return tx.Bucket(bucketCertsBySubject).Put(namespaceKeyRaw(c.Subject, c.UUID), []byte{})
	})
	if err != nil {
		return storage.VersionRef{}, err
	}
	return storage.VersionRef{UUID: c.UUID, CID: id}, nil
}

// GetCertification returns the current version of a self-certification.
func (w *WorkingStore) GetCertification(ctx context.Context, uuid string) (*model.SelfCertification, storage.VersionRef, error) {
	var data []byte
	var cid string
	err := w.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketCerts).Get([]byte(uuid))
		if c == nil {
			return storage.ErrNotFound
		}
		cid = string(c)
		v := tx.Bucket(bucketCertVersions).Get(c)
		if v == nil {
			return fmt.Errorf("working store: certification %s points at missing version %s", uuid, cid)
		}
		data = append([]byte(nil), v...)
		return nil
	})
	if err != nil {
		return nil, storage.VersionRef{}, err
	}
	rec, err := model.SelfCertificationFromBytes(data)
	return rec, storage.VersionRef{UUID: uuid, CID: cid}, err
}

// GetCertificationVersion returns one stored version by CID.
func (w *WorkingStore) GetCertificationVersion(ctx context.Context, cid string) (*model.SelfCertification, error) {
	data, err := w.get(bucketCertVersions, []byte(cid))
	if err != nil {
		return nil, err
	}
	if err := storage.VerifyContentID(cid, data); err != nil {
		return nil, fmt.Errorf("working store: stored certification corrupt: %w", err)
	}
	return model.SelfCertificationFromBytes(data)
}

// CertificationsForSubject returns the current version of every
// self-certification recorded for a subject URL, current and obsolete alike,
// in UUID order.
func (w *WorkingStore) CertificationsForSubject(ctx context.Context, subject string) ([]*model.SelfCertification, error) {
	var ids []string
	prefix := namespaceKeyRaw(subject, "")
	err := w.db.View(func(tx *bolt.Tx) error {
		cur := tx.Bucket(bucketCertsBySubject).Cursor()
		for k, _ := cur.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = cur.Next() {
			ids = append(ids, string(k[len(prefix):]))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]*model.SelfCertification, 0, len(ids))
	for _, id := range ids {
		c, _, err := w.GetCertification(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// namespaceKeyRaw joins two strings with the 0x00 delimiter used by the
// store's prefix indexes.
func namespaceKeyRaw(a, b string) []byte {
	k := make([]byte, 0, len(a)+1+len(b))
	k = append(k, a...)
	k = append(k, attestationIndexDelim)
	k = append(k, b...)
	return k
}
