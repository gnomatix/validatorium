# Curation (validatory)

**Status:** Draft from the project owner's direction of 2026-11-16 (worklog
prompts P-20261116-005-sess and P-20261116-006-sess); wording pending the
owner's review.

## 1. Purpose

Curation keeps the data valid for the nature of what it is. It applies to every
kind of record in Validatorium. It does not decide truth: validity still comes
only from evaluation against evidence (`data-model.md` §1). Curation adds
annotations and applies rules that act on them.

## 2. The graph

Validatorium's records form a graph of typed things: premises, evidence,
terms, attestations, and the relations between them. Because terms are
grounded to proper ontologies, every node has a type, and curation is chosen
by type.

## 3. Layers of curation

1. **Submitter curation.** The person who submits a premise provides its
   starting curation: their wording, their term groundings, their evidence.
2. **Validatory.** «validatory» means the namespaces of Validatorium
   curation: an internal layer in which curation workers add annotations
   within their own namespaces. Validatory annotations need not be shown to
   users.

## 4. Annotations

An annotation is an authored record attached to a node of the graph, kept in
a validatory namespace, and attributed to whoever or whatever produced it
(`data-model.md` §12: annotations retain their authorship). Like every
record, annotations are versioned and obsoleted, never deleted.

## 5. Classifiers

A classifier is any runnable tool that takes records of a type as input and
returns a result: a local model such as FunctionGemma, or a classical
natural-language-processing tool. The graph's records are reformatted into
the input format each tool needs; that the records can serve as its input is
what shows a classifier applies to that type. Each result is stored as an
annotation, and so becomes available for curation.

## 6. Curation workers and rules

A curation worker applies a set of rules to the annotations on records of
the types it covers. Its rules are established from the literature for that
domain (for example, philosophical rigor tests such as argumentation schemes
and their critical questions), not invented. Applying rules adds further
annotations, and annotations determine what is done next, according to the
rules: for example, flagging a premise for review or obsoleting a record,
with the agent and reason recorded.

## 7. Development process

1. **Inventory types.** List the types of data held.
2. **Coverage check.** Whenever a new type appears, ask: is there a curation
   worker for it?
3. **Research.** A research worker surveys the literature and existing tools
   for that type and proposes curation workers, classifiers and rule sets to
   the owner.
4. **Owner decision.** The owner accepts or rejects each proposal before it
   is built.
5. **Iterate on real data.** The Wikipedia ingestion exercise is the first
   iteration: its articles and their citations also point to the methods and
   literature curation draws on.
