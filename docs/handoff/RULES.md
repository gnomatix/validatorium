# Working rules

Each rule exists because breaking it damaged this project on 2026-11-17.
`scripts/guard.sh` checks the ones a script can check; a reviewer checks the
rest (checklist at the end).

## 1. The specification is fixed and owner-only

- `docs/specs/` is written and revised by the project owner only. A worker
  never edits it, never drafts replacement text, never adds a new spec file.
- Conversation about usage, use cases or data sources changes nothing in the
  specification and creates no new scope.
- A gap between the specification and a need is reported as a Beads record
  of type `decision`, assigned to the owner, quoting the section. Work on
  that point stops there; other work continues.
- Check: `scripts/guard.sh` compares `docs/specs/` with
  `docs/specs.sha256`. Any difference fails. Only the owner regenerates the
  manifest (`scripts/guard.sh --owner-accept-specs`).

## 2. Reuse; do not invent

Validatorium encodes work that already exists (Micropublications,
ontologies, Wikidata, IPFS, OrbitDB, OpenPGP) in Go. Before adding a field,
a label, a category or a format:

1. Read the specification section for that record (`data-model.md`).
2. Look for the existing ontology or standard that covers the need, and
   verify its identifier at its primary source.
3. If neither covers it, stop and file a `decision` record (rule 1).

Free-text labels standing in for controlled vocabulary are a defect.
`domain-system.md`: "Annotation is mandatory, not optional. Ungrounded
jargon renders a premise structurally invalid."

## 3. Every code change belongs to a Beads task

- Code under `cmd/` or `internal/` is changed only for an open Beads task
  whose text describes the change. The commit message carries the task id
  (`validatorium-xxx`).
- No task, no code. A use-case walkthrough, a discussion or an example is
  not a task.
- Check: `scripts/guard.sh` fails for any commit that touches `cmd/` or
  `internal/` without a `validatorium-` id in its message.

## 4. Use cases are examples, not requirements

Nothing from an example (names, phrases, scenarios) appears in code, test
names, field names or labels. While a walkthrough record is open
(`validatorium-845`), its flags are recorded verbatim and nothing else is
produced from it.

## 5. Validity

Validity comes only from an evaluation method returning true against
evidence (`data-model.md` §1). No voting, no consensus, no counting of
sources, no weight for authority or reputation
(`project-principles.md`, core rules).

## 6. Verification

- Go to the primary source first. Absence from a search result is not
  evidence against a statement.
- Report what was run and what it printed. "Unit-tested" and "run on real
  data" are different states; say which.

## 7. Repository conduct

- Branch `claude/validatorium-implementation-td1o9u`; no pull requests; no
  merges or releases unless the owner says so.
- Commits carry no tool or model attribution.
- No secrets, no personal details of any person, in any file or tracker
  record. The tracker export `.beads/issues.jsonl` is in the repository.
- No uploads to third-party services.
- Sub-agents are launched only with the owner's approval of agent type and
  model.
- Quality gates before every commit: `scripts/guard.sh`.

## 8. Sources for building

Code is built from two sources only: text in `docs/specs/`, or the owner's
own words, quoted with where they were said. Tracker records that summarize
a decision ("DECISION: ...", "Owner direction ...") were mostly written by
agents and are not sources on their own; check the summary against the
specification or the owner's wording before building from it. On
2026-11-17 two features built from such summaries (record signing, a
validation route) had to be reverted.

## Reviewer checklist (for the reviewing model or person)

For each delivered change, answer each line with evidence (file and line,
or command and output):

1. Which Beads task authorizes it? Does the diff stay inside that task?
2. Does it touch `docs/specs/`? (Must be no.)
3. Does it add a field, label, enum value or format? If yes: which spec
   section defines it, or which ontology term, verified where?
4. Does any test, name or comment come from a use-case example?
5. Did `scripts/guard.sh` pass? Paste the last lines.
6. Is every "works" claim backed by a command that was run, with output?
7. Which specification section or quoted owner statement does the change
   implement? (A tracker summary alone is not enough, rule 8.)
