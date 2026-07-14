# Project Principles

This document defines the intended Validatorium system. `docs/BUILD_STATE.md`
records what the current code implements.

## Purpose

Validatorium is a datastore of validated, evidence-backed facts about observable
reality, represented as atomic premises. It applies one method: observation,
evidence, and reproducible evaluation.

It is general-purpose. The same foundation can represent scientific observations,
historical records, legal documents, genealogy, temporal facts, local machine
state, or any other claim that can be checked against evidence.

It is designed for local and distributed use. A person can run a complete local
node; multiple nodes can replicate versioned premises and evidence without a
central authority.

## Boundaries

Validatorium is not a debate platform, social network, voting system, editorial
board, or repository of opinion. Truth is not determined by consensus,
credentials, popularity, authority, or attention. These properties have no effect
on whether a factual premise evaluates true.

Observable reality is the domain. Evidence-based verification—the scientific
method—is the validation method. The method is not a restriction to an academic or
professional field.

## Core Rules

1. **A valid premise evaluates true.** Structural well-formedness is necessary but
   is not semantic validity. Evidence and applicable context determine the result.
2. **Only valid premises enter the validated store.** Unsupported submissions are
   not premises in that store; there is no `unsupported` validity state.
3. **Premises are atomic.** Each premise contains one independently evaluable
   factual assertion with explicit scope.
4. **Terms are grounded and versioned.** Significant terms reference versioned term
   records from versioned ontologies. Ontologies define meaning, not truth.
5. **Evidence is inspectable.** Every validity result identifies the evidence,
   method, scope, evaluator, and time required to reproduce it.
6. **Authority has no evidential weight.** Attribution records who performed work;
   it never makes the resulting premise true.
7. **Support and challenge are evidence relations.** They describe what evidence
   does to a premise. They are not positions in a debate.
8. **UUIDs provide stable identity.** Stored versions are content-addressed for
   integrity. Superseded records remain addressable and point to replacement UUIDs.
9. **Releases are exact.** Every release references exact versions of premises,
   ontology terms, evidence, assets, and artifacts. A premise receives a release
   tag only when valid and current for that release.
10. **Validity is checked when needed.** Expiry, TTL, recurrence, dependency
    changes, new evidence, ontology remapping, and runtime use can trigger
    re-evaluation. Checks create immutable attestations rather than rewriting the
    premise.
11. **Temporal inactivity is not obsolescence.** A recurring premise may be current
    in each applicable interval and inactive between intervals. `obsolete` means
    superseded by replacement records.
12. **Derived premises remain accountable.** A derived or materialized premise
    identifies the premises, observations, method, and evaluation context from
    which it was produced.
13. **Facts are not owned.** Authored methods, work logs, annotations, and analyses
    retain authorship. External ontologies and standards retain their maintainers,
    licenses, versions, and citations.

## Higher-Level Systems

Validatorium is foundational infrastructure for systems that automatically
re-evaluate claims. Runtime adapters can check local files, services, compute
infrastructure, sensors, or other observable state and emit timestamped evidence.

Automated hypothesis-testing systems can be built above the same layer. An
orchestrator may execute an authorized experimental method through an instrument,
including a lab-on-a-chip, and return observations with complete provenance.
Validatorium records the premises, methods, observations, evidence relationships,
and validity attestations. Hardware control, safety constraints, and experimental
orchestration remain responsibilities of the higher-level system.

## Anti-Patterns

The following are incompatible with Validatorium:

- deciding factual validity through debate, consensus, voting, reputation, or
  credentials;
- treating structural schema checks as proof that a premise is true;
- storing unsupported submissions as validated premises;
- treating ontology maintainers or source authors as authorities on truth;
- overwriting historical records instead of creating versioned replacements;
- treating an old release tag as permanent validity;
- treating a recurring premise as obsolete while it is merely outside its active
  interval;
- hiding evidence, method, temporal scope, or provenance; and
- describing intended architecture as though it is already implemented.

## Summary

Validatorium stores atomic facts that evaluate true against evidence. It preserves
exact meaning, evidence, provenance, versions, and evaluation context so those facts
can be checked again by people, software, or instruments.
