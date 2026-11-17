package promote

import (
	"bytes"
	"context"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"

	"gnomatix/validatorium/internal/identity"
	"gnomatix/validatorium/internal/model"
)

func testSigner(t *testing.T) (*identity.Signer, string) {
	t.Helper()
	e, err := openpgp.NewEntity("signer", "", "signer@example.org", nil)
	if err != nil {
		t.Fatal(err)
	}
	var priv, pub bytes.Buffer
	w, _ := armor.Encode(&priv, openpgp.PrivateKeyType, nil)
	if err := e.SerializePrivate(w, nil); err != nil {
		t.Fatal(err)
	}
	w.Close()
	w, _ = armor.Encode(&pub, openpgp.PublicKeyType, nil)
	_ = e.Serialize(w)
	w.Close()
	s, err := identity.NewSigner(priv.String())
	if err != nil {
		t.Fatal(err)
	}
	return s, pub.String()
}

func TestPromoteSignsPublishedVersion(t *testing.T) {
	f := setup(t)
	signer, pubKey := testSigner(t)
	f.p.Signer = signer
	res, err := f.p.Promote(context.Background(), f.cand.UUID, f.attestation(t, model.ResultTrue))
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if res.SignatureID == "" {
		t.Fatal("no signature recorded")
	}
	sigs, err := f.ws.SignaturesBySigner(context.Background(), signer.Fingerprint())
	if err != nil || len(sigs) != 1 {
		t.Fatalf("signatures by signer: %d, %v", len(sigs), err)
	}
	s := sigs[0]
	if s.SubjectCID != f.cid || s.Action != ActionPromote {
		t.Fatalf("signature record: %+v", s)
	}
	data, _ := f.cand.CanonicalBytes()
	fp, err := identity.VerifyDetached(pubKey, string(data), s.ArmoredSignature)
	if err != nil || fp != signer.Fingerprint() {
		t.Fatalf("verify: %q, %v", fp, err)
	}
	if _, err := identity.VerifyDetached(pubKey, string(data)+"x", s.ArmoredSignature); err == nil {
		t.Fatal("signature verified over different bytes")
	}
}

func TestPromoteWithoutSignerRecordsNoSignature(t *testing.T) {
	f := setup(t)
	res, err := f.p.Promote(context.Background(), f.cand.UUID, f.attestation(t, model.ResultTrue))
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if res.SignatureID != "" {
		t.Fatalf("unexpected signature %s", res.SignatureID)
	}
}

func TestPromoteRefusalWithSignerWritesNothing(t *testing.T) {
	f := setup(t)
	signer, _ := testSigner(t)
	f.p.Signer = signer
	if _, err := f.p.Promote(context.Background(), f.cand.UUID, f.attestation(t, model.ResultFalse)); err == nil {
		t.Fatal("false attestation promoted")
	}
	f.assertUnchanged(t)
	if sigs, _ := f.ws.SignaturesBySigner(context.Background(), signer.Fingerprint()); len(sigs) != 0 {
		t.Fatalf("refused promotion left %d signatures", len(sigs))
	}
}
