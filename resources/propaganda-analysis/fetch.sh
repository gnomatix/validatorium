#!/bin/sh
# Download the Institute for Propaganda Analysis PDFs listed in MANIFEST.tsv
# from the Internet Archive and verify each against its recorded SHA-256.
# The PDFs are not stored in git because of their size; the OCR text files
# are. Run from any directory.
set -eu
here=$(cd "$(dirname "$0")" && pwd)
status=0
tail -n +2 "$here/MANIFEST.tsv" | while IFS="$(printf '\t')" read -r path size sha1 sha256 url; do
	out="$here/$path"
	if [ -f "$out" ] && [ "$(sha256sum "$out" | cut -d' ' -f1)" = "$sha256" ]; then
		echo "present  $path"
		continue
	fi
	mkdir -p "$(dirname "$out")"
	curl -fsSL --retry 3 -o "$out.part" "$url"
	got=$(sha256sum "$out.part" | cut -d' ' -f1)
	if [ "$got" = "$sha256" ]; then
		mv "$out.part" "$out"
		echo "fetched  $path"
	else
		rm -f "$out.part"
		echo "MISMATCH $path: expected $sha256, got $got" >&2
		status=1
	fi
done
exit $status
