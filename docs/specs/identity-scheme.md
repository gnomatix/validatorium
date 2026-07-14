# Identity and Versioning Specification

**Status:** Draft
**Date:** 2026-07-14

## 1. Identity Rules

Validatorium uses UUIDs for stable record identity.

- Premises, term mappings, evidence records, attestations, assets, and artifacts
  receive UUIDs.
- A UUID identifies the continuing record.
- A CID identifies the immutable bytes of one version of that record.
- Content hashes provide integrity and deduplication; they are not logical identity.

## 2. Record Versions

```yaml
VersionedRecord:
  uuid: uuid
  version_cid: cid
  previous_version_cid: cid | null
  created_at: datetime
```

Updating a record creates a new immutable version under the same UUID when its
identity has not changed. Historical CIDs remain addressable through IPFS.

## 3. Supersession

A semantic replacement, split, or merge creates new UUIDs. The superseded record is
marked obsolete and points to all replacement UUIDs.

```yaml
Lifecycle:
  state: current | obsolete
  replacement_uuids: [uuid]
```

Consumers can cite the obsolete UUID or any historical CID directly, or follow the
replacement pointers to current records.

## 4. Premise Identity

A premise UUID is assigned at creation. Wording, evidence, term mappings, or other
record details may receive new versions without changing that UUID when the factual
identity remains the same.

When refinement changes the factual identity—or one premise is decomposed into
several independently evaluable premises—the original becomes obsolete and points
to the replacement premise UUIDs.

Statement text is not hashed to create premise identity. Two similar or identical
strings can differ in scope, term grounding, evidence, method, or temporal context.
Deduplication is an evaluation and indexing concern, not an identity rule.

## 5. Versioned Ontology Terms

Premises reference versioned term-record UUIDs from versioned ontology records.
When a term mapping is replaced, its old record becomes obsolete and points to the
new term-record UUIDs. Old records remain addressable.

## 6. Releases

A release manifest pins exact UUID/CID pairs for premises, terms, evidence, assets,
and artifacts. Release tags do not move. A current-record index may advance to
newer versions without changing a historical release.

## 7. Runtime Attestations

Every release-time or runtime evaluation receives its own UUID and immutable CID.
An attestation identifies the exact premise version, evidence versions, term
records, method, context, evaluation time, and result. New checks append new
attestations rather than editing old results.

## 8. Serialization

UUIDs use canonical lowercase RFC 4122 text. CIDs use canonical multibase encoding.
References must state whether they identify a logical record UUID or a specific
version CID; the two are never interchangeable.
