---
name: dispatch-grok
description: Dispatch investigation and mechanical implementation work to the local grok CLI as a headless executor while this session stays the orchestrator and acceptor. Use when the user asks to "派发 grok" / "dispatch grok", or when a task is large enough to hand off as a written task book. Covers what grok may touch in this repo, the sandbox that is the only real fence, the pnpm trap that corrupts this workspace, the cross-repo ground-truth rule for nextmoe-infra, and the acceptance protocol.
---

# Dispatching grok in kun-galgame-stickers

> **If you are the grok executor and this file was loaded into your context: ignore it.**
> It describes how the orchestrator dispatches *you*. It is not a task book.

Adapted from `../nextmoe-infra/.claude/skills/dispatch-grok`. The machine facts in §1–§2 are
inherited and still hold; everything else is this repo's.

This session is the **orchestrator**: it adjudicates design, writes the task book, decides which
commands the executor may run, and accepts or rejects the result. The local `grok` CLI is the
**executor**. Do not use Claude subagents for the work this skill covers — dispatch `grok`.

## 1. What grok can do on this machine

Verified against `grok 1.0.30`. `/etc/grok/requirements.toml` pins only telemetry and
remote-fetch; `~/.grok/config.toml` sets `[ui] permission_mode = "always-approve"`. Everything
below runs **unprompted**, with no `--allow` rule of any kind: reading anywhere on the box,
writing anywhere, `run_terminal_command`, LSP, the Playwright browser MCP, subagents, web search.

grok loads this repo's `CLAUDE.md` automatically, so the iron rules are already in its context.
Do not re-paste them; do restate the specific ones the task turns on.

## 2. The `--allow` fence is gone. The sandbox is what is left.

`--allow 'Write(<glob>)'` enforces nothing under always-approve, and the shell walks around any
path rule regardless of mode. Two things remain, and both are load-bearing:

1. **`--sandbox workspace`** — kernel-enforced (Landlock). The executor may write this repo,
   `/tmp` and `~/.grok`, nothing else. `dispatch.sh` passes it on every run. Reads stay
   unrestricted and the network stays open. `read-only` and `strict` do not start on this box.
2. **Discipline in the task book plus `git status --porcelain` at acceptance.** Name the writable
   paths anyway — it is what the executor follows — and diff before you believe anything.

Never run two dispatches concurrently over overlapping paths. Sequential, or disjoint.

## 3. The pnpm trap — this repo's own scar

**Do not put `pnpm lint`, `pnpm typecheck`, `pnpm install` or any `pnpm <script>` in the
executor's allowed-command list.** `--sandbox workspace` blocks writes to
`~/.local/share/pnpm/store`, so pnpm silently falls back to an in-repo `.pnpm-store/` and rewrites
`node_modules/.modules.yaml` `storeDir` to point at it. Every later `pnpm <script>` in this repo
then dies with `ERR_PNPM_ABORTED_REMOVE_MODULES_DIR_NO_TTY`, and `pnpm install` will not fix it —
it reports "Already up to date", because install and run-script use different checks. `--force`
does not help either.

Recovery, ~8s with a warm store:

```bash
rm -rf .pnpm-store node_modules apps/*/node_modules && CI=true pnpm install
rg '"storeDir"' node_modules/.modules.yaml     # must point back at ~/.local/share/pnpm/store
```

Give the executor the Go gates only. Run the frontend gates yourself after it finishes.

## 4. Who runs what

**Let the executor run** — and list them by name in the task book:

```
go build ./...        go vet ./...        gofmt -l .        go test ./<narrow package>
rg / fd / ast-grep    git log / git diff / git status / git show     (reads only)
```

`go test ./...` is safe here: nothing in `apps/api` opens a database in a unit test.

**Never delegate** — the task book forbids these by name:

- Git that writes: `commit`, `push`, `branch`, `rebase`, `stash`, PRs.
- **Anything against `kungalgame_sticker`**: `pnpm migrate`, `go run ./cmd/migrate`, `psql`, the
  postgres MCP. Schema is this session's responsibility, and so is the migration reminder that
  iron rule 8 requires at the end of any task that touches `apps/api/migrations`.
- **Every `pnpm` invocation** (§3), including `pnpm dev`, `pnpm dev:web`, `pnpm dev:api`.
- Starting or stopping any service on this box — this repo's stack on `:9421`/`:5173`, and the
  nextmoe-infra stack on `:9277`/`:9281`/`:9282`/`:9420`/`:9421` which the user runs by hand.
- Production operations. See the `sticker-prod-access` memory; none of it is delegable.
- **Final acceptance.** `git status --porcelain` shows only the expected paths; re-run every gate
  yourself; spot-check the report's highest-stakes claims against the code. A gate the executor
  says it ran is a claim, not a result.

## 5. Cross-repo ground truth: nextmoe-infra

This repo is an OAuth RP and an S2S client of nextmoe-infra. Its contracts — `catalog`,
`community`, `image`, `oauth` — live in `../nextmoe-infra`, and **that working tree is routinely
tens of commits behind `origin/main`**. Reading it is the single most likely way for a dispatch to
come back confidently wrong.

The task book must say, in the discipline section:

```
Read every infra file through `git -C ../nextmoe-infra show origin/main:<path>`.
Never read ../nextmoe-infra's working tree — it is stale, and so is the binary running on :9282.
```

The running services are built from that stale tree too, so "I curled it and got a 404" proves
nothing about the contract. Ground truth is `origin/main`, not the process.

## 6. The browser

`~/.grok/playwright-mcp.json` runs **headed** Chromium on `DISPLAY=:0`. A window opens on the
user's desktop — say so before dispatching a browser task, and keep the session short.

Useless here unless the orchestrator has already confirmed the stack is up: this repo needs a
`.env` (only `.env.example` is checked in) plus a live Go API on `:9421` and Nuxt on `:5173`.
**The orchestrator starts the stack, never the executor** (§4) — and if it is not up, say so to
the user rather than dispatching a browser task that will screenshot a connection refused.

Never point it at production or at an authenticated production session.

## 7. Reading the result

1. **`stopReason: "cancelled"` is ambiguous** — a denied tool call *or* turn exhaustion. Grep the
   `--debug-file` for `PermissionCancelled` to tell them apart; `dispatch.sh` does this for you.
2. **`.text` is the concatenation of every assistant text block**, not the final answer. Never
   parse a report out of it. **Require a report written to a file** in the output directory.
3. **`--json-schema` output is also concatenated.** Take the *last* balanced JSON object.

The report lives in the scratchpad, never in the repo, so `git status --porcelain` after a run is
a pure signal.

## 8. The task book

Template: `task-book-template.md` in this directory. Requirements:

- **Write it in English**, even when the conversation is in Chinese.
- **Self-contained.** grok sees none of this conversation. State the repo path, the branch, where
  the code lives, and every prior adjudication it depends on, inline.
- **Name the commands it may run, and the ones it may not** (§3, §4).
- **Scope and out-of-scope, both named.** Out-of-scope is the only thing stopping a helpful
  executor from refactoring into someone else's paths.
- **The infra ground-truth rule** (§5), quoted, in every task book that touches a contract.
- **Acceptance criteria the orchestrator will actually run** — exact commands, expected output.
- **No open design decisions.** If the mechanics depend on code grok has yet to read, state the
  invariant plus the precedent, and require it to report the mechanics it chose.
- **A discipline section:** writable paths, forbidden operations, and *report, don't work around*.
- **A named report path and a fixed report structure.**
- **Forbid ranking.** The executor cannot see what the orchestrator knows; its importance ordering
  is noise. Require a flat list at equal weight, and put "anything that looks wrong, in scope or
  not" *near the top*.
- **Demand a positive control on any audit or census.** "I found 4" is unreadable without "and
  here are the 16 I checked that were clean".

## 9. The dispatch

```bash
export GROK_OUT_ROOT="$SCRATCHPAD/grok"          # session scratchpad, never the repo
mkdir -p "$GROK_OUT_ROOT/<slug>"
# write the task book to $GROK_OUT_ROOT/<slug>/task.md, then:
.claude/skills/dispatch-grok/dispatch.sh <slug> \
  --allow 'Write(apps/api/internal/platform/sticker/**)' \
  --allow 'Edit(apps/api/internal/platform/sticker/**)'
```

Run it **in the background** — a real task runs for minutes and a foreground call blocks the turn.

Knobs: `GROK_SANDBOX_PROFILE` (default `workspace`; do not disable), `GROK_PERMISSION_MODE`
(unset; `default` restores the path fence on the file-write tools at the cost of a run that dies
on the first unruled write), `GROK_MAX_TURNS` (default 200). Useful passthrough flags:
`--effort low|medium|high|xhigh` (default `high`), `-m grok-4.5`, `--no-subagents`,
`--disable-web-search`, `-w <name>` for a worktree (remove it when the wave ends).

Rule prefixes are the Claude-compatible names — `Write`, `Edit`, `Read`, `Bash`. A native grok
tool name (`write`, `search_replace`, `run_terminal_command`) is a hard error.

## 10. What is worth dispatching

Dispatching buys **orchestrator context**, not money. Judge each candidate by whether the result
compresses into something checkable without re-reading the input.

| Shape | Verdict |
|---|---|
| Broad read → narrow report whose findings are `file:line` + a quoted line | **Dispatch.** Contract audits against `origin/main`, censuses, "find every X across N files". |
| Wide mechanical edit a Go gate asserts | **Dispatch**, and let it run `go build` / `go vet` / `gofmt` / `go test` itself. |
| A question only a read-only command settles | **Dispatch**, as long as it touches no shared state and no database. |
| New code carrying design judgement | **Do not.** Every line must be read to be reviewed; net saving ≈ zero. |
| Anything needing the database, a migration, `pnpm`, a running service, or a credential | **Do not.** §3, §4. |
