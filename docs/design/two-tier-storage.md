# Two-Tier Storage: System Design

**Status:** Implemented core, with proposed next layers marked as such
**Date:** 2026-11-05
**Beads:** validatorium-bqe.8 (this document); decisions validatorium-gka,
-aqo, -aql, -yv2, -7oj, -o2u, -z5a, -egu; tasks -bqe.2, -bqe.4, -bqe.5,
-bqe.6, -bqe.7

This document describes the storage system as built, the decisions it rests
on, the evidence that it works, and the trade-offs taken. Sections marked
**Proposed** describe layers that are not implemented; `docs/BUILD_STATE.md`
is the authority on what exists.

Terms used here: a *record* is a UUID-identified thing (premise, attestation,
term record). A *version* is one immutable byte encoding of a record,
identified by its content identifier (CID). A *pointer* says which version a
UUID currently refers to. *Promotion* is the only path by which a record
enters the global store.

## 1. Requirements

### 1.1 Functional

| Requirement | Source | Status |
|---|---|---|
| Stable record identity across versions (UUID at creation, not content hash) | decision yv2; data-model.md §2 | Implemented (`model.New`, `storage.VersionRef`) |
| Immutable, content-addressed versions; history never rewritten | project-principles.md rule 8; storage-architecture.md §2 | Implemented (`storage.ContentID`, `ErrImmutable`) |
| Full local operation with zero connectivity | decision gka; bqe.4 | Implemented (`bbolt.WorkingStore`) |
| Replication of published records between nodes without a central authority | decision gka; distributed-datastore.md §6 | Implemented (`orbit.GlobalStore`), tested two-node |
| A record enters the global store only under a true evaluation | project-principles.md rule 2; storage-architecture.md §4; decision o2u | Implemented (`promote.Promoter`) |
| Writing to the global store is publication; no separate push | decision o2u | Implemented (promotion writes blocks and pointers directly) |
| Obsolete records remain addressable and point to replacements | decision egu; data-model.md §4, §7 | Implemented for term records and pointers |
| Per-domain namespaces as a pointer layer over UUID records | decision 7oj; bqe.6 | Implemented (root namespace only ships) |
| Evaluation that produces attestations from evidence | evidence-verification.md §4 | Not implemented |
| Release manifests over exact versions | decision z5a; data-model.md §9 | Not implemented |
| Runtime/JIT re-validation producing new attestations | data-model.md §10 | Not implemented |
| Signed records | decision ilv (open) | Not implemented |

### 1.2 Non-functional

- **Correctness over throughput.** A premise store that admits an unsupported
  record is wrong even if fast. Every admission check runs before any write.
- **Integrity.** Every version fetched from either tier is re-hashed and
  compared with the CID it was requested by (`storage.VerifyContentID`).
- **Offline first.** Tier 1 must work with the network down and the IPFS node
  not started.
- **Single binary.** Go only; Node.js only for Argdown tooling (decision aql).
- **Scale target (current).** One node holds on the order of 10^5 to 10^6
  premises in bbolt comfortably; the global tier is bounded by IPFS block
  storage and OrbitDB log length, discussed in §5.

### 1.3 Constraints

- Team of one plus agents; vendored dependency tree, no network at build.
- Kubo v0.41, go-orbit-db v1.22, bbolt v1.5 (go.mod).
- Validity semantics are fixed by the specifications and must not be
  re-derived from storage behaviour (replication is not consensus).

## 2. High-Level Design

### 2.1 Components

```
                 +---------------------------+
                 |        cmd/val (CLI)      |
                 +------------+--------------+
                              |
             +----------------+-----------------+
             |                                  |
   +---------v----------+            +----------v-----------+
   |  internal/promote  |            |  (evaluators, release |
   |  admission seam    |            |   builders: proposed) |
   +----+----------+----+            +-----------------------+
        |          |
        | tier 1   | tier 2
        |          |
+-------v------+ +-v-------------------------------------------+
| bbolt        | | orbit.GlobalStore                            |
| WorkingStore | |  blocks: Kubo CoreAPI Block().Put/Get (raw) |
|              | |  pointers: OrbitDB Documents store          |
| candidates   | |  namespaces: one Documents store each       |
| versions     | +----------------------+----------------------+
| current      |                        |
| attestations | +----------------------v----------------------+
| terms        | | ipfsnode.Node (embedded Kubo, libp2p,       |
| namespaces   | |  pubsub, bitswap)                           |
+--------------+ +---------------------------------------------+

   storage.WorkingStore / storage.GlobalStore / storage.NamespaceStore
   are the only interfaces promote and callers depend on.
```

Packages and their single responsibilities:

- `internal/model`: record types, well-formedness checks, canonical bytes.
  No I/O. Checks structure, never truth.
- `internal/storage`: tier interfaces, pointer records, `ContentID`.
- `internal/bbolt`: tier 1. Beside the IPFS datastore, not under it.
- `internal/ipfsnode`, `internal/orbit`: tier 2. Moves bytes; decides nothing.
- `internal/promote`: the one write path from tier 1 to tier 2.
- `internal/namespace`: validated namespace names.
- `internal/storage/memory`: tier-2 test double with the same contract.

### 2.2 Data flow: promotion

```
candidate draft (bbolt)        attestation (from an evaluator)
        |                               |
        +-------------+-----------------+
                      v
          promote.PromoteInto(uuid, att, namespaces)
                      |
      1 load draft; ValidateRecord on draft and attestation
      2 draft.State must be current
      3 cid := ContentID(draft.CanonicalBytes())
        att.PremiseUUID == uuid && att.PremiseVersionCID == cid
      4 att.Result == true && not expired at now()
        -- any failure: return error, nothing written --
      5 tier 1: PutVersion(draft), PutAttestation(att)
      6 tier 2: PutBlock(version bytes)   -> must equal cid
                PutBlock(attestation bytes) -> attCID
                PutPointer{uuid, cid, attCID, status}
      7 tier 1: SetCurrent(uuid, cid); DeleteCandidate(uuid)
      8 both tiers: PutNamespacedPointer{ns, uuid, cid} per namespace
```

Step 6 is publication. Peers holding the pointer store receive the pointer
through OrbitDB's pubsub-driven log exchange and fetch the two blocks by CID
through bitswap. `promote.TestPromoteThroughOrbitReplicatesToPeer` exercises
exactly this across two in-process Kubo nodes.

### 2.3 Storage choices

| Need | Choice | Decision / rationale |
|---|---|---|
| Local embedded KV | bbolt | aqo: single-file, transactional, Go-native, no CGO; sits beside IPFS |
| Immutable version bytes | IPFS raw blocks (CIDv1, SHA2-256) | gka; a CID computed locally equals the IPFS block CID, so tier 1 and tier 2 agree without a lookup |
| Mutable per-UUID pointer | OrbitDB Documents store keyed `_id` = UUID | gka, 7oj; last-writer-wins CRDT on a per-key basis is the right shape for a pointer |
| Namespaced pointers | One Documents store per namespace | 7oj; address derives from name + write list, so peers converge by name |
| Identity key | UUIDv4 from `crypto/rand` | yv2; no external dependency |

## 3. Deep Dive

### 3.1 Data model

Records and their canonical bytes (`model/canonical.go`, `model/term.go`):

- `Micropublication` (premise): UUID, claim text and term groundings, support
  and challenge graphs of MP evidence elements, attribution, lifecycle.
- `ValidityAttestation`: UUID, premise UUID, premise version CID, result in
  {true, false, indeterminate}, evaluated-at, evaluator, method, evidence
  versions, term record UUIDs, scope, dependencies, optional expiry.
- `TermRecord`: UUID, label, ontology URI, ontology record UUID, source
  version, definition, attribution, lifecycle with replacement UUIDs.

Canonical form is the `encoding/json` encoding of the struct. Field order is
declaration order and map keys are sorted, so equal records give equal bytes.
This is a deliberate, cheap choice; see §6 for its limits.

Lifecycle is two-valued, `current` or `obsolete`, and obsolescence means
supersession only (project-principles.md rule 11). Expiry and recurrence live
in attestations and policy, never in lifecycle.

### 3.2 Tier 1 schema (`bbolt/working.go`)

| Bucket | Key | Value | Mutability |
|---|---|---|---|
| `candidates` | UUID | canonical premise bytes | mutable draft |
| `versions` | CID | canonical premise bytes | immutable |
| `current` | UUID | CID | pointer |
| `attestations` | attestation UUID | canonical attestation bytes | immutable |
| `attestations_by_premise` | premise UUID `0x00` attestation UUID | attestation CID | index |
| `term_versions` | CID | canonical term bytes | immutable |
| `terms` | term UUID | CID | pointer |
| `namespaces` | namespace `0x00` UUID | JSON `NamespacedPointer` | pointer |

Invariants enforced in code: `SetCurrent` refuses a CID not present in
`versions`; `putImmutable` returns `ErrImmutable` when a key would change
bytes and is a no-op for identical bytes; `PutAttestation` and
`PutTermRecord` reject malformed records; index keys use a `0x00` delimiter
that cannot appear in a UUID or namespace name, so prefix scans never leak
across neighbours (tests `TestAttestationsAppendOnlyAndIndexedByPremise`,
`TestNamespacedPointersAreAPointerLayer`).

Measured on the session container (`go test -bench`, 200 iterations, 4 vCPU):

| Operation | ns/op |
|---|---|
| PutCandidate (one fsync'd bbolt transaction) | ~243,000 |
| GetCandidate | ~12,400 |
| PutVersion (hash + transaction) | ~294,000 |
| ListAttestations (100 entries) | ~266,000 |

Writes are dominated by bbolt's per-transaction fsync. Batching several
records per transaction is the obvious lever if ingestion rate matters.

### 3.3 Tier 2 (`orbit/orbit.go`, `orbit/namespace.go`)

- `PutBlock` puts a raw block with SHA2-256 and re-verifies the CID Kubo
  returns against `ContentID`. A mismatch is an error, never a silent
  acceptance.
- `GetBlock` fetches by CID under a bounded context (10 s default). Not found
  or a timeout while the caller's context is still live maps to
  `storage.ErrNotFound`. Fetched bytes are re-hashed.
- Pointer and namespace stores are OrbitDB Documents stores. The write list is
  part of the manifest, so every peer opening the same store must use the same
  list; the main store exposes `Address()` and `Identity()` for that.
- Namespace stores are opened lazily and cached. OrbitDB's address is a hash
  of the manifest (name, type, access-controller address), so two nodes with
  the same write list reach the same namespace store by name alone;
  `orbit/namespace_test.go` asserts this on two nodes.

### 3.4 Interfaces

`storage.WorkingStore` and `storage.GlobalStore` are the tier boundary
(decision bqe.5). `promote` depends only on them, which is why
`storage/memory` can stand in for the global tier in unit tests and why a
different replication backend could replace go-orbit-db without touching
admission logic.

### 3.5 Error handling

- Every promotion refusal happens before the first write (verified by
  `assertUnchanged` in the promote tests).
- Tier-1 writes precede tier-2 writes so the working store never points at a
  version it does not hold.
- A failure between step 6 and step 7 of §2.2 leaves a published version with
  no local `current` pointer. Promotion is idempotent for identical bytes
  (`PutVersion`, `PutBlock` and `PutAttestation` are no-ops on repeat), so
  re-running the same promotion completes it. There is no automatic retry;
  see §6.
- Namespaced pointer writes after step 7 return a partial `Result` with the
  error; the record is published and current, and the namespace placement can
  be retried.

## 4. Scale and Reliability

### 4.1 Load estimate

Per promoted premise: one version block (typically 0.5 to 5 KB), one
attestation block (similar), one pointer entry, and one entry per namespace.
At 10^6 premises this is a few GB of blocks and ~10^6 OrbitDB log entries per
store, which is within what Kubo's flatfs or pebble datastores handle but is
past what a single OrbitDB Documents store loads quickly (it replays its log
into memory on open).

### 4.2 Scaling levers

- **Horizontal (by namespace).** Each namespace is already its own store and
  log. Splitting domains across namespaces bounds the per-store log length
  and lets a node subscribe to the domains it needs (storage-architecture.md
  §7).
- **Vertical.** bbolt and Kubo both benefit from SSD and memory; nothing in
  the design assumes a single thread.
- **Pointer store partitioning.** The main pointer store is the one unbounded
  log. Proposed: shard by UUID prefix or route all pointers through namespace
  stores and keep the main store as an index of namespaces. Revisit when a
  node's pointer log exceeds roughly 10^5 entries.

### 4.3 Failover and redundancy

Every node holds the blocks it fetched; any node that has a pointer store and
its referenced blocks can serve them. There is no primary. Pinning
(`Options.PinBlocks`) keeps blocks through garbage collection. The working
store is a single file; its durability is the host's.

### 4.4 Monitoring

Not implemented. The natural signals are: promotion count and refusal count by
reason; block put/get latency and not-found rate; OrbitDB replication lag
(time from a peer's pointer write to local visibility); working-store file
size and transaction latency.

## 5. Trade-offs Taken

| Decision | Benefit | Cost |
|---|---|---|
| CID as the local version identifier | One identifier across tiers; integrity check is free | Hashing on every version write |
| Canonical JSON via struct encoding | Zero dependencies, deterministic for Go writers | Not a cross-language canonical form (see §6) |
| Pointer is the only mutable record | Versions are trivially immutable and shareable | Readers do two lookups (pointer, then block) |
| One OrbitDB store per namespace | Natural sharding, per-domain subscription, convergent addresses | Open cost per namespace; many tiny namespaces are wasteful |
| Promotion writes tier 1 first | Local store never dangles | A crash mid-promotion needs a re-run |
| Attestation expiry checked at promotion | Stale evaluations cannot admit a record | A record admitted today can be past expiry tomorrow; re-validation is a separate concern |
| Delete candidate after promotion | Drafts and versions are distinct classes | Editing a promoted premise starts from `GetVersion`, not the draft |
| Only the root namespace ships | Structure stays the owner's call | Every deployment must define its own domains |

## 6. What to Revisit

1. **Canonical encoding.** If any non-Go writer must produce the same CIDs,
   adopt DAG-CBOR (already vendored through Kubo) or JSON Canonicalization
   Scheme (RFC 8785). Do this before any external writer exists; changing it
   later changes every CID.
2. **Pointer store growth.** Partition or route through namespaces once a log
   is large enough to make open time noticeable.
3. **Crash recovery for promotion.** Record an intent entry in the working
   store before step 6 of §2.2 and reconcile on open.
4. **Access control.** The write list is a static manifest. Multi-writer
   namespaces need either a shared list agreed out of band or an OrbitDB
   access controller that grants per key; this intersects decision ilv.
5. **Tier-1 batching.** Group writes per transaction if ingestion throughput
   matters.

## 7. Proposed Next Layers

These are designs, not implementations. Each is a separate Beads task.

### 7.1 Evaluation produces attestations (epic validatorium-th7)

An evaluator takes a premise version, resolves its evidence references,
executes the stated method, and returns a `ValidityAttestation`. The storage
layer needs nothing new; the contract is already `model.NewAttestation` plus
`promote.PromoteInto`. The design question is the evaluator interface:

```go
type Evaluator interface {
    // Evaluate executes the premise's method against its evidence and
    // the given scope. It never writes; the caller promotes.
    Evaluate(ctx, premise *model.Micropublication, scope map[string]any) (*model.ValidityAttestation, error)
}
```

Evaluators are method-specific (document inspection, dataset lookup,
reproducible computation, runtime observation) and are selected by the
method URI. A result of `indeterminate` is a legitimate outcome and is stored
in the attestation log without promotion.

### 7.2 Release manifests (decision z5a)

A release is an immutable block listing exact `VersionRef`s of premises,
terms, evidence, attestations, assets and artifacts (data-model.md §9), plus a
pointer in a `validatorium:releases` namespace keyed by release version.
Construction: for each candidate premise, require a current pointer, a true
attestation unexpired at release time, term records that resolve to
themselves (not obsolete), and dependencies that are themselves in the
release. Everything needed to implement this exists in the working store
API; the missing piece is the manifest record type and the builder.

### 7.3 Runtime / JIT re-validation (data-model.md §10)

A consumer calls an evaluator with a runtime scope before using a premise and
receives a new attestation. The attestation is appended locally
(`PutAttestation`) and published only if the user chooses
(distributed-datastore.md §8). No pointer changes; the premise and earlier
attestations are untouched. Triggers (expiry, recurrence windows, dependency
or evidence changes, term replacement) are policy over the attestation log and
term resolution, both of which are queryable today.
