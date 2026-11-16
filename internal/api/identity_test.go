package api

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"

	"gnomatix/validatorium/internal/identity"
	"gnomatix/validatorium/internal/model"
)

func newKey(t *testing.T, name string) (*openpgp.Entity, string, string) {
	t.Helper()
	e, err := openpgp.NewEntity(name, "", name+"@example.org", nil)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w, _ := armor.Encode(&buf, openpgp.PublicKeyType, nil)
	_ = e.Serialize(w)
	w.Close()
	fp, err := identity.Fingerprint(buf.String())
	if err != nil {
		t.Fatal(err)
	}
	return e, buf.String(), fp
}

func detach(t *testing.T, e *openpgp.Entity, msg string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := openpgp.ArmoredDetachSignText(&buf, e, strings.NewReader(msg), nil); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestSignedSelfCertificationLifecycle(t *testing.T) {
	srv, _ := newServer(t)
	alice, aliceKey, aliceFP := newKey(t, "alice")
	mallory, _, _ := newKey(t, "mallory")

	// Registration needs proof of key possession.
	if r, _ := post(t, srv.URL+"/api/identities", map[string]string{"public_key": aliceKey, "proof": detach(t, mallory, identity.ProofText(aliceFP))}); r.StatusCode != 403 {
		t.Fatalf("proof by another key: %d", r.StatusCode)
	}
	r, out := post(t, srv.URL+"/api/identities", map[string]string{"public_key": aliceKey, "proof": detach(t, alice, identity.ProofText(aliceFP))})
	if r.StatusCode != 201 || out["fingerprint"] != aliceFP {
		t.Fatalf("register identity: %d %v", r.StatusCode, out)
	}
	if r, _ := post(t, srv.URL+"/api/identities", map[string]string{"public_key": aliceKey, "proof": detach(t, alice, identity.ProofText(aliceFP))}); r.StatusCode != 409 {
		t.Fatalf("re-registration: %d", r.StatusCode)
	}

	page := "https://example.org/signed"
	cert := &model.SelfCertification{Subject: page, Standard: model.SealOfTruth, Method: "frontier-model-analysis", Date: "2026-11-16"}
	body := map[string]string{"subject": page, "method": cert.Method, "date": cert.Date, "signer": aliceFP}

	// A signature by the wrong key, or over different content, is refused.
	body["signature"] = detach(t, mallory, cert.SignedPayload())
	if r, _ := post(t, srv.URL+"/api/certifications", body); r.StatusCode != 403 {
		t.Fatalf("wrong-key signature: %d", r.StatusCode)
	}
	body["signature"] = detach(t, alice, strings.Replace(cert.SignedPayload(), "2026-11-16", "2026-01-01", 1))
	if r, _ := post(t, srv.URL+"/api/certifications", body); r.StatusCode != 403 {
		t.Fatalf("signature over other content: %d", r.StatusCode)
	}

	body["signature"] = detach(t, alice, cert.SignedPayload())
	if r, out := post(t, srv.URL+"/api/certifications", body); r.StatusCode != 201 {
		t.Fatalf("signed certification: %d %v", r.StatusCode, out)
	}
	_, v := verify(t, srv.URL, page)
	if !v.Verified || len(v.Validatorium.Current) != 1 || !v.Validatorium.Current[0].SignerVerified || v.Validatorium.Current[0].Signer != aliceFP {
		t.Fatalf("signed certification view: %+v", v.Validatorium)
	}

	// Obsoleting the identity withdraws signer verification everywhere; the
	// certification itself stays on record.
	if r, _ := post(t, srv.URL+"/api/identities/"+aliceFP+"/obsolete", map[string]string{"by": "moderator:1", "reason": "key reported compromised"}); r.StatusCode != 200 {
		t.Fatalf("obsolete identity: %d", r.StatusCode)
	}
	_, v = verify(t, srv.URL, page)
	if len(v.Validatorium.Current) != 1 || v.Validatorium.Current[0].SignerVerified {
		t.Fatalf("after identity obsoleted: %+v", v.Validatorium)
	}
	// An obsoleted identity can no longer sign new certifications.
	cert2 := &model.SelfCertification{Subject: "https://example.org/other", Standard: model.SealOfTruth, Method: "m", Date: "2026-11-16"}
	if r, _ := post(t, srv.URL+"/api/certifications", map[string]string{"subject": cert2.Subject, "method": "m", "date": cert2.Date, "signer": aliceFP, "signature": detach(t, alice, cert2.SignedPayload())}); r.StatusCode != 403 {
		t.Fatalf("signing with an obsoleted identity: %d", r.StatusCode)
	}
}
