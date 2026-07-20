# Hacker News (Show HN)

## Title
Show HN: oos — search all your OpenCode sessions across all projects at once

## Body

You have 400+ OpenCode sessions spread across 50+ projects. Which project was that deep-dive discussion about async error handling in? Which session was that perf optimization chat? oos solves this.

It reads your local OpenCode SQLite db and lets you search across every session from every project in a single TUI. Type a keyword, find the right conversation instantly, press Enter — and you're back in that session, right where you left off.

- Cross-project global search: one query covers all projects
- Real-time filtering, <1ms per keystroke
- Best-match message from full conversation history (not just the first question)
- 3-column layout: directory / best-match message / timestamp
- Single binary, no dependencies

MIT, one-liner install:
```bash
curl -fsSL https://raw.githubusercontent.com/wsaaaqqq/oos/master/install.sh | bash
```

GitHub: https://github.com/wsaaaqqq/oos
