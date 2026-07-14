# Validatorium — Build State

Last updated: 2026-07-14.

This document records what the repository implements now. Intended behavior is
defined in the specifications. Durable work state and architecture decisions are
tracked in Beads; use `bd ready` and `bd list --type decision` for current records.

## Implemented now

- Go 1.26.5 module at `gnomatix/validatorium`
- CLI skeleton at `cmd/val`
- UUID-based micropublication record model
- bbolt local storage scaffold
- embedded Kubo node and go-orbit-db integration
- two-node replication and partition-reconvergence spike
- vendored Go dependency tree

This is an initial record/storage scaffold and tested replication spike, not the
complete Validatorium validation, release, retrieval, or ingestion system.

## Storage scaffold

The implemented scaffold contains:

1. A local bbolt working store.
2. An embedded Kubo/IPFS and go-orbit-db replication layer.

UUIDs provide stable record identity. Content addressing provides integrity for
stored versions; it does not by itself implement Validatorium release manifests or
runtime validity.

## Not yet implemented

- Evaluation that establishes whether linked evidence makes a premise true
- Admission enforcement that keeps unsupported submissions out of the validated
  premise store
- Release manifests over exact versioned premises, terms, evidence, assets, and
  artifacts
- Per-release valid/current premise tags
- Expiry, TTL, recurrence, and temporal eligibility
- Runtime/JIT re-validation and immutable runtime attestations
- Versioned ontology-term records and replacement UUID traversal
- Extensible provenance-preserving adapters for machine or instrument observations
- Automated hypothesis-testing orchestration
- Complete user-facing premise workflows

The current `Micropublication.ValidateRecord()` method checks record structure; it does
not establish semantic validity.

## Package layout

- `cmd/val`: CLI skeleton
- `internal/bbolt`: local bbolt store
- `internal/ipfsnode`: embedded Kubo node setup
- `internal/orbit`: go-orbit-db integration
- `internal/model`: micropublication record model
- `internal/storage`: storage abstractions
- `internal/spike`: replication-spike tests

## Repository quality gates

Use the required vendored dependency tree:

```bash
go mod verify
go test -mod=vendor ./...
go build -mod=vendor ./...
```

Run `go mod vendor` after every `go mod tidy` operation.

## Next work

Run `bd ready` to select the next unblocked task. Do not infer implementation status
from a design specification; verify it from code and tests.
