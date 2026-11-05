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
- Tests: unit tests per package, two-node replication of blocks and pointers,
  and an end-to-end test that promotes from bbolt through Kubo/go-orbit-db and
  reads the pointer, version block and attestation block on a second node
- two-node replication and partition-reconvergence spike
- vendored Go dependency tree

Writing to the global store is the publication step; there is no separate push.

## Not yet implemented

- Evaluation that establishes whether linked evidence makes a premise true. The
  promotion seam consumes attestations; nothing in the repository produces one
  by executing a method against evidence.
- Signing of promoted records (Beads decision `validatorium-ilv` is open)
- Core namespace structure beneath the root, private/shared namespaces via
  native IPFS mechanisms, and per-namespace access policy
- Release manifests over exact versioned premises, terms, evidence, assets, and
  artifacts, and per-release valid/current premise tags
- TTL, recurrence, and temporal eligibility beyond attestation expiry
- Runtime/JIT re-validation workflows
- Publication of term records and attestations other than through premise
  promotion; evidence records as a separate record class
- Extensible provenance-preserving adapters for machine or instrument observations
- Automated hypothesis-testing orchestration
- Complete user-facing premise workflows (the CLI opens stores; it does not yet
  create, evaluate, or promote premises)

`Micropublication.ValidateRecord()` and `ValidityAttestation.ValidateRecord()`
check record structure; they do not establish semantic validity.

## Package layout

- `cmd/val`: node CLI
- `internal/model`: micropublication, attestation and term record models
- `internal/namespace`: namespace names
- `internal/storage`: tier interfaces, pointer records, content identifiers
- `internal/storage/memory`: in-process global store for tests
- `internal/bbolt`: bbolt working store (tier 1) and the earlier key/value scaffold
- `internal/ipfsnode`: embedded Kubo node (on-disk or mock-network)
- `internal/orbit`: go-orbit-db global store (tier 2)
- `internal/promote`: tier-1 to tier-2 admission seam
- `internal/spike`: replication-spike tests

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
