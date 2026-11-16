#!/bin/sh
# Grid Engine array job: decompose one chunk of articles per task.
#
#   N=$(ls chunks/articles.* | wc -l)
#   qsub -t 1-$N -tc 8 -cwd -o logs -e logs ingest/wikipedia/sge-array.sh
#
# Set WORKER_CMD to the command that runs your model: it must read the
# instructions file named by $1 and one article JSON object on standard
# input, and write the worker's JSONL to standard output. It is a placeholder
# here because the model runtime (local or hosted) is your choice.
#$ -S /bin/sh
set -eu
: "${WORKER_CMD:?set WORKER_CMD to the model command}"
here=$(cd "$(dirname "$0")" && pwd)
prompt="$here/WORKER_PROMPT.md"
chunk=$(ls chunks/articles.* | sed -n "${SGE_TASK_ID}p")
mkdir -p out
dest="out/$(basename "$chunk").jsonl"
[ -s "$dest" ] && exit 0 # this chunk is already done
tmp="$dest.part"
: > "$tmp"
while IFS= read -r article; do
	printf '%s\n' "$article" | $WORKER_CMD "$prompt" >> "$tmp" || echo "task $SGE_TASK_ID: worker failed on one article" >&2
done < "$chunk"
mv "$tmp" "$dest"
