# Brief: grammar-based wikitext parser (validatorium-x3g.8)

For an implementing worker. Not specification. Rules: `docs/handoff/RULES.md`.

## What the owner asked for

- P-20261117-009: the ingestion tools are "real production code", "testable
  with Wikipedia type stuff", for "free text edited by different people with
  markup", using "standard ways": "parsers for different formats", "a parser
  definition language".
- P-20261117-011: the parser-definition languages he named are Ragel and
  ANTLR.

## What exists now (and why it is replaced)

`cmd/wikiextract/strip.go`, `templates.go`, `expand.go` remove and render
markup with hand-written Go regular expressions. That is not the
parser-definition approach above; this task replaces it.

## Scope

- A wikitext grammar in ANTLR or Ragel (the worker states which and why,
  from the two the owner named), with generated Go code checked in and the
  generator command documented.
- The same output contract: `ingest/wikipedia/schemas/article.schema.json`
  (version 1). Changing the contract needs the owner.
- Out of scope: templates that call Lua modules (no Lua runtime); they
  render as nothing, as now.

## Acceptance tests (run, paste the output in the task)

1. `go test -mod=vendor ./cmd/wikiextract` passes, including the schema test.
2. On the Simple English dump (fetch with
   `ingest/wikipedia/fetch-dump.sh simplewiki 20261101 DIR`), with the same
   flags, compare leftover markup against the current extractor; the counts
   the current extractor gives (2026-11-17, recorded on the task): lines
   containing `}}` 416, `]]` 55, `[[` 10, list-character starts 1,449,
   named entities 2. The new parser must not be worse on any of them.
3. Throughput with `-index -workers 4` on the same dump: the current
   extractor takes 58 s. Report the new figure.
4. Read 20 articles by eye against the live pages; list defects found.

## Hand-back

Commit message carries `validatorium-x3g.8`; `scripts/guard.sh` passes;
results pasted into the task notes.
