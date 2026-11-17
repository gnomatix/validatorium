#!/bin/sh
# Worker command for a local model served over the OpenAI-compatible chat
# endpoint POST /v1/chat/completions. Documented by llama.cpp (tools/server/README.md)
# and by Ollama (docs/api/openai-compatibility); both read 2026-11-17.
#
#   WORKER_CMD="ingest/wikipedia/worker-local.sh" (see sge-array.sh)
#   usage: worker-local.sh INSTRUCTIONS_FILE < one-article.json > premises.jsonl
#
# Environment:
#   WORKER_URL    default http://127.0.0.1:8080/v1/chat/completions
#   WORKER_MODEL  model name the server expects (default "local")
#   WORKER_TIMEOUT seconds per article (default 600)
#
# Prints only lines that are JSON objects; anything else the model wrote is
# dropped and counted on standard error. Exits non-zero when the server
# fails or no valid line comes back.
set -eu
prompt=${1:?usage: worker-local.sh INSTRUCTIONS_FILE}
url=${WORKER_URL:-http://127.0.0.1:8080/v1/chat/completions}
model=${WORKER_MODEL:-local}
article=$(cat)
[ -n "$article" ] || { echo "worker-local: empty input" >&2; exit 2; }
# Instructions start after the first line that is exactly "---".
sys=$(awk 'f{print} /^---$/{f=1}' "$prompt")
[ -n "$sys" ] || sys=$(cat "$prompt")
req=$(jq -n --arg m "$model" --arg s "$sys" --arg a "$article" \
	'{model:$m, temperature:0, stream:false, messages:[{role:"system",content:$s},{role:"user",content:$a}]}')
resp=$(printf '%s' "$req" | curl -sS --fail-with-body -m "${WORKER_TIMEOUT:-600}" \
	-H 'Content-Type: application/json' --data-binary @- "$url") || {
	echo "worker-local: request failed: $resp" >&2; exit 3; }
out=$(printf '%s' "$resp" | jq -r '.choices[0].message.content // empty')
good=$(printf '%s\n' "$out" | jq -R -c 'fromjson? | select(type=="object")')
total=$(printf '%s\n' "$out" | grep -c . || true)
kept=$(printf '%s\n' "$good" | grep -c . || true)
[ "$kept" -gt 0 ] || { echo "worker-local: no valid JSON line in the reply ($total lines)" >&2; exit 4; }
[ "$kept" -eq "$total" ] || echo "worker-local: dropped $((total-kept)) of $total lines that were not JSON objects" >&2
printf '%s\n' "$good"
