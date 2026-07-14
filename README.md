


# Validatorium

![Validatorium](./assets/images/validatorium-panel.png)

A distributed knowledge base of evidence-grounded atomic premises about observable
reality.

## Purpose

Validatorium is being built to store factual premises that can be evaluated against
evidence. A premise is a single, unambiguous assertion whose significant terms are
grounded to versioned ontology records. A premise is valid when evaluation against
its evidence and applicable context returns true.

Debate, consensus, voting, credentials, popularity, and institutional endorsement
do not determine validity. Evidence does.

Validated premises can support encyclopedias, textbooks, reference databases,
fact-checking tools, educational software, and other systems that need factual
claims with explicit provenance.

## Atomic Premises

https://github.com/user-attachments/assets/eee02365-842b-41f9-92f3-6d03e8275578

An atomic premise:

- makes exactly one factual assertion;
- uses explicit scope and versioned ontology terms;
- links to evidence and a reproducible evaluation method;
- can be independently re-evaluated; and
- enters the validated store only when it evaluates true.

Examples include:

> The normal human diploid genome contains 46 chromosomes.

> Pure water boils at 100°C at standard atmospheric pressure.

> It is currently July.

The third example is temporal. It can be valid during July, inactive outside July,
and valid again the following July. Temporal inactivity does not make a premise
obsolete; obsolescence means that a record has been superseded by replacement
UUIDs.

## Releases and Runtime Validity

A Validatorium release references exact versions of its premises, ontology terms,
evidence, assets, and artifacts. A premise receives a release tag only when it is
valid and current for that release. Historical release tags preserve the exact
versions used.

Validity is not assumed to remain true forever. Premises may define expiry times,
TTLs, recurrence windows, or other re-evaluation conditions. Runtime consumers can
perform just-in-time re-validation before using a premise. Each check produces a
timestamped attestation containing its result, method, evidence, and scope without
mutating the underlying premise.

Versioned ontology mappings follow the same immutable record model. A superseded
term record is marked obsolete, remains addressable through IPFS, and points to its
replacement record UUIDs.

## Foundation for Automated Validation

Validatorium provides infrastructure on which higher-level auto-validation systems
can be built. A runtime evaluator can observe a local environment—such as compute
infrastructure, file contents, services, or instrument output—and evaluate a
premise immediately before use.

The same foundation can support automated hypothesis testing. An authorized
orchestrator can submit a hypothesis and method to an instrument such as a
lab-on-a-chip, collect timestamped observations, preserve method and instrument
provenance, and return the resulting evidence for premise evaluation. Instrument
control, safety policy, and experimental orchestration remain higher-level systems;
Validatorium supplies the versioned premises, evidence records, provenance, and
validity infrastructure beneath them.

## Technical Foundation

### Micropublication

Validatorium builds on the [Micropublication](http://purl.org/mp/) semantic model
(Clark et al., 2014). A micropublication represents a claim, its evidence,
attribution, and support or challenge relationships as structured data. These
relationships describe evidence; they are not social debate or voting mechanisms.

### Ontology Grounding

Every significant term resolves to a versioned term record from a versioned
ontology. [Wikidata](https://www.wikidata.org/) acts as a language-independent pivot
for linking surface forms to domain ontologies such as OBO, ChEBI, and UniProt. An
ontology identifies what a term means; it does not determine whether a premise is
true.

### Distributed Storage

The intended architecture combines a local bbolt working store with an embedded
Kubo/IPFS and go-orbit-db replication layer. UUIDs provide stable record identity;
content addressing provides integrity for each stored version.

## Current Status

Early development. The repository currently contains:

- a Go record and storage scaffold;
- a bbolt local store;
- embedded Kubo and go-orbit-db integration;
- a tested two-node replication and reconvergence spike; and
- an initial UUID-based micropublication model.

Truth evaluation, release manifests, runtime/JIT re-validation, temporal scheduling,
and instrument adapters are intended architecture and are not yet complete.

## Rights and Licensing

Factual propositions describe reality; Validatorium claims no ownership over facts.
Authored work logs retain their authorship and provenance. External ontologies,
standards, and datasets remain under the terms of their maintainers and are fully
cited.

The software licensing model is being finalized, with source-available commercial
licensing under consideration. No root software license has yet been granted.

