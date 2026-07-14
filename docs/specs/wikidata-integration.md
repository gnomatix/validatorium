# Wikidata/SPARQL Integration Specification

**Status:** Draft
**Last Updated:** 2026-07-14
**Component:** validatorium core / term grounding / fact verification

---

## Overview

validatorium uses Wikidata as a **pivot layer** — not as an authority, but as a stable coordinate system for entity resolution, term grounding, and cross-referencing to domain ontologies. Wikidata Q-numbers serve as language-independent identifiers that link surface-form text to structured knowledge.

This spec defines how the system queries Wikidata, what it uses the results for, and what it explicitly does NOT treat Wikidata as (evidence).

---

## 1. Role of Wikidata in the System

### 1.1 Entity Disambiguation

Wikidata Q-numbers provide stable, cross-language identifiers for entities. When a premise mentions "Mercury," the system needs to distinguish:

- Q308 (planet Mercury)
- Q925 (element mercury)
- Q7957 (Freddie Mercury)

Q-numbers are permanent, language-neutral, and machine-readable. They serve as the canonical pivot between surface forms and ontology URIs.

### 1.2 Classification via P31 and P279

Two properties form the backbone of Wikidata's type system:

| Property | Name | Role |
|----------|------|------|
| P31 | instance of | "this entity IS A ___" |
| P279 | subclass of | "this class IS A SUBCLASS OF ___" |

These allow the system to answer "what kind of thing is this?" — critical for validating that a premise's subject-predicate structure is coherent.

Example: Q308 → P31 → Q3504248 (inner planet) → P279 → Q17362350 (inferior planet) → P279 → Q634 (planet)

### 1.3 External Ontology Links

Wikidata acts as a hub connecting to domain ontologies:

| Property | Name | Use |
|----------|------|-----|
| P1709 | equivalent class | Links to OWL/RDFS classes in external ontologies |
| P2888 | exact match | SKOS exact match to external URI |
| P4390 | mapping relation type | Qualifies the precision of a mapping (see §7) |

These let the system resolve from a Q-number outward to the authoritative ontology for a given domain.

### 1.4 Biographical and Temporal Facts

Key temporal properties used for fact verification:

| Property | Name |
|----------|------|
| P569 | date of birth |
| P570 | date of death |
| P571 | inception |
| P576 | dissolved/abolished |
| P580 | start time |
| P582 | end time |
| P585 | point in time |

These support temporal compatibility checking (e.g., "Could person X have met person Y?").

### 1.5 Cross-Reference to Domain Ontologies

Wikidata stores external identifiers that link entities to authoritative domain databases:

| Property | Domain | Target System |
|----------|--------|---------------|
| P486 | Medicine | MeSH descriptor ID |
| P683 | Chemistry | ChEBI ID |
| P685 | Biology | NCBI Taxonomy ID |
| P352 | Proteomics | UniProt protein ID |
| P1709 | Various | Gene Ontology (via equivalent class URI) |
| P699 | Disease | Disease Ontology ID |
| P2892 | Biochemistry | UMLS CUI |
| P231 | Chemistry | CAS Registry Number |

This allows validatorium to ground a term first to Wikidata, then resolve outward to the authoritative identifier in the relevant domain.

---

## 2. SPARQL Query Patterns

All SPARQL queries target `https://query.wikidata.org/sparql` unless otherwise noted.

### 2.1 Entity Lookup by Label

**Primary method:** Use the `wbsearchentities` REST API for initial candidate retrieval (faster, fuzzy-match capable), then SPARQL for structured follow-up.

**REST API call:**
```
GET https://www.wikidata.org/w/api.php?action=wbsearchentities
    &search=apoptosis
    &language=en
    &type=item
    &limit=10
    &format=json
```

**SPARQL equivalent (exact label match):**
```sparql
SELECT ?item ?itemLabel ?itemDescription WHERE {
  ?item rdfs:label "apoptosis"@en .
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
LIMIT 20
```

**SPARQL (substring match via FILTER):**
```sparql
SELECT ?item ?itemLabel ?itemDescription WHERE {
  ?item rdfs:label ?label .
  FILTER(LANG(?label) = "en")
  FILTER(CONTAINS(LCASE(?label), "apoptosis"))
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
LIMIT 20
```

### 2.2 Classification Query ("What is this thing?")

Given Q-number, retrieve its type hierarchy:

```sparql
SELECT ?item ?itemLabel ?class ?classLabel ?superclass ?superclassLabel WHERE {
  VALUES ?item { wd:Q14599311 }  # apoptotic process
  ?item wdt:P31 ?class .
  OPTIONAL { ?class wdt:P279 ?superclass . }
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
```

**Full transitive subclass chain:**
```sparql
SELECT ?item ?itemLabel ?ancestor ?ancestorLabel WHERE {
  VALUES ?item { wd:Q14599311 }
  ?item wdt:P31/wdt:P279* ?ancestor .
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
```

### 2.3 Biographical Fact Retrieval

```sparql
SELECT ?person ?personLabel ?birthDate ?deathDate ?birthPlace ?birthPlaceLabel WHERE {
  VALUES ?person { wd:Q7186 }  # Marie Curie
  OPTIONAL { ?person wdt:P569 ?birthDate . }
  OPTIONAL { ?person wdt:P570 ?deathDate . }
  OPTIONAL { ?person wdt:P19 ?birthPlace . }
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
```

**With qualifiers (for sourced claims):**
```sparql
SELECT ?person ?personLabel ?birthDate ?refURL WHERE {
  VALUES ?person { wd:Q7186 }
  ?person p:P569 ?birthStatement .
  ?birthStatement ps:P569 ?birthDate .
  OPTIONAL {
    ?birthStatement prov:wasDerivedFrom ?ref .
    ?ref pr:P854 ?refURL .
  }
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
```

### 2.4 External ID and Ontology Mapping Retrieval

**Retrieve all external identifiers for an entity:**
```sparql
SELECT ?item ?itemLabel ?prop ?propLabel ?value WHERE {
  VALUES ?item { wd:Q14599311 }  # apoptotic process
  ?item ?p ?statement .
  ?statement ?ps ?value .
  ?prop wikibase:claim ?p .
  ?prop wikibase:statementProperty ?ps .
  ?prop wikibase:propertyType wikibase:ExternalId .
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
```

**Retrieve specific domain ontology links:**
```sparql
SELECT ?item ?itemLabel ?meshId ?chebiId ?ncbiId ?equivClass WHERE {
  VALUES ?item { wd:Q2996394 }  # programmed cell death
  OPTIONAL { ?item wdt:P486 ?meshId . }
  OPTIONAL { ?item wdt:P683 ?chebiId . }
  OPTIONAL { ?item wdt:P685 ?ncbiId . }
  OPTIONAL { ?item wdt:P1709 ?equivClass . }
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
```

**Retrieve mappings with P4390 precision qualifiers:**
```sparql
SELECT ?item ?itemLabel ?externalURI ?mappingRelation ?mappingRelationLabel WHERE {
  VALUES ?item { wd:Q2996394 }
  ?item p:P2888 ?matchStatement .
  ?matchStatement ps:P2888 ?externalURI .
  OPTIONAL { ?matchStatement pq:P4390 ?mappingRelation . }
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
```

### 2.5 Temporal Compatibility Checking

**Check if two people's lifetimes overlapped:**
```sparql
SELECT ?person1Label ?birth1 ?death1 ?person2Label ?birth2 ?death2
       (IF(?death1 >= ?birth2 && ?death2 >= ?birth1, "OVERLAP", "NO OVERLAP") AS ?compatible)
WHERE {
  VALUES ?person1 { wd:Q7186 }   # Marie Curie
  VALUES ?person2 { wd:Q937 }    # Albert Einstein
  ?person1 wdt:P569 ?birth1 . ?person1 wdt:P570 ?death1 .
  ?person2 wdt:P569 ?birth2 . ?person2 wdt:P570 ?death2 .
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
```

**Check if an event falls within an entity's existence:**
```sparql
SELECT ?entityLabel ?inception ?dissolved
       (IF(?eventDate >= ?inception && ?eventDate <= COALESCE(?dissolved, NOW()), "VALID", "INVALID") AS ?temporalValidity)
WHERE {
  VALUES ?entity { wd:Q131454 }  # Austro-Hungarian Empire
  VALUES (?eventDate) { ("1910-01-01"^^xsd:dateTime) }
  ?entity wdt:P571 ?inception .
  OPTIONAL { ?entity wdt:P576 ?dissolved . }
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
```

### 2.6 Federated Queries

**Query WikiPathways for pathways involving a protein:**
```sparql
PREFIX wp: <http://vocabularies.wikipathways.org/wp#>

SELECT ?pathway ?pathwayLabel ?gene WHERE {
  # Get UniProt ID from Wikidata
  wd:Q21171311 wdt:P352 ?uniprotId .

  # Federate to WikiPathways
  SERVICE <http://sparql.wikipathways.org/sparql> {
    ?gene a wp:Protein .
    ?gene wp:bdbUniprot ?uniprotUri .
    ?pathway a wp:Pathway .
    ?pathway wp:organismName "Homo sapiens" .
    ?gene dcterms:isPartOf ?pathway .
    ?pathway dc:title ?pathwayLabel .
    FILTER(CONTAINS(STR(?uniprotUri), ?uniprotId))
  }
}
LIMIT 20
```

**Federated query to UniProt SPARQL:**
```sparql
PREFIX up: <http://purl.uniprot.org/core/>

SELECT ?protein ?proteinLabel ?uniprotId ?function WHERE {
  VALUES ?protein { wd:Q21171311 }
  ?protein wdt:P352 ?uniprotId .

  BIND(IRI(CONCAT("http://purl.uniprot.org/uniprot/", ?uniprotId)) AS ?uniprotURI)

  SERVICE <https://sparql.uniprot.org/sparql> {
    ?uniprotURI up:annotation ?annotation .
    ?annotation a up:Function_Annotation .
    ?annotation rdfs:comment ?function .
  }
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
```

---

## 3. Term Grounding Workflow

Term grounding maps surface-form jargon in premise text to stable identifiers with known semantics.

### 3.1 Workflow Steps

```
┌─────────────────┐
│ Premise Text     │  "Apoptosis is triggered by caspase-3 activation"
└────────┬────────┘
         │ LLM extracts jargon terms
         ▼
┌─────────────────┐
│ Terms Identified │  ["apoptosis", "caspase-3"]
└────────┬────────┘
         │ wbsearchentities query per term
         ▼
┌─────────────────┐
│ Candidates       │  apoptosis → [Q14599311, Q2996394, ...]
└────────┬────────┘
         │ Retrieve P31, description, external IDs
         ▼
┌─────────────────┐
│ Enriched Candidates │  Q14599311: instance of biological process,
│                      │  GO:0006915, MeSH D017209
└────────┬────────┘
         │ LLM/human selects correct grounding
         ▼
┌─────────────────┐
│ Grounding Record │  surface_form → wikidata_id → ontology_uri
└─────────────────┘
```

### 3.2 Step 1: Term Extraction

The LLM identifies terms in premise text that require grounding — domain-specific jargon, proper nouns, or ambiguous terms. Output is a list of `(surface_form, context_hint)` pairs.

### 3.3 Step 2: Candidate Retrieval

For each term, call `wbsearchentities`:

```
GET https://www.wikidata.org/w/api.php?action=wbsearchentities
    &search=caspase-3
    &language=en
    &type=item
    &limit=5
    &format=json
```

Response includes:
- `id`: Q-number (e.g., Q21171311)
- `label`: canonical English label
- `description`: disambiguating description
- `aliases`: alternative names

### 3.4 Step 3: Classification and External Links

For each candidate, retrieve classification and ontology links:

```sparql
SELECT ?item ?itemLabel ?itemDescription ?class ?classLabel
       ?equivClass ?exactMatch ?meshId ?goId ?chebiId WHERE {
  VALUES ?item { wd:Q21171311 }
  OPTIONAL { ?item wdt:P31 ?class . }
  OPTIONAL { ?item wdt:P1709 ?equivClass . }
  OPTIONAL { ?item wdt:P2888 ?exactMatch . }
  OPTIONAL { ?item wdt:P486 ?meshId . }
  OPTIONAL { ?item wdt:P683 ?chebiId . }
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
```

### 3.5 Step 4: Selection

The LLM (or human reviewer) selects the correct grounding based on:
1. P31 classification matches expected domain
2. Description matches context
3. External ontology links confirm identity

### 3.6 Step 5: Recording the Grounding

The grounding is recorded as a structured mapping:

```json
{
  "surface_form": "caspase-3",
  "wikidata_id": "Q21171311",
  "wikidata_label": "caspase-3",
  "classification": {
    "P31": ["Q8054"],
    "P31_label": ["protein"]
  },
  "ontology_mappings": [
    {
      "uri": "http://purl.uniprot.org/uniprot/P42574",
      "source_property": "P352",
      "mapping_relation": "exact_match",
      "ontology": "UniProt"
    },
    {
      "uri": "http://purl.obolibrary.org/obo/PR_000005076",
      "source_property": "P1709",
      "mapping_relation": "exact_match",
      "ontology": "Protein Ontology"
    }
  ],
  "mapping_precision": "Q39893184",
  "mapping_precision_label": "exact match"
}
```

### 3.7 Mapping Precision via P4390

Every grounding record MUST include the mapping precision (P4390 value) indicating the relationship between the Wikidata entity and the external ontology term:

| Q-number | Label | Meaning |
|----------|-------|---------|
| Q39893184 | exact match | Same concept, transitive |
| Q39894595 | close match | Interchangeable in many contexts, NOT transitive |
| Q39894604 | narrow match | External concept is narrower |
| Q39894556 | broad match | External concept is broader |
| Q39893967 | related match | Associated but non-hierarchical |

This precision is critical for downstream reasoning — an `exact_match` grounding can be used interchangeably, while a `broad_match` requires additional qualification.

### 3.8 Ontology Mapping Versions

Validatorium tags premises with versioned terms from versioned ontologies. When a
term mapping is superseded, its record is marked obsolete and retains reference
pointers to the replacement record UUIDs. The obsolete record remains addressable
through IPFS. Each Validatorium release checks these references and applies any
required remapping before tagging a premise as valid and current for that release.

---

## 4. Fact Verification Patterns

Wikidata is used to CHECK claims against structured data — not as authoritative evidence, but as a structured source that can flag inconsistencies or confirm alignment with community-maintained data.

### 4.1 Temporal Claims

**Pattern:** "X happened during Y's lifetime" or "X preceded Y"

```sparql
# Verify: "Darwin was alive when Origin of Species was published (1859)"
SELECT ?person ?personLabel ?birthDate ?deathDate ?pubDate
       (IF(?pubDate >= ?birthDate && ?pubDate <= ?deathDate, "CONSISTENT", "INCONSISTENT") AS ?verdict)
WHERE {
  VALUES ?person { wd:Q1035 }       # Charles Darwin
  VALUES ?work { wd:Q20124 }        # On the Origin of Species
  ?person wdt:P569 ?birthDate .
  ?person wdt:P570 ?deathDate .
  ?work wdt:P577 ?pubDate .
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
```

**Temporal ordering check:**
```sparql
# Verify: "Pasteur's germ theory preceded Koch's postulates"
SELECT ?event1Label ?date1 ?event2Label ?date2
       (IF(?date1 < ?date2, "CORRECT ORDER", "INCORRECT ORDER") AS ?verdict)
WHERE {
  VALUES ?event1 { wd:Q846672 }   # germ theory
  VALUES ?event2 { wd:Q1068008 }  # Koch's postulates
  ?event1 wdt:P571|wdt:P585 ?date1 .
  ?event2 wdt:P571|wdt:P585 ?date2 .
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
```

### 4.2 Classificatory Claims

**Pattern:** "X is a type of Y"

```sparql
# Verify: "Apoptosis is a type of programmed cell death"
ASK {
  wd:Q14599311 wdt:P31/wdt:P279* wd:Q2996394 .
}
```

**With path details:**
```sparql
# Show the classification chain from apoptosis to biological process
SELECT ?step ?stepLabel WHERE {
  wd:Q14599311 wdt:P31/wdt:P279* ?step .
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
```

**Negative check (something is NOT a type of something):**
```sparql
# Check: "Is mercury (element) a planet?" — should return false
ASK {
  wd:Q925 wdt:P31/wdt:P279* wd:Q634 .
}
```

### 4.3 Relational Claims

**Pattern:** "X has property P with value Y"

```sparql
# Verify: "Marie Curie won the Nobel Prize in Physics"
ASK {
  wd:Q7186 wdt:P166 wd:Q38104 .
}
```

**With qualifiers:**
```sparql
# Verify: "Marie Curie won the Nobel Prize in Physics in 1903"
SELECT ?awardDate WHERE {
  wd:Q7186 p:P166 ?statement .
  ?statement ps:P166 wd:Q38104 .
  ?statement pq:P585 ?awardDate .
  FILTER(YEAR(?awardDate) = 1903)
}
```

### 4.4 Existence Claims

**Pattern:** "X exists/is a known entity"

```sparql
# Check if an entity with this label exists
ASK {
  ?item rdfs:label "CRISPR-Cas9"@en .
}
```

**CRITICAL CAVEAT:** Absence from Wikidata does NOT prove non-existence. Wikidata has:
- Coverage gaps in specialized domains
- Bias toward well-known entities
- Lag behind new discoveries

The system MUST report existence checks as:
- `FOUND` — entity exists in Wikidata (can proceed with grounding)
- `NOT_FOUND` — entity not in Wikidata (does NOT imply non-existence; flag for manual review)

### 4.5 Verification Result Schema

Every fact verification returns a structured result:

```json
{
  "claim": "Marie Curie won the Nobel Prize in Physics in 1903",
  "verification_type": "relational_with_temporal",
  "wikidata_query_type": "qualified_property_check",
  "result": "CONSISTENT",
  "confidence_notes": [
    "Wikidata confirms P166 (award received) = Q38104 (Nobel Prize in Physics)",
    "Qualifier P585 (point in time) = 1903-12-10",
    "Source: Nobel Foundation records (P854)"
  ],
  "caveats": [
    "Wikidata is community-edited; treat as corroboration, not proof"
  ]
}
```

---

## 5. API Access Methods

### 5.1 SPARQL Endpoint

**Endpoint:** `https://query.wikidata.org/sparql`

**Request format:**
```http
GET /sparql?query={URL-encoded SPARQL}&format=json HTTP/1.1
Host: query.wikidata.org
User-Agent: validatorium/0.1 (https://github.com/gnomatix/validatorium)
Accept: application/sparql-results+json
```

Or via POST:
```http
POST /sparql HTTP/1.1
Host: query.wikidata.org
User-Agent: validatorium/0.1
Content-Type: application/x-www-form-urlencoded
Accept: application/sparql-results+json

query={URL-encoded SPARQL}
```

**Response format:** SPARQL JSON Results (`application/sparql-results+json`)

### 5.2 REST API (MediaWiki Action API)

**Base URL:** `https://www.wikidata.org/w/api.php`

**wbsearchentities** — fuzzy entity search:
```
GET /w/api.php?action=wbsearchentities
    &search=apoptosis
    &language=en
    &type=item
    &limit=10
    &format=json
```

**wbgetentities** — retrieve full entity data:
```
GET /w/api.php?action=wbgetentities
    &ids=Q14599311|Q21171311
    &props=labels|descriptions|claims|sitelinks
    &languages=en
    &format=json
```

### 5.3 Rate Limiting

**Wikidata Query Service (SPARQL):**
- No hard rate limit published, but aggressive querying will be throttled
- Guideline: max 1 concurrent query, max ~60 queries/minute for batch operations
- Queries that take >60 seconds are killed server-side
- Set a reasonable `User-Agent` header (required by Wikimedia policy)

**REST API:**
- Soft limit: ~200 requests/second per IP (lower for unauthenticated)
- Respect `Retry-After` headers
- Use `maxlag` parameter for writes (not applicable for reads in our use case)

**Implementation requirements:**
- Exponential backoff on 429/503 responses
- Minimum 100ms between sequential SPARQL queries
- Batch `wbgetentities` calls (up to 50 IDs per request)
- Include descriptive `User-Agent` per Wikimedia User-Agent policy

### 5.4 Caching Strategy (Go CLI)

The validatorium Go CLI implements a tiered caching strategy:

**Layer 1: In-memory LRU cache (session-scoped)**
- Capacity: 1000 entities
- TTL: duration of CLI session
- Key: Q-number or query hash
- Use: avoid redundant lookups within a single grounding session

**Layer 2: On-disk SQLite cache**
- Location: `~/.validatorium/cache/wikidata.db`
- Schema:
  ```sql
  CREATE TABLE entity_cache (
      qid TEXT PRIMARY KEY,
      data_json TEXT NOT NULL,
      fetched_at TIMESTAMP NOT NULL,
      expires_at TIMESTAMP NOT NULL
  );

  CREATE TABLE query_cache (
      query_hash TEXT PRIMARY KEY,
      result_json TEXT NOT NULL,
      fetched_at TIMESTAMP NOT NULL,
      expires_at TIMESTAMP NOT NULL
  );

  CREATE INDEX idx_entity_expires ON entity_cache(expires_at);
  CREATE INDEX idx_query_expires ON query_cache(expires_at);
  ```

**TTL policy:**
- Entity data (labels, descriptions, P31): 7 days
- External IDs (P486, P683, etc.): 30 days (these rarely change)
- SPARQL query results: 24 hours
- Classification chains (P279): 7 days
- Temporal facts (P569, P570): 30 days

**Cache refresh:**
- `validatorium cache clear` — purge all cached data
- `validatorium cache refresh <QID>` — force re-fetch of specific entity
- Automatic eviction of expired entries on startup

**Offline mode:**
- If network is unavailable, serve from cache with a `STALE` warning
- Never serve stale data silently — always annotate with fetch timestamp

---

## 6. Limitations and Caveats

### 6.1 Wikidata is NOT Authoritative Evidence

**This is the single most important design principle:**

> Wikidata is used for **entity resolution** and **cross-referencing** — NEVER as evidence for premises.

Wikidata is community-edited. Any statement in Wikidata could be:
- Incorrect (vandalism or error)
- Outdated (the world changed but the record did not)
- Conflicting with another sourced statement
- Incomplete (relevant qualifiers missing)

**Correct usage:**
- "Wikidata identifies Q7186 as Marie Curie" → use for entity resolution ✓
- "Wikidata says this, therefore it is true" → WRONG ✗

A Wikidata statement used in a premise requires the normal Validatorium evidence
evaluation with exact sources, method, scope, and provenance.

### 6.2 Coverage Gaps

Wikidata coverage is uneven:

| Domain | Coverage | Notes |
|--------|----------|-------|
| Famous people | Excellent | Most notable figures well-represented |
| Countries, cities | Excellent | Comprehensive geographic data |
| Common species | Good | Major species covered |
| Proteins, genes | Moderate | Bulk-imported but may lag databases |
| Specialized chemistry | Variable | Less common compounds may be missing |
| Cutting-edge research | Poor | New discoveries take months/years to appear |
| Historical events | Variable | Western-centric bias |
| Domain-specific concepts | Variable | Depends on community effort |

**Implication:** A `NOT_FOUND` result from Wikidata should NEVER be interpreted as "this thing doesn't exist." It means only that Wikidata doesn't (yet) have an entry for it.

### 6.3 Temporal Precision Issues

Wikidata dates come with precision indicators, but the system must handle:

- **Year-only dates:** `"1867-00-00"` with precision 9 (year) — don't compare at day level
- **Conflicting date statements:** Some entities have multiple P569 values with different sources
- **Calendar differences:** Julian vs. Gregorian calendar for pre-1582 dates
- **Approximate dates:** `"1200-00-00"` with precision 7 (century) for medieval figures

**Implementation rule:** When comparing dates, respect the minimum precision of both operands. If one date is year-only, comparison granularity must be year-level.

```go
// Temporal precision levels (matching Wikidata model)
const (
    PrecisionBillionYears = 0
    PrecisionMillionYears = 3
    PrecisionThousandYears = 4
    PrecisionCentury      = 7
    PrecisionDecade       = 8
    PrecisionYear         = 9
    PrecisionMonth        = 10
    PrecisionDay          = 11
)
```

### 6.4 Multilingual Considerations

- Labels may differ across languages; always specify language in queries
- Some entities lack English labels (use `SERVICE wikibase:label` with fallback chain)
- Aliases may catch alternative names but aren't exhaustive

### 6.5 Query Complexity Limits

- Wikidata Query Service has a 60-second query timeout
- Property path queries (`wdt:P279*`) can be slow on deep hierarchies
- Federated queries add network latency and may fail if remote endpoints are down
- Use `LIMIT` aggressively and prefer targeted queries over exploratory ones

---

## 7. P4390 Mapping Relation Types

P4390 (mapping relation type) is **critical** for term grounding precision. It qualifies the relationship between a Wikidata entity and an external ontology concept, determining how the grounding can be used in downstream reasoning.

### 7.1 Relation Types

| Wikidata Value | SKOS Equivalent | Meaning | Transitivity | Use in Reasoning |
|----------------|-----------------|---------|--------------|------------------|
| Q39893184 | skos:exactMatch | Equivalent meaning in all contexts | **Transitive** | Fully interchangeable; can substitute freely |
| Q39894595 | skos:closeMatch | Interchangeable in many but not all contexts | **NOT transitive** | Use with caution; note contexts where they diverge |
| Q39894604 | skos:narrowMatch | External concept is narrower than Wikidata entity | N/A | External is more specific; valid for specialization only |
| Q39894556 | skos:broadMatch | External concept is broader than Wikidata entity | N/A | External is more general; valid for generalization only |
| Q39893967 | skos:relatedMatch | Non-hierarchical association | N/A | Cannot substitute; use only for discovery/navigation |

### 7.2 Transitivity Rules

**exact_match is transitive:**
If A `exact_match` B and B `exact_match` C, then A `exact_match` C.

This means: if Wikidata Q14599311 (apoptotic process) is an exact match to GO:0006915, and GO:0006915 is an exact match to MeSH D017209, then all three can be used interchangeably in premise grounding.

**close_match is NOT transitive:**
If A `close_match` B and B `close_match` C, we CANNOT conclude A `close_match` C.

This is critical — "close enough" does not chain. Each close_match must be evaluated independently.

### 7.3 Impact on Term Grounding

The mapping relation determines what the system can DO with a grounding:

```
exact_match:
  - Can replace surface_form with ontology_uri in formal reasoning
  - Can merge evidence across equivalent identifiers
  - Can assert "X in Wikidata IS the same as Y in Gene Ontology"

close_match:
  - Can suggest the ontology term as likely equivalent
  - CANNOT merge evidence without human confirmation
  - Must flag: "this is close but may diverge in edge cases"

narrow_match:
  - External concept is MORE SPECIFIC than the Wikidata entity
  - If premise uses broad term, narrow_match external ID is a valid specialization
  - Cannot generalize from the external concept back to the Wikidata entity

broad_match:
  - External concept is LESS SPECIFIC than the Wikidata entity
  - If premise uses specific term, broad_match external ID loses precision
  - Can generalize from Wikidata entity to the external concept

related_match:
  - For discovery/navigation only
  - CANNOT be used for substitution in any direction
  - Records association for human exploration
```

### 7.4 Query: Retrieve Mapping Relations for an Entity

```sparql
SELECT ?item ?itemLabel ?externalURI ?relationType ?relationTypeLabel WHERE {
  VALUES ?item { wd:Q14599311 }  # apoptotic process
  {
    ?item p:P2888 ?stmt .
    ?stmt ps:P2888 ?externalURI .
    OPTIONAL { ?stmt pq:P4390 ?relationType . }
  }
  UNION
  {
    ?item p:P1709 ?stmt .
    ?stmt ps:P1709 ?externalURI .
    OPTIONAL { ?stmt pq:P4390 ?relationType . }
  }
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
```

### 7.5 Default Behavior When P4390 is Missing

Many Wikidata mappings lack an explicit P4390 qualifier. In this case:

- **P2888 (exact match) without P4390:** Assume `exact_match` (P2888 semantics imply this)
- **P1709 (equivalent class) without P4390:** Assume `exact_match` (equivalence is the stated relation)
- **External ID properties (P486, P683, etc.) without P4390:** Assume `exact_match` (these are direct identifier lookups)
- **Any other mapping without P4390:** Assume `related_match` (most conservative)

---

## 8. Implementation Notes

### 8.1 Go CLI Integration Points

```go
// Core interface for Wikidata integration
type WikidataClient interface {
    // Entity search
    SearchEntities(ctx context.Context, query string, lang string, limit int) ([]EntityCandidate, error)

    // Entity retrieval
    GetEntity(ctx context.Context, qid string) (*Entity, error)
    GetEntities(ctx context.Context, qids []string) ([]*Entity, error)

    // SPARQL
    ExecuteSPARQL(ctx context.Context, query string) (*SPARQLResult, error)

    // High-level operations
    ClassifyEntity(ctx context.Context, qid string) ([]Classification, error)
    GetExternalMappings(ctx context.Context, qid string) ([]OntologyMapping, error)
    CheckTemporalClaim(ctx context.Context, claim TemporalClaim) (*VerificationResult, error)
}

// Grounding record stored in validatorium
type TermGrounding struct {
    SurfaceForm     string            `json:"surface_form"`
    WikidataID      string            `json:"wikidata_id"`
    WikidataLabel   string            `json:"wikidata_label"`
    Classification  []string          `json:"classification_p31"`
    OntologyMappings []OntologyMapping `json:"ontology_mappings"`
    SelectedBy      string            `json:"selected_by"`  // "llm" or "human"
    GroundedAt      time.Time         `json:"grounded_at"`
}

type OntologyMapping struct {
    URI              string `json:"uri"`
    SourceProperty   string `json:"source_property"`
    MappingRelation  string `json:"mapping_relation"`  // exact|close|narrow|broad|related
    Ontology         string `json:"ontology"`
}
```

### 8.2 Error Handling

| Error | Response |
|-------|----------|
| Network timeout | Serve from cache if available; mark as STALE |
| 429 Too Many Requests | Exponential backoff, retry up to 3 times |
| SPARQL timeout (60s) | Simplify query, reduce depth, retry |
| Entity not found | Return NOT_FOUND with explicit caveat |
| Ambiguous results | Return all candidates for human/LLM disambiguation |

### 8.3 Security Considerations

- All queries are read-only (no Wikidata editing)
- No authentication required for reads
- Validate/sanitize any user-supplied entity labels before interpolating into SPARQL queries (prevent SPARQL injection)
- Cache files should not be world-readable (may reveal research interests)

---

## Appendix A: Property Quick Reference

| Property | Name | Domain | Use |
|----------|------|--------|-----|
| P31 | instance of | General | Classification |
| P279 | subclass of | General | Hierarchy |
| P1709 | equivalent class | Ontology | OWL class mapping |
| P2888 | exact match | Ontology | SKOS exact match |
| P4390 | mapping relation type | Ontology | Mapping precision qualifier |
| P486 | MeSH descriptor ID | Medicine | Medical concepts |
| P683 | ChEBI ID | Chemistry | Chemical entities |
| P685 | NCBI Taxonomy ID | Biology | Organisms |
| P352 | UniProt protein ID | Proteomics | Proteins |
| P699 | Disease Ontology ID | Medicine | Diseases |
| P569 | date of birth | Biography | Temporal |
| P570 | date of death | Biography | Temporal |
| P571 | inception | Temporal | Start of existence |
| P576 | dissolved/abolished | Temporal | End of existence |
| P580 | start time | Temporal | Period start |
| P582 | end time | Temporal | Period end |
| P585 | point in time | Temporal | Single date |
| P577 | publication date | Publishing | When published |
| P166 | award received | Biography | Awards |
| P19 | place of birth | Biography | Location |
| P854 | reference URL | Sourcing | Source link |
| P248 | stated in | Sourcing | Source work |
