# Premise Decomposition Specification

How compound claims are broken into atomic premises in Validatorium.

## 1. What Makes a Premise Atomic

A premise is atomic when it satisfies ALL of the following:

1. **Contains exactly one claim.** A single factual assertion, no more.
2. **Cannot be decomposed into simpler, independently verifiable sub-claims.** If you can split it into parts that each stand alone as verifiable facts, it is not yet atomic.
3. **Contains no derivation connectives.** The following signal compound or derived claims:
   - `because` (joins a fact with its explanation)
   - `therefore` (signals a conclusion follows)
   - `and` when joining separate claims (not when part of a single atomic fact)
   - `which means` (signals a derivation)
4. **Does not smuggle its conclusion into its own statement.** A premise that presupposes the very thing it's meant to support is circular, not atomic.
5. **Contains no operation or derivation.** If the claim requires performing a computation, applying a rule, or drawing an inference to arrive at its truth value, it is a conclusion — not a premise.

### Quick Test

> Can this claim be verified by direct observation or direct reference to a definition?

- YES → it may be a premise.
- NO, it requires applying rules/operations to other claims → it is a conclusion.

## 2. The Premise vs. Conclusion Distinction

This is the foundational distinction in Validatorium. Getting it wrong collapses the entire system.

| | Premise | Conclusion |
|---|---------|-----------|
| **Verified by** | Direct observation or reference | Derivation from premises |
| **Role** | Axiom — the thing you check against evidence | Theorem — the thing that follows |
| **Decomposable?** | No (atomic) | Yes (into supporting premises) |
| **Contains operations?** | No | Yes (the derivation itself) |

### The Critical Insight

**If you can derive a claim from more basic claims, it is a conclusion, not a premise.**

Premises are the bedrock. They are what you verify directly. Conclusions follow from them. If you put a conclusion in as a premise, you have done nothing useful — you have not shown WHY it is true.

### Example: Why '2 + 2 = 4' Is a Conclusion

The statement `2 + 2 = 4` is **not** a premise. It is derivable from more basic claims:

**Premises:**
- P1: `2 is a natural number`
- P2: `The successor of 1 is 2`
- P3: `The successor of 3 is 4`
- P4: `Addition of n + m means applying the successor operation m times starting from n`

**Derivation:**
- From P4: `2 + 2` means "apply successor twice starting from 2"
- First application: successor of 2 = 3 (from the definition of successor on natural numbers)
- Second application: successor of 3 = 4 (from P3)
- Therefore: `2 + 2 = 4`

Each premise is independently verifiable — you can observe the definition of natural numbers, the successor function, and the addition operation directly. The conclusion `2 + 2 = 4` **follows** from applying the operation defined in the premises.

Storing `2 + 2 = 4` as a premise hides the reasoning. It asserts the result without showing the structure that produces it.

## 3. Decomposition Rules

### 3.1 Conjunction: 'X and Y'

**Pattern:** `X and Y` where X and Y are independent claims.

**Decomposition:**
- Premise: `X`
- Premise: `Y`

**Example:**
> "The Earth orbits the Sun and the Moon orbits the Earth"

→ Premise 1: `The Earth orbits the Sun`
→ Premise 2: `The Moon orbits the Earth`

**Exception:** Do NOT decompose when `and` joins elements of a genuinely single atomic fact:
> "DNA contains adenine and thymine"

This describes a single observable property of DNA's composition. Whether to decompose depends on whether the parts are independently significant in context.

### 3.2 Causal Explanation: 'X because Y'

**Pattern:** `X because Y`

**Decomposition:**
- Premise: `Y` (the explaining fact)
- Premise: `X` (the explained fact)
- Relation: `Y SUPPORTS X` (the causal/explanatory link — this relation itself may require additional premises to establish)

The causal link between Y and X is itself a claim that may need its own premises.

### 3.3 Inference: 'X therefore Y'

**Pattern:** `X therefore Y`

**Decomposition:**
- Premise: `X`
- Conclusion: `Y` (supported by `X`, but `Y` may require additional premises to be fully derived)

Note that `Y` is explicitly marked as a **conclusion**, not stored as a premise.

### 3.4 Implication: 'X, which means Y'

**Pattern:** `X, which means Y`

**Decomposition:**
- Premise: `X`
- Conclusion: `Y` (the derivation from X to Y must be made explicit through additional premises)

The phrase "which means" signals a derivation step. The derived claim is a conclusion.

### 3.5 Compound Subjects

**Pattern:** `A and B are Z`

**Decomposition:**
- Premise: `A is Z`
- Premise: `B is Z`

**Example:**
> "Cats and dogs are mammals"

→ Premise 1: `Cats are mammals`
→ Premise 2: `Dogs are mammals`

## 4. The Derivation Test

The derivation test determines whether a claim is a premise or a conclusion.

### Procedure

```
Given a claim C:

1. Ask: "Can I derive C from more basic claims?"

2. If YES:
   → C is a CONCLUSION.
   → Identify the premises it derives from.
   → Apply the derivation test recursively to each of those premises.

3. If NO:
   → C is a PREMISE.
   → Verify it directly against a specific retrievable observable source.
```

### Recursion

The derivation test is recursive. Keep decomposing until you reach claims that can ONLY be verified by direct observation or direct reference to definitions. These terminal claims are your premises.

### Termination Criteria

Stop decomposing when:
- The claim is verifiable by direct observation (empirical measurement, sensory data)
- The claim is verifiable by direct reference (looking up a definition, consulting a standard)
- Further decomposition would manufacture philosophical claims about existence rather than identify genuinely independent verifiable facts (see Section 6)

## 5. Worked Examples

### Example A: Direct Observable Fact

> "Water boils at 100°C at standard atmospheric pressure"

**Derivation test:** Can this be derived from simpler claims?

No. This is a directly observable empirical fact. You measure it.

**Result:** This IS a premise.
- Status: `established`
- Evidence type: direct measurement
- Epistemic basis: reproducible experimental observation

---

### Example B: Fact + Explanation

> "Water boils at 100°C because of the hydrogen bonding between water molecules"

**Derivation test:** This contains `because` — it joins a fact with a causal explanation.

**Decomposition:**

| ID | Type | Statement |
|----|------|-----------|
| P1 | Premise | Water boils at 100°C at standard atmospheric pressure |
| P2 | Premise | Water molecules form hydrogen bonds |
| P3 | Premise | Hydrogen bond strength determines boiling point in polar liquids |
| C1 | Conclusion | Water boils at 100°C because of hydrogen bonding |

**Structure:** C1 is derived from P1 + P2 + P3. Each premise is independently verifiable:
- P1: direct measurement
- P2: spectroscopic observation, computational chemistry
- P3: comparative study of polar liquids with varying hydrogen bond strengths

---

### Example C: Conjunction of Independent Facts

> "The Earth orbits the Sun and the Moon orbits the Earth"

**Derivation test:** Contains `and` joining two independent observable facts.

**Decomposition:**

| ID | Type | Statement |
|----|------|-----------|
| P1 | Premise | The Earth orbits the Sun |
| P2 | Premise | The Moon orbits the Earth |

Each is independently verifiable through astronomical observation.

---

### Example D: Arithmetic as Derivation

> "2 + 2 = 4"

**Derivation test:** Can this be derived from more basic claims?

Yes. It requires performing the addition operation on the premises about numbers.

**Decomposition:**

| ID | Type | Statement |
|----|------|-----------|
| P1 | Premise | 2 is a natural number |
| P2 | Premise | The successor of 1 is 2 |
| P3 | Premise | The successor of 3 is 4 |
| P4 | Premise | Addition of n + m means applying the successor operation m times starting from n |
| C1 | Conclusion | 2 + 2 = 4 |

**Why this matters:** Each premise is verifiable by reference to the Peano axioms (direct definitional reference). The conclusion requires *performing the operation* — applying successor twice starting from 2. That application is the derivation. The result is not observable; it is computed.

---

### Example E: Implicit Derivation

> "Aspirin reduces inflammation and therefore helps with arthritis pain"

**Decomposition:**

| ID | Type | Statement |
|----|------|-----------|
| P1 | Premise | Aspirin reduces inflammation |
| P2 | Premise | Arthritis pain is caused by inflammation |
| C1 | Conclusion | Aspirin helps with arthritis pain |

Note: The original statement only explicitly mentions P1, but the derivation of C1 requires P2 as an additional premise. Decomposition exposes the hidden assumptions.

## 6. When NOT to Decompose

### Already Atomic Claims

These are already premises. Do not decompose further:

- `The human diploid genome contains 46 chromosomes` — directly verifiable by karyotype observation.
- `A is a letter in the English alphabet` — directly verifiable by reference to the alphabet.
- `The speed of light in a vacuum is 299,792,458 metres per second` — directly verifiable by measurement.
- `Helium has atomic number 2` — directly verifiable by reference to the periodic table.

### The Over-Decomposition Anti-Pattern

**Do NOT decompose into existence claims or manufactured philosophical sub-claims.**

Bad decomposition of "A is a letter in the English alphabet":
- ~~"A exists"~~
- ~~"The English alphabet exists"~~
- ~~"Letters are elements of alphabets"~~
- ~~"A is an element of the English alphabet"~~

This is manufacturing unnecessary philosophy. The English alphabet is an observable empirical reality. The premise `A is a letter in the English alphabet` is verifiable as stated — you look at the alphabet, you see A in it. Done.

### The Rule

> If further decomposition produces sub-claims that are LESS directly verifiable than the original, you have gone too far. The original is your premise.

Decomposition must move toward more direct verifiability, not away from it.

## 7. Decomposition for Evidence Localization

### Purpose

A compound candidate can hide several independently evaluable factual assertions
and an unstated inference. Decomposition exposes those parts so evidence can be
mapped to each one.

### Method

```text
Given a compound candidate C:

1. Decompose C into atomic premises and explicit derived assertions.
2. Ground the significant terms in each premise.
3. Expose every inference required to derive a further assertion.
4. Map supporting and challenging evidence to each atomic premise.
5. Evaluate each premise independently.
6. Materialize a derived premise only when its stated method evaluates true.
```

The first atomic premise or inference not established by the linked evidence is the
point requiring additional evidence or refinement. Participant agreement is not an
input.

### Example

Consider the candidate “Vaccines cause autism.” Decomposition exposes the distinct
claims and the causal inference:

| ID | Type | Statement | Evaluation |
|----|------|-----------|------------|
| P1 | Premise | Vaccines are administered to children | Evaluate independently |
| P2 | Premise | Autism spectrum disorder is diagnosed in children | Evaluate independently |
| P3 | Premise | Autism diagnoses have increased over time | Evaluate independently |
| P4 | Premise | Vaccine administration has increased over time | Evaluate independently |
| P5 | Premise | Correlation between two time series does not establish causation | Evaluate independently |
| P6 | Premise | Controlled studies show no difference in autism rates between vaccinated and unvaccinated populations | Evaluate against cited studies |
| C1 | Candidate derivation | Vaccines cause autism | Does not follow from P1–P4 and is challenged by P5–P6 |

The system records the atomic premises, evidence relations, and explicit evaluation.
It does not model social positions around the candidate.

### Principle

> Decompose until each factual assertion and inference can be evaluated directly
> against evidence.
