package bbolt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/storage"

	bolt "go.etcd.io/bbolt"
)

// Bucket layout of the tier-1 working store. All keys are UTF-8 strings.
//
//	candidates            UUID            -> canonical micropublication bytes (mutable draft)
//	versions              CID             -> canonical micropublication bytes (immutable)
//	current               UUID            -> CID of the current local version
//	attestations          attestation UUID -> canonical attestation bytes (immutable)
//	attestations_by_premise premiseUUID + 0x00 + attestation UUID -> CID of the attestation bytes
//	term_versions         CID             -> canonical term record bytes (immutable)
//	terms                 term UUID       -> CID of the current term record version
//
// Candidates are the only mutable record class. Versions, term versions and
// attestations are content-addressed or append-only; a write that would change
// stored bytes is refused with storage.ErrImmutable. The current and terms
// buckets hold pointers only.
var (
	bucketCandidates      = []byte("candidates")
	bucketVersions        = []byte("versions")
	bucketCurrent         = []byte("current")
	bucketAttestations    = []byte("attestations")
	bucketAttestIdx       = []byte("attestations_by_premise")
	bucketTermVersions    = []byte("term_versions")
	bucketTerms           = []byte("terms")
	workingStoreBuckets   = [][]byte{bucketCandidates, bucketVersions, bucketCurrent, bucketAttestations, bucketAttestIdx, bucketTermVersions, bucketTerms}
	attestationIndexDelim = byte(0)
)

// WorkingStore is the bbolt implementation of storage.WorkingStore. It sits
// beside the IPFS datastore, not underneath it, and supports full local
// operation with zero connectivity.
type WorkingStore struct {
	db *bolt.DB
}

var _ storage.WorkingStore = (*WorkingStore)(nil)

// OpenWorkingStore opens (creating if needed) the working store at dbPath.
func OpenWorkingStore(dbPath string) (*WorkingStore, error) {
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create working store directory: %w", err)
		}
	}
	db, err := bolt.Open(dbPath, 0o600, nil)
	if err != nil {
		return nil, fmt.Errorf("open working store: %w", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		for _, b := range workingStoreBuckets {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return fmt.Errorf("create bucket %s: %w", b, err)
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return &WorkingStore{db: db}, nil
}

// Close closes the underlying database.
func (w *WorkingStore) Close() error { return w.db.Close() }

func (w *WorkingStore) PutCandidate(ctx context.Context, m *model.Micropublication) error {
	if m == nil || m.UUID == "" {
		return errors.New("working store: candidate has no UUID")
	}
	data, err := m.CanonicalBytes()
	if err != nil {
		return err
	}
	return w.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketCandidates).Put([]byte(m.UUID), data)
	})
}

func (w *WorkingStore) GetCandidate(ctx context.Context, uuid string) (*model.Micropublication, error) {
	data, err := w.get(bucketCandidates, []byte(uuid))
	if err != nil {
		return nil, err
	}
	return model.MicropublicationFromBytes(data)
}

func (w *WorkingStore) ListCandidates(ctx context.Context) ([]string, error) {
	var ids []string
	err := w.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketCandidates).ForEach(func(k, _ []byte) error {
			ids = append(ids, string(k))
			return nil
		})
	})
	return ids, err
}

func (w *WorkingStore) DeleteCandidate(ctx context.Context, uuid string) error {
	return w.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketCandidates).Delete([]byte(uuid))
	})
}

func (w *WorkingStore) PutVersion(ctx context.Context, m *model.Micropublication) (storage.VersionRef, error) {
	if m == nil || m.UUID == "" {
		return storage.VersionRef{}, errors.New("working store: version has no UUID")
	}
	data, err := m.CanonicalBytes()
	if err != nil {
		return storage.VersionRef{}, err
	}
	id, err := storage.ContentID(data)
	if err != nil {
		return storage.VersionRef{}, err
	}
	err = w.db.Update(func(tx *bolt.Tx) error {
		return putImmutable(tx.Bucket(bucketVersions), []byte(id), data)
	})
	if err != nil {
		return storage.VersionRef{}, err
	}
	return storage.VersionRef{UUID: m.UUID, CID: id}, nil
}

func (w *WorkingStore) GetVersion(ctx context.Context, cid string) (*model.Micropublication, error) {
	data, err := w.get(bucketVersions, []byte(cid))
	if err != nil {
		return nil, err
	}
	if err := storage.VerifyContentID(cid, data); err != nil {
		return nil, fmt.Errorf("working store: stored version corrupt: %w", err)
	}
	return model.MicropublicationFromBytes(data)
}

func (w *WorkingStore) SetCurrent(ctx context.Context, ref storage.VersionRef) error {
	if ref.UUID == "" || ref.CID == "" {
		return errors.New("working store: current pointer needs UUID and CID")
	}
	return w.db.Update(func(tx *bolt.Tx) error {
		if tx.Bucket(bucketVersions).Get([]byte(ref.CID)) == nil {
			return fmt.Errorf("working store: set current %s -> %s: version %w", ref.UUID, ref.CID, storage.ErrNotFound)
		}
		return tx.Bucket(bucketCurrent).Put([]byte(ref.UUID), []byte(ref.CID))
	})
}

func (w *WorkingStore) GetCurrent(ctx context.Context, uuid string) (storage.VersionRef, error) {
	data, err := w.get(bucketCurrent, []byte(uuid))
	if err != nil {
		return storage.VersionRef{}, err
	}
	return storage.VersionRef{UUID: uuid, CID: string(data)}, nil
}

func (w *WorkingStore) PutAttestation(ctx context.Context, a *model.ValidityAttestation) error {
	if a == nil {
		return errors.New("working store: nil attestation")
	}
	if err := a.ValidateRecord(); err != nil {
		return err
	}
	data, err := a.CanonicalBytes()
	if err != nil {
		return err
	}
	id, err := storage.ContentID(data)
	if err != nil {
		return err
	}
	return w.db.Update(func(tx *bolt.Tx) error {
		if err := putImmutable(tx.Bucket(bucketAttestations), []byte(a.UUID), data); err != nil {
			return err
		}
		return tx.Bucket(bucketAttestIdx).Put(attestationIndexKey(a.PremiseUUID, a.UUID), []byte(id))
	})
}

func (w *WorkingStore) GetAttestation(ctx context.Context, uuid string) (*model.ValidityAttestation, error) {
	data, err := w.get(bucketAttestations, []byte(uuid))
	if err != nil {
		return nil, err
	}
	return model.AttestationFromBytes(data)
}

func (w *WorkingStore) ListAttestations(ctx context.Context, premiseUUID string) ([]*model.ValidityAttestation, error) {
	var out []*model.ValidityAttestation
	prefix := attestationIndexKey(premiseUUID, "")
	err := w.db.View(func(tx *bolt.Tx) error {
		atts := tx.Bucket(bucketAttestations)
		c := tx.Bucket(bucketAttestIdx).Cursor()
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			attUUID := k[len(prefix):]
			data := atts.Get(attUUID)
			if data == nil {
				return fmt.Errorf("working store: attestation index names missing record %s", attUUID)
			}
			a, err := model.AttestationFromBytes(data)
			if err != nil {
				return err
			}
			out = append(out, a)
		}
		return nil
	})
	return out, err
}

func (w *WorkingStore) PutTermRecord(ctx context.Context, t *model.TermRecord) (storage.VersionRef, error) {
	if t == nil {
		return storage.VersionRef{}, errors.New("working store: nil term record")
	}
	if err := t.ValidateRecord(); err != nil {
		return storage.VersionRef{}, err
	}
	data, err := t.CanonicalBytes()
	if err != nil {
		return storage.VersionRef{}, err
	}
	id, err := storage.ContentID(data)
	if err != nil {
		return storage.VersionRef{}, err
	}
	err = w.db.Update(func(tx *bolt.Tx) error {
		if err := putImmutable(tx.Bucket(bucketTermVersions), []byte(id), data); err != nil {
			return err
		}
		return tx.Bucket(bucketTerms).Put([]byte(t.UUID), []byte(id))
	})
	if err != nil {
		return storage.VersionRef{}, err
	}
	return storage.VersionRef{UUID: t.UUID, CID: id}, nil
}

func (w *WorkingStore) GetTermRecord(ctx context.Context, uuid string) (*model.TermRecord, error) {
	var data []byte
	err := w.db.View(func(tx *bolt.Tx) error {
		cid := tx.Bucket(bucketTerms).Get([]byte(uuid))
		if cid == nil {
			return storage.ErrNotFound
		}
		v := tx.Bucket(bucketTermVersions).Get(cid)
		if v == nil {
			return fmt.Errorf("working store: term %s points at missing version %s", uuid, cid)
		}
		data = make([]byte, len(v))
		copy(data, v)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return model.TermRecordFromBytes(data)
}

func (w *WorkingStore) GetTermRecordVersion(ctx context.Context, cid string) (*model.TermRecord, error) {
	data, err := w.get(bucketTermVersions, []byte(cid))
	if err != nil {
		return nil, err
	}
	if err := storage.VerifyContentID(cid, data); err != nil {
		return nil, fmt.Errorf("working store: stored term version corrupt: %w", err)
	}
	return model.TermRecordFromBytes(data)
}

func (w *WorkingStore) ResolveTerm(ctx context.Context, uuid string) ([]*model.TermRecord, error) {
	return model.ResolveTermReplacements(func(id string) (*model.TermRecord, error) {
		return w.GetTermRecord(ctx, id)
	}, uuid)
}

// get copies a value out of a bucket so it stays valid after the transaction.
func (w *WorkingStore) get(bucket, key []byte) ([]byte, error) {
	var val []byte
	err := w.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bucket).Get(key)
		if v == nil {
			return storage.ErrNotFound
		}
		val = make([]byte, len(v))
		copy(val, v)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return val, nil
}

// putImmutable writes key -> data unless the key already holds different
// bytes, in which case it returns storage.ErrImmutable. Writing identical
// bytes is a no-op.
func putImmutable(b *bolt.Bucket, key, data []byte) error {
	if existing := b.Get(key); existing != nil {
		if bytes.Equal(existing, data) {
			return nil
		}
		return fmt.Errorf("%w: %s", storage.ErrImmutable, key)
	}
	return b.Put(key, data)
}

func attestationIndexKey(premiseUUID, attestationUUID string) []byte {
	k := make([]byte, 0, len(premiseUUID)+1+len(attestationUUID))
	k = append(k, premiseUUID...)
	k = append(k, attestationIndexDelim)
	k = append(k, attestationUUID...)
	return k
}
