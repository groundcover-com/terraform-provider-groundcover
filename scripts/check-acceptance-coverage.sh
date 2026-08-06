#!/usr/bin/env bash
# Assert that every TestAcc test in ./internal/provider is selected by one of the
# acceptance-test matrix groups in a workflow. A test that matches no group is never
# run by that workflow — silently, since a group whose regex matches nothing still
# reports success.
#
# Usage: scripts/check-acceptance-coverage.sh <workflow-file>

set -euo pipefail

workflow="${1:-}"
if [ -z "$workflow" ]; then
  echo "usage: $0 <workflow-file>" >&2
  exit 2
fi
if [ ! -f "$workflow" ]; then
  echo "workflow file not found: $workflow" >&2
  exit 2
fi

all_tests="$(mktemp)"
covered_tests="$(mktemp)"
regexes="$(mktemp)"
trap 'rm -f "$all_tests" "$covered_tests" "$regexes"' EXIT

go test ./internal/provider -list '^TestAcc' | awk '/^TestAcc/ { print }' | sort -u > "$all_tests"

# Only the acceptance-test-group job's regexes count. Other jobs may name TestAcc
# tests for unrelated reasons, and matching those would mask a real coverage gap. The
# block ends at the next job key rather than a specific job name, so renaming the
# following job cannot silently widen the range to the rest of the file.
awk -F"'" '
  /^  acceptance-test-group:/ { in_job = 1; next }
  in_job && /^  [A-Za-z0-9_-]+:/ { in_job = 0 }
  in_job && /^[[:space:]]*run_regex: / { print $2 }
' "$workflow" > "$regexes"

if [ ! -s "$regexes" ]; then
  echo "No acceptance test groups were found in $workflow."
  echo "Expected an 'acceptance-test-group:' job with 'run_regex:' matrix entries."
  exit 1
fi

while IFS= read -r regex; do
  go test ./internal/provider -list "$regex" | awk '/^TestAcc/ { print }'
done < "$regexes" | sort -u > "$covered_tests"

uncovered="$(comm -23 "$all_tests" "$covered_tests")"
if [ -n "$uncovered" ]; then
  echo "The acceptance-test matrix in $workflow does not cover every TestAcc test:"
  printf '%s\n' "$uncovered"
  echo
  echo "Add each test to a group's run_regex, or rename it off the TestAcc prefix if"
  echo "it is a unit test that needs no backend."
  exit 1
fi

echo "All $(wc -l < "$all_tests" | tr -d ' ') acceptance tests are covered by the matrix groups in $workflow."
