# Argdown Conventions for validatorium

> Version: 0.1.0 | Status: Draft | Last updated: 2026-07-14

This document defines how the `validatorium` project uses [Argdown](https://argdown.org) notation for representing premises, evidence, and arguments. It covers syntax conventions, metadata handling, file organization, and integration with the Argdown toolchain.

---

## 1. Premise Representation

A **premise** is an atomic propositional claim represented as an Argdown statement.

### Syntax

```argdown
[Premise Title]: Statement text.
```

### Rules

| Rule | Description |
|------|-------------|
| **Atomicity** | Each premise expresses exactly one claim. No conjunctions or disjunctions. |
| **Formal wording** | Use precise, domain-appropriate language. Avoid hedging or colloquial phrasing. |
| **Domain scoping** | The statement must be interpretable within a single knowledge domain. |
| **Title convention** | Title is a concise noun-phrase descriptor (2–5 words). Use Title Case. |
| **Punctuation** | Statement text ends with a period. |

### Examples

```argdown
[Human Diploid Count]: The normal human diploid genome contains 46 chromosomes.

[Earth Orbital Period]: Earth completes one orbit around the Sun in approximately 365.25 days.

[Water Boiling Point STP]: Pure water boils at 100°C at standard atmospheric pressure.
```

### Anti-patterns

```argdown
// BAD: Compound claim (two assertions joined by "and")
[Cell Facts]: Human cells contain 46 chromosomes and are eukaryotic.

// BAD: Vague title
[Thing]: The normal human diploid genome contains 46 chromosomes.

// BAD: Hedged/colloquial wording
[Human Diploid Count]: Humans pretty much have 46 chromosomes usually.
```

---

## 2. Metadata Convention

Metadata is attached to premises and evidence records using YAML front matter and
Argdown data attributes.

### 2.1 YAML Front Matter

```argdown
---
domain: molecular_biology
subdomain: cytogenetics
version: 1
created: 2026-03-15
ontology_release: "obo:2026-03"
---

[Human Diploid Count]: The normal human diploid genome contains 46 chromosomes.
```

### 2.2 Premise Metadata

```argdown
[Human Diploid Count]: The normal human diploid genome contains 46 chromosomes. {
  premise_uuid: "018f...",
  term_record_uuids: ["term-uuid-1", "term-uuid-2"],
  evidence_links: ["Tjio-Levan 1956", "Modern karyotyping"],
  evaluation_method: "cid:method-version",
  release_tags: ["validatorium-2026.07"]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `premise_uuid` | UUID | Yes | Stable premise identity |
| `term_record_uuids` | UUID[] | Yes | Versioned ontology-term records |
| `evidence_links` | string[] | Yes | Evidence records used by the evaluation method |
| `evaluation_method` | URI/CID | Yes | Reproducible method that determines whether the premise evaluates true |
| `release_tags` | string[] | No | Releases in which the premise was valid and current |
| `expires_at` | datetime | No | Optional re-evaluation boundary |
| `recurrence` | object | No | Optional recurring validity schedule |

Subjective confidence, consensus, popularity, and authority are not premise
metadata. Evaluation results belong in immutable validity attestations.

---

## 3. Evidence Representation

Evidence items use Argdown argument syntax because it provides readable support and
attack edges. These edges encode evidential relationships, not debate positions.

```argdown
<Evidence Tag> {
  evidence_uuid: "evidence-uuid",
  source: "URI-or-CID",
  type: "measurement",
  observed_at: "2026-03-15T10:30:00Z",
  method: "URI-or-CID"
}
```

Evidence types include measurements, observations, publications, records,
datasets, computations, runtime observations, and instrument output. Every record
preserves source, method, scope, time, and provenance.

```argdown
[Human Diploid Count]: The normal human diploid genome contains 46 chromosomes. {
  premise_uuid: "018f...",
  term_record_uuids: ["term-uuid-1", "term-uuid-2"],
  evaluation_method: "cid:chromosome-count-method"
}
  + <Tjio-Levan 1956> {
      evidence_uuid: "evidence-uuid-1",
      source: "PMID:13369537",
      type: "empirical_observation",
      method: "improved hypotonic pretreatment and squash technique"
    }
  + <Modern karyotyping> {
      evidence_uuid: "evidence-uuid-2",
      source: "DOI:10.1038/ng1005-1068",
      type: "replication"
    }
```

### Evidence Argument Bodies (Optional)

When an evidence argument needs its own internal structure, define it with premises:

```argdown
<Tjio-Levan 1956> {source: "PMID:13369537", type: "empirical_observation"}
  (1) Tjio and Levan used improved hypotonic pretreatment and squash techniques.
  (2) They observed 46 chromosomes in human embryonic lung fibroblasts.
  (3) The count was confirmed across multiple cell preparations.
  ----
  (4) The human diploid chromosome number is 46.
```

---

## 4. Relations Between Premises

### 4.1 Support (+)

A premise or evidence item supports another premise:

```argdown
[Mammalian Cell Division]: Mammalian somatic cells divide by mitosis.
  + [Human Diploid Count]
  + <Observed cell divisions> {source: "cid:observation-set", type: "empirical_observation"}
```

### 4.2 Attack (-)

A premise or evidence item contradicts another:

```argdown
[Flat Earth Candidate]: The Earth is flat. {premise_uuid: "candidate-uuid"}
  - <Satellite imagery> {source: "NASA:ISS", type: "empirical_observation"}
  - [Earth Shape Measurements]
```

### 4.3 Refinement

A premise refines or narrows another. Marked with `relation: "refines"` in the data attributes of the supporting connection, or by using the `#refinement` tag:

```argdown
[Human Diploid Count]: The normal human diploid genome contains 46 chromosomes.

[Human Autosome Count]: The normal human genome contains 22 pairs of autosomes. #refinement
  + [Human Diploid Count]
```

Or with explicit metadata on the relation (using a comment convention):

```argdown
[Human Autosome Count]: The normal human genome contains 22 pairs of autosomes.
  +> [Human Diploid Count] <!-- relation: "refines" -->
```

### 4.4 Temporal Incompatibility

When two premises cannot be simultaneously true due to temporal constraints, use the `#temporal_incompatibility` tag and a dedicated argument:

```argdown
[Jesus Birth Era]: The historical Jesus of Nazareth was born circa 4 BCE. #dated_claim

[Francis Birth Date]: Jorge Mario Bergoglio (Pope Francis) was born on 17 December 1936. #dated_claim

<Temporal exclusion: Jesus-Francis> #temporal_incompatibility {
  type: "logical_derivation",
  rule: "temporal_non_overlap",
  window_years: 1940
}
  (1) [Jesus Birth Era]
  (2) [Francis Birth Date]
  (3) Two distinct persons born >1900 years apart cannot be the same individual.
  ----
  (4) The historical Jesus and Pope Francis are distinct persons.
```

### 4.5 Relation Summary

| Prefix/Tag | Meaning | Example |
|------------|---------|---------|
| `+` | Supports | `+ [Other Premise]` |
| `-` | Attacks/contradicts | `- [Contradicting Premise]` |
| `#refinement` | Narrows/specifies | Tag on the refining premise |
| `#temporal_incompatibility` | Cannot co-hold due to time | Tag on evaluating argument |
| `#supersedes` | Replaces outdated premise | Tag on the newer premise |

---

## 5. Term Annotation Convention

Domain terms within premises can be grounded to ontology URIs using three approaches (in order of preference):

### 5.1 Inline Term Annotations (Preferred)

Use Argdown's tag system combined with a terms block in the YAML front matter:

```argdown
---
domain: molecular_biology
terms:
  diploid:
    uri: "http://purl.obolibrary.org/obo/PATO_0001393"
    ontology: "PATO"
    label: "diploid"
  chromosome:
    uri: "http://purl.obolibrary.org/obo/GO_0005694"
    ontology: "GO"
    label: "chromosome"
  Homo_sapiens:
    uri: "http://purl.obolibrary.org/obo/NCBITaxon_9606"
    ontology: "NCBITaxon"
    label: "Homo sapiens"
---

[Human Diploid Count]: The normal human diploid genome contains 46 chromosomes. {
  grounding: ["PATO:0001393", "GO:0005694", "NCBITaxon:9606"]
}
```

### 5.2 HTML Comment Annotations

For inline, human-readable annotations that don't affect parsing:

```argdown
[Human Diploid Count]: The normal human <!-- term: Homo_sapiens, uri: NCBITaxon:9606 -->
diploid <!-- term: diploid, uri: PATO:0001393 --> genome contains 46
chromosomes <!-- term: chromosome, uri: GO:0005694 -->.
```

### 5.3 Companion `.terms.yaml` File

For complex domains, a sidecar file provides full term definitions:

```
premises/
  molecular_biology/
    cytogenetics.argdown
    cytogenetics.terms.yaml
```

`cytogenetics.terms.yaml`:

```yaml
terms:
  - id: "diploid"
    label: "diploid"
    uri: "http://purl.obolibrary.org/obo/PATO_0001393"
    ontology: "PATO"
    definition: "A ploidy quality inhering in a cell or organism with two complete sets of chromosomes."

  - id: "chromosome"
    label: "chromosome"
    uri: "http://purl.obolibrary.org/obo/GO_0005694"
    ontology: "GO"
    definition: "A structure composed of a very long molecule of DNA and associated proteins."

  - id: "Homo_sapiens"
    label: "Homo sapiens"
    uri: "http://purl.obolibrary.org/obo/NCBITaxon_9606"
    ontology: "NCBITaxon"
    definition: "The species Homo sapiens."
```

### 5.4 Fully Annotated Premise Example

```argdown
---
domain: molecular_biology
subdomain: cytogenetics
ontology_release: "obo:2026-03"
terms:
  diploid: {record_uuid: "term-uuid-1", uri: "PATO:0001393"}
  chromosome: {record_uuid: "term-uuid-2", uri: "GO:0005694"}
  Homo_sapiens: {record_uuid: "term-uuid-3", uri: "NCBITaxon:9606"}
---

[Human Diploid Count]: The normal human diploid genome contains 46 chromosomes. {
  premise_uuid: "premise-uuid",
  term_record_uuids: ["term-uuid-1", "term-uuid-2", "term-uuid-3"],
  evidence_links: ["Tjio-Levan 1956", "Modern karyotyping"],
  evaluation_method: "cid:chromosome-count-method",
  release_tags: ["validatorium-2026.07"]
}
  + <Tjio-Levan 1956> {source: "PMID:13369537", type: "empirical_observation"}
  + <Modern karyotyping> {source: "DOI:10.1038/ng1005-1068", type: "replication"}
```

---

## 6. File Organization

### 6.1 Strategy: Domain-Grouped Files

Premises are grouped by domain into `.argdown` files. Each file contains related premises that share a domain context and term vocabulary. A single file should not exceed ~50 premises to remain manageable.

### 6.2 Directory Structure

```
validatorium/
├── premises/
│   ├── biology/
│   │   ├── cytogenetics.argdown
│   │   ├── cytogenetics.terms.yaml
│   │   ├── evolution.argdown
│   │   └── molecular_biology.argdown
│   ├── physics/
│   │   ├── thermodynamics.argdown
│   │   └── mechanics.argdown
│   ├── history/
│   │   ├── ancient_near_east.argdown
│   │   ├── roman_empire.argdown
│   │   └── modern_papacy.argdown
│   └── _meta/
│       ├── temporal_relations.argdown
│       └── cross_domain_arguments.argdown
├── evidence/
│   ├── biology/
│   │   └── cytogenetics_evidence.argdown
│   └── history/
│       └── dating_evidence.argdown
├── arguments/
│   ├── evaluations/
│   │   └── jesus_francis_temporal.argdown
│   └── derivations/
│       └── chromosome_implications.argdown
└── docs/
    └── specs/
        └── argdown-conventions.md  (this file)
```

### 6.3 Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Directory | `snake_case`, domain name | `molecular_biology/` |
| Premise file | `snake_case.argdown` | `cytogenetics.argdown` |
| Terms sidecar | `<same_name>.terms.yaml` | `cytogenetics.terms.yaml` |
| Evidence file | `<domain>_evidence.argdown` | `cytogenetics_evidence.argdown` |
| Argument file | descriptive `snake_case.argdown` | `jesus_francis_temporal.argdown` |
| Cross-domain | placed in `_meta/` | `_meta/temporal_relations.argdown` |

### 6.4 When to Split Files

- **New file**: When adding premises in a new subdomain
- **Split existing**: When a file exceeds 50 premises or covers multiple subdomains
- **Cross-domain arguments**: Always in `arguments/` or `_meta/`
- **Evidence-heavy premises**: Evidence bodies go in `evidence/` directory; premise files use references

### 6.5 File Header Template

Every `.argdown` file must begin with:

```argdown
---
domain: <domain>
subdomain: <subdomain>
version: <integer>
created: <ISO date>
maintainer: <handle>
description: "Brief description of the premises in this file"
---
```

---

## 7. Integration with Argdown Tooling

### 7.1 Existing Argdown Features We Use

The validatorium relies on the following standard Argdown parser features:

| Feature | Usage |
|---------|-------|
| **Statements** `[Title]: text` | All premises |
| **Arguments** `<Tag>` | All evidence items and evaluative arguments |
| **Support** `+` | Evidence supporting premises, premise-premise support |
| **Attack** `-` | Contradictions, counter-evidence |
| **Tags** `#tag` | Categorization (`#refinement`, `#temporal_incompatibility`, `#dated_claim`) |
| **Data attributes** `{key: "val"}` | All structured metadata |
| **YAML front matter** | File-level configuration and term definitions |
| **Argument reconstruction** `(1)...(n)` | Evidence body structure |
| **Inference** `----` | Logical step separator in arguments |

### 7.2 Extensions Required

The standard Argdown parser supports data attributes as opaque key-value pairs. We require additional tooling for:

| Extension | Purpose | Implementation |
|-----------|---------|----------------|
| **Metadata validation** | Ensure required fields are present and correctly typed | Custom Argdown plugin |
| **Term resolution** | Resolve `grounding` CURIEs to full URIs via front matter or sidecar | Post-parse transformer |
| **Cross-file references** | Resolve `[Premise Title]` references across files | Multi-file loader plugin |
| **Evidence aggregation** | Collect all evidence for a premise across `evidence/` directory | Custom query tool |
| **Confidence propagation** | Compute derived confidence from evidence strengths | Reasoning engine plugin |
| **Export to JSON-LD** | Serialize premises + metadata as linked data | Export plugin |

### 7.3 Node.js Module Architecture

```
src/
├── parser/
│   ├── index.ts              # Re-exports @argdown/core with our plugins
│   ├── metadata-plugin.ts    # Validates and normalizes metadata attributes
│   ├── term-resolver.ts      # Resolves CURIE → full URI
│   └── multi-file-loader.ts  # Loads and links across .argdown files
├── query/
│   ├── premise-query.ts      # Find premises by domain, status, term
│   ├── evidence-query.ts     # Find evidence for a given premise
│   └── relation-query.ts     # Traverse support/attack graph
├── transform/
│   ├── to-jsonld.ts          # Export to JSON-LD
│   ├── to-graph.ts           # Export to property graph (Neo4j compatible)
│   └── from-yaml.ts          # Import from structured YAML sources
└── validate/
    ├── schema.ts             # JSON Schema for metadata fields
    └── lint.ts               # Argdown linting rules (atomicity, title format, etc.)
```

### 7.4 Key Dependencies

```json
{
  "@argdown/core": "^1.7.0",
  "@argdown/node": "^1.7.0",
  "@argdown/map-views": "^1.7.0"
}
```

### 7.5 Parser Usage Example

```typescript
const result = await app.run({
  process: ["parse-input", "build-model", "metadata-plugin", "term-resolver"],
  input: `
    [Human Diploid Count]: The normal human diploid genome contains 46 chromosomes. {
      premise_uuid: "premise-uuid",
      term_record_uuids: ["term-uuid-1", "term-uuid-2"]
    }
  `
});

for (const statement of result.statements) {
  console.log(statement.title);
  console.log(statement.text);
  console.log(statement.data.premise_uuid);
  console.log(statement.data.term_record_uuids);
}
```

---

## 8. Complete Example: Temporal Evaluation

This example derives a time-scoped premise from versioned birth and death records.

```argdown
[Ada Birth]: Ada Lovelace was born on 10 December 1815. {
  premise_uuid: "birth-premise-uuid",
  term_record_uuids: ["ada-term-uuid", "birth-term-uuid"],
  evidence_links: ["Ada birth record"],
  evaluation_method: "cid:document-date-check"
}

[Ada Death]: Ada Lovelace died on 27 November 1852. {
  premise_uuid: "death-premise-uuid",
  term_record_uuids: ["ada-term-uuid", "death-term-uuid"],
  evidence_links: ["Ada death record"],
  evaluation_method: "cid:document-date-check"
}

[Ada Alive in 1843]: Ada Lovelace was alive on 1 January 1843. {
  premise_uuid: "alive-premise-uuid",
  term_record_uuids: ["ada-term-uuid", "alive-term-uuid"],
  evaluation_method: "cid:half-open-life-interval",
  release_tags: ["validatorium-2026.07"]
}
  + [Ada Birth]
  + [Ada Death]
```

The evaluator constructs the half-open alive interval from the birth and death
premises and tests whether `1843-01-01` falls inside it. The resulting validity
attestation records the method, dependencies, evaluation time, and result. The
example uses Argdown support edges to represent dependencies, not participant
positions.

---

## Appendix A: Quick Reference

### Statement (Premise)

```argdown
[Title]: Atomic claim text. {
  premise_uuid: "UUID",
  term_record_uuids: ["UUID"],
  evidence_links: ["Evidence Tag"],
  evaluation_method: "URI-or-CID",
  release_tags: ["release-version"]
}
```

### Evidence Record

```argdown
<Evidence Tag> {
  evidence_uuid: "UUID",
  source: "URI-or-CID",
  type: "measurement|observation|publication|record|dataset|computation|runtime|instrument",
  method: "URI-or-CID",
  observed_at: "RFC3339 timestamp"
}
```

### Relations

```argdown
[A]: Claim A.
  + [B]                          // B supports A
  - [C]                          // C attacks A
  + <Evidence> {source: "..."}   // Evidence supports A
```

### Tags

```argdown
#dated_claim              // Premise with temporal content
#historical_person        // Premise about a person
#temporal_incompatibility // Argument based on temporal exclusion
#refinement              // Premise that narrows another
#supersedes              // Premise that replaces another
```

---

## Appendix B: Validation Checklist

Before committing an `.argdown` file, verify:

- [ ] YAML front matter identifies the domain, file version, and ontology release
- [ ] Every premise has a stable `premise_uuid`
- [ ] Significant terms reference versioned records through `term_record_uuids`
- [ ] Every premise identifies evidence links and a reproducible evaluation method
- [ ] Premise titles are Title Case, 2–5 words
- [ ] Premise texts are atomic and explicitly scoped
- [ ] Premise texts end with a period
- [ ] Evidence records identify their UUID, source, type, method, time, and provenance
- [ ] Cross-file premise references use exact title matches
- [ ] Tags are from the approved set or documented in this specification
- [ ] The file is parseable by `@argdown/core` without errors
