# Models for the work

Named by role. Check each vendor's current model list before starting; the
names change faster than this file.

| Role | Requirement | Suggested |
|---|---|---|
| Implementation of a Beads task in Go | Top coding model of the tool in use, highest reasoning setting | Codex: the newest `-codex` model in its model picker, effort high or max. Kiro: its default top model. |
| Review of delivered changes | A different vendor's model from the implementer, using the checklist in `RULES.md` | Gemini (top Pro model) |
| Article decomposition (bulk) | Local, instruction-following, emits strict JSON Lines | To be measured in stage 4 of the work plan; candidates are whatever GGUF models fit the machine |
| Decomposition cross-check | A second, different local model on the same articles | Same |

Constraint from `llm-integration.md`: "No data leaves the local machine.
Ever." applies to the model pipeline; bulk decomposition runs on local
models. The model proposes; evaluation determines validity.
