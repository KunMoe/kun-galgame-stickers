#!/usr/bin/env bash
# Dispatch one grok executor run. See SKILL.md in this directory.
#
#   GROK_OUT_ROOT=<dir under /tmp> dispatch.sh <slug> [extra grok args...]
#
# Expects <GROK_OUT_ROOT>/<slug>/task.md to exist. Writes run.json, stderr.log and
# debug.log beside it. Extra arguments are passed straight through to grok — that is
# where the per-task --allow 'Write(<glob>)' / 'Edit(<glob>)' rules go.
#
# Knobs: GROK_SANDBOX_PROFILE (default workspace), GROK_PERMISSION_MODE (unset),
# GROK_MAX_TURNS (default 200).
#
# Run this in the background: a real dispatch takes minutes.
set -euo pipefail

slug="${1:?usage: dispatch.sh <slug> [extra grok args...]}"
shift

root="${GROK_OUT_ROOT:?set GROK_OUT_ROOT to a writable directory under /tmp}"
out="$root/$slug"
[ -f "$out/task.md" ] || { echo "dispatch: missing $out/task.md" >&2; exit 2; }

# The sandbox is the only fence left (SKILL.md §2): since grok 1.0.30 + this box's
# always-approve config, --allow rules enforce nothing and the executor has a shell.
# `workspace` = kernel-enforced write access to the repo, /tmp and ~/.grok, nothing
# else; reads and the network stay open. `read-only` and `strict` are NOT options
# here — they need bubblewrap to bind /run/containerd and die before the first turn.
MODE_ARGS=()
[ -n "${GROK_PERMISSION_MODE:-}" ] && MODE_ARGS=(--permission-mode "$GROK_PERMISSION_MODE")

grok --prompt-file "$out/task.md" \
     --sandbox "${GROK_SANDBOX_PROFILE:-workspace}" \
     "${MODE_ARGS[@]+"${MODE_ARGS[@]}"}" \
     --allow "Write($out/**)" \
     --allow "Read($out/**)" \
     --output-format json \
     --max-turns "${GROK_MAX_TURNS:-200}" \
     --debug-file "$out/debug.log" \
     "$@" \
     >"$out/run.json" 2>"$out/stderr.log" || true

if [ ! -s "$out/run.json" ]; then
  echo "dispatch: grok produced no JSON; see $out/stderr.log" >&2
  tail -c 2000 "$out/stderr.log" >&2 || true
  exit 1
fi

stop=$(jq -r '.stopReason // "?"' "$out/run.json")
turns=$(jq -r '.num_turns // "?"' "$out/run.json")
cost=$(jq -r '.total_cost_usd // "?"' "$out/run.json")
printf 'stopReason=%s turns=%s cost_usd=%s out=%s\n' "$stop" "$turns" "$cost" "$out"

[ -s "$out/stderr.log" ] && { echo '--- stderr ---' >&2; tail -c 2000 "$out/stderr.log" >&2; }

if [ "$stop" != "end_turn" ]; then
  # stopReason "cancelled" covers both a denied tool call and turn exhaustion.
  # Only the debug log distinguishes them, which is why it is always written.
  if rg -q 'PermissionCancelled' "$out/debug.log" 2>/dev/null; then
    echo 'dispatch: a tool call was DENIED — a missing --allow rule under GROK_PERMISSION_MODE, or a deny rule' >&2
    rg -o 'confirmation floor tool="[^"]*"' "$out/debug.log" | sort -u >&2 || true
  elif [ "$turns" -ge "${GROK_MAX_TURNS:-200}" ] 2>/dev/null; then
    echo 'dispatch: turn budget exhausted — raise GROK_MAX_TURNS or split the task' >&2
  else
    echo "dispatch: run ended as '$stop'; inspect $out/debug.log" >&2
  fi
  exit 1
fi
