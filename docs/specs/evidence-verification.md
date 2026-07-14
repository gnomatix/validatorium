# Evidence and Verification Specification

## 1. Verification Standard

A premise is valid when it can be evaluated to true against observable evidence and
its applicable context. The method is always evidence-based verification: identify
what can be observed, examine it, and record how the result was obtained.

The domain is observable reality, not a particular academic discipline. A
measurement, document, database record, file, service response, computation,
historical artifact, or instrument observation can serve as evidence when it is
retrievable and its relevance can be checked.

## 2. Evidence Requirements

Evidence is an observable record that can be independently examined.

| Evidence type | Examples |
|---|---|
| Measurement or observation | Experimental data, sensor output, field observation |
| Published study | Identified by DOI, PMID, or another persistent identifier |
| Dataset or database record | NCBI record, PubChem measurement, Wikidata statement |
| Public record | Civil record, court document, patent, census record |
| Artifact | Standard, dictionary, codebase, configuration file |
| Reproducible computation | Program, inputs, environment, and resulting output |
| Runtime observation | File digest, service response, infrastructure state |
| Instrument output | Timestamped result with method, calibration, and instrument provenance |

Every evidence record identifies:

- its stable URI or content identifier;
- source type and provenance;
- the exact material relevant to the premise;
- whether it supports or challenges the premise;
- the method used to inspect or produce it;
- applicable time, environment, and scope; and
- the last successful accessibility and integrity check.

Attribution identifies who created or examined a record. Credentials and reputation
add no evidential weight.

## 3. Evidence-Link Checks

For each link:

1. **Resolution:** Can the referenced source be retrieved?
2. **Authenticity and integrity:** Is it the identified source, and does its digest
   or signature match when available?
3. **Relevance:** Does it address the exact scoped premise?
4. **Relationship:** Does the relevant material support or challenge that premise?
5. **Reproducibility:** Can the stated method be repeated from the recorded inputs
   and context?

A failed evidence link is marked inapplicable with the reason. It is not kept active
until someone produces an opposing source.

## 4. Premise Evaluation

Structural validation checks atomicity, schema, scope, term grounding, identifiers,
and evidence-link shape. It does not establish truth.

Semantic evaluation executes the premise's method against its evidence and context.
It produces an immutable attestation:

```yaml
ValidityAttestation:
  premise_uuid: uuid
  premise_version: cid
  result: true | false | indeterminate
  evaluated_at: datetime
  evaluator: string
  method: uri-or-cid
  evidence: [uri-or-cid]
  scope: object
  expires_at: datetime | null
  dependencies: [uuid-or-cid]
```

- `true` means the premise is valid for the recorded scope and evaluation time.
- `false` means it does not evaluate true for that scope and time.
- `indeterminate` means the available evidence or context cannot establish a
  Boolean result.

Only a premise with a `true` attestation enters the validated store or receives a
valid/current release tag. False and indeterminate attestations remain part of the
evaluation history; they are not valid premise states.

Support and challenge relations describe evidence. They are not participants,
sides, votes, or competing opinions.

## 5. Release and Runtime Validity

Every Validatorium release references exact premise, ontology-term, evidence,
method, asset, and artifact versions. A premise is tagged for a release only when it
is valid and current under those versions.

Runtime consumers may re-evaluate just in time before using a premise. Re-evaluation
is triggered by policy or by changes such as:

- expiry or TTL;
- entry into or exit from a recurrence window;
- dependency updates;
- new supporting or challenging evidence;
- ontology-term replacement; or
- a current observation requested by the consumer.

Each check adds an attestation. It never mutates the premise or an earlier release.

## 6. Temporal Premises

Temporal facts use explicit valid-time context. A state may be represented as an
interval—for example, a person's alive interval can be evaluated from birth and
death records. Release and runtime evaluators test whether the applicable time falls
inside the interval.

Premises may also be recurring. “It is currently July” evaluates true during each
applicable July interval and false outside it. The same premise UUID can be valid in
multiple recurrence windows. Inactivity outside a window is not obsolescence;
obsolescence means supersession by replacement UUIDs.

## 7. Automated Evaluation and Experimentation

Validatorium supplies the premise, evidence, provenance, versioning, and attestation
layer for higher-level automation.

A local evaluator can inspect files, services, compute infrastructure, or sensors
and return a runtime observation. An automated hypothesis-testing system can use an
authorized orchestrator to execute a method through an instrument such as a
lab-on-a-chip, collect observations, and submit them as evidence. Safety controls,
hardware authorization, and experiment orchestration remain outside Validatorium;
the resulting evidence is evaluated under the same rules as any other evidence.

## 8. “Go Look” Principle

Do not substitute authority, consensus, popularity, or rhetoric for evidence. Find
the observable source, inspect the relevant material, execute the method, and record
the result. If the premise cannot be evaluated to true, it does not enter the
validated premise store.
