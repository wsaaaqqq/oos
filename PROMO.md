# oos — OpenCode Session Finder

> TUI interactive fuzzy finder for OpenCode sessions.
> Find any session instantly and restore your workflow with one keystroke.

Repository: https://github.com/wsaaaqqq/oos

---

## One-liner (通用)

```
oos — 跨所有项目的 OpenCode 会话搜索引擎。输入关键字，瞬间定位到你要找的对话，回车继续。Go TUI，零配置，一条命令安装。
```

---

## 各平台推文速查

| 平台 | 标题 | 描述 |
|---|---|---|
| **V2EX** | OpenCode 会话太多找不到？跨项目全局搜索工具 oos | 几十个项目几百条会话，想不起在哪个项目里聊过哪个话题？oos 跨所有项目搜索全部会话，输入关键字瞬间定位，回车直接继续对话。支持多关键字 AND 逻辑 + `!` 排除，智能消息匹配，目录自动缩略。MIT 开源，一条脚本安装。 |
| **Hacker News** | Show HN: oos — search all OpenCode sessions across all projects at once | You have 400+ sessions scattered across 50+ projects. Which project was that bug discussion in? oos reads your local SQLite db and searches every session from every project in one TUI. Type, find, Enter — back to work. Cross-project global search, real-time filtering, best-match from full history, pure Go single binary. MIT, one-liner install. |
| **Reddit /r/golang** | oos — cross-project session search for OpenCode (Go + bubbletea) | Got 50+ projects and 400+ OpenCode sessions? oos searches across all sessions from all projects at once — type a keyword, find it instantly, press Enter to resume. Cross-project global search, real-time filtering, full-history best-match. MIT, one-liner install. |
| **Reddit /r/commandline** | oos — search all OpenCode sessions across all projects at once | If you use OpenCode across many projects, you know the pain — sessions pile up and `session list` doesn't search. oos reads your local SQLite db and lets you search every session from every project. Type a keyword, find the right conversation, Enter to resume. Cross-project global search, real-time filtering. Works on Windows/Linux/macOS. |
| **抖音** | OpenCode 用户必备：跨项目秒搜全部会话 | 几十个项目几百个会话，想找上次聊的 bug——忘了在哪个项目里？oos 一条命令，跨所有项目搜索全文，输关键词秒定位，回车继续对话。全终端，零配置，一行安装。#开源 #程序员 #效率工具 |
| **小红书** | 开源分享｜OpenCode用户的必备效率工具 oos | 🛠️ 用 OpenCode 写代码久了，会话太多找不到之前的对话？写了 oos——跨所有项目搜索全部会话，输关键词秒定位，回车直接继续聊。Go TUI，三列展示项目目录+匹配消息+时间，一行命令安装。GitHub 开源✨ |
| **Twitter/X** | oos — search all your OpenCode sessions across all projects | Type, filter, Enter. 400+ sessions across 50+ projects in milliseconds. Cross-project global search, Go TUI, one-liner install. github.com/wsaaaqqq/oos #golang #opencode #cli #tui |
| **掘金** | OpenCode 会话太多找不到？我写了 oos 一行命令搞定 | 面对几百条分散在几十个项目里的会话，如何快速找回三周前和 AI 的某次对话？用 Go + bubbletea 写了个 TUI 工具 oos，跨项目全局搜索，输关键字瞬间定位，回车继续对话。 |

---

## Install

```bash
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/wsaaaqqq/oos/master/install.sh | bash

# Windows
iwr -useb https://raw.githubusercontent.com/wsaaaqqq/oos/master/install.ps1 | iex
```

## Tech

- Go 1.24 + bubbletea + lipgloss
- modernc.org/sqlite (pure Go, no CGO)
- SQLite reads `~/.local/share/opencode/opencode.db`
- In-memory filtering, <1ms per keystroke

## License

MIT
