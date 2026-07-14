# Fine-Tuning Data Generation Specification

How the Validatorium prototype generates training data for fine-tuning FunctionGemma as a specialized tool-routing model.

## Overview

The prototype uses gemma4 (full-size) as a general-purpose model for all tasks: premise extraction, term grounding, validation, evidence assessment, and formatting. Every interaction where a human validates the output is captured as a training example. Once sufficient examples accumulate, we fine-tune [FunctionGemma](https://ai.google.dev/gemma/docs/functiongemma/finetuning-with-functiongemma) (270M params) to handle mechanical tool-routing decisions at high speed, freeing gemma4 for reasoning-heavy work.

The result is a two-tier architecture:
- **FunctionGemma (270M):** fast tool router — decides *which* tool to call and with what arguments.
- **gemma4 (full-size):** reasoning engine — handles the actual intellectual work within each tool.

---

## 1. The Training Data Flywheel

### Phase 1: Data Accumulation

The prototype runs with gemma4 handling everything. Every interaction follows this cycle:

1. User submits input (text, query, claim, source material).
2. gemma4 decides which tool to invoke and with what arguments.
3. The tool executes and produces output.
4. A human reviews the output and provides a verdict.
5. The complete interaction is logged.

Every validated interaction is stored as:

```
[input, action_taken, tool_called, arguments, output, human_verdict]
```

### Phase 2: First Fine-Tune

When **500+ validated examples** accumulate (with reasonable coverage across all tools), trigger the first fine-tuning run. Options:

- **No-code:** Upload CSV to [FunctionGemma Tuning Lab](https://huggingface.co/spaces/google/functiongemma-tuning-lab)
- **Programmatic:** Use the FunctionGemma fine-tuning pipeline with HuggingFace datasets

### Phase 3: Deploy and Iterate

Deploy the fine-tuned FunctionGemma as the routing layer. Continue logging ALL interactions — both the router's decisions and their human verdicts. Re-tune periodically with the growing dataset.

**The flywheel:** More usage → more training data → better routing → faster system → more usage.

---

## 2. What Gets Logged

Every interaction produces a log entry containing:

| Field | Description |
|-------|-------------|
| `input_text` | The text, query, or claim submitted by the user |
| `tool_called` | The tool that was actually invoked |
| `tool_arguments` | Arguments passed to the tool (JSON) |
| `output` | The output produced by the tool |
| `human_verdict` | `correct`, `incorrect`, or `partially_correct` |
| `correction` | If incorrect: what SHOULD have happened (correct tool + arguments) |
| `timestamp` | ISO 8601 timestamp |
| `session_id` | Unique session identifier |
| `model_version` | Which model made the routing decision (gemma4, functiongemma-v1, etc.) |

---

## 3. Training Data Schema

The canonical training format is CSV with four columns:

```csv
input_text,correct_tool,correct_arguments,context
```

| Column | Description |
|--------|-------------|
| `input_text` | The user's input or the current state of processing |
| `correct_tool` | The tool that should be called |
| `correct_arguments` | JSON object of the correct arguments |
| `context` | Relevant surrounding context: current domain, previous steps in the pipeline, premise state |

### Example Rows

```csv
"Aspirin inhibits COX-2 enzyme activity",ground_term,"{""term"": ""COX-2"", ""domain"": ""biochemistry""}","{""step"": ""term_identification"", ""domain"": ""pharmacology""}"
"PMID:12345678 shows aspirin reduces inflammation",assess_evidence,"{""source"": ""PMID:12345678"", ""claim"": ""aspirin reduces inflammation""}","{""step"": ""evidence_linking"", ""premise_id"": ""p-0042""}"
"Aspirin was discovered in 1897 and inhibits platelet aggregation",decompose_premise,"{""text"": ""Aspirin was discovered in 1897 and inhibits platelet aggregation""}","{""step"": ""extraction"", ""reason"": ""compound_claim""}"
```

---

## 4. Tool Definitions

FunctionGemma routes to these tools:

| Tool | Purpose | Key Arguments |
|------|---------|---------------|
| `query_wikidata` | Look up an entity for grounding | `entity`, `domain`, `language` |
| `search_pubmed` | Find primary literature evidence | `query`, `date_range`, `mesh_terms` |
| `validate_structure` | Check premise against validation rules | `premise_text`, `rules` |
| `decompose_premise` | Split compound claim into atomic sub-premises | `text`, `reason` |
| `check_temporal` | Verify temporal compatibility between entities | `entity_a`, `entity_b`, `relation` |
| `ground_term` | Resolve a jargon term to ontology URI | `term`, `domain`, `preferred_ontology` |
| `assess_evidence` | Evaluate if a source supports a claim | `source`, `claim`, `evidence_type` |
| `format_argdown` | Convert premise to Argdown notation | `premise_text`, `grounded_terms`, `evidence_links` |

Each tool has a defined schema. FunctionGemma's job is to select the correct tool AND produce valid arguments for it.

---

## 5. Quality Criteria for Training Examples

### Inclusion Rules

1. **Human-validated only.** No example enters the training set without a human verdict.
2. **Corrections are gold.** Incorrect model outputs that were corrected are especially valuable — they teach the model what NOT to do.
3. **Log both sides of corrections.** If the human corrected the tool selection, store both:
   - The wrong choice (as a negative example)
   - The right choice (as a positive example)

### Balance Requirements

4. **Tool coverage.** Examples must cover all 8 tools. If one tool dominates (e.g., `ground_term` accounts for 60% of examples), oversample underrepresented tools or undersample the dominant one during training.
5. **Edge cases are premium.** Ambiguous situations where the correct tool isn't obvious are the most valuable training examples. Flag these during logging.

### Minimum Viable Training Set

- **500 examples** minimum for first fine-tune
- **≥40 examples per tool** (minimum coverage threshold)
- **≥50 correction examples** (negative + corrected positive pairs)
- **≥30 edge-case examples** (human flagged as ambiguous)

---

## 6. Data Format for Fine-Tuning

### FunctionGemma Conversational Format

FunctionGemma expects training data in conversational format with function_call tokens:

```
User: Aspirin inhibits COX-2 enzyme activity
Model: <start_function_call>call:ground_term{term:COX-2, domain:biochemistry}<end_function_call>
```

```
User: PMID:12345678 shows aspirin reduces inflammation in rat models
Model: <start_function_call>call:assess_evidence{source:PMID:12345678, claim:aspirin reduces inflammation, evidence_type:animal_model}<end_function_call>
```

### Conversion Pipeline

Raw JSONL logs → filtered by verdict → converted to conversational format → output as training file.

```
training_logs.jsonl  →  filter (verdict=correct OR correction exists)
                     →  transform to FunctionGemma format
                     →  training_data.txt (conversational) OR training_data.csv (Tuning Lab)
```

### FunctionGemma Tuning Lab Format

The [no-code Tuning Lab](https://huggingface.co/spaces/google/functiongemma-tuning-lab) accepts CSV with columns:

```csv
user_prompt,tool_name,tool_arguments
```

Map from canonical schema:
- `user_prompt` ← `input_text` (optionally prepend `context` as system prompt)
- `tool_name` ← `correct_tool`
- `tool_arguments` ← `correct_arguments`

---

## 7. Iteration Cycle

| Period | Activity | Target |
|--------|----------|--------|
| Week 1–4 | Accumulate examples from prototype usage | 500+ validated examples |
| Week 5 | First fine-tune | Deploy FunctionGemma v1 |
| Week 5+ | Deploy fine-tuned model alongside gemma4 | Measure routing accuracy |
| Monthly | Re-tune with accumulated data | Improving accuracy each cycle |

### Metrics to Track

| Metric | Definition | Target |
|--------|------------|--------|
| Routing accuracy | % of tool selections matching human verdict | >90% after first tune |
| Latency | Time from input to tool selection | <100ms (FunctionGemma vs ~2s gemma4) |
| False routing rate | % of calls routed to wrong tool | <5% at steady state |
| Coverage | % of tools with >40 examples | 100% before first tune |
| Correction rate | % of interactions requiring human correction | Decreasing over time |

### Continuous Improvement

After deployment:
1. FunctionGemma makes routing decisions.
2. ALL decisions are still logged (with model_version = functiongemma-vN).
3. Humans continue to validate a sample of outputs.
4. Incorrect routings from FunctionGemma become high-value correction examples.
5. Monthly re-tune incorporates new data.
6. Each iteration: routing gets faster and more accurate.

---

## 8. Storage of Training Data

### Location

Training logs are stored locally at:

```
.validatorium/training/
├── logs/
│   ├── 2026-07.jsonl       # Monthly log files
│   ├── 2026-08.jsonl
│   └── ...
├── exports/
│   ├── training_v1.csv     # Exported for Tuning Lab
│   └── training_v1.txt     # Conversational format
└── metrics/
    └── accuracy.jsonl       # Per-iteration metrics
```

### Format

**JSONL** — one JSON object per line, append-only. Example entry:

```json
{
  "input_text": "Aspirin inhibits COX-2 enzyme activity",
  "tool_called": "ground_term",
  "tool_arguments": {"term": "COX-2", "domain": "biochemistry"},
  "output": {"uri": "https://www.wikidata.org/wiki/Q21107848", "label": "Prostaglandin-endoperoxide synthase 2"},
  "human_verdict": "correct",
  "correction": null,
  "timestamp": "2026-07-14T10:23:41Z",
  "session_id": "sess-a1b2c3",
  "model_version": "gemma4"
}
```

### Boundaries

- **Separate from the premise store.** Training data is operational data, not knowledge. It lives outside the distributed premise database.
- **Not distributed.** This is per-node data. It does not sync via `refs/dolt/data` or any P2P mechanism.
- **Exportable.** Scripts convert JSONL to CSV (for Tuning Lab) or HuggingFace datasets format (for programmatic training).

---

## 9. Privacy Considerations

| What | Shareable? | Reason |
|------|-----------|--------|
| Raw training logs | **No** | Contains user inputs and session data |
| Fine-tuned model weights | **Yes** | Model weights don't contain recoverable raw inputs |
| Anonymized routing examples | **Opt-in** | Community can pool anonymized tool-routing pairs |
| Aggregated metrics | **Yes** | No PII in accuracy/latency numbers |

### Default Posture

**Your data, your model, your node.**

- Training data stays local by default.
- Fine-tuned weights can be shared if the node operator chooses.
- A community could pool anonymized routing examples (input text stripped or generalized, only tool + arguments + verdict retained) to train a shared routing model.
- No training data leaves the node without explicit operator action.

### Anonymization for Sharing

If a node operator opts to share routing examples:

1. Replace specific entity names with type placeholders: `"Aspirin"` → `"[DRUG]"`
2. Remove session IDs and timestamps.
3. Retain only: tool, arguments (with placeholders), verdict, correction.
4. This preserves the routing signal while removing identifying content.

---

## References

- [FunctionGemma Fine-Tuning Guide](https://ai.google.dev/gemma/docs/functiongemma/finetuning-with-functiongemma)
- [FunctionGemma Tuning Lab (no-code)](https://huggingface.co/spaces/google/functiongemma-tuning-lab)
- [Validatorium Architecture](../../README.md)
