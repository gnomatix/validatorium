# Working terms

> **Status (2026-11-17).** Agent-written, without the owner's approval. The
> "Meaning here" column is the agent's paraphrase; entries sourced to the
> owner are not quotations. Where a definition differs from the owner's
> words, the owner's words govern. Not specification.

Terms established in working sessions on Validatorium, with the meaning they
carry in this project. This is a glossary for people and agents working on the
project, not specification: where a term also appears in `docs/specs/`, the
spec governs.

| Term | Meaning here | Source |
|---|---|---|
| premise | One atomic factual assertion about observable reality, with its significant terms grounded to versioned ontology records. | `docs/specs/data-model.md` §1, §3 |
| self-referential premise | A premise whose author, subject and source of evidence are the same person (the same key): what a person states about themselves. | Use-case walkthrough, 2026-11-16 (`docs/case-studies/jeffs-favorite-food.md`) |
| anti-premise | The negation of a premise, established by the same evidence through exclusion within a defined partition ("the sky is not red" when the measured value lies in the blue range). | Working session, 2026-11-16 |
| core premise | A premise that much other knowledge rests on, measured by its centrality in the premise graph. Centrality measures importance, not truth. | Working session, 2026-11-16 |
| support | The independent evidence and orthogonal methods behind a premise. Never a count of people or sources asserting it. | `docs/specs/project-principles.md` §6 |
| orthogonal methods | Independent ways of evaluating the same premise that share no assumptions; where they agree, the premise has passed independent checks. | Owner, claude.ai conversation "Initial greeting", 2026-07-14, messages 88 and 94 |
| fail bucket | Where a premise goes when orthogonal methods disagree: it is examined, the cause found, and the methods improved. | Owner, same conversation, message 94 |
| evidential weight | What counts toward a premise being true or false. Only evidence, examined by the stated method, carries it; attribution, authority, credentials and reputation carry none. | `docs/specs/project-principles.md` §6; `docs/specs/distributed-datastore.md` §9 |
| candidate | A premise recorded in a working store but not yet evaluated true and published. "Provisional" refers to this state. | `internal/storage` working store; owner, 2026-11-16 |
| indeterminate | The attestation result when evidence is absent or inconclusive. Never the same as false: absence from a source counts nothing against a premise. | `docs/specs/data-model.md` §6 |
| overloaded premise | Wording that means two things at once, both intended. It is split into one premise per reading, each evaluated independently. | Owner, 2026-11-16 |
| piggybacking | A weak premise borrowing credibility from its proximity or link to a core premise, without support of its own. | Owner, 2026-11-16 |
| "yes, and" traversal | The interaction pattern for entering statements: accept the statement, assume nothing about the person, walk back to the root premise, then build forward one premise at a time in the person's own words. | Owner, 2026-11-16 (walkthrough flag F9) |
