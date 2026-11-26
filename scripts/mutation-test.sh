#!/usr/bin/env bash
# Runs mutation testing with Gremlins and fails when the tests are too weak or the result cannot be trusted.
# Gremlins 0.6.0 exits successfully even when below its own threshold, so the result is checked here.
set -euo pipefail

min_efficacy=${MIN_EFFICACY:-90}
max_timed_out_percent=5

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT

# Gremlins derives its timeout from a first test run. If that run comes from the test cache,
# the timeout is so short that every mutant times out.
GOFLAGS="${GOFLAGS:-} -count=1" gremlins unleash \
	--integration --coverpkg ./... --timeout-coefficient 5 \
	--invert-assignments --invert-logical --invert-loopctrl \
	--invert-bitwise --invert-bwassign --remove-self-assignments \
	--output "$workdir/report.json" | tee "$workdir/log"

timed_out=$(sed -n 's/^Timed out: \([0-9]*\),.*/\1/p' "$workdir/log")
if [[ -z $timed_out ]]; then
	echo "mutation testing: no result found in the Gremlins output" >&2
	exit 1
fi

# mutants_total counts the killed and surviving mutants only.
read -r tested efficacy < <(jq -r '"\(.mutants_total) \(.test_efficacy)"' "$workdir/report.json")

jq -en --argjson tested "$tested" --argjson timed_out "$timed_out" --argjson max "$max_timed_out_percent" \
	'$tested > 0 and $timed_out * 100 <= $max * ($tested + $timed_out)' >/dev/null || {
	echo "mutation testing: $timed_out mutants timed out and $tested were tested" >&2
	exit 1
}

jq -en --argjson efficacy "$efficacy" --argjson min "$min_efficacy" '$efficacy >= $min' >/dev/null || {
	echo "mutation testing: efficacy $efficacy% is below $min_efficacy%" >&2
	exit 1
}

echo "mutation testing passed: efficacy $efficacy%, $timed_out timed out"
