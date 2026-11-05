# Signing schemes for pseudonymous, authority-free attribution (Beads decision validatorium-ilv)

**Status: UNVERIFIED.** Extracted on 2026-11-05 from a deep-research run that was stopped
during its fetch phase, before adversarial verification and synthesis. Each line is a
claim as extracted from one fetched source by a reader agent; none has been checked
against a second source. Treat as leads to confirm, not as findings.

Fetched sources with extracted claims: 0; claims: 45

## Claims

- The W3C Data Integrity EdDSA Cryptosuites v1.0 specification (W3C Recommendation, 15 May 2025) defines exactly two conforming cryptosuite identifiers, eddsa-rdfc-2022 (RDF Dataset Canonicalization) and eddsa-jcs-2022 (JSON Canonicalization Scheme, RFC 8785), and relegates the older Ed25519Signature2020 suite to a deprecated appendix.  
  Fetch label: fetch:w3.org
- Verification methods in this suite are expressed as Multikey objects whose publicKeyMultibase value must be base-58-btc encoded with the 'z' prefix and the Ed25519 multicodec prefix 0xed01; the proofValue is a detached 64-byte pure Ed25519 (RFC 8032) signature, also multibase base58btc encoded. This makes the key fingerprint and signature directly compatible with did:key identifiers and libp2p-style Ed25519 keys.  
  Fetch label: fetch:w3.org
- The specification requires the pure Ed25519 variant (not Ed25519ph or Ed25519ctx) for both signing and verification, and warns that known flaws in Ed25519 libraries occur mainly on the verification side, recommending testing against the 'Taming the many EdDSAs' test vectors and rejecting small-order public keys to achieve SUF-CMA.  
  Fetch label: fetch:w3.org
- The EdDSA cryptosuites do not provide selective disclosure or unlinkable disclosure; every signed document is fully linkable to the signer's key, which is consistent with (and only with) a stable pseudonymous attribution model.  
  Fetch label: fetch:w3.org
- A documented interoperability pitfall for eddsa-jcs-2022 is failing to copy the document's @context into the proof options before canonicalization; the spec also mandates that RDF canonicalization implementations detect and abort on dataset poisoning. Key rotation and revocation are not addressed in this specification itself but are deferred to the Controlled Identifiers / DID document layer.  
  Fetch label: fetch:w3.org
- RFC 9580 defines a stable key fingerprint usable as a pseudonymous signer identifier: a v6 fingerprint is a 256-bit SHA2-256 hash over the public key packet (prefix 0x9B, 4-octet length, packet body) and the v6 Key ID is its high-order 64 bits, whereas a v4 fingerprint is a 160-bit SHA-1 hash (prefix 0x99, 2-octet length, packet body) with the Key ID as the low-order 64 bits; the spec warns Key IDs can collide and should not be assumed unique.  
  Fetch label: fetch:rfc-editor.org
- OpenPGP signatures carry the signer's identity in-band via the Issuer Fingerprint subpacket, which the spec says SHOULD be in all signatures; a version mismatch between the subpacket and the signature must be treated as malformed, and a v6 signature should avoid the older Issuer Key ID subpacket. This makes third-party attribution of a signed record to a fingerprint possible without external lookup.  
  Fetch label: fetch:rfc-editor.org
- RFC 9580 makes Ed25519 (algorithm ID 27) mandatory-to-implement for signatures and deprecates the RFC 4880-era EdDSALegacy (ID 22) encoding; v6 key material MUST NOT use the legacy Ed25519 OIDs, and Ed25519 signatures must use a hash with a digest of at least 256 bits. This affects interoperability between OpenPGP Ed25519 keys and raw libp2p Ed25519 keys.  
  Fetch label: fetch:rfc-editor.org
- Revocation in OpenPGP is a self-signed Key Revocation Signature (type 0x20) that should carry a Reason for Revocation subpacket (codes 0 no reason, 1 superseded, 2 compromised, 3 retired, 32 User ID no longer valid); the delegated Revocation Key (designated revoker) subpacket is deprecated and MUST NOT be generated, with the spec noting enforcement support for it is unreliable. Key expiry is signalled via a Key Expiration Time subpacket (absent or zero means never expires).  
  Fetch label: fetch:rfc-editor.org
- Documented pitfalls: SHA-1 (used for v4 fingerprints) is not collision resistant, so v4 fingerprints are weaker signer identifiers than v6; v3 signatures must not be created; v6 signatures are salted and therefore non-deterministic, so the same record signed twice by the same key yields different signature bytes (relevant to content-addressed/IPLD storage of signatures).  
  Fetch label: fetch:rfc-editor.org
- did:key is a purely generative DID method with no support for key rotation, updates, deactivation, or revocation; a did:key identifier is bound permanently to one public key.  
  Fetch label: fetch:w3c-ccg.github.io
- The did:key identifier is encoded as the did:key prefix followed by a Multibase (base58-btc 'z' or base64url 'u') string whose payload is a multicodec header plus the raw public key bytes; supported signature key codecs include secp256k1 (0xe7, 33 bytes), X25519 (0xec, 32 bytes), Ed25519 (0xed, 32 bytes), P-256 (0x1200, 33 bytes) and P-384 (0x1201, 49 bytes).  
  Fetch label: fetch:w3c-ccg.github.io
- The spec strongly discourages using did:key for interactions lasting weeks to months and says long-lived use is only recommended with high confidence in hardware isolation; it is 'not ideal' for high-value transactions.  
  Fetch label: fetch:w3c-ccg.github.io
- Because the identifier is derived directly from the public key, a did:key is globally unique and globally correlatable (a stable pseudonymous fingerprint), and deriving the X25519 keyAgreement key from the Ed25519 signing key violates the principle of key separation.  
  Fetch label: fetch:w3c-ccg.github.io
- A resolved did:key DID document by default lists the key under authentication, assertionMethod, capabilityInvocation and capabilityDelegation (keyAgreement only when encryption key derivation is enabled), with verification methods expressed as Multikey (publicKeyMultibase) or JsonWebKey (publicKeyJwk), and legacy Ed25519VerificationKey2020/X25519KeyAgreementKey2020 only when experimental public key types are disabled.  
  Fetch label: fetch:w3c-ccg.github.io
- Data Integrity proofs are embedded in the secured data as a proof object whose type is DataIntegrityProof; the cryptosuite property is mandatory for that type and the actual signing algorithms (e.g. eddsa-rdfc-2022, eddsa-jcs-2022, ecdsa-rdfc-2019, ecdsa-jcs-2019, bbs-2023) are defined in separate cryptosuite specifications, so a Go implementation must pick and implement a specific suite beyond this base spec.  
  Fetch label: fetch:w3.org
- A proof must carry a proofPurpose (e.g. assertionMethod, authentication, capabilityInvocation, capabilityDelegation, keyAgreement), and the verifier MUST reject a proof whose purpose does not match the expected one; assertionMethod is the purpose for signing attributable statements rather than conferring authority.  
  Fetch label: fetch:w3.org
- Third-party verification depends on dereferencing the verificationMethod URL to a controlled identifier document (e.g. a DID document) and checking that the key's controller matches the asserted issuer; key rotation/revocation is expressed through the verification method's 'revocation' and 'expires' properties in that controller document rather than inside the signature itself.  
  Fetch label: fetch:w3.org
- Data Integrity signatures rely on canonicalization (RDF Dataset Canonicalization or JSON Canonicalization Scheme) before hashing and signing, and the spec itself warns that this transformation step is a security-critical dependency and that enveloping formats such as JWTs/CWTs avoid it.  
  Fetch label: fetch:w3.org
- A static digital signature is itself a correlatable identifier: the spec's privacy section states that signatures enable tracking across interactions, which is exactly the property a pseudonymous-but-traceable attribution scheme intends, and only selective-disclosure suites like bbs-2023 provide unlinkability.  
  Fetch label: fetch:w3.org
- The draft ties signer attribution to the Issuer Fingerprint subpacket and requires consistency between Issuer Key ID and v4 Issuer Fingerprint subpackets: each Key ID must match the low 64 bits of exactly one v4 fingerprint subpacket and every v4 fingerprint subpacket must have a matching Key ID; 'same key pair' for signature cumulation is determined by these issuer subpackets.  
  Fetch label: fetch:datatracker.ietf.org
- Revocation semantics are split into hard and soft: hard revocations are treated as if created in the infinite past (invalidating all signatures by that key), while signatures from a revoked certificate are only accepted when the revocation is soft and the signature predates it; in the IANA registry only 'Key is Superseded' and 'Key is retired and no longer used' are soft, all other reasons are hard.  
  Fetch label: fetch:datatracker.ietf.org
- The draft makes Key Flags mandatory for signing: a signature made by a key lacking the corresponding Key Flag must be considered invalid, and a key with no Key Flags subpacket must not create signatures at all - a pitfall for minimal self-generated OpenPGP keys used purely for attribution.  
  Fetch label: fetch:datatracker.ietf.org
- For third-party verifiability the draft recommends placing all signature subpackets in the hashed (signed) area, forbids more than one subpacket of a given type in the hashed area (receivers MUST invalidate violating signatures), and requires bitwise-identical unhashed subpackets to be deduplicated.  
  Fetch label: fetch:datatracker.ietf.org
- The document is an individual Internet-Draft (not a standard) that proposes updates to RFC 9580 without new wire formats; version -02 is dated 3 November 2025, expired 7 May 2026 per the datatracker, and still contains unresolved TODO/FIXME/TBC items (for example on revocation-reason mismatch, temporary identities, and cross-protocol key reuse), so its rules cannot yet be relied on as implemented library behaviour.  
  Fetch label: fetch:datatracker.ietf.org
- GopenPGP v3 (ProtonMail/gopenpgp) is actively maintained in Go as of late 2025, with the latest release v3.5.2 (30 Sep) fixing compression tests for Go 1.27 and v3.5.0 bumping the go-crypto fork dependency to v1.5.2; the v2 line (v2.11.x) is still receiving parallel maintenance releases.  
  Fetch label: fetch:github.com
- GopenPGP v3.0.0 added full support for RFC 9580 (OpenPGP v6), including generating v6 keys, so a Go application can produce RFC 9580-format keys and signatures via this library.  
  Fetch label: fetch:github.com
- GopenPGP v3's API is not backward-compatible with v2; applications migrating from v2 must rewrite all previous API calls.  
  Fetch label: fetch:github.com
- GopenPGP v3 introduces configurable 'profiles' for algorithm selection and, since v3.5.1, ships a post-quantum (PQC) profile; PQC algorithms (draft-ietf-openpgp-pqc draft-09) are available in the '-proton' tagged variants that also include non-standardized features (persistent symmetric keys, automatic forwarding).  
  Fetch label: fetch:github.com
- GopenPGP v3 supports signing a message with multiple keys and, on verification, considers all signatures in a message and allows each to be inspected individually, which is relevant for multi-signer attribution of records.  
  Fetch label: fetch:github.com
- IANA's OpenPGP Public Key Algorithms registry assigns ID 27 to Ed25519 and ID 28 to Ed448 as first-class algorithms, while the older EdDSA algorithm ID 22 is marked deprecated; this means Ed25519 keys (the same curve used by libp2p peer keys) can be carried natively in modern OpenPGP (RFC 9580) rather than only via the legacy EdDSA encoding.  
  Fetch label: fetch:iana.org
- OpenPGP defines standardized, machine-readable revocation semantics: Signature Types 0x20 (Key Revocation), 0x28 (Subkey Revocation) and 0x30 (Certification Revocation), plus a Reason for Revocation registry with codes 0 (no reason), 1 (superseded), 2 (compromised), 3 (retired) and 32 (User ID no longer valid), carried in signature subpacket 29. Competing schemes like raw Ed25519/libp2p PeerIDs and did:key have no equivalent registry-defined revocation vocabulary.  
  Fetch label: fetch:iana.org
- OpenPGP fingerprints, which would serve as the stable pseudonymous signer identity, differ by key version: v4 fingerprints are 160-bit SHA-1 of the normalized public key packet with the Key ID being the last 64 bits, whereas v6 fingerprints are 256-bit SHA-256 with the Key ID being the first 64 bits. A system that pins identity to a fingerprint must therefore fix the key version, and v4 fingerprints depend on SHA-1.  
  Fetch label: fetch:iana.org
- Per the IANA Key and Signature Versions registry, only v4 keys with v4 signatures and v6 keys with v6 signatures are flagged as permitted to generate ('Generate? Yes'); v3 signatures are marked 'Generate? No'. Implementations in Go (go-crypto, GopenPGP) must therefore support v4 and optionally v6 to be compliant and interoperable, and the registry references RFC 9580 exclusively (the string 'RFC 4880' does not appear on the page), indicating RFC 9580 has superseded RFC 4880 as the normative reference.  
  Fetch label: fetch:iana.org
- The Signature Subpacket Types registry marks subpacket 12 (Revocation Key, i.e. designated third-party revoker) as deprecated, while subpacket 33 (Issuer Fingerprint) is the current mechanism for binding a signature to the full issuer fingerprint rather than the 64-bit Issuer Key ID (subpacket 16); and the registry already includes post-quantum signature algorithms (IDs 30-34, ML-DSA and SLH-DSA, referenced to RFC 9980).  
  Fetch label: fetch:iana.org
- ProtonMail/go-crypto v1.5.0 adds post-quantum OpenPGP algorithm support (ML-KEM, ML-DSA, and SLH-DSA), meaning the Go OpenPGP fork is actively tracking the PQC OpenPGP drafts as of September 2026.  
  Fetch label: fetch:github.com
- go-crypto v1.5.0 added a Signature.IssuerKeyVersion field, which exposes the version of the issuing key on a parsed signature (relevant for distinguishing v4 from v6/RFC 9580 issuers when verifying signatures by third parties).  
  Fetch label: fetch:github.com
- go-crypto v1.5.0 is largely a robustness/hardening release: the majority of its changelog entries fix panics or invalid slice access on malformed or nil-issuer input in the OpenPGP packet parsing and verification paths (including v5/v6 SKESK packets and PKESK/ECDH session-key decoding).  
  Fetch label: fetch:github.com
- go-crypto v1.5.0 fixed a defect in RSA key generation where generated primes did not satisfy the P < Q constraint, and a follow-up precompute bug after the prime swap, indicating RSA key generation in prior versions could emit non-conformant keys.  
  Fetch label: fetch:github.com
- go-crypto v1.5.0 restricts embedded signatures to embeddable signature types, closing a path where arbitrary signature types could be embedded in a signature subpacket (a verification-pitfall fix).  
  Fetch label: fetch:github.com
- ProtonMail go-crypto's packet package is a maintained Go implementation of OpenPGP packet parsing/serialization that supports both RFC 4880 v4 keys and RFC 9580 v6 keys, with v6 key generation opt-in via Config.V6Keys (default output remains v4 keys).  
  Fetch label: fetch:pkg.go.dev
- The library natively supports Ed25519 signing under both the legacy EdDSA algorithm ID (22) and the RFC 9580 Ed25519 (27) and Ed448 (28) algorithm IDs, with dedicated constructors (NewEd25519PublicKey/NewEd25519PrivateKey), so a libp2p-style Ed25519 key can be wrapped in OpenPGP packets.  
  Fetch label: fetch:pkg.go.dev
- Key and subkey revocation is a first-class API: Signature.RevokeKey / RevokeSubkey produce standard revocation signatures (SigTypeKeyRevocation 0x20, SigTypeSubkeyRevocation 0x28, SigTypeCertificationRevocation 0x30) carrying an RFC-defined ReasonForRevocation code (NoReason=0, KeySuperseded=1, KeyCompromised=2, KeyRetired=3, UserIDNotValid=32), and PublicKey.VerifyRevocationSignature checks them.  
  Fetch label: fetch:pkg.go.dev
- The PublicKey packet exposes a stable signer identifier via Fingerprint []byte and KeyId uint64, but the short 32-bit key ID is undefined for v5/v6 keys, so pseudonymous identity for new-format keys must use the full fingerprint or full key ID.  
  Fetch label: fetch:pkg.go.dev
- The module is at a stable v1 tag (v1.5.2 published Sep 25, 2026, with a newer module version existing) under BSD-3-Clause, is imported by 200+ known packages, and has started adding post-quantum algorithm IDs (ML-DSA, SLH-DSA, ML-KEM), indicating active maintenance in 2026; Go 1.26 changes mean Config.Rand is ignored for standard-library key generation unless a GODEBUG setting is enabled.  
  Fetch label: fetch:pkg.go.dev
