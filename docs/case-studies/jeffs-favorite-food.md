# Case study: Jeff's favorite food

How one casual sentence decomposes into atomic premises. "Jeff" is a
hypothetical speaker.

## The statement

> "My name's Jeff, and I like, my favorite food is Pop-Tarts, strawberry
> Pop-Tarts, but only the kind that they used to make."

It reads as one statement. It contains at least nine.

## Explicit premises

Each phrase of the sentence asserts something of its own.

| # | Phrase | What it asserts | Kind of claim |
|---|---|---|---|
| 1 | "My name's Jeff" | The speaker's name is Jeff. | About the speaker |
| 2 | "my favorite food is Pop-Tarts" | Of all foods, Jeff prefers Pop-Tarts. "Favorite" is a comparison against every other food. | Self-reported preference |
| 3 | "strawberry Pop-Tarts" | Among Pop-Tarts, Jeff prefers the strawberry flavor. | Self-reported preference; refines 2 |
| 4 | "the kind that they used to make" | An earlier version of strawberry Pop-Tarts existed and differs from the current one. | About the product's history; checkable against the manufacturer's records |
| 5 | "but only" | Jeff prefers the earlier version to the current one. | Self-reported preference; refines 3 |

## Implicit premises

The sentence also depends on claims it never states.

| # | Depends on | What is assumed |
|---|---|---|
| 6 | 5 | Jeff has eaten both the earlier and the current version; otherwise he cannot prefer one to the other. |
| 7 | 4 | "They" is a specific manufacturer, never named. |
| 8 | 4 | "Used to" refers to a period of time that is never dated. |
| 9 | 2, 3, 5 | "Is" means now: the preference holds as of when Jeff said it. Favorites change. |

## In Argdown

Following `docs/specs/argdown-conventions.md`. Groundings are left as
`TBD` where no identifier has been checked; none is guessed.

```argdown
---
domain: personal_preference
terms:
  Pop-Tarts:
    uri: TBD
    label: "Pop-Tarts (manufactured toaster pastry product)"
  strawberry_flavor:
    uri: TBD
    label: "strawberry flavor"
---

[Speaker Name]: The speaker's name is Jeff. {
  evidence_type: "self_report"
}

[Jeff Favorite Food]: Jeff's favorite food is Pop-Tarts. {
  evidence_type: "self_report",
  scope: {as_of: "date of statement"}
}

[Jeff Favorite Flavor]: Jeff's favorite Pop-Tarts flavor is strawberry. #refinement {
  evidence_type: "self_report",
  scope: {as_of: "date of statement"}
}
  + [Jeff Favorite Food]

[Earlier Strawberry Formulation]: Strawberry Pop-Tarts were made to a formulation, during a dated period, that differs from the current formulation. {
  evidence_type: "document_inspection",
  scope: {manufacturer: "TBD", period: "TBD"}
}

[Jeff Prefers Earlier Version]: Jeff prefers the earlier strawberry Pop-Tarts formulation to the current one. #refinement {
  evidence_type: "self_report",
  scope: {as_of: "date of statement"}
}
  + [Jeff Favorite Flavor]
  + [Earlier Strawberry Formulation]
  + [Jeff Ate Both Versions]

[Jeff Ate Both Versions]: Jeff has eaten both the earlier and the current strawberry Pop-Tarts formulation. {
  evidence_type: "self_report"
}
```

## What it shows

- **Even a casual sentence carries many premises.** Writing it into the
  record format brings each into the open, including those the speaker never
  said out loud.
- **Different premises have different kinds of evidence.** Premises 1, 2, 3,
  5 and 6 rest on Jeff's own report, and the record says so. Premise 4 is
  about a product's history and is checkable against documents, independently
  of Jeff.
- **Scope has to be stated.** "Is" (now), "used to" (when), and "they" (who)
  are all scope the sentence leaves out.
- **No one needs to approve it.** There is no notability test. Jeff states
  it; the record shows exactly what is claimed and what supports each part.
