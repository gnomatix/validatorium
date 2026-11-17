# Wikipedia ingestion run: work plan

A data feed, not specification. Assumes the owner's machine is up: Grid
Engine, about 24 threads, 96 GB RAM, ZFS with compression, a local model
runtime, and a 1 Gbit connection. Pipeline details: `ingest/wikipedia/README.md`.

## Numbers to plan against

Each figure names where it came from.

| Quantity | Value | Source |
|---|---|---|
| `enwiki-latest-pages-articles-multistream.xml.bz2` | 27,019,419,473 bytes (27.0 GB), dated 2026-11-07 | <https://dumps.wikimedia.org/enwiki/latest/>, read 2026-11-17 |
| Size as XML after decompression | not verified | measure in stage 1 |
| Article count (main namespace, non-redirect) | not verified | stage 2 prints it (`wikiextract: wrote N articles`) |
| Download of 27.0 GB at 50 MB/s | about 9 minutes | arithmetic |
| Extractor rate, one thread | about 1,440 articles per second; about 2.6 MB of compressed input per second | measured 2026-11-17 on simplewiki-20261101 in a cloud container (`validatorium-x3g.14`) |
| Extraction of 27.0 GB at that rate | about 2.9 hours on one thread | arithmetic from the row above; the machine differs |
| Extractor with `-templates` (template expansion) | about 120 articles/s with 4 workers, about 30 times slower; leave it off for the run | measured 2026-11-17 on simplewiki, partial run (`validatorium-x3g.8`) |
| Extractor with `-index`, 4 workers | 58 s for the same dump (150 s serial); output byte-identical | measured 2026-11-17, same container (`validatorium-x3g.11`) |
| Full decomposition in 2 hours needs | N articles / 7,200 seconds | N from stage 2 |

The decomposition rate of local models on this machine is unknown until
stage 4 measures it. The 2-hour full run is decided by that number.

## End state of this run

Candidate premises in a bbolt working store, each citing its article
revision and quote. Candidates are drafts. Nothing becomes valid or is
published until an evaluator for these premises exists
(`validatorium-x3g.3`).

## Stages

| # | Stage | Beads | Command or action | Done when |
|---|---|---|---|---|
| 0 | Build | none (no code change) | `go build -mod=vendor -o val ./cmd/val && go build -mod=vendor -o wikiextract ./cmd/wikiextract` | both binaries run `-h` |
| 1 | Get the dump and verify it | `validatorium-x3g.12` | `ingest/wikipedia/fetch-dump.sh enwiki latest DIR` downloads dump and index and verifies them; after a torrent download, put the files in DIR and run it with `VERIFY_ONLY=1` | exit status 0; `DIR/manifest.json` records dump date, sizes, SHA-1 and SHA-256 |
| 2 | Extract | `validatorium-x3g.14` | `./wikiextract -in DUMP -max 500 > sample.jsonl`; read 20 by eye; then `./wikiextract -in DUMP -index INDEX -workers 24 > articles.jsonl` | counts and wall time recorded; defects filed |
| 2a | Title list | `validatorium-x3g.1` | list of chemical-element article titles from Wikidata; `./wikiextract -in DUMP -titles elements.txt > elements.jsonl` | about 118 articles extracted |
| 3 | Worker command | `validatorium-x3g.15` | start a local model server; `export WORKER_CMD=ingest/wikipedia/worker-local.sh WORKER_URL=... WORKER_MODEL=...`; set `WORKER_MAX_CHARS` to fit the model context | one article in, valid JSON Lines out |
| 4 | Sample decomposition | `validatorium-x3g.16` | `split -l 10 -d -a 5 elements.jsonl chunks/articles.`; `qsub -t 1-$N -tc 8 -cwd -o logs -e logs ingest/wikipedia/sge-array.sh` | all chunks have `out/*.jsonl`; articles per hour recorded per model |
| 5 | Load and measure | `validatorium-x3g.16` | `cat out/*.jsonl \| ./val import -working-store wiki.db -file - -agent "worker:<model>@<date>" -articles elements.jsonl` | created / present / skipped / rejected counts recorded; 50 premises read by eye |
| 6 | Second model, same articles | `validatorium-x3g.13` | repeat 4-5 with a different `-agent` and model | agreement between the two workers measured |
| 7 | Go / no-go for a larger run | `validatorium-x3g.16` | compare the measured rate with the table above | decision recorded with the numbers |
| 8 | Evaluator for these premises | `validatorium-x3g.3` | implement per the task; needs the owner's approval of agent and model | a premise can receive an attestation |

## Without Grid Engine

The same task script runs without a scheduler, one task per chunk:

```sh
N=$(ls chunks/articles.* | wc -l)
seq 1 $N | xargs -P 8 -I{} env SGE_TASK_ID={} ingest/wikipedia/sge-array.sh
```

Checked 2026-11-17 in a cloud container against a mock model server: 13 chunks
of the element articles (130 articles), 148 output lines, all imported
(created 148, rejected 0). The mock returns one premise per request; a real
model has not been run.

## Known limits going in

- `wikiextract` is single-threaded and strips markup with regular
  expressions; expect defects in stage 2 (`validatorium-x3g.8`,
  `validatorium-x3g.9` are the planned replacements).
- The worker instructions have never been given to a model. Open decision
  on its output fields: `validatorium-x3g.17`.
- Query is a scan, not an index (`validatorium-x3g.5`).

## Checks that already run in the pipeline

- `val import -articles` rejects any premise whose `quote` is not verbatim
  in the source article, and counts the rejects.
- Import is idempotent: the premise UUID derives from agent, wiki, page id,
  revision id and local id.
- Different `-agent` values keep separate records for cross-checking.
