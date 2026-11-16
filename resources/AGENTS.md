# Resources: guide for agents

This folder holds reference material. It is not specification: the specs in
`docs/specs/` define Validatorium, and nothing here changes them. Read these
materials for background when working on how claims, links and evidence can
be manipulated, and on what the record model must make visible.

## Why these resources are here

Validatorium records premises, the evidence other people produced for them,
and the links between premises. Truth is decided by evidence and method only;
attribution, authority, popularity and consensus carry no evidential weight
(`docs/specs/project-principles.md`).

The ways a false claim is made to look true are therefore directly relevant.
A record system that links premises can be used to borrow credibility: a weak
claim is placed next to, or linked to, a well-supported one, and readers
accept it because of the proximity. Knowing the documented techniques tells
us what the records, links and signatures must expose so that such a move can
be seen and examined.

## propaganda-analysis/

Publications of the Institute for Propaganda Analysis (New York, 1937–1942):
the four bound volumes of the *Propaganda Analysis* bulletin, *The Fine Art of
Propaganda* (1939), and *Propaganda, How to Recognize It and Deal With It*
(1938). They are useful because they:

- define the classic propaganda devices with worked examples, including
  "transfer" (carrying the prestige of something respected over to something
  else) and "card stacking" (selective use of true facts to support a false
  impression), which are the record-level manipulations Validatorium must
  make detectable;
- analyse real campaigns of the period line by line, which gives concrete
  test cases for how a manipulative chain of claims is built;
- describe countermeasures aimed at the reader, the precursor of what is now
  called prebunking or inoculation.

All items are public domain (Creative Commons Public Domain Mark 1.0 on the
Internet Archive). Each folder has the scanned PDF (Git LFS; run
`git lfs pull` after cloning) and the OCR text (`*_djvu.txt`). If the PDFs
are missing, run `propaganda-analysis/fetch.sh`: it downloads them from the
Internet Archive and verifies each against the SHA-256 in
`propaganda-analysis/MANIFEST.tsv`. Search the OCR text, but quote
from the PDF: the OCR has recognition errors.

## references.md

Bibliographic references to research literature that bears on the same
questions: why some premises become cores that much else depends on (network
hubs and essentiality), how citation chains create unfounded authority, and
the cognitive effects by which a statement seems true through familiarity or
closeness to a true one. Full texts are not stored because they are under
copyright. Entries marked "unverified" must be checked before being cited.

## Rules for agents

- Do not treat anything here as a requirement. Proposed changes to specs or
  scope go to the project owner.
- Cite a resource by its folder and file, and quote the PDF, not the OCR text.
- Keep `MANIFEST.tsv` and the README table in step with the files when adding or
  removing material.
