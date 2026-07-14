# Indexing and Retrieval Strategy

## Overview

Validatorium requires efficient storage, indexing, and retrieval of premises across a distributed network. This spec draws on algorithms proven in computational genomics — specifically those designed for handling massive, redundant, overlapping short sequences — applied to the problem of indexing premise statements by their grounded keyword content.

The core insight: premise statements, like genomic reads, are short text sequences with high redundancy in their component terms. The same domain terms appear across thousands of premises. The same conceptual neighborhoods recur. Algorithms that exploit this redundancy for compression and fast lookup in genomics apply directly.

## Design Principles

- Local-first: every node maintains its own compressed index
- UUID-addressable: premise UUIDs are the primary logical keys; CIDs select exact stored versions
- Keyword-based: retrieval is driven by grounded terms, not raw text search
- Compression-native: redundancy is exploited, not tolerated
- Domain-clustered: related premises are physically co-located in the index

---

## 1. The Indexing Unit

Premises are not indexed as raw strings. They are indexed as **term sequences** — the ordered list of ontology-grounded canonical terms extracted from the statement.

Example:

```
Statement: "The normal human diploid genome contains 46 chromosomes."

Term sequence: [normal, human(Q15978631), diploid(PATO:0001393), genome(SO:0001026), 46, chromosome(GO:0005694)]
```

This term sequence is the indexable unit. Raw surface text is stored separately and
retrieved by logical premise UUID or exact version CID.

---

## 2. Suffix Array with LCP (Longest Common Prefix)

### What it solves

Finding all premises that share keyword subsequences. Premises sharing long term subsequences are conceptually related.

### How it works

1. Concatenate all premise term sequences with unique delimiters between them
2. Build a suffix array over the concatenated sequence
3. Compute the LCP array
4. Regions of high LCP values = clusters of premises sharing long term runs

### Application

```
Query: "find all premises involving human + chromosome"
→ Search suffix array for the bigram [human(Q15978631), chromosome(GO:0005694)]
→ All suffixes starting with this bigram → all premises containing these adjacent terms
```

### Complexity

- Build: O(n) with SA-IS algorithm
- Query: O(m log n) where m = query length
- Space: O(n) integers (compressible with wavelet trees)

---

## 3. FM-Index / BWT (Burrows-Wheeler Transform)

### What it solves

Full-text substring search over the entire premise corpus in compressed space. The same technique used by BWA/Bowtie for genome read alignment.

### How it works

1. Compute BWT of the concatenated term sequence corpus
2. Build the FM-index (occurrence arrays + suffix array samples)
3. Substring search via backward extension — O(m) per query regardless of corpus size

### Application

- Find all premises containing any given term or term sequence
- Count occurrences without decompressing
- Locate positions (which premise, which position in premise)

### Why this over a simple inverted index

An inverted index maps term → list of premises containing it. Fine for single-term lookup. But the FM-index gives you:
- Arbitrary substring (multi-term sequence) search in O(m)
- Compression built in (BWT is highly compressible for redundant data)
- No need to pre-specify which n-grams to index
- Scales to millions of premises in memory that would be gigabytes uncompressed

---

## 4. MinHash / Locality-Sensitive Hashing

### What it solves

Fast approximate similarity between premises without pairwise comparison. From metagenomic binning (Mash, sourmash).

### How it works

1. Each premise's term set (unordered) is hashed into a fixed-size signature (e.g., 256 minhash values)
2. Jaccard similarity between two premises ≈ fraction of matching minhash values
3. Signatures are tiny — can be stored in DHT, compared across network without transferring full premise data

### Application

- **Near-duplicate detection**: two premises with Jaccard > 0.8 are flagged for human review (possible refinement relation)
- **Cluster discovery**: LSH bands group similar premises into buckets without exhaustive comparison
- **Swarm queries**: "find me premises similar to this one" = broadcast signature, each node checks locally in O(1)

### Parameters (analogous to metagenomic k-mer sketches)

- Term-level k-grams (k=2 or k=3) as the shingling unit
- Sketch size: 256 hashes (good balance of accuracy vs. space)
- Bands: 16 bands of 16 rows for ~0.5 threshold detection

---

## 5. De Bruijn Graph on Term K-grams

### What it solves

Discovering the conceptual topology of the premise corpus. Which terms are used together? What are the "paths" through knowledge space?

### How it works

1. Extract term k-grams (k=3) from every premise: sliding window of 3 consecutive grounded terms
2. Each unique k-gram is a node. Overlapping k-grams are connected by edges.
3. The resulting graph reveals conceptual connectivity — heavily-traversed edges are foundational relationships

### Application

- **Domain boundary detection**: dense subgraphs = coherent domains. Sparse connections between subgraphs = interdisciplinary bridges.
- **Premise suggestion**: "given these terms, what other terms commonly co-occur?" = graph neighborhood traversal
- **Redundancy measurement**: high-coverage nodes (many premises traverse them) = foundational premises that everything depends on
- **Gap detection**: expected edges that are missing = premises that probably should exist but don't

### Analogy to genome assembly

In genome assembly, the de Bruijn graph reveals the structure of the genome. Here it reveals the structure of the knowledge space — which concepts connect to which, where the dense regions are, where the gaps are.

---

## 6. Domain-Based Sharding and Clustering

### Physical storage layout

Premises are stored clustered by domain. Within a domain, they're ordered by suffix array rank (which groups premises with similar term content together).

```
/store
  /biology/
    /molecular-genetics/    ← premises sharing molecular genetics terms
    /cell-biology/          ← premises sharing cell biology terms
  /history/
    /modern/
    /ancient/
  /mathematics/             ← (excluded for now, reserved)
  /linguistics/
```

### Why domain clustering matters for the suffix tree approach

The FM-index and suffix array work best when related content is physically proximal. Domain clustering means:
- Higher BWT compression ratios (similar sequences adjacent = longer runs)
- Faster domain-scoped queries (search only the relevant sub-index)
- Natural sharding unit for the swarm (nodes can host specific domains)

---

## 7. Compression Strategy

### Exploiting redundancy (the metagenomic insight)

In metagenomic datasets, the same organisms' sequences appear thousands of times. You don't store them thousands of times. You store the *distinct* sequences and count coverage.

Similarly: the same grounded terms appear across thousands of premises. The index stores each unique term once and references it everywhere.

### Layers of compression

1. **Term dictionary**: each grounded term (ontology URI + canonical form) gets a compact integer ID. Premises stored as arrays of integer IDs, not strings.
2. **BWT compression**: the Burrows-Wheeler transform of the term-ID corpus is highly compressible with run-length encoding due to redundancy.
3. **Succinct data structures**: rank/select bit vectors for the FM-index use ~1.05n bits instead of raw arrays.
4. **Delta encoding**: within a domain cluster, consecutive premises in suffix order often differ by only a few terms. Store deltas.

### Expected compression ratios

For a corpus of 1M premises averaging 10 terms each:
- Raw storage: ~10M term references × 4 bytes = 40MB
- With BWT + run-length: estimated 5-10MB
- With succinct structures: estimated 3-7MB
- Full premise text (for retrieval): stored separately, gzipped by domain cluster

---

## 8. Query Patterns

### Exact term lookup
"All premises containing chromosome(GO:0005694)"
→ FM-index backward search, O(m), returns count + positions

### Term co-occurrence
"All premises containing both human(Q15978631) AND chromosome(GO:0005694)"
→ Intersect FM-index results, or search for adjacency in suffix array

### Similar premise search
"Premises similar to this one"
→ Compute MinHash signature, LSH bucket lookup, return candidates above threshold

### Domain enumeration
"All premises in molecular genetics"
→ Scan domain cluster directly (pre-partitioned)

### Gap analysis
"What terms commonly co-occur with X that have no premise linking them?"
→ De Bruijn graph neighborhood of X, find edges with zero coverage

### Ancestry/dependency
"What premises does this conclusion depend on?"
→ Graph traversal on the `supports` / `requires` relation index (separate from term index)

---

## 9. Distributed Index Architecture

### Local node

Each node maintains:
- Full FM-index for locally-stored premises (domains it hosts)
- MinHash signatures for ALL known premises (tiny — 256 bytes each, storable for millions)
- De Bruijn graph for its hosted domains
- Relation graph (supports/contradicts/refines) for locally-relevant premises

### Swarm coordination

- **Signature broadcast**: when a new premise is added, its MinHash signature is gossiped to the network
- **Index shards**: domain-specific FM-indexes can be requested from domain-hosting nodes
- **Query routing**: if a query involves terms outside the local domain, route to the node hosting that domain's index

### Sync model

- Signatures sync always (small, universal)
- Full index syncs by domain subscription
- Premise content syncs on demand or by domain subscription

---

## 10. Technology: Go Implementation

### Libraries to evaluate

| Need | Candidate | Notes |
|------|-----------|-------|
| Suffix array | `index/suffixarray` (stdlib) | Built into Go, but basic |
| FM-index | Custom or port from sdsl-lite | May need to implement |
| BWT | Custom | Straightforward to implement in Go |
| MinHash | `dgryski/go-minhash` | Lightweight |
| LSH | `ekzhu/minhash-lsh` or custom | |
| Compression | `klauspost/compress` (zstd) | For raw text storage |
| Succinct bitvectors | `hillbig/rsdic` or custom | |
| Graph | `dominikbraun/graph` | For de Bruijn / relation graphs |

### Build vs. buy decision

The FM-index and suffix array are well-understood algorithms. A custom Go implementation tailored to our term-ID corpus (fixed alphabet size = number of distinct grounded terms) would likely outperform a generic library. The alphabet is much smaller than nucleotides mapped to integers — it's the set of all known ontology-grounded terms, which is large but bounded.

MinHash/LSH: use existing libraries. No need to reimplement.

De Bruijn graph: custom, built on top of a general graph library. The semantics are domain-specific.

---

## 11. Relationship to OrbitDB / Distributed Store

The indexing layer sits *on top of* the distributed storage layer. OrbitDB (or whatever CRDT-based store is chosen) handles:
- Replication and sync of raw premise records
- Conflict-free merge (append-only log of premises)
- Peer discovery and networking

The index layer handles:
- Fast local query resolution
- Similarity search
- Domain clustering
- Compression

These are separate concerns. A node receives new premises via the CRDT store, then indexes them locally. The index is not replicated directly — it's rebuilt from the premise data. (Signatures are the exception — they're small enough to replicate.)

---

## Open Questions

1. **Alphabet size**: how many distinct grounded terms will exist? Tens of thousands? Millions? Affects FM-index parameter choices.
2. **Update frequency**: how often do new premises arrive? Affects whether to use a static or dynamic suffix array.
3. **Term ordering within premises**: canonical ordering of terms matters for suffix-based search. Need deterministic rules.
4. **Cross-domain queries**: how to efficiently query across domain shards without a global index?
5. **Custom vs. off-the-shelf FM-index**: implement from scratch for optimal fit, or adapt an existing bioinformatics library?
