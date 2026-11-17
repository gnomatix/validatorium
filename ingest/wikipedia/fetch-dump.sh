#!/bin/sh
# Fetch a Wikipedia multistream dump and its index, verify both against the
# SHA-1 sums Wikimedia publishes, and write a manifest (dump date, sizes,
# SHA-1, SHA-256). Files already present (for example from a torrent) are
# resumed or left as they are, then verified the same way.
#
#   ingest/wikipedia/fetch-dump.sh [WIKI] [DATE] [DIR]
#     WIKI  default enwiki
#     DATE  YYYYMMDD of the dump run, or "latest" (default)
#     DIR   default ./dump-WIKI-DATE
#   VERIFY_ONLY=1  do not download; only verify files already in DIR
#
# Exit status is non-zero if any checksum does not match.
set -eu
wiki=${1:-enwiki}
date=${2:-latest}
base=https://dumps.wikimedia.org/$wiki
sums_url=$base/$date/$wiki-$date-sha1sums.txt
sums=$(curl -sS --fail -m 60 "$sums_url") || { echo "fetch-dump: cannot read $sums_url" >&2; exit 2; }
# Resolve the dated run from the names inside the sums file.
run=$(printf '%s\n' "$sums" | sed -n "s/.*  $wiki-\([0-9]\{8\}\)-pages-articles-multistream\.xml\.bz2\$/\1/p" | head -1)
[ -n "$run" ] || { echo "fetch-dump: no multistream dump listed in $sums_url" >&2; exit 2; }
dir=${3:-dump-$wiki-$run}
mkdir -p "$dir"
printf '%s\n' "$sums" > "$dir/$wiki-$run-sha1sums.txt"
status=0
entries=""
for f in "$wiki-$run-pages-articles-multistream.xml.bz2" "$wiki-$run-pages-articles-multistream-index.txt.bz2"; do
	want=$(printf '%s\n' "$sums" | awk -v f="$f" '$2==f {print $1}')
	[ -n "$want" ] || { echo "fetch-dump: $f not in the sums file" >&2; exit 2; }
	if [ "${VERIFY_ONLY:-0}" != 1 ]; then
		curl -sS --fail -C - -o "$dir/$f" "$base/$run/$f" || { echo "fetch-dump: download failed: $f" >&2; exit 3; }
	fi
	[ -f "$dir/$f" ] || { echo "fetch-dump: missing $dir/$f" >&2; exit 3; }
	got=$(sha1sum "$dir/$f" | cut -d' ' -f1)
	sha256=$(sha256sum "$dir/$f" | cut -d' ' -f1)
	bytes=$(wc -c < "$dir/$f" | tr -d ' ')
	if [ "$got" = "$want" ]; then ok=true; echo "ok        $f"; else ok=false; status=1; echo "MISMATCH  $f (sha1 $got, published $want)" >&2; fi
	entries="$entries$(jq -n -c --arg f "$f" --arg u "$base/$run/$f" --arg s1 "$got" --arg p "$want" --arg s256 "$sha256" --argjson b "$bytes" --argjson ok "$ok" \
		'{file:$f, url:$u, bytes:$b, sha1:$s1, published_sha1:$p, sha256:$s256, verified:$ok}')
"
done
printf '%s' "$entries" | jq -s --arg w "$wiki" --arg r "$run" --arg s "$sums_url" --arg t "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
	'{wiki:$w, dump_date:$r, sums_url:$s, checked_at:$t, files:.}' > "$dir/manifest.json"
echo "manifest: $dir/manifest.json"
exit $status
