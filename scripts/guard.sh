#!/bin/sh
# Repository guard. Run before every commit and push:  scripts/guard.sh
# Rules: docs/handoff/RULES.md.
#   --owner-accept-specs   regenerate docs/specs.sha256 (project owner only)
set -eu
cd "$(git rev-parse --show-toplevel)"
manifest=docs/specs.sha256
sums() { find docs/specs -type f | LC_ALL=C sort | xargs sha256sum; }
if [ "${1:-}" = "--owner-accept-specs" ]; then
	sums > "$manifest"
	echo "specs manifest regenerated: $manifest"
	exit 0
fi
fail=0

# Rule 1: the specification is unchanged.
if ! sums | diff -u "$manifest" - >/dev/null; then
	echo "FAIL rule 1: docs/specs differs from $manifest (owner-only):" >&2
	sums | diff -u "$manifest" - >&2 || true
	fail=1
fi

# Rule 3: commits touching code carry a Beads task id. Checked for commits
# after the guard was introduced (GUARD_BASE overrides the start point).
base=${GUARD_BASE:-$(git log --format=%H --diff-filter=A -- scripts/guard.sh | tail -1)}
if [ -n "$base" ]; then
	for c in $(git rev-list "$base"..HEAD); do
		if git diff-tree --no-commit-id --name-only -r "$c" | grep -Eq '^(cmd|internal)/'; then
			if ! git log -1 --format=%B "$c" | grep -Eq 'validatorium-[a-z0-9]+'; then
				echo "FAIL rule 3: commit $(git log -1 --format='%h %s' "$c") changes code without a Beads task id" >&2
				fail=1
			fi
		fi
	done
fi

# Rule 7: no attribution trailers.
if [ -n "$base" ] && git log --format=%B "$base"..HEAD | grep -Eiq '^(co-authored-by|generated (by|with)|claude-session):'; then
	echo "FAIL rule 7: a commit message carries a tool or model attribution line" >&2
	fail=1
fi

# Quality gates.
go vet -mod=vendor ./... || fail=1
go test -mod=vendor -count=1 -short ./... || fail=1

[ "$fail" -eq 0 ] && echo "guard: ok" || { echo "guard: FAILED" >&2; exit 1; }
