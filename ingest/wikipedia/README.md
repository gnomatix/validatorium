# Wikipedia ingestion (data feed)

A data feed into Validatorium, not part of the specification. It turns a
Wikipedia dump into **candidate** premises in a working store. Candidates are
drafts: importing makes nothing valid and publishes nothing. Each premise cites
the exact article revision it came from, as a pointer to the source.

## Pipeline

1. **Dump.** `enwiki-latest-pages-articles-multistream.xml.bz2` and its index
   from <https://dumps.wikimedia.org/enwiki/latest/>, checked against the
   published checksums: `ingest/wikipedia/fetch-dump.sh enwiki latest DIR`
   (`VERIFY_ONLY=1` checks files already downloaded, e.g. by torrent, and
   writes `DIR/manifest.json`).
2. **Extract articles** with `wikiextract` (in this repository,
   `cmd/wikiextract`):

   ```sh
   go build -o wikiextract ./cmd/wikiextract
   ./wikiextract -in enwiki-latest-pages-articles-multistream.xml.bz2 > articles.jsonl
   # parallel: decode the bz2 streams concurrently using the published index
   ./wikiextract -in enwiki-latest-pages-articles-multistream.xml.bz2 \
     -index enwiki-latest-pages-articles-multistream-index.txt.bz2 -workers 24 > articles.jsonl
   ./wikiextract -in enwiki-latest-pages-articles-multistream.xml.bz2 -max 500 > sample.jsonl
   ./wikiextract -in enwiki-latest-pages-articles-multistream.xml.bz2 -titles elements.txt > elements.jsonl
   ```

   One JSON object per article: `wiki`, `page_id`, `title`, `rev_id`, plain
   `text`, and `refs`. Citations are kept: each `<ref>` becomes a `[ref:N]`
   marker in the text and an entry in `refs` with its URL, DOI, PMID, ISBN
   and title. Redirects and non-article pages are skipped.
3. **Split** the article file into chunks, e.g. `split -l 1000 -d -a 5
   articles.jsonl chunks/articles.`
4. **Decompose** each article with a model using
   [`WORKER_PROMPT.md`](WORKER_PROMPT.md). Output: JSONL, one premise per
   line. `sge-array.sh` runs this as a Grid Engine array job, one task per
   chunk.
5. **Load** into a working store:

   ```sh
   cat out/*.jsonl | val import -working-store wiki.db -file - -agent "worker:<model>@<date>" -articles articles.jsonl
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

- every `quote` occurs verbatim in the article text (done by `val import -articles`: lines whose quote is not found are rejected and counted);
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
| `citations` | support-graph entries pointing at each cited source (DOI, then PMID, then URL, then ISBN) |
| `depends_on` | support-graph entries `urn:validatorium:premise:<uuid>` |
| `kind: conclusion` | not stored as a premise (counted only) |
| `scope` | kept in the JSONL only; the statement text carries the scope |
