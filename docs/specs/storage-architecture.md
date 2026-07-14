# Storage Architecture

**Status:** Draft
**Date:** 2026-07-14

## 1. Goals

Validatorium stores and distributes versioned premises, evidence, ontology-term
mappings, validity attestations, releases, assets, and artifacts while preserving
UUID identity and immutable version history.

Storage does not determine truth. Evaluation determines whether a premise is valid;
storage preserves the exact inputs and results.

## 2. Two Storage Tiers

### Tier 1: Local Working Store

bbolt provides the embedded local working store for:

- candidate records and local workflows;
- current UUID-to-version pointers;
- indexes and reverse references;
- evaluation queues and cached attestations; and
- release construction state.

### Tier 2: Distributed Store

Embedded Kubo/IPFS provides content-addressed immutable versions. go-orbit-db
provides replicated document/log behavior between nodes.

- UUIDs identify records.
- CIDs identify immutable versions.
- Obsolete records remain addressable and point to replacement UUIDs.
- Release manifests pin exact UUID/CID pairs.

## 3. Record Classes

- premises and premise versions;
- versioned ontology and term-mapping records;
- evidence and authored work logs;
- validity attestations;
- release manifests;
- referenced assets and artifacts; and
- current/obsolete pointer records.

## 4. Promotion

A local candidate is promoted to the validated distributed premise store only after
its evidence-based evaluation returns true. Structural checks run first but do not
establish truth.

Promotion records:

1. the premise UUID and immutable version CID;
2. exact term-mapping versions;
3. exact evidence and method versions;
4. the true validity attestation; and
5. provenance and attribution.

False or indeterminate evaluations remain local evaluation records unless a user
explicitly publishes them as evidence/work artifacts; they are not valid premises.

## 5. Releases

Release construction selects premises that are valid and current at the release
time. It rechecks ontology replacement pointers, dependencies, expiry, TTL, and
recurrence policies, then writes an immutable release manifest.

A historical release never changes. Later premise, term, evidence, asset, or
artifact versions appear only in later releases.

## 6. Runtime Validity

A consumer can request just-in-time re-validation before using a premise. Runtime
evaluators may inspect local files, services, compute infrastructure, sensors, or
other authorized observable state. Every check appends an immutable attestation.

Runtime state and private observations remain local unless the user explicitly
publishes them.

## 7. Replication

go-orbit-db replicates immutable records and pointer updates. IPFS retrieves content
by CID. Replication convergence means nodes obtain the same published records; it
is not a vote or consensus mechanism for truth.

Nodes may subscribe to selected domains and retrieve other content on demand.
Offline nodes continue to use local records and reconcile published updates when
connectivity returns.

## 8. Integrity and Security

- Verify CIDs for all retrieved content.
- Authenticate signed records where signatures are provided.
- Preserve source licenses, citations, and authorship.
- Validate record size and schema before indexing.
- Treat remote content as untrusted input.
- Require explicit authorization for local probes or instrument access.
- Never place local credentials or private runtime state in replicated records.

## 9. Implementation Status

Implemented now:

- bbolt local-store scaffold;
- embedded Kubo startup;
- go-orbit-db integration; and
- a tested two-node replication and reconvergence spike.

Promotion enforcement, release manifests, runtime/JIT evaluation, ontology update
processing, and instrument adapters remain intended architecture.
