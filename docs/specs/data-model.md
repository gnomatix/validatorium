# Data Model Specification

**Status:** Draft
**Date:** 2026-07-14

## 1. Core Invariant

A Validatorium premise is an atomic factual assertion about observable reality. It
is valid when its evaluation method returns true against its evidence and applicable
context. Only valid premises enter the validated premise store.

Record validation establishes that a record is well-formed. It does not
establish truth.

## 2. Premise Identity and Versions

Every premise receives a UUID when created. The UUID is stable across versions of
the same premise. Each immutable stored version has a content identifier (CID) for
integrity and distribution.

```yaml
Premise:
  uuid: uuid
  version_cid: cid
  statement: string
  terms: [TermGrounding]
  support_graph: [EvidenceRelation]
  challenge_graph: [EvidenceRelation]
  attribution: Attribution
  temporal_policy: TemporalPolicy | null
  lifecycle: Lifecycle
```

A content hash is not premise identity. It identifies the bytes of one stored
version.

## 3. Statement

A statement:

- contains one independently evaluable factual assertion;
- has explicit scope and conditions;
- uses versioned ontology terms for significant concepts; and
- does not hide additional premises or inferences.

A derived statement may become a premise only when its dependencies, method,
context, and evaluation are recorded and the method returns true.

## 4. Versioned Term Grounding

```yaml
TermGrounding:
  surface_form: string
  term_record_uuid: uuid
  ontology_record_uuid: uuid
  source_uri: uri
  scope_note: string | null
```

Term records and ontology records are versioned. When a mapping is superseded, the
old record is marked obsolete, remains addressable through IPFS, and stores pointers
to replacement record UUIDs. Historical premise versions retain their exact term
record references.

Ontologies define what terms mean. They do not determine whether premises are true.

## 5. Evidence and Micropublication Relations

Validatorium uses the Micropublication model for claims, attribution, and evidence
relationships.

```yaml
EvidenceRecord:
  uuid: uuid
  version_cid: cid
  source: uri | cid
  type: string
  method: uri | cid
  observed_at: datetime | null
  scope: object | null
  provenance: object
  attribution: Attribution

EvidenceRelation:
  evidence_uuid: uuid
  relationship: supports | challenges
  relevance: string
```

Support and challenge are evidence relationships, not debate positions. Attribution
records authorship and provenance; it adds no evidential weight.

## 6. Validity Attestations

Every evaluation creates an immutable attestation:

```yaml
ValidityAttestation:
  uuid: uuid
  premise_uuid: uuid
  premise_version_cid: cid
  result: true | false | indeterminate
  evaluated_at: datetime
  evaluator: string
  method: uri | cid
  evidence_versions: [uri | cid]
  term_record_uuids: [uuid]
  scope: object
  dependencies: [uuid | cid]
  expires_at: datetime | null
```

A `true` result establishes validity only for the recorded versions, scope, and
evaluation time. False and indeterminate attestations remain in the evaluation log
but do not create valid premise states.

## 7. Lifecycle

```yaml
Lifecycle:
  state: current | obsolete
  replacement_uuids: [uuid]
```

- `current` means the premise has not been superseded.
- `obsolete` means it was superseded and points to replacement UUIDs.

Temporal inactivity, expiry, or a false runtime result does not make a premise
obsolete.

## 8. Temporal Policy

```yaml
TemporalPolicy:
  valid_time: interval | null
  expires_at: datetime | null
  ttl: duration | null
  recurrence: object | null
  timezone: string | null
  calendar: string | null
```

Temporal states can be represented as intervals. For example, a person's alive
state may be evaluated over `[birth_time, death_time)`. Recurring premises can be
valid in multiple windows. “It is currently July” evaluates true during each
applicable July interval and is inactive between them.

## 9. Releases

```yaml
ReleaseManifest:
  version: string
  created_at: datetime
  premise_versions: [uuid + cid]
  term_versions: [uuid + cid]
  evidence_versions: [uuid + cid]
  asset_versions: [uuid + cid]
  artifact_versions: [uuid + cid]
  validity_attestations: [uuid]
```

A premise receives a release tag only when it is valid and current for that release.
Release construction checks expiry, TTL, recurrence, dependencies, evidence, and
term replacement pointers. Historical releases remain immutable and reproducible.

## 10. Runtime/JIT Re-validation

Consumers may request evaluation immediately before using a premise. Runtime checks
can observe local files, services, compute infrastructure, sensors, or other scoped
state. The result is a new attestation; the premise and prior attestations remain
unchanged.

## 11. Automated Experimentation

Higher-level systems can use Validatorium as the premise and evidence layer for
automated hypothesis testing. An authorized orchestrator may execute a method
through an instrument, collect observations, and submit provenance-preserving
evidence for evaluation. Instrument control and safety remain outside the core data
model.

## 12. Authorship and External Sources

Factual propositions are not owned by Validatorium. Authored methods, observations,
work logs, annotations, and analyses retain their authorship. External ontologies,
standards, and datasets retain their source licenses, versions, maintainers, and
citations.
