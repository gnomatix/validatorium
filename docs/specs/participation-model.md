# Participation Model

How people and software contribute to Validatorium.

## 1. Who Can Participate

Anyone. Credentials, institutional affiliation, popularity, and authority have no
role in factual validity.

Interfaces should use plain language and require no formal-logic notation. Domain
knowledge may help someone locate evidence or formulate a precise premise, but it
does not grant special standing.

## 2. Activities

### Propose

Submit a candidate atomic premise with:

- one factual assertion;
- explicit scope and evaluation context;
- versioned ontology-term references;
- retrievable evidence; and
- a reproducible evaluation method.

A candidate is not a stored valid premise until evaluation returns true.

### Add Evidence

Link an observable, retrievable source as supporting or challenging evidence.
Record what part of the source is relevant, how it relates to the exact premise,
and the method used to examine it. Attribution records who performed the work; it
adds no evidential weight.

### Evaluate

Run the premise's method against its evidence and applicable context. Evaluation
produces an immutable, timestamped attestation with a result of `true`, `false`, or
`indeterminate`.

- `true`: the premise is valid for the attestation's scope and time.
- `false`: the premise is not valid for that scope and time.
- `indeterminate`: the available evidence or context cannot establish either
  result.

Only a premise that evaluates true enters the validated premise store or receives
a valid/current release tag.

### Clarify or Decompose

If a candidate is compound, ambiguous, or insufficiently scoped, create atomic,
unambiguous premises. Each resulting premise receives its own UUID, versioned term
references, evidence, method, and evaluation. Original records remain addressable
and may point to replacement UUIDs.

### Re-evaluate

A release, expiry, TTL, recurrence boundary, dependency change, ontology remapping,
new evidence item, or runtime use may trigger evaluation again. Re-evaluation adds
an attestation; it does not rewrite the premise or earlier results.

## 3. Evidence and Ambiguity Processing

Evidence processing is impersonal:

1. Resolve and authenticate the source.
2. Locate the material relevant to the exact premise.
3. Determine whether that material supports or challenges the premise.
4. Execute the stated evaluation method.
5. Record the result and provenance.

A source that fails resolution, authenticity, provenance, methodology, relevance,
or entailment checks is marked inapplicable with recorded reasons. A separate
contradicting source is not required to reject a defective evidence link.

Ambiguity processing is structural:

1. Identify the ambiguous term, scope, or compound assertion.
2. Ground terms to versioned ontology records.
3. Refine or decompose the candidate.
4. Evaluate every resulting atomic premise independently.

Agreement between participants is not an input to either process.

## 4. What Does Not Exist

- Debate, voting, polling, or consensus mechanisms
- Reputation-weighted truth
- Editorial authority over factual validity
- Participant agreement as an acceptance criterion
- An `unsupported` premise state in the validated store
- In-place edits that erase prior records

Argdown support/attack edges and Micropublication support/challenge relations are
retained because they represent evidence relationships, not social positions.

## 5. Adversary Resistance

| Attack | Defense |
|---|---|
| False candidate | It cannot enter the validated store unless evaluation returns true. |
| Fabricated source | Resolution, authenticity, provenance, and content checks fail. |
| Irrelevant source | Relevance to the exact scoped premise is evaluated explicitly. |
| Ambiguous wording | Atomicity, scope, and ontology-grounding checks require refinement. |
| Identity manipulation | Contributor identity has no evidential weight. |
| Historical rewriting | Immutable records and attestations preserve prior versions. |
| Stale validity | Release and runtime/JIT re-evaluation apply temporal and dependency rules. |

No mechanism can guarantee that every evaluator or evidence source is honest.
Validatorium therefore preserves methods, observations, provenance, versions, and
results so checks can be reproduced and challenged by additional evidence.

## 6. Accessibility

User-facing actions should remain direct:

| Action | Plain-language label |
|---|---|
| Propose | Propose a premise |
| Add evidence | Add evidence |
| Evaluate | Check this premise |
| Clarify | Refine this premise |
| Decompose | Split this premise |
| Re-evaluate | Check again now |

Premise relations are displayed visually. Ontology grounding remains
language-independent, while statements and interfaces may be localized.
