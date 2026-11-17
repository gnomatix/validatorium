# Validatorium — Build State

Last updated: 2026-11-05.

This document records what the repository implements now. Intended behavior is
defined in the specifications. Durable work state and architecture decisions are
tracked in Beads; use `bd ready` and `bd list --type decision` for current records.

## Implemented now

- Go 1.26.5 module at `gnomatix/validatorium`
- CLI at `cmd/val` that opens both storage tiers (on-disk Kubo repo, or an
  isolated in-process network with `-offline`) and reports their addresses
- UUID-based micropublication record model with canonical bytes for
  content-addressed immutable versions
- Immutable validity attestation records (`true` / `false` / `indeterminate`,
  optional expiry) with well-formedness checks
- Versioned ontology-term records with obsolete-record replacement pointers and
  replacement traversal (splits, merges, cycle and dangling-pointer detection)
- Tier-boundary interfaces: `storage.WorkingStore` (tier 1) and
  `storage.GlobalStore` (tier 2); `storage.ContentID` computes the CIDv1
  raw/SHA2-256 identifier that IPFS assigns to the same bytes
- bbolt working store: mutable candidate drafts, immutable versions by CID,
  UUID-to-current-CID pointers, an append-only attestation log indexed by
  premise, and versioned term records; basic-operation benchmarks
- Tier-2 global store over embedded Kubo and go-orbit-db: raw blocks by CID
  (verified on put and get) and a replicated UUID-keyed pointer Documents store
- Promotion seam: a candidate is published to the global store only under a
  true, unexpired attestation that names the exact premise version; refusals
  leave both tiers unchanged; version and attestation history is retained
- Namespace pointer layer: validated `namespace:subdomain:...` names
  (`internal/namespace`); namespaced pointers map (namespace, UUID) to an exact
  version CID in both tiers, one record referenceable from many namespaces;
  promotion can place a version into named namespaces; only the root
  namespace `validatorium` ships with the binary
- Release manifests (`internal/release`): a builder tags a premise only when,
  at the release time, it is current, a true unexpired attestation names its
  exact current version, every term record that attestation relied on is
  current, and every premise it depends on is also tagged; excluded candidates
  are reported with reasons. Publishing stores the manifest and its exact term
  versions as blocks, requires the tagged premise versions to be in the global
  store already, and lists the manifest under its label in
  `validatorium:releases`; a published label never changes
- Wikidata (`internal/wikidata`): revision-pinned entity reads; a structured
  snapshot that cites the source as a pointer (revision-pinned URI plus the
  content identifier of the bytes read) with an equivalence re-check; term
  records pinned to an entity revision; and a `wikidata-statement` evaluator
  whose claim is about what the pinned revision holds
- Self-certification registry for the ITVB Seal of Truth: a
  `SelfCertification` record class, separate from premise validity, that is
  versioned and can be obsoleted by anyone with the obsoleting agent and
  reason recorded; `val api` serves `GET /api/verification?url=` in the ITVB
  response format (with ITVB headers and CORS for browser plugins), plus
  registration and obsoleting endpoints
- OpenPGP identities for the registry (`internal/identity`, `Identity`
  record class): `POST /api/identities` registers an armored public key under
  its fingerprint after a detached-signature proof of key possession over
  `identity.ProofText(fingerprint)`; `POST /api/identities/{fp}/obsolete`
  obsoletes it with agent and reason recorded (no revocation; versions stay
  addressable). A certification registered with a detached signature over
  `SignedPayload()` by a current identity is recorded as signer-verified;
  verification responses report `signer_verified` only while that identity is
  still current. Who may obsolete an identity is not restricted yet
- Evaluation (`internal/evaluate`): an `Evaluator` interface and a registry
  that selects the evaluator by the premise's MP:Method reference and returns
  an immutable attestation for the exact premise version, without writing.
  In this package one evaluator exists: runtime file digest
  (`urn:validatorium:method:file-sha256`), which returns true on a SHA-256
  match, false on a mismatch, and indeterminate when the file cannot be read,
  and records the content identifier of the bytes it read as evidence. A
  second evaluator, `wikidata-statement`, is in `internal/wikidata` (above)
- Automatic publication on validity (`internal/autopublish`): evaluates a
  working-store candidate with the evaluator registered for its method,
  records the attestation whatever its result, and publishes the candidate
  (with namespaced pointers) only when the attestation establishes validity;
  admission rules stay in `internal/promote`. `val publish` runs it against
  the on-disk Kubo/go-orbit-db global store (or `-offline` for an isolated node).
  With `-sign-key` (an armored OpenPGP private key file), every published
  version is signed and the signature record kept in the working store, so
  every version a key published can be listed (`SignaturesBySigner`).
  `val api` serves `GET /view/premises/{uuid}`: a cytoscape.js graph of the
  premise with its support and challenge evidence coloured by kind (data,
  method, material) and its terms; the library is embedded in the binary.
  `GET /builder` is the argument builder: search premises, keep them in a
  toolkit (stored in the browser), mark a claim and its supporting or
  attacking premises, and get the argument in Argdown
  (`argdown-conventions.md` §1, §2.2, §4.1–4.2).
  `val api -publish` serves `POST /api/premises/{uuid}/validate` (optional body
  `{"scope": {...}}`), which runs the same evaluation and publication for one
  candidate and answers with the result, the method and, when published, the
  version and attestation CIDs
- Tests: an end-to-end test evaluates a real file, promotes on the true
  result, refuses the false one, and builds and publishes a release; unit
  tests per package, two-node replication of blocks and pointers,
  and an end-to-end test that promotes from bbolt through Kubo/go-orbit-db and
  reads the pointer, version block and attestation block on a second node
- two-node replication and partition-reconvergence spike
- vendored Go dependency tree

Writing to the global store is the publication step; there is no separate push.

## Not yet implemented

- Evaluators beyond the runtime file digest: document inspection, dataset
  lookup, reproducible computation, instrument observation.
- Signing of promoted records (Beads decision `validatorium-ilv` is open)
- Core namespace structure beneath the root, private/shared namespaces via
  native IPFS mechanisms, and per-namespace access policy
- Evidence, asset and artifact versions in release manifests (manifests list
  premises, attestations and term records only)
- TTL, recurrence, and temporal eligibility beyond attestation expiry
- Runtime/JIT re-validation workflows
- Publication of term records and attestations other than through premise
  promotion; evidence records as a separate record class
- Extensible provenance-preserving adapters for machine or instrument observations
- Automated hypothesis-testing orchestration
- CLI release publication to the global store (`val release` builds the
  manifest against the working store only)

`Micropublication.ValidateRecord()` and `ValidityAttestation.ValidateRecord()`
check record structure; they do not establish semantic validity.

## Package layout

- `cmd/val`: node CLI; subcommands `create`, `evaluate`, `attestations`, `release` for the local workflow, `publish` for evaluate-then-publish to the global store, and `api` for the self-certification registry API
- `internal/model`: micropublication, attestation and term record models
- `internal/namespace`: namespace names
- `internal/storage`: tier interfaces, pointer records, content identifiers
- `internal/storage/memory`: in-process global store for tests
- `internal/bbolt`: bbolt working store (tier 1) and the earlier key/value scaffold
- `internal/ipfsnode`: embedded Kubo node (on-disk or mock-network)
- `internal/orbit`: go-orbit-db global store (tier 2)
- `internal/promote`: tier-1 to tier-2 admission seam
- `internal/autopublish`: evaluate-then-publish (automatic publication on validity)
- `internal/release`: release manifest builder and publication
- `internal/evaluate`: evaluator interface, method registry, file-digest evaluator
- `internal/wikidata`: Wikidata pointer records, term resolution, statement evaluator
- `internal/identity`: OpenPGP fingerprints and detached-signature verification
- `internal/api`: self-certification registry HTTP API (ITVB response format)
- `internal/spike`: replication-spike tests

Design: `docs/design/two-tier-storage.md` records the storage system design,
its trade-offs, and the proposed next layers.

## Repository quality gates

Use the required vendored dependency tree:

```bash
go mod verify
go vet -mod=vendor ./...
go test -mod=vendor ./...
go build -mod=vendor ./...
```

`go test -short` skips the multi-node tests. Run `go mod vendor` after every
`go mod tidy` operation.

## Next work

Run `bd ready` to select the next unblocked task. Do not infer implementation status
from a design specification; verify it from code and tests.
