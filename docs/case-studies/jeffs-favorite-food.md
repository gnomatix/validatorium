# Case study: Jeff's favorite food

How one casual sentence decomposes into atomic premises. "Jeff" is a
hypothetical speaker.

## The statement

> "My name's Jeff, and I like, my favorite food is Pop-Tarts, strawberry
> Pop-Tarts, but only the kind that they used to make."

It reads as one statement. It contains at least nine premises, and they share
one root: **there is a person named Jeff.** Every other premise is about that
person and is established by that same person. They are *self-referential
premises*.

## Explicit premises

Each phrase of the sentence asserts something of its own.

| # | Phrase | What it asserts | Kind of premise |
|---|---|---|---|
| 1 | "My name's Jeff" | There is a person named Jeff, and he is the speaker. | Root: existence of the person; self-referential |
| 2 | "my favorite food is Pop-Tarts" | Of all foods, Jeff prefers Pop-Tarts. "Favorite" is a comparison against every other food. | Self-referential preference |
| 3 | "strawberry Pop-Tarts" | Among Pop-Tarts, Jeff prefers the strawberry flavor. | Self-referential preference; refines 2 |
| 4 | "the kind that they used to make" | An earlier version of strawberry Pop-Tarts existed and differs from the current one. | About the product's history; not self-referential; checkable against the manufacturer's records |
| 5 | "but only" | Jeff prefers the earlier version to the current one. | Self-referential preference; refines 3 |

## Implicit premises

The sentence also depends on claims it never states.

| # | Depends on | What it depends on without saying so |
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

[Person Named Jeff]: There is a person named Jeff, who is the author of these records. {
  evidence_type: "self_report"
}

[Jeff Favorite Food]: Jeff's favorite food is Pop-Tarts. {
  evidence_type: "self_report",
  scope: {as_of: "date of statement"}
}
  + [Person Named Jeff]

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

## Who is Jeff?

The first question the example raises. Jeff matters in two separate roles.

**Jeff as author.** Identity is attribution only and pseudonymous (Beads
decision `validatorium-ilv`). What is needed from Jeff the author is a GPG key
that signs his records, so each action traces to the same author; nothing
else. His legal name, credentials, reputation and notability are not required
and carry no evidential weight. The key exists so that, if it is ever found to
have corrupted data, every record it signed can be listed and re-examined.

**Jeff as subject.** "Jeff's favorite food is Pop-Tarts" is about Jeff, so the
record must say which Jeff. The simplest reference is the author's own key:
these are premises about the person who signed them. All of Jeff's premises
about himself then link together without his proving who he is.

**What is not done.** Jeff is not refused. He needs no permission, credentials
or notability: only a key and well-formed premises.

**What is protected.** These are personal facts about a private person.
Observations are local by default and private data is not replicated
implicitly (`docs/specs/distributed-datastore.md` §8). Jeff's premises about
himself stay on his own node unless he publishes them.

**"My name's Jeff" itself** is recorded like his other self-reports. If Jeff
wants it checkable, he can link evidence such as an identity document. Nothing
requires it; the system works the same whether or not "Jeff" is his legal
name.

## What it shows

- **Even a casual sentence carries many premises.** Writing it into the
  record format brings each into the open, including those the speaker never
  said out loud.
- **Self-referential premises share a recognisable shape.** For premises 1,
  2, 3, 5 and 6 the author, the subject and the source of the evidence are the
  same key, and all depend on the root "there is a person named Jeff". A node
  syncing these records can detect that shape from the records alone. A
  signature proves a key holder exists; that the key holder is a person named
  Jeff rests on his own statement.
- **Different premises have different kinds of evidence.** Premises 1, 2, 3,
  5 and 6 rest on Jeff's own report, and the record says so. Premise 4 is
  about a product's history and is checkable against documents, independently
  of Jeff.
- **Scope has to be stated.** "Is" (now), "used to" (when), and "they" (who)
  are all scope the sentence leaves out.
- **No one needs to approve it.** There is no notability test. Jeff states
  it; the record shows exactly what is claimed and what supports each part.
