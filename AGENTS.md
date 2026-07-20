# AGENTS.md — oos

Single-binary Go TUI to search across all OpenCode sessions from all projects at once — find any session by keyword and resume the conversation instantly.

## Commands

```bash
go build -o oos.exe .             # build
go vet ./...                       # lint
python scripts/sync-gitee.py       # sync GitHub relase → Gitee

# Release
git tag v1.x.x && git push --tags  # triggers .github/workflows/release.yml
```

No test suite. `go build` + `go vet` is the only verification. CI runs `go vet ./...` on push/PR to master.

## Architecture

```
session.go   — SQLite read-only: loads session + part tables, only at startup
filter.go    — keyword parsing + in-memory matching, called on every keystroke
tui.go       — bubbletea TUI: search bar, result list (3-column fixed-width), status bar
main.go      — entrypoint, runs TUI, then execs `opencode -s <id>` on Enter
```

All files are `package main`. No internal packages.

## Database details

Path: `~/.local/share/opencode/opencode.db` (SQLite, read-only via `modernc.org/sqlite`).

Key tables:
- `session` — id, title, slug, directory, model (JSON), agent, parent_id, time_updated
- `part` — linked to `message` via `message_id`; `data` is JSON with `type`/`text` fields
- `message` — has `data` JSON with `role` but NO text content; actual text is in `part`

Sessions with `parent_id IS NOT NULL AND parent_id != ''` are subagent sessions (explore/general), excluded by SQL filter. Changing this requires updating the WHERE clause in `loadSessionRows`.

User's first question: first `part` row for the first user `message` with `type: "text"`. Stored as `Session.FirstUserMsg` for display and keyword matching.

`LoadAllMessages` queries all text parts across all sessions for MSGS ON mode (~2-5s startup cost).

## Display quirks

Column widths are hardcoded constants in `tui.go`: `colTitle=24`, `colMsg=56`, `colTime=11`. Column separator is ` │ ` (3 cols). Total line width is 2 + 24 + 3 + 56 + 3 + 11 = 99 cols.

`displayWidth()` counts runes > 127 as 2 columns (CJK). This must match how the terminal renders wide chars, or columns misalign.

`truncateCols()` hard-cuts at display-width boundary with no ellipsis.

`highlightMatches()` must be called AFTER `truncateCols()` — if highlighting inserts ANSI codes first, truncation counts them as display width and breaks alignment.

Directory column uses `formatDir()`: leaf directory in full, parents abbreviated to first char moving right-to-left, `!` prefix if parts were dropped.

Message column uses `ctxSnippet()`: finds first matching keyword, shifts view so keyword is ~20% from left edge.

`findBestMsg()`: when keywords exist but not in `FirstUserMsg`, searches all loaded messages for first hit to display.

## Search flow

```
ParseKeys() → Filter (Include/Exclude via space-split + '!' prefix)
FilterSessions() → MatchSession() → sessionContains() OR msgsContain()
```

`msgMap()` returns `m.allMsgs` only when `searchMsgs=true`, nil otherwise. This gates whether message history is searched.

MSGS ON is default (`searchMsgs: true` in `initialModel`). On startup, sessions load first, then all messages load async.

## Release

Do NOT tag or publish unless the user explicitly says "发版", "发布", "release", or similar.

### Versioning

`v<major>.<minor>.<bugfix>`

| bump | trigger | example |
|---|---|---|
| major | breaking changes / major feature overhaul | TUI rewrite |
| minor | new features / functional improvements | new shortcut, Enter-keep-open |
| bugfix | bug fixes only | cross-compile fix, fallback fix |

### Workflow

- `.goreleaser.yml` produces 6 binaries (windows/linux/darwin × amd64/arm64) as raw binaries (no archive, no zip)
- CI: `ci.yml` runs on push/PR to master, just `go build` + `go vet`
- Release: `release.yml` triggered by `v*` tag push, uses GoReleaser
- `PROMO.md` is promo copy for social platforms, not part of the tool
- `scripts/sync-gitee.py` syncs binaries from GitHub release to Gitee, using `git credential fill` for token

## Publishing to Gitee

Two repos: GitHub (`wsaaaqqq/oos`) and Gitee (`haitao666/oos`). Code syncs automatically via Gitee import. Release binaries must be uploaded manually.

Full release workflow:

```bash
# 1. Code is pushed to GitHub (already done)
# 2. Tag and push to trigger GitHub Release
git tag v1.0.2 && git push --tags
# → .github/workflows/release.yml builds 6 platform binaries via GoReleaser

# 3. Wait for GitHub Release to finish (~2 min)
gh run watch

# 4. Sync binaries to Gitee
python scripts/sync-gitee.py --tag v1.0.2
# → downloads all assets from GitHub Release, uploads to Gitee Release
# → skips assets that already exist (idempotent)
```

Gitee token: stored in Windows Credential Manager (`git:https://gitee.com`), retrieved by `git credential fill` inside the script. No manual token config needed.

## Gotchas

- `oos.exe` binary is NOT committed; `.gitignore` excludes `*.exe`
- GoReleaser uses `archives.format: binary`, meaning release assets are single files with no .zip/.tar.gz wrapper
- `install.sh`/`install.ps1` expect release assets named `oos_{os}_{arch}`, NOT `{project_name}_{os}_{arch}`
- The GoReleaser `name_template` is `oos_{{ .Os }}_{{ .Arch }}`
- On Windows, `openSession()` uses `exec.Command` (not `syscall.Exec`) to launch `opencode -s <id>`
