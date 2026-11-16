package bbolt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/namespace"
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
//	namespaces            namespace + 0x00 + UUID -> JSON NamespacedPointer
//	cert_versions         CID             -> canonical self-certification bytes (immutable)
//	certs                 cert UUID       -> CID of the current self-certification version
//	certs_by_subject      subject + 0x00 + cert UUID -> empty (index)
//	identity_versions     CID             -> canonical identity bytes (immutable)
//	identities            fingerprint     -> CID of the current identity version
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
	bucketNamespaces      = []byte("namespaces")
	bucketCertVersions    = []byte("cert_versions")
	bucketCerts           = []byte("certs")
	bucketCertsBySubject  = []byte("certs_by_subject")
	bucketIdentityVers    = []byte("identity_versions")
	bucketIdentities      = []byte("identities")
	bucketSignatures      = []byte("signatures")
	bucketSigsBySigner    = []byte("signatures_by_signer")
	bucketSigsBySubject   = []byte("signatures_by_subject")
	workingStoreBuckets   = [][]byte{bucketCandidates, bucketVersions, bucketCurrent, bucketAttestations, bucketAttestIdx, bucketTermVersions, bucketTerms, bucketNamespaces, bucketCertVersions, bucketCerts, bucketCertsBySubject, bucketIdentityVers, bucketIdentities, bucketSignatures, bucketSigsBySigner, bucketSigsBySubject}
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

func (w *WorkingStore) PutNamespacedPointer(ctx context.Context, p storage.NamespacedPointer) error {
	if _, err := namespace.Parse(string(p.Namespace)); err != nil {
		return err
	}
	if p.UUID == "" || p.CID == "" {
		return errors.New("working store: namespaced pointer needs UUID and CID")
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return w.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketNamespaces).Put(namespaceKey(p.Namespace, p.UUID), data)
	})
}

func (w *WorkingStore) GetNamespacedPointer(ctx context.Context, ns namespace.Name, uuid string) (storage.NamespacedPointer, error) {
	data, err := w.get(bucketNamespaces, namespaceKey(ns, uuid))
	if err != nil {
		return storage.NamespacedPointer{}, err
	}
	var p storage.NamespacedPointer
	if err := json.Unmarshal(data, &p); err != nil {
		return storage.NamespacedPointer{}, fmt.Errorf("working store: decode namespaced pointer: %w", err)
	}
	return p, nil
}

func (w *WorkingStore) ListNamespace(ctx context.Context, ns namespace.Name) ([]storage.NamespacedPointer, error) {
	var out []storage.NamespacedPointer
	prefix := namespaceKey(ns, "")
	err := w.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(bucketNamespaces).Cursor()
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			var p storage.NamespacedPointer
			if err := json.Unmarshal(v, &p); err != nil {
				return fmt.Errorf("working store: decode namespaced pointer %s: %w", k, err)
			}
			out = append(out, p)
		}
		return nil
	})
	return out, err
}

func (w *WorkingStore) DeleteNamespacedPointer(ctx context.Context, ns namespace.Name, uuid string) error {
	return w.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketNamespaces).Delete(namespaceKey(ns, uuid))
	})
}

// namespaceKey orders the namespaces bucket by namespace, then UUID. The 0x00
// delimiter cannot occur in a namespace name, so a prefix scan over one
// namespace never matches another namespace that shares a textual prefix.
func namespaceKey(ns namespace.Name, uuid string) []byte {
	k := make([]byte, 0, len(ns)+1+len(uuid))
	k = append(k, ns...)
	k = append(k, attestationIndexDelim)
	k = append(k, uuid...)
	return k
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

// ListCurrent returns the UUIDs of every premise with a current published
// version, in key order.
func (w *WorkingStore) ListCurrent(ctx context.Context) ([]string, error) {
	var out []string
	err := w.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketCurrent).ForEach(func(k, _ []byte) error {
			out = append(out, string(k))
			return nil
		})
	})
	return out, err
}
