package identity

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
)

// testKey returns a fresh entity and its armored public key.
func testKey(t *testing.T, name string) (*openpgp.Entity, string) {
	t.Helper()
	e, err := openpgp.NewEntity(name, "", name+"@example.org", nil)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w, _ := armor.Encode(&buf, openpgp.PublicKeyType, nil)
	if err := e.Serialize(w); err != nil {
		t.Fatal(err)
	}
	w.Close()
	return e, buf.String()
}

func sign(t *testing.T, e *openpgp.Entity, msg string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := openpgp.ArmoredDetachSignText(&buf, e, strings.NewReader(msg), nil); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestFingerprintAndSignatures(t *testing.T) {
	alice, aliceKey := testKey(t, "alice")
	bob, _ := testKey(t, "bob")

	fp, err := Fingerprint(aliceKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(fp) != 40 || strings.ToUpper(fp) != fp {
		t.Fatalf("fingerprint %q", fp)
	}

	msg := ProofText(fp)
	got, err := VerifyDetached(aliceKey, msg, sign(t, alice, msg))
	if err != nil || got != fp {
		t.Fatalf("valid signature: %q %v", got, err)
	}
	if _, err := VerifyDetached(aliceKey, msg+"tampered", sign(t, alice, msg)); err == nil {
		t.Fatal("signature over different text accepted")
	}
	if _, err := VerifyDetached(aliceKey, msg, sign(t, bob, msg)); err == nil {
		t.Fatal("signature by a different key accepted")
	}
	if _, err := Fingerprint("not a key"); err == nil {
		t.Fatal("garbage key accepted")
	}
}
