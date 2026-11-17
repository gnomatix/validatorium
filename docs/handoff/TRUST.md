# Trust register

State at commit `d59bb96` plus this handoff. "Tests" means
`go test -mod=vendor -count=1 -short ./...` passes (run 2026-11-17, all
packages ok).

## A. Verified by tests; safe to continue from

| Component | Path | Evidence |
|---|---|---|
| Record model, canonical bytes, content IDs | `internal/model`, `internal/storage` | unit tests |
| Attestations (true / false / indeterminate, expiry) | `internal/model/attestation.go` | unit tests |
| Term records and replacement traversal | `internal/model/term.go` | unit tests |
| bbolt working store | `internal/bbolt` | unit tests, benchmarks |
| Global store on embedded Kubo + go-orbit-db | `internal/ipfsnode`, `internal/orbit` | unit tests; two-node replication test (skipped by `-short`) |
| Promotion seam | `internal/promote` | unit and end-to-end tests |
| Namespaces, releases | `internal/namespace`, `internal/release` | unit tests |
| Evaluation registry, file-digest evaluator | `internal/evaluate` | unit tests |
| Automatic publication, `val publish` | `internal/autopublish`, `cmd/val/publish.go` | unit tests; one manual offline end-to-end run |
| Wikidata revision-pinned reads and evaluator | `internal/wikidata` | unit tests against recorded responses |
| OpenPGP identities, detached signatures | `internal/identity`, `internal/api` | unit tests |
| Premise query and candidate API | `internal/api/premises.go` | unit tests |

## B. Unit-tested only; never run on real data

| Component | Path | What is missing |
|---|---|---|
| Wikipedia extractor | `cmd/wikiextract` | Run on simplewiki-20261101 (215,035 articles after the 2026-11-17 template and formula fixes, 214,908 before; 150 s serial, 58 s with `-index -workers 4`, outputs byte-identical); not on enwiki; output read for one article only. Markup stripping is regular expressions (`strip.go`), a heuristic. Single-threaded. |
| Import of worker output | `cmd/val/import.go` | Never run on model output. |
| Sentence spans (`wikiextract -sentences`) | `cmd/wikiextract/sentences.go` | Run on simplewiki-20261101: 3,917,627 sentences, 602,984 with citations; byte spans checked non-empty on 1,233,975 of them; about 20 read by eye. Rules only (no clause splitting). |
| Dump fetch and verify | `ingest/wikipedia/fetch-dump.sh` | Run on simplewiki 20261101 (both files verified; tampered file fails with status 1); not on enwiki. |
| Worker command | `ingest/wikipedia/worker-local.sh` | Tested against a mock server only; never against a model. |
| Grid Engine template | `ingest/wikipedia/sge-array.sh` | Run locally with `SGE_TASK_ID` set (13 chunks, mock model, all output imported); never submitted to a scheduler. |
| Worker instructions | `ingest/wikipedia/WORKER_PROMPT.md` | Never given to a model. `supported_by` (the SUPPORTS relation of `premise-decomposition.md` §3.2) added 2026-11-17. |
| Template expansion (`wikiextract -templates`) | `cmd/wikiextract/expand.go` | About 30 times slower than without it (partial run); Lua-module templates expand to nothing. Off by default; leave it off. |
| Graph view, argument builder | `internal/api/view.go`, `builder.go` | Built from requests that the tracker records (`72i`, `dmz`, created 2026-07-14) mark as the owner's verbatim words; that conversation is not available in this session, so the quotes are unconfirmed. Rendered in headless Chromium only. |
| Draft text from premises (`val draft`) | `cmd/val/draft.go`, `internal/llm` | Built from `llm-integration.md` and a request the tracker record `3xe` (created 2026-07-14) marks as the owner's verbatim words, unconfirmed here; tested against a mock model only. |

## C. Do not build on without the owner's decision

| Item | Path | Problem |
|---|---|---|
| Worker output `scope` object with free keys (`time`, `place`, ...) | `WORKER_PROMPT.md`, `cmd/val/import.go` (`ImportRecord.Scope`) | Invented free-form structure; not defined by the specification; not stored by import. |
| Worker output allows empty `ontology_uri` | `WORKER_PROMPT.md` rule 7 | The specification requires grounding; the prompt lets a term through with only a free-text `sense`. |
| Case study field names (`evidence_type: "self_report"`, `scope: {as_of}`) | `docs/case-studies/jeffs-favorite-food.md` | Invented labels where evidence and time ontologies exist. Not a model for any implementation. |
| Glossary | `docs/working-terms.md` | Contains category names coined in conversation. Not specification. |
| Reference list | `resources/references.md` | Some entries are marked unverified. |
| Wikipedia epic `validatorium-1tc` | Beads | Created by the agent from a conversation, without a request to create records; overlaps `validatorium-x3g`. Use `x3g`. |
| Decision records `elu`, `7oj`, `o2u`, `wfl`, `8gv`, `cfb`, `ilv` | Beads | Their DECISION text is an agent-written summary, not the owner's words (labelled 2026-11-17). Build only from the specification or the owner's quoted words (`RULES.md` rule 8). |
| Method execution inside `val` | `internal/evaluate`, `internal/wikidata` | `data-model.md` §11 places method execution with an orchestrator outside the core; decision `validatorium-1hx`. |
| Curation epic `validatorium-8e6` | Beads | Task titles came from conversation; the specification text is the owner's to write. |

## D. Not implemented

See `docs/BUILD_STATE.md` "Not yet implemented". Notably: `temporal_policy`
(`data-model.md` §2, §8) has no code; records are not signed (promotion
signing was built from a tracker summary and reverted on 2026-11-17);
no indexed search (the API scans).

## E. Known incident state

- An audit of the 2026-11-16/17 session completed on 2026-11-17. Its findings
  on tracker records, saved rules and documents are not yet applied
  (`validatorium-1pa`).
- Personal details of a private individual were removed from the
  `validatorium-845` notes on 2026-11-17. They remain in earlier commits of
  `.beads/issues.jsonl` (from `b0ec3ff`); removing them there rewrites
  published history (`validatorium-59m`).
