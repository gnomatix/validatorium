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
| Wikipedia extractor | `cmd/wikiextract` | Never run on a real dump. Markup stripping is regular expressions (`strip.go`), a heuristic. Single-threaded. |
| Import of worker output | `cmd/val/import.go` | Never run on model output. |
| Grid Engine template | `ingest/wikipedia/sge-array.sh` | Never submitted to a scheduler. `WORKER_CMD` is a placeholder. |
| Worker instructions | `ingest/wikipedia/WORKER_PROMPT.md` | Never given to a model. |

## C. Do not build on without the owner's decision

| Item | Path | Problem |
|---|---|---|
| Worker output `scope` object with free keys (`time`, `place`, ...) | `WORKER_PROMPT.md`, `cmd/val/import.go` (`ImportRecord.Scope`) | Invented free-form structure; not defined by the specification; not stored by import. |
| Worker output allows empty `ontology_uri` | `WORKER_PROMPT.md` rule 7 | The specification requires grounding; the prompt lets a term through with only a free-text `sense`. |
| Case study field names (`evidence_type: "self_report"`, `scope: {as_of}`) | `docs/case-studies/jeffs-favorite-food.md` | Invented labels where evidence and time ontologies exist. Not a model for any implementation. |
| Glossary | `docs/working-terms.md` | Contains category names coined in conversation. Not specification. |
| Reference list | `resources/references.md` | Some entries are marked unverified. |
| Wikipedia epic `validatorium-1tc` | Beads | Created from usage talk; overlaps `validatorium-x3g`. Use `x3g`. |
| Curation epic `validatorium-8e6` | Beads | Task titles came from conversation; the specification text is the owner's to write. |

## D. Not implemented

See `docs/BUILD_STATE.md` "Not yet implemented". Notably: `temporal_policy`
(`data-model.md` §2, §8) has no code; promotion does not sign records;
no indexed search (the API scans).

## E. Known incident state

- An audit of the 2026-11-16/17 session's records was started and did not
  complete. No verified audit report exists.
- The tracker export `.beads/issues.jsonl` contains personal details of a
  private individual in `validatorium-845` notes. Removing them from
  history rewrites published commits; that is the owner's decision.
