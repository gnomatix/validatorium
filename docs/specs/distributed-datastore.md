# Distributed Datastore Specification

**Status:** Draft
**Date:** 2026-07-14

## 1. Scope

The distributed datastore replicates published Validatorium records. It preserves
immutable versions and provenance; it does not decide whether a premise is true.

## 2. Components

- **Kubo/IPFS:** content-addressed storage and retrieval by CID
- **go-orbit-db:** replicated document/log stores
- **libp2p:** peer connectivity supplied through Kubo
- **bbolt:** local working state and indexes

The intended deployment is a single Go process containing the CLI, bbolt, embedded
Kubo, and go-orbit-db.

## 3. Addressing

Every logical record has a UUID. Every immutable version has a CID. Current-record
indexes map UUIDs to current version CIDs. Historical CIDs remain resolvable.

Premises use versioned term-record UUIDs from versioned ontologies. Obsolete records
remain available and point to replacement UUIDs.

## 4. Published Records

Replicated record classes include:

- valid premise versions;
- evidence and provenance records;
- versioned ontology/term mappings;
- validity attestations;
- release manifests; and
- published assets and artifacts.

Candidate submissions do not become distributed valid premises merely because they
are structurally well-formed. Promotion requires an evidence-based evaluation that
returns true.

## 5. Stores

Separate logical stores may be used for premises, evidence, mappings, attestations,
releases, and pointer records. Store names and access policies are versioned
configuration.

Document keys use stable UUIDs where a current logical record is required. Immutable
payloads and release members reference exact CIDs.

## 6. Replication

Published operations are append-only or version-creating. Nodes replicate records,
verify CIDs, and update local indexes. Partitioned nodes retain local availability
and converge after reconnection.

Replication agreement is technical convergence over published bytes. It is not
social consensus, voting, or evidence of factual validity.

## 7. Release Retrieval

A release manifest is the reproducible retrieval unit. It lists exact premise,
term, evidence, asset, artifact, and attestation versions. A consumer can retrieve
and verify those CIDs independently of later updates.

## 8. Runtime/JIT Evaluation

Runtime observations and attestations are local by default. A user may publish an
attestation and its permitted evidence, but private paths, credentials, machine
identifiers, and restricted data must not be replicated implicitly.

## 9. Security

- Treat all peer data as untrusted.
- Enforce schema and size limits before indexing.
- Verify content identifiers.
- Preserve signatures and authorship without assigning authority weight.
- Apply store access controls for non-public namespaces.
- Require user authorization for publication and local observation.

## 10. Current Implementation

The repository contains embedded Kubo/go-orbit-db integration and a two-node
replication/reconvergence spike. Production store separation, promotion, releases,
access policy, and runtime-attestation publication are not yet complete.
