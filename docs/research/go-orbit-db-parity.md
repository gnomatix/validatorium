# go-orbit-db capability check (validatorium-bqe.3)

Checked 2026-11-17. Go side: the vendored `berty.tech/go-orbit-db` v1.22.2
and `berty.tech/go-ipfs-log` (read in `vendor/`, tested in
`internal/orbit/parity_test.go`). JavaScript side: `orbitdb/orbitdb` at
commit `2ee8387` (2026-05-14, package version 4.0.0), read from its source
tree and `docs/ENCRYPTION.md`. JavaScript versions 2 and 3 were not checked.
Supersedes the leads in `UNVERIFIED-go-orbit-db-parity.md`.

## Capability matrix

| Capability | JS OrbitDB 4.0.0 (source) | go-orbit-db 1.22.2 | How the Go column was established |
|---|---|---|---|
| Event log database | `src/databases/events.js` | `stores/eventlogstore` | test `TestParityEventLogStoreAndPlaintextPayload` |
| Key-value database | `src/databases/keyvalue.js` | `stores/kvstore` | test `TestParityKeyValueStore` |
| Indexed key-value database | `src/databases/keyvalue-indexed.js` | absent | `vendor/` has no counterpart |
| Documents database | `src/databases/documents.js` | `stores/documentstore` | used by `internal/orbit` (existing tests) |
| Access controllers | `ipfs`, `orbitdb` | `ipfs`, `orbitdb`, `simple` | `vendor/.../accesscontroller/` |
| Write list enforced on local writes | yes (access controller) | yes | test `TestParityWriteAccessRefusesOtherIdentity` |
| Identities | `src/identities/`, provider `publickey` | `go-ipfs-log/identityprovider`, provider `orbitdb` (secp256k1) | test `TestParityIdentitySignVerify` |
| Payload encryption | modular, `encryption: { data }` | absent | no encryption option in `CreateDBOptions`; test shows the entry block carries the value base64-encoded, readable by any peer holding the block |
| Log-entry (replication) encryption | modular, `encryption: { replication }` | partial: `go-ipfs-log` can encrypt entry links (`cbor.Options.LinkKey`, NaCl secretbox) through `CreateDBOptions.IO`; payload stays readable | read in `vendor/berty.tech/go-ipfs-log/io/cbor/cbor.go`; not tested |

## Findings for the namespace and signing work

- **Private or shared namespaces cannot rely on go-orbit-db for
  confidentiality.** Any peer that holds an entry block can read its value.
  Confidential records would have to be encrypted by Validatorium before
  they are written.
- **Entry signatures are OrbitDB identities (secp256k1), not OpenPGP.**
  Validatorium's OpenPGP signatures (`internal/identity`) stay separate
  records.
- **Defect in the vendored code:** `Identity.GetPublicKey()` fails
  ("cannot parse invalid wire-format data") on the default provider's
  keys, because it expects a protobuf-wrapped key. The provider's
  `UnmarshalPublicKey` reads them. Code that verifies OrbitDB identities
  must use the provider method.
- **Write access** is enforced on local writes when a peer is outside the
  write list (single-node test). Enforcement on replicated entries from a
  peer outside the list was not tested here.

## Not checked

JavaScript OrbitDB 2.x and 3.x; link encryption in practice; access-control
enforcement on replication; interoperability between the Go and JavaScript
implementations.
