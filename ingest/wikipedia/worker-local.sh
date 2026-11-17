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
#   WORKER_ESCALATE space-separated further model names, tried in order on the
#                 same endpoint when a model returns no valid premise line for
#                 a part (small model first, larger models after)
#   WORKER_TIMEOUT seconds per request (default 600)
#   WORKER_MAX_CHARS split the article text at paragraph breaks into parts of at
#                 most this many characters, one request per part (default 0:
#                 no split). A single paragraph longer than the limit is sent
#                 alone. Local ids are prefixed with the part number (s2-p1) so
#                 they stay unique within the article; depends_on is rewritten
#                 the same way.
#
# Prints only lines that are JSON objects; anything else the model wrote is
# dropped and counted on standard error. Exits non-zero when the server
# fails or no valid line comes back.
set -eu
prompt=${1:?usage: worker-local.sh INSTRUCTIONS_FILE}
url=${WORKER_URL:-http://127.0.0.1:8080/v1/chat/completions}
model=${WORKER_MODEL:-local}
first_model=$model
article=$(cat)
[ -n "$article" ] || { echo "worker-local: empty input" >&2; exit 2; }
# Instructions start after the first line that is exactly "---".
sys=$(awk 'f{print} /^---$/{f=1}' "$prompt")
[ -n "$sys" ] || sys=$(cat "$prompt")
ask() { # $1 = article JSON; prints the model reply text
	req=$(jq -n --arg m "$model" --arg s "$sys" --arg a "$1" \
		'{model:$m, temperature:0, stream:false, messages:[{role:"system",content:$s},{role:"user",content:$a}]}')
	resp=$(printf '%s' "$req" | curl -sS --fail-with-body -m "${WORKER_TIMEOUT:-600}" \
		-H 'Content-Type: application/json' --data-binary @- "$url") || {
		echo "worker-local: request failed: $resp" >&2; return 3; }
	printf '%s' "$resp" | jq -r '.choices[0].message.content // empty'
}
max=${WORKER_MAX_CHARS:-0}
parts=$(printf '%s' "$article" | jq -c --argjson max "$max" '
	. as $a
	| if $max <= 0 then [$a.text] else
		reduce ($a.text | split("\n\n")[]) as $p ([];
			if length > 0 and ((.[-1] | length) + 2 + ($p | length)) <= $max
			then .[-1] += "\n\n" + $p else . + [$p] end)
	  end
	| to_entries[] | .value as $t | {i: (.key + 1), article: ($a | .text = $t)}' 2>/dev/null) || true
total=$(printf '%s\n' "$parts" | grep -c . || true)
[ "$total" -gt 0 ] || { echo "worker-local: input is not an article object" >&2; exit 2; }
out=""
printf '%s\n' "$parts" > "${TMPDIR:-/tmp}/worker-local.$$"
while IFS= read -r part; do
	i=$(printf '%s' "$part" | jq -r .i)
	reply=""
	for m in $first_model ${WORKER_ESCALATE:-}; do
		model=$m
		reply=$(ask "$(printf '%s' "$part" | jq -c .article)") || { rm -f "${TMPDIR:-/tmp}/worker-local.$"; exit 3; }
		if printf '%s\n' "$reply" | jq -R -e 'fromjson? | select(type=="object")' >/dev/null 2>&1; then
			break
		fi
		echo "worker-local: part $i: no valid line from $m" >&2
	done
	if [ "$total" -gt 1 ]; then
		reply=$(printf '%s\n' "$reply" | jq -R -r --arg pre "s$i-" 'fromjson? // . | if type=="object" then (.local_id |= ($pre + tostring) | .depends_on |= ((. // []) | map($pre + tostring)) | .supported_by |= ((. // []) | map($pre + tostring))) | tojson else . end')
	fi
	out="$out$reply
"
done < "${TMPDIR:-/tmp}/worker-local.$$"
rm -f "${TMPDIR:-/tmp}/worker-local.$$"
good=$(printf '%s\n' "$out" | jq -R -c 'fromjson? | select(type=="object")')
total_lines=$(printf '%s\n' "$out" | grep -c . || true)
kept=$(printf '%s\n' "$good" | grep -c . || true)
[ "$kept" -gt 0 ] || { echo "worker-local: no valid JSON line in the reply ($total_lines lines)" >&2; exit 4; }
[ "$kept" -eq "$total_lines" ] || echo "worker-local: dropped $((total_lines-kept)) of $total_lines lines that were not JSON objects" >&2
printf '%s\n' "$good"
