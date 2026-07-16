# oos — OpenCode Session Finder

> TUI interactive fuzzy finder for [OpenCode](https://opencode.ai) sessions. Find any session instantly and restore your workflow with one keystroke.

<p align="center">
  <img src="https://img.shields.io/badge/go-%2300ADD8.svg?style=flat&logo=go&logoColor=white">
  <img src="https://img.shields.io/badge/license-MIT-blue.svg">
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey">
</p>

## Why

OpenCode sessions pile up fast. With 400+ sessions, the built-in `opencode session list` becomes a scrolling nightmare. `oos` gives you a real-time TUI fuzzy search — type a keyword, see matching sessions instantly, press Enter to jump right back in.

```
🥲 Before: scroll through 400+ sessions
😎 After:  oos key1 key2 → Enter → back to work
```

## Demo

```
╭───────────────────────────────────────────────────────────────╮
│ java spring !test                                   msgs ON   │
╰───────────────────────────────────────────────────────────────╯
> C:/ws/java/sgsms-qx      │ Search the project for all control │ 16:32
  !w/s/sgsms-tyzy/ext-app  │ 整理yjcrm项目...                   │ 15:20
  C:/ws/java/sgsms-lp      │ unit test for GdController         │ 13:40
─────────────────────────────────────────────────────────────────
esc quit                                             12 matches
```

- **3 columns**: project directory | user question (context-matched) | timestamp
- **Real-time filter**: every keystroke instantly updates results
- **`!` marker**: directory path truncated → `!` indicates more parents above

## Install

```bash
go install github.com/wsaaaqqq/oos@latest
```

Or build from source:

```bash
git clone https://github.com/wsaaaqqq/oos.git
cd oos && go build -o oos .
```

## Usage

```bash
oos                  # list all sessions
oos java spring      # sessions matching both "java" AND "spring"
oos build !plan      # build sessions, excluding plan agent ones
```

### Keyboard Shortcuts

| Key | Action |
|---|---|
| type keywords | real-time filter, space-separated AND logic, `!key` to exclude |
| `↑` / `↓` | move selection |
| `Enter` | `cd` to project dir + open session with `opencode -s <id>` |
| `Alt+S` | toggle full-message search (ON by default) |
| `Ctrl+W` | delete last keyword |
| `Esc` | quit |

### Display Columns

| Column | Width | Description |
|---|---|---|
| Project dir | 24 cols | leaf dir in full, parents abbreviated to first char, `!` marks truncation |
| User question | 56 cols | context-centered on first keyword match, hard-cut at boundary |
| Timestamp | 11 cols | `HH:MM` (today) or `MM-DD HH:MM` (older) |

## Search Modes

| Mode | Scope | Default |
|---|---|---|
| **MSGS ON** | title + slug + dir + model + agent + user question + **all message history** | ✅ |
| MSGS OFF | title + slug + dir + model + agent + user question | — |

Toggle with `Alt+S`. MSGS OFF is faster; MSGS ON searches deep into conversation history.

## How It Works

Reads `~/.local/share/opencode/opencode.db` (SQLite) — the same database OpenCode uses to store sessions and messages. All filtering happens in-memory after a one-time load at startup.

| Phase | Data | Time |
|---|---|---|
| Startup load | 437 sessions + first user messages | ~200ms |
| Per-keystroke filter | in-memory O(n) scan | <1ms |
| Full-message load (MSGS ON) | all text parts from `part` table | ~2-5s |

## Tech Stack

| Component | Purpose |
|---|---|
| [Go](https://go.dev/) 1.24+ | language |
| [bubbletea](https://github.com/charmbracelet/bubbletea) | Elm-style TUI framework |
| [lipgloss](https://github.com/charmbracelet/lipgloss) | terminal styling |
| [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) | pure-Go SQLite, no CGO |

## Related

- [OpenCode](https://opencode.ai) — the agentic coding tool this is built for
- [opencode session list](https://opencode.ai/docs/cli#session) — built-in session list command
