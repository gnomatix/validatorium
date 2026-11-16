package bbolt

import (
	"bytes"
	"context"
	"errors"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"

	bolt "go.etcd.io/bbolt"
)

// PutSignature stores an immutable signature record under its own content
// identifier and indexes it by signer and by signed version. It returns the
// record's CID.
func (w *WorkingStore) PutSignature(ctx context.Context, s *model.Signature) (string, error) {
	if s == nil {
		return "", errors.New("working store: nil signature")
	}
	data, err := s.CanonicalBytes()
	if err != nil {
		return "", err
	}
	id, err := storage.ContentID(data)
	if err != nil {
		return "", err
	}
	err = w.db.Update(func(tx *bolt.Tx) error {
		if err := putImmutable(tx.Bucket(bucketSignatures), []byte(id), data); err != nil {
			return err
		}
		if err := tx.Bucket(bucketSigsBySigner).Put(pairKey(s.Signer, id), nil); err != nil {
			return err
		}
		return tx.Bucket(bucketSigsBySubject).Put(pairKey(s.SubjectCID, id), nil)
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

// SignaturesBySigner returns every signature record made with the key whose
// fingerprint is given, in CID order. It is the remediation query: the
// subjects of these records are every version that key signed.
func (w *WorkingStore) SignaturesBySigner(ctx context.Context, fingerprint string) ([]*model.Signature, error) {
	return w.signaturesByIndex(bucketSigsBySigner, fingerprint)
}

// SignaturesForSubject returns every signature over the version with this CID.
func (w *WorkingStore) SignaturesForSubject(ctx context.Context, subjectCID string) ([]*model.Signature, error) {
	return w.signaturesByIndex(bucketSigsBySubject, subjectCID)
}

func (w *WorkingStore) signaturesByIndex(index []byte, key string) ([]*model.Signature, error) {
	var out []*model.Signature
	prefix := append([]byte(key), 0)
	err := w.db.View(func(tx *bolt.Tx) error {
		recs := tx.Bucket(bucketSignatures)
		c := tx.Bucket(index).Cursor()
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			data := recs.Get(k[len(prefix):])
			if data == nil {
				return storage.ErrNotFound
			}
			s, err := model.SignatureFromBytes(data)
			if err != nil {
				return err
			}
			out = append(out, s)
		}
		return nil
	})
	return out, err
}

func pairKey(a, b string) []byte {
	k := make([]byte, 0, len(a)+1+len(b))
	k = append(k, a...)
	k = append(k, 0)
	return append(k, b...)
}
