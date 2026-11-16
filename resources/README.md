# Resources

Source material collected for Validatorium work. Nothing in this folder is
part of the specification (`docs/specs/`); it is reference material.

## propaganda-analysis/

Publications of the Institute for Propaganda Analysis (New York, 1937–1942),
downloaded 2026-11-16 from the Internet Archive. Each item there carries the
Creative Commons Public Domain Mark 1.0
(<https://creativecommons.org/publicdomain/mark/1.0/>). Every file was checked
against the SHA-1 the Internet Archive publishes for it; sizes, SHA-1 and
SHA-256 values are in `propaganda-analysis/MANIFEST.tsv`.

| Folder (Internet Archive identifier) | Title | Date |
|---|---|---|
| `LeeFineArt` | Alfred McClung Lee and Elizabeth Briant Lee (eds.), *The Fine Art of Propaganda: A Study of Father Coughlin's Speeches* | 1939 |
| `IPAVol1` | *Propaganda Analysis*, Volume I of the Publications of the Institute for Propaganda Analysis | 1938 |
| `IPAVol2` | *Propaganda Analysis*, Volume II | 1939 |
| `IPAVol3` | *Propaganda Analysis*, Volume III | 1940 |
| `IPAVol4` | *Propaganda Analysis*, Volume IV | 1942 |
| `ipa_paganda_howtorecognizeitanddealwithit` | *Propaganda, How to Recognize It and Deal With It* | 1938 |
| `ipa_whatispropaganda` | *Propaganda, How to Recognize It and Deal With It* (separate scan) | 1938 |

Source pages: `https://archive.org/details/<identifier>`.

Each folder holds the Internet Archive's OCR text (`*_djvu.txt`), which is
stored in git. The scanned PDFs (up to 131 MB each) are not stored in git:
`propaganda-analysis/MANIFEST.tsv` lists each PDF's Internet Archive URL,
size, SHA-1 and SHA-256, and `propaganda-analysis/fetch.sh` downloads them
into place and verifies them. The one small PDF (0.4 MB) is stored in git.
The OCR text is machine output and contains recognition errors; quote from
the PDF.

## references.md

Bibliographic references (not full texts) to research on network centrality
and essentiality, citation distortion, and the cognitive effects by which a
claim seems true through proximity or familiarity. See `AGENTS.md` for why
these materials are relevant.
