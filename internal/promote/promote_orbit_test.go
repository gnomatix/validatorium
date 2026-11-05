package promote

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"berty.tech/go-orbit-db/stores"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"

	"gnomatix/validatorium/internal/bbolt"
	"gnomatix/validatorium/internal/ipfsnode"
	"gnomatix/validatorium/internal/model"
	"gnomatix/validatorium/internal/orbit"
	"gnomatix/validatorium/internal/storage"
)

// TestPromoteThroughOrbitReplicatesToPeer exercises the full seam: a candidate
// in a bbolt working store on node A is promoted under a true attestation into
// the Kubo/go-orbit-db global store, and node B, which only opened the same
// pointer store by address, obtains the pointer, the version block and the
// attestation block. Writing to the global store is the publication step;
// there is no separate push.
func TestPromoteThroughOrbitReplicatesToPeer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	mn := mocknet.New()
	defer mn.Close()
	nodeA, err := ipfsnode.NewMockNode(ctx, mn)
	if err != nil {
		t.Fatalf("node A: %v", err)
	}
	defer nodeA.Close()
	nodeB, err := ipfsnode.NewMockNode(ctx, mn)
	if err != nil {
		t.Fatalf("node B: %v", err)
	}
	defer nodeB.Close()
	if _, err := mn.LinkPeers(nodeA.PeerID(), nodeB.PeerID()); err != nil {
		t.Fatalf("link: %v", err)
	}
	if _, err := mn.ConnectPeers(nodeA.PeerID(), nodeB.PeerID()); err != nil {
		t.Fatalf("connect: %v", err)
	}

	gsA, err := orbit.Open(ctx, nodeA, filepath.Join(t.TempDir(), "orbitA"), "promotion-e2e", nil)
	if err != nil {
		t.Fatalf("open global store A: %v", err)
	}
	defer gsA.Close()
	ws, err := bbolt.OpenWorkingStore(filepath.Join(t.TempDir(), "working.db"))
	if err != nil {
		t.Fatalf("open working store: %v", err)
	}
	defer ws.Close()

	cand, _ := model.New("The normal human diploid genome contains 46 chromosomes.", model.Attribution{Agent: "key:abc", AtTime: "2026-07-14T00:00:00Z"})
	cand.Claim.Terms = []model.TermGrounding{{SurfaceForm: "chromosomes", OntologyURI: "http://purl.obolibrary.org/obo/GO_0005694"}}
	cand.SupportGraph = []model.Evidence{{Kind: model.EvidenceData, Reference: "doi:10.1111/j.1601-5223.1956.tb03010.x"}}
	if err := ws.PutCandidate(ctx, cand); err != nil {
		t.Fatal(err)
	}
	data, _ := cand.CanonicalBytes()
	versionCID, _ := storage.ContentID(data)
	att, _ := model.NewAttestation(cand.UUID, versionCID, model.ResultTrue, "evaluator:test", "ipfs://method", time.Now())

	p := &Promoter{Working: ws, Global: gsA}
	res, err := p.Promote(ctx, cand.UUID, att)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}

	// Node B joins the pointer store by address and waits for the pointer.
	gsB, err := orbit.Open(ctx, nodeB, filepath.Join(t.TempDir(), "orbitB"), gsA.Address(), &orbit.Options{
		WriteAccess:       []string{gsA.Identity()},
		BlockFetchTimeout: 20 * time.Second,
	})
	if err != nil {
		t.Fatalf("open global store B: %v", err)
	}
	defer gsB.Close()
	sub, err := gsB.Docs().EventBus().Subscribe(new(stores.EventReplicated))
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()

	deadline := time.After(30 * time.Second)
	var ptr storage.Pointer
	for {
		ptr, err = gsB.GetPointer(ctx, cand.UUID)
		if err == nil {
			break
		}
		if !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("B GetPointer: %v", err)
		}
		select {
		case <-sub.Out():
		case <-time.After(500 * time.Millisecond):
		case <-deadline:
			t.Fatal("timed out waiting for pointer replication")
		case <-ctx.Done():
			t.Fatal("context cancelled")
		}
	}
	if ptr.CID != res.Version.CID || ptr.AttestationCID != res.AttestationCID || ptr.Status != model.Current {
		t.Fatalf("replicated pointer = %+v, want %+v", ptr, res.Pointer)
	}

	// Node B fetches the exact version and attestation blocks by CID.
	vb, err := gsB.GetBlock(ctx, ptr.CID)
	if err != nil {
		t.Fatalf("B GetBlock version: %v", err)
	}
	got, err := model.MicropublicationFromBytes(vb)
	if err != nil || got.UUID != cand.UUID || got.Claim.Text != cand.Claim.Text {
		t.Fatalf("replicated version drift: %+v %v", got, err)
	}
	ab, err := gsB.GetBlock(ctx, ptr.AttestationCID)
	if err != nil {
		t.Fatalf("B GetBlock attestation: %v", err)
	}
	gotAtt, err := model.AttestationFromBytes(ab)
	if err != nil || gotAtt.PremiseVersionCID != ptr.CID || gotAtt.Result != model.ResultTrue {
		t.Fatalf("replicated attestation drift: %+v %v", gotAtt, err)
	}
}
