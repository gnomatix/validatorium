# Scalability Requirements — Validatorium Storage Layer

**Status:** Draft
**Last updated:** 2026-07-14

## Overview

This document specifies scalability requirements for Validatorium's distributed storage layer. The storage backend must work at scale — this is non-negotiable. All targets are measurable and must be validated by benchmarks before advancing between phases.

**Technology stack:**
- **Distributed storage:** go-orbit-db (or OrbitDB JS) on IPFS/libp2p
- **Text indexing:** FM-index, suffix arrays
- **Similarity indexing:** MinHash/LSH (drawn from genomic assembly algorithms)
- **Addressing:** UUID logical identity with content-addressed immutable versions

---

## 1. Scale Targets

Each phase must be validated before advancing to the next. There is no skipping.

| Phase | Premise Count | Topology | Success Criteria |
|-------|--------------|----------|-----------------|
| **Phase 1 — Prototype** | 10,000 | Single node | Concept proven. CRUD works. Index queries return correct results. |
| **Phase 2 — Local** | 100,000 | Single node | Query performance within targets below. No degradation cliffs. |
| **Phase 3 — Distributed** | 1,000,000+ | Multi-node | Replication working. Nodes sync correctly. CRDTs merge without conflicts. |
| **Phase 4 — Public** | 10,000,000+ | Open network | Any node can join. Domain sharding operational. Malicious input rejected. |

### Phase Gate Criteria

- **Phase 1 → 2:** All CRUD operations pass. FM-index and MinHash queries return correct results. No data corruption on restart.
- **Phase 2 → 3:** All performance targets met at 100K scale. Index build time acceptable. Storage within budget.
- **Phase 3 → 4:** Replication latency within target. Partition recovery verified. Concurrent writes handled correctly.
- **Phase 4:** Malformed input is rejected, and only premises with true validity attestations enter validated indexes. Domain sharding reduces per-node storage. Cold start completes in minutes.

---

## 2. Performance Requirements

All latency targets measured at p95 unless otherwise stated. Measured on commodity hardware (4-core, 16GB RAM, SSD).

| Operation | Target | Scale | Notes |
|-----------|--------|-------|-------|
| Premise lookup by UUID | < 10ms | Any scale | Current-version pointer lookup; must remain constant-time. |
| Term search (FM-index) | < 100ms | 1M premises | Single term query. Multi-term conjunction may be higher. |
| Similarity search (MinHash) | < 500ms | 1M premises | k-nearest neighbors. k ≤ 20. |
| Write (validated premise version) | < 1s | Any scale | Excludes evidence-evaluation time; includes structural checks and index update. |
| Replication to connected peer | < 5s | Any scale | From write-commit to availability on peer. |
| Cold start (domain sync) | < 10 minutes | 100K premises in domain | New node joins, syncs one domain to queryable state. |

### Degradation Policy

- Performance may degrade linearly with scale, never super-linearly.
- If any target is missed by >2x, the phase gate fails.
- UUID and CID lookup must remain O(1) regardless of premise count.

---

## 3. Storage Efficiency

### Premise Size Assumptions

- Typical premise: < 1KB (Argdown text + metadata + evidence URIs)
- Maximum premise size: 64KB (hard limit, reject anything larger)
- Evidence links are URIs stored as strings — the system does not store evidence content

### Storage Budget by Scale

| Scale | Raw Storage | Compressed Index Target | MinHash Signatures | Total Budget |
|-------|-------------|------------------------|-------------------|--------------|
| 10K premises | ~10MB | < 1MB | ~2.5MB | < 50MB |
| 100K premises | ~100MB | < 10MB | ~25MB | < 200MB |
| 1M premises | ~1GB | < 100MB | ~256MB | < 2GB |
| 10M premises | ~10GB | < 1GB | ~2.5GB | < 15GB |

### MinHash Signature Storage

- 256 bytes per premise (fixed-size signature)
- At 1M premises: 256MB — fits in memory on any modern machine
- At 10M premises: 2.5GB — requires memory-mapped access or domain sharding

### Index Storage

- FM-index compresses well (BWT-based)
- Suffix arrays: ~4 bytes per character in indexed text
- Index rebuild must be possible from raw premise data (indexes are derived, not canonical)

---

## 4. Benchmarking Requirements

All benchmarks must be automated, reproducible, and run as part of phase gate validation.

### 4.1 go-orbit-db Benchmarks

| Benchmark | Method | Pass Criteria |
|-----------|--------|---------------|
| Basic CRUD | Insert, read, update, delete 1000 records. Measure p50/p95/p99 latency. | Read < 10ms p95. Write < 100ms p95. |
| Document store at 10K records | Load 10K premises, run 100 random reads. | No degradation vs 1K baseline (< 2x). |
| Document store at 100K records | Load 100K premises, run 100 random reads. | Read < 10ms p95. |
| Document store at 1M records | Load 1M premises, run 100 random reads. | Read < 10ms p95. Write < 200ms p95. |
| Replication (two local nodes) | Write on node A, measure time to availability on node B. | < 5s for single premise. |
| Replication (cold sync) | Node B joins with 0 records. Node A has 100K. Measure full sync time. | < 10 minutes. |
| Concurrent writes | 10 goroutines writing simultaneously for 60s. Check for data loss or corruption. | Zero data loss. No unresolved conflicts. |

### 4.2 FM-Index Benchmarks

| Benchmark | Method | Pass Criteria |
|-----------|--------|---------------|
| Build time (100K terms) | Index 100K premise term sequences. Measure wall time. | < 30s. |
| Build time (1M terms) | Index 1M premise term sequences. Measure wall time. | < 5 minutes. |
| Query time (100K) | 1000 random single-term queries. Measure p95. | < 50ms p95. |
| Query time (1M) | 1000 random single-term queries. Measure p95. | < 100ms p95. |
| Incremental update | Add 100 new premises to existing 1M index. Measure time. | < 5s (or document rebuild strategy). |

### 4.3 MinHash/LSH Benchmarks

| Benchmark | Method | Pass Criteria |
|-----------|--------|---------------|
| Signature computation | Compute MinHash for 10K premises. Measure throughput. | > 1000 signatures/second. |
| Similarity query (100K) | k=10 nearest query against 100K signatures. Measure p95. | < 200ms p95. |
| Similarity query (1M) | k=10 nearest query against 1M signatures. Measure p95. | < 500ms p95. |
| LSH bucket distribution | Verify hash distribution is uniform across buckets at 1M scale. | No bucket > 10x average size. |

### 4.4 Identity and Version Storage Benchmarks

| Benchmark | Method | Pass Criteria |
|-----------|--------|---------------|
| CID existence check | Check if a version CID exists with 1M records. | < 1ms p95. |
| Deduplication | Insert same premise twice. Verify no duplication. | Exactly one record stored. |
| CID verification | Verify content identifiers for 10K premise versions. | > 50,000 versions/second. |

---

## 5. Failure Modes

Each failure mode must be tested with an automated scenario before the relevant phase gate.

### 5.1 Node Offline (Phase 3+)

- **Scenario:** Node B goes offline. Node A continues writing.
- **Expected:** Node A operates normally (local-first). Queries on A return correct results.
- **Verification:** Node A's read/write performance unchanged during B's absence.

### 5.2 Network Partition and Recovery (Phase 3+)

- **Scenario:** Two nodes write independently during partition. Partition heals.
- **Expected:** CRDT merge produces union of both nodes' writes. No data loss. No conflicts requiring manual resolution.
- **Verification:** After merge, both nodes return identical query results. Premise count = union of both sets.

### 5.3 Malicious Input (Phase 4)

- **Scenario:** A node sends malformed records or premise versions without true validity attestations.
- **Expected:** Malformed records are rejected. Unvalidated records do not enter the validated premise index. No partial writes or index corruption occur.
- **Verification:** Validated queries return only premises with applicable true attestations. Concurrent validated writes are unaffected.

### 5.4 Concurrent Writes (Phase 3+)

- **Scenario:** 10+ nodes write rapidly and simultaneously.
- **Expected:** OrbitDB CRDT log handles concurrent appends. All writes eventually visible on all nodes.
- **Verification:** After quiescence, all nodes agree on premise set. No lost writes.

### 5.5 Oversized Premise (All Phases)

- **Scenario:** Attempt to write a premise exceeding 64KB.
- **Expected:** Rejected by structural checks with a clear error. No partial storage.
- **Maximum size:** 64KB hard limit. Anything larger is structurally invalid.
- **Verification:** Oversized write returns error. Subsequent reads/writes unaffected.

---

## 6. Technology Evaluation Criteria

### 6.1 go-orbit-db Evaluation

Must be validated before committing to implementation:

| Criterion | Method | Fail Condition |
|-----------|--------|---------------|
| Compatibility with OrbitDB JS (Helia-based) | Attempt cross-implementation replication. | Cannot replicate between Go and JS nodes. |
| Maintenance status | Check last meaningful commit, open issues, maintainer responsiveness. | No meaningful commit in > 12 months. Unresponsive maintainers. |
| Document schema support | Implement Validatorium premise schema in go-orbit-db document store. | Cannot express required fields/types. |
| Performance at scale | Run benchmarks from §4.1. | Fails phase 2 performance targets. |
| CRDT correctness | Run concurrent write + partition tests from §5. | Data loss or unresolvable conflicts. |

### 6.2 Fallback: Custom go-libp2p + CRDT

If go-orbit-db fails evaluation:

- Build custom document store on go-libp2p
- Implement operation-based CRDT (append-only log with causal ordering)
- Use IPFS content addressing directly
- **Cost:** Higher implementation effort, but full control over storage semantics
- **Timeline impact:** +2-4 weeks for Phase 3

### 6.3 Fallback: Hypercore/Hyperswarm

If OrbitDB ecosystem entirely fails (both Go and JS):

- Hypercore provides append-only logs with replication
- Hyperswarm provides DHT-based peer discovery
- **Tradeoff:** Smaller ecosystem, less battle-tested for document stores
- **Constraint:** Primarily Node.js — may require polyglot architecture or Go bindings
- **Timeline impact:** +4-6 weeks, architecture rethink required

### 6.4 Decision Timeline

- go-orbit-db evaluation: complete within Phase 1
- Decision point: before Phase 2 implementation begins
- If fallback needed: document rationale, re-estimate Phase 2-4 timelines

---

## 7. Horizontal Scaling Strategy

No single node needs to store everything. The network scales by specialization.

### 7.1 Node Types

| Node Type | Stores | Indexes | Role |
|-----------|--------|---------|------|
| **Full node** | All premises for subscribed domains | Full FM-index + MinHash for those domains | Anchor node serving replicated domain data. |
| **Index node** | MinHash signatures + FM-index | Full indexes, premise content fetched on demand | Search infrastructure. Fast queries without full storage. |
| **Light node** | Own premises only + MinHash signatures | Local index only | Contributor node. Discovers related premises via MinHash, fetches on demand. |

### 7.2 Domain-Based Sharding

- Premises are tagged with domain(s) (e.g., `molecular-biology`, `organic-chemistry`)
- Nodes subscribe to domains they care about
- A node only replicates premises within its subscribed domains
- Cross-domain queries route to relevant domain nodes via DHT

### 7.3 Storage Budget by Node Type (at 10M total premises)

| Node Type | Subscribed Domains | Approximate Storage |
|-----------|-------------------|-------------------|
| Full node (1 domain, ~500K premises) | 1 | ~1GB |
| Full node (all domains) | All | ~15GB |
| Index node (all domains) | All | ~4GB (signatures + compressed index) |
| Light node | N/A | < 100MB (own premises + signatures) |

### 7.4 Query Routing

1. Local index checked first (always fast)
2. If result insufficient, query forwarded to domain's full/index nodes
3. DHT-based discovery for finding relevant domain nodes
4. Results deduplicated by logical UUID and exact version CID

---

## 8. Data Integrity

### 8.1 Identity and Content Addressing

- UUIDs identify logical premises and other records.
- CIDs identify immutable stored versions.
- Every retrieved payload must verify against its CID.
- A new version under the same UUID does not change logical identity.
- A semantic replacement receives a new UUID; the obsolete record points to its
  replacement UUIDs.

### 8.2 Immutability

Premise versions, evidence, release manifests, and validity attestations are
immutable. Updates append versions or records. Historical release manifests retain
exact UUID/CID pairs.

### 8.3 Signing

Published records may be signed for authentication and attribution. Signatures show
who produced a record; they do not make its factual contents true. Invalid claimed
signatures cause record rejection.

### 8.4 Verification Invariants

1. Every version payload matches its CID.
2. Every logical record has a valid UUID.
3. Every record passes structural schema and size checks before indexing.
4. Every premise in a validated index has an applicable true validity attestation.
5. Every release member matches the exact UUID/CID pair in its manifest.
6. Obsolete records remain addressable and replacement-pointer graphs are acyclic.
7. Every claimed signature verifies against the identified signer.

---

## Appendix: Benchmark Tooling

Benchmarks should be implemented as:

- Go benchmark tests (`testing.B`) for Go components
- Automated scripts that generate synthetic premise data at required scale
- CI-runnable for Phase 1-2 targets (single node)
- Manual execution acceptable for Phase 3-4 (multi-node) until infrastructure automation exists

### Synthetic Data Requirements

- Premises generated with realistic term distributions (Zipfian)
- Domain tags drawn from a fixed set (20 domains for testing)
- Evidence URIs generated as valid DOI/URL formats
- Argdown structure follows actual premise grammar
