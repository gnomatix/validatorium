# Brief: compiled multi-regex stage with Vectorscan (validatorium-x3g.9)

For an implementing worker. Not specification. Rules: `docs/handoff/RULES.md`.

## What the owner asked for

- P-20261117-009: "vector scan, hyperscan allows you to basically construct a
  library of different regexes"; "it's all scripted"; then FunctionGemma,
  escalating to larger models.
- P-20261117-011: Ragel is a Vectorscan build dependency.

## What exists now

- `wikiextract -sentences`: rule-based sentence spans with citations bound
  per sentence, Go `regexp` only (`cmd/wikiextract/sentences.go`).
- Escalation between models: `ingest/wikipedia/worker-local.sh`
  (`WORKER_ESCALATE`).

## Scope

- Vectorscan (or Hyperscan) through a Go binding, with Go `regexp` as the
  fallback when the library is absent; the build must still pass without it.
- The pattern library is a file of named patterns, loaded at start, so
  patterns are data, not code.
- Clause splitting: no rules for it are written by the owner or in the
  specification. Do not invent them; ask the owner, or leave clause
  splitting out.

## Environment

Not installed in the cloud container (checked 2026-11-17). Install on the
owner's machine; record the library version used.

## Acceptance tests

1. The current sentence output (`-sentences`) is unchanged on the Simple
   English dump when the Vectorscan path is used (compare byte for byte).
2. Report time for both paths on that dump with `-index -workers 4`.
3. `scripts/guard.sh` passes on a machine without the library.

## Hand-back

Commit message carries `validatorium-x3g.9`; results pasted into the task.
