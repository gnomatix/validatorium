# Wikipedia ingestion (data feed)

A data feed into Validatorium, not part of the specification. It turns a
Wikipedia dump into **candidate** premises in a working store. Candidates are
drafts: importing makes nothing valid and publishes nothing. Each premise cites
the exact article revision it came from, as a pointer to the source.

## Pipeline

1. **Dump.** `enwiki-latest-pages-articles-multistream.xml.bz2` and its index
   from <https://dumps.wikimedia.org/enwiki/latest/>, checked against the
   published checksums.
2. **Extract articles** to one JSON object per line with `wiki`, `page_id`,
   `title`, `rev_id` and plain `text` (wikitext markup removed). Any
   extractor that keeps the revision id works.
3. **Split** the article file into chunks, e.g. `split -l 1000 -d -a 5
   articles.jsonl chunks/articles.`
4. **Decompose** each article with a model using
   [`WORKER_PROMPT.md`](WORKER_PROMPT.md). Output: JSONL, one premise per
   line. `sge-array.sh` runs this as a Grid Engine array job, one task per
   chunk.
5. **Load** into a working store:

   ```sh
   cat out/*.jsonl | val import -working-store wiki.db -file - -agent "worker:<model>@<date>"
   ```

   It prints how many premises were created, already present, skipped as
   conclusions, and rejected; rejected lines and reasons go to standard error.
   Importing is idempotent: each premise UUID is derived from the agent,
   wiki, page id, revision id and local id, so re-running a file creates
   nothing new. Different workers (different `-agent`) decomposing the same
   article keep separate records, so their outputs can be cross-checked:
   where independent workers agree, the decomposition is consistent; where
   they differ, the article goes back for review.

## Iterating on quality

Run steps 4–5 on a small sample first (a few hundred articles), read the
output, tighten `WORKER_PROMPT.md`, and repeat before the full run. Checks
worth automating between iterations:

- every `quote` occurs verbatim in the article text;
- no statement contains a pronoun whose referent is outside the statement;
- no statement contains "and" joining two independent assertions;
- every `ontology_uri` that is not empty resolves to a Wikidata entity whose
  label matches `surface_form`.

## What is kept where

| Worker field | In the working store |
|---|---|
| `statement` | the premise statement |
| `terms` | the statement's term groundings |
| `source.url`, `quote` | support-graph entry (kind `data`) pointing at the revision, with the quote as its note |
| `depends_on` | support-graph entries `urn:validatorium:premise:<uuid>` |
| `kind: conclusion` | not stored as a premise (counted only) |
| `scope` | kept in the JSONL only; the statement text carries the scope |
