# Wikipedia ingestion run: work plan

A data feed, not specification. Assumes the owner's machine is up: Grid
Engine, about 24 threads, 96 GB RAM, ZFS with compression, a local model
runtime, and a 1 Gbit connection. Pipeline details: `ingest/wikipedia/README.md`.

## Numbers to plan against

Sizes are from recall; confirm at <https://dumps.wikimedia.org/enwiki/latest/>.

| Quantity | Value |
|---|---|
| `enwiki-latest-pages-articles-multistream.xml.bz2` | about 25 GB compressed, about 110 GB as XML |
| Articles (main namespace, non-redirect) | about 7 million |
| Download at 50 MB/s | 25 GB: about 9 minutes. 118 GB: about 40 minutes |
| Full decomposition in 2 hours needs | about 970 articles per second |

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
| 1 | Get the dump and verify it | `validatorium-x3g.12` | download dump and index (torrent or HTTPS); compare with the published `sha1sums`/`md5sums` file | checksum matches; dump date recorded in the task |
| 2 | Extract | `validatorium-x3g.14` | `./wikiextract -in DUMP -max 500 > sample.jsonl`; read 20 by eye; then `./wikiextract -in DUMP > articles.jsonl` | counts and wall time recorded; defects filed |
| 2a | Title list | `validatorium-x3g.1` | list of chemical-element article titles from Wikidata; `./wikiextract -in DUMP -titles elements.txt > elements.jsonl` | about 118 articles extracted |
| 3 | Worker command | `validatorium-x3g.15` | wrapper around the local model runtime; set `WORKER_CMD` | one article in, valid JSON Lines out |
| 4 | Sample decomposition | `validatorium-x3g.16` | `split -l 10 -d -a 5 elements.jsonl chunks/articles.`; `qsub -t 1-$N -tc 8 -cwd -o logs -e logs ingest/wikipedia/sge-array.sh` | all chunks have `out/*.jsonl`; articles per hour recorded per model |
| 5 | Load and measure | `validatorium-x3g.16` | `cat out/*.jsonl \| ./val import -working-store wiki.db -file - -agent "worker:<model>@<date>" -articles elements.jsonl` | created / present / skipped / rejected counts recorded; 50 premises read by eye |
| 6 | Second model, same articles | `validatorium-x3g.13` | repeat 4-5 with a different `-agent` and model | agreement between the two workers measured |
| 7 | Go / no-go for a larger run | `validatorium-x3g.16` | compare the measured rate with the table above | decision recorded with the numbers |
| 8 | Evaluator for these premises | `validatorium-x3g.3` | implement per the task; needs the owner's approval of agent and model | a premise can receive an attestation |

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
