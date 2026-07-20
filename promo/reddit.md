# Reddit 帖子

## /r/golang

**Title:** oos — cross-project session search for OpenCode (Go + bubbletea)

**Body:**

Got 50+ projects and 400+ OpenCode sessions? Can't remember which project you discussed that threading bug in three weeks ago? oos searches across all sessions from all projects at once — type a keyword, find it instantly, press Enter to resume.

- Cross-project global search — one query hits all projects
- Real-time in-memory filtering, <1ms per keystroke
- Multi-keyword AND + `!` exclusion
- Best-match message from full conversation history
- MIT, one-liner install scripts for all platforms

GitHub: https://github.com/wsaaaqqq/oos

---

## /r/commandline

**Title:** oos — search all your OpenCode sessions across all projects at once

**Body:**

If you use OpenCode across many projects, you know the pain — sessions pile up, and `session list` doesn't search. oos reads your local SQLite db and lets you search every session from every project in one place. Type a keyword, find the right conversation, Enter to resume.

- Cross-project global search
- Real-time filtering on every keystroke
- 3-column layout: directory / best-match message / timestamp
- Directory auto-abbreviation, `!` to exclude
- Works on Windows/Linux/macOS

Install:
```bash
curl -fsSL https://raw.githubusercontent.com/wsaaaqqq/oos/master/install.sh | bash
```

GitHub: https://github.com/wsaaaqqq/oos
