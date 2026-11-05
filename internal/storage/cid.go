package storage

import (
	"fmt"

	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

// ContentID returns the content identifier for a stored version: a CIDv1 with
// the raw codec and a SHA2-256 multihash, rendered in its default base32 form.
//
// This is exactly the CID an IPFS node assigns to the same bytes when they are
// put as a raw block with SHA2-256, so a locally computed version reference
// resolves to the same block once published.
func ContentID(data []byte) (string, error) {
	h, err := mh.Sum(data, mh.SHA2_256, -1)
	if err != nil {
		return "", fmt.Errorf("hash content: %w", err)
	}
	return cid.NewCidV1(cid.Raw, h).String(), nil
}

// VerifyContentID reports whether data hashes to the given CID string.
func VerifyContentID(id string, data []byte) error {
	want, err := cid.Decode(id)
	if err != nil {
		return fmt.Errorf("decode cid %q: %w", id, err)
	}
	h, err := mh.Sum(data, mh.SHA2_256, -1)
	if err != nil {
		return fmt.Errorf("hash content: %w", err)
	}
	got := cid.NewCidV1(cid.Raw, h)
	if !want.Equals(got) {
		return fmt.Errorf("content does not match cid %s (hashes to %s)", id, got)
	}
	return nil
}
