# Anti-Patterns: LLM Behavior

This document records failure modes that software and agents working on
Validatorium must not reproduce.

## 1. Manufacturing Categories for Simple Facts

If a claim about observable reality can be checked against evidence, it is
empirical. Do not invent philosophical sub-categories, hedge a directly observable
fact, or narrow empirical verification to the natural sciences. Go check.

## 2. Claiming to Remember Without Persisting Anything

Words without action are not durable memory. If project knowledge must survive a
session, record it in Beads with `bd remember`. Do not claim something was
“permanently noted” unless it was actually stored.

## 3. Turning Facts into Debate

Do not frame factual validation as disagreement between participants, competing
positions, consensus, voting, moderation, or social adjudication. Examine the
evidence and evaluate the premise. Legitimate Micropublication support/challenge
and Argdown support/attack edges are evidence relationships, not debate roles.

## 4. Confusing Structure with Validity

A schema check can establish that a record is well-formed. It cannot establish that
the premise is true. Reserve `valid` for a premise that evaluates true against its
evidence and context; call shape, type, and grounding checks structural validation.

## 5. Treating Validity as Permanent

A release tag records validity for an exact release and its versioned assets. It
does not eliminate runtime checks. Respect expiry, TTL, recurrence, dependency,
evidence, and ontology-remapping triggers. Runtime re-validation creates a new
attestation; it does not rewrite history.

## 6. Treating Temporal Inactivity as Obsolescence

A recurring premise may be valid during multiple intervals and inactive between
them. `obsolete` means superseded by replacement UUIDs, not merely false outside a
temporal window.

## 7. Excluding Machine Observation from Evidence

A timestamped observation from a file check, service probe, sensor, or scientific
instrument can be empirical evidence when its method, scope, evaluator, and
provenance are retained. Instruments do not bypass validation requirements.

## 8. Claiming Intended Design Is Implemented

Specifications define the intended system. `docs/BUILD_STATE.md`, executable code,
and test results establish current implementation. Never present planned release
validation, runtime evaluation, networking, or instrument integration as completed
behavior.
