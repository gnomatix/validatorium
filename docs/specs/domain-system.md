# Domain System Specification

## Overview

This document specifies how knowledge domains function within Validatorium. Domains are the primary organizational and scoping mechanism for premises, determining which ontologies apply to term grounding and disambiguating terms that carry different meanings across fields of knowledge.

---

## 1. What a Domain Is

A **domain** is a field of knowledge — biology, chemistry, history, linguistics, physics, etc.

Domains serve three functions in the system:

1. **Ontology selection.** A domain determines which ontologies are relevant when grounding terms within a premise.
2. **Scope provision.** The same word may carry different meanings in different domains. The domain disambiguates. "Expression" in genetics is not "expression" in mathematics or art.
3. **Organizational unit.** Premises are indexed by domain for retrieval, subscription, and storage sharding.

A premise belongs to **one or more** domains. There is no requirement for exclusivity — a premise about the biochemistry of neurotransmitter synthesis naturally spans multiple domains.

---

## 2. Domain Classification

### Wikidata Alignment

Domains are tied to Wikidata's property hierarchies:

- **P31** (instance of) — classifies entities
- **P279** (subclass of) — establishes hierarchical relationships between classes

Domains map to top-level knowledge classifications derivable from these hierarchies. This provides a stable, externally-maintained classification backbone without requiring Validatorium to define its own taxonomy.

### Properties

- **Not a rigid hierarchy.** Domains overlap. Biochemistry spans biology and chemistry. Neurolinguistics spans neuroscience and linguistics. The system does not force premises into a single branch of a tree.
- **Automatable assignment.** Domain assignment can be inferred from the terms used in a premise. If every grounded term in a premise resolves to ontologies associated with biology, the premise is classified as a biology premise. Mixed groundings produce multi-domain classification.
- **Emergent.** The set of recognized domains is not fixed at design time. New domains emerge as premises begin using terms from coherent new ontology sets.

---

## 3. Relationship to Ontologies

Each domain has one or more associated ontologies used for term grounding. The ontology used to ground a term determines or confirms the domain classification.

### Domain–Ontology Mapping

| Domain | Primary Ontologies |
|--------|-------------------|
| Biology | OBO Foundry: Gene Ontology (GO), Cell Ontology (CL), ChEBI, NCBITaxon |
| Chemistry | ChEBI, PubChem |
| Medicine | MeSH, SNOMED-CT, Disease Ontology (DOID) |
| History | Wikidata temporal properties, biographical properties |
| Linguistics | Wikidata lexemes, ISO 639 language codes |

This table is illustrative, not exhaustive. Additional domains use their respective community-standard ontologies.

### Grounding–Domain Feedback Loop

When a term is grounded to a specific ontology, that grounding acts as evidence for the premise's domain classification. This creates a bidirectional relationship:

1. The domain suggests which ontologies to search for groundings.
2. The ontology a grounding resolves to confirms or adjusts the domain assignment.

---

## 4. Jargon Terms

### Definition

**Jargon** is any word or phrase that carries a domain-specific meaning differing from its common usage or from its meaning in other domains.

### Requirements

- **All jargon MUST be annotated** with its domain and ontology URI.
- **The same word in different domains produces different groundings.** Each grounding is a distinct semantic entity:
  - "expression" in genetics → Gene Ontology: gene expression (GO:0010467)
  - "expression" in mathematics → a symbolic formula
  - "expression" in art → creative manifestation
- **Annotation is mandatory, not optional.** Ungrounded jargon renders a premise structurally invalid.

### Anti-Equivocation Mechanism

This is the core mechanism for preventing **equivocation** — a logical fallacy in which the same word is used with different meanings within an argument, creating the illusion of logical connection where none exists.

By requiring every domain-specific term to carry an explicit grounding, the system makes equivocation structurally impossible at the data layer. Two premises using the same word with different groundings are recognized as discussing different concepts. They cannot be silently conflated in argument composition.

---

## 5. Mathematics Exclusion (Current Phase)

### Status

Mathematics is **excluded** from the initial implementation. The domain identifier is **reserved** — it will be addressed in a future phase.

### Rationale

Mathematical premises decompose into axioms and derivations. Their validation requires **proof** — formal deduction from accepted axioms — rather than **empirical observation** from evidence. This demands a fundamentally different validation pipeline:

- Empirical domains: premise → evidence and method → true/false/indeterminate attestation
- Mathematics: premise → proof chain → result under declared axioms

This is a **scope decision**, not a philosophical one. The mathematical domain requires purpose-built validation machinery that is out of scope for the initial system.

### Boundary Cases

| Case | Included? | Domain |
|------|-----------|--------|
| "Euler was born in Basel in 1707" | Yes | History/Biography |
| "The Pythagorean theorem was known to Babylonian mathematicians" | Yes | History |
| "This algorithm has O(n log n) time complexity" | Conditional | Computer Science |
| "The sum of interior angles of a Euclidean triangle is 180°" | No | Mathematics (reserved) |
| "2 + 2 = 4" | No | Mathematics (reserved) |

**Rule:** Claims *about* mathematics as a human activity (historical, biographical, sociological) are included under their respective domains. Claims *within* mathematics (theorems, proofs, derivations) are excluded.

**Computational claims** that can be independently verified by execution (e.g., algorithmic complexity demonstrable via benchmarking, or program correctness verifiable by testing) may be included if the verification method is empirical rather than purely deductive.

---

## 6. Domain Governance

### No Privileged Roles

There are no domain "owners," "experts," or "moderators" with elevated privileges. Ontology grounding and structural checks make premises evaluable; evidence evaluation determines whether they are valid. No human authority grants or revokes domain access.

### External Ontologies

Domain-specific ontologies are maintained by external organizations (OBO Foundry, NLM, WHO, etc.). Validatorium **references** these ontologies. It does not create, fork, or modify them.

Premises are tagged with versioned terms from versioned ontologies. A superseded
term-mapping record is marked obsolete and stores reference pointers to its
replacement record UUIDs; the obsolete record remains addressable through IPFS.

### Wikidata as Fallback

If a term requires grounding but no suitable domain ontology exists, Wikidata serves as the universal fallback. Wikidata's broad coverage ensures that any identifiable concept can be grounded, even if the grounding is less precise than a specialized ontology would provide.

### Domain Emergence

New domains are not declared by fiat. A domain is **recognized** when premises begin consistently using terms that ground to a coherent, previously-unrepresented set of ontologies. The system detects clustering in ontology usage and surfaces candidate new domains.

---

## 7. Cross-Domain Premises

### Behavior

A premise may legitimately span multiple domains. This is common and expected.

**Example:** "Serotonin is synthesized from tryptophan via tryptophan hydroxylase in raphe nuclei neurons."

This premise spans:
- **Biochemistry** — synthesis pathway, enzyme identification (ChEBI, UniProt)
- **Neuroscience** — neuronal cell type, brain region (Cell Ontology, neuroanatomy ontologies)

### Properties of Cross-Domain Premises

- The premise carries groundings from **all** relevant domains.
- It appears in **all** associated domain indexes.
- There is **no conflict** — multiple domain tags coexist without priority or hierarchy.
- Each grounding is independently valid within its own domain's ontology.
- Cross-domain premises are a natural consequence of the system's design, not an edge case requiring special handling.

---

## 8. Domain and Storage

### Sharding

Domains are a natural sharding unit for the distributed premise store.

- Premises are indexed by domain.
- Each domain's premises form a logically coherent subset of the total store.
- Multi-domain premises appear in multiple shards (stored once, indexed multiply).

### Subscription Model

Nodes in the distributed network can **subscribe** to specific domains:

- A node interested only in biology need not replicate chemistry premises.
- Subscription is domain-granular — subscribe to one, several, or all domains.
- Cross-domain premises are delivered to subscribers of any of their associated domains.

### Index Compression

Domain-based clustering improves index compression:

- Terms within a domain exhibit high co-occurrence patterns.
- Ontology URIs within a domain share common prefixes.
- Domain-local indexes exploit this regularity for efficient storage and lookup.

---

## Design Decisions Summary

| Decision | Rationale |
|----------|-----------|
| Domains are not exclusive | Real knowledge crosses boundaries; forcing single-domain would lose information |
| Classification via Wikidata hierarchies | Stable external authority; avoids inventing a taxonomy |
| Automated domain inference from groundings | Reduces manual classification burden; ensures consistency |
| Jargon annotation is mandatory | Core anti-equivocation mechanism; structural correctness depends on it |
| Mathematics excluded in current phase | Different validation model (proof vs. evidence); future phase |
| No domain governance roles | Consistent with overall system principle of method-based, not authority-based, validation |
| Wikidata as universal fallback | Ensures every term can be grounded even without a specialized ontology |
