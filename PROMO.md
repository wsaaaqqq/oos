# oos — OpenCode Session Finder

> TUI interactive fuzzy finder for OpenCode sessions.
> Find any session instantly and restore your workflow with one keystroke.

Repository: https://github.com/wsaaaqqq/oos

---

## One-liner (通用)

```
oos — OpenCode 会话的 fzf 替代品。输入关键字实时模糊搜索 400+ 历史会话，回车即恢复项目上下文。Go + TUI，零配置，一条命令安装。
```

---

## 各平台推文速查

| 平台 | 标题 | 描述 |
|---|---|---|
| **V2EX** | 为 OpenCode 写了个 terminal 会话模糊搜索工具 oos | OpenCode 用久了会话堆积 400+，内置的 list 只能翻页不能搜。用 Go + bubbletea 写了个 TUI 工具，输入关键字实时过滤，回车直接打开指定会话。支持多关键字 AND 逻辑 + `!` 排除，目录列自动缩略，消息列上下文匹配。MIT 开源，一条脚本安装。 |
| **Hacker News** | Show HN: Oos — a fuzzy TUI session finder for OpenCode | I built a terminal fuzzy finder for my OpenCode sessions. 400+ sessions, built-in list is scroll-only. oos is a Go TUI that loads the local SQLite db, filters in-memory on every keystroke, and restores any session with Enter. 3-column layout: project dir (auto-truncated) / user question (context-matched) / timestamp. MIT, one-liner install. |
| **Reddit /r/golang** | oos — TUI fuzzy session finder for OpenCode (Go + bubbletea) | Wrote a Go TUI tool to search OpenCode sessions by keyword. Uses bubbletea + lipgloss for the UI, modernc.org/sqlite (pure Go, no CGO) to read the local session DB. Real-time in-memory filtering, multi-keyword AND + `!` exclusion, context-centered message snippets. MIT, one-liner install scripts for all platforms. |
| **Reddit /r/commandline** | oos — replace `opencode session list` with a fuzzy TUI finder | If you use OpenCode and have 400+ sessions, this is for you. Terminal fuzzy finder with real-time keyword search, Enter to open. 24/56/11 column layout, directory auto-abbreviation, `!` to exclude. Works on Windows/Linux/macOS. |
| **抖音** | 开源了一个 OpenCode 的黑科技搜索工具 | 敲代码神器！OpenCode 会话太多找不到？我写了个 oos，输入关键词一秒搜索，回车直接打开历史项目。全终端显示，三列布局，自动目录缩写。GitHub 开源，一行命令安装，Windows/Mac/Linux 都能用。#开源 #程序员 #效率工具 |
| **小红书** | 开源分享｜一个OpenCode用户的效率工具 oos | 🛠️ 折腾了一个终端工具 oos。OpenCode 用久了会话太多，每次找之前的会话都要翻半天。干脆自己写了个 TUI 模糊搜索，输入关键词实时过滤，回车直接打开对应会话。Go 写的，三列展示项目目录+用户提问+时间，安装就一行命令。GitHub 开源✨ |
| **Twitter/X** | oos — OpenCode session fuzzy finder | Type, filter, Enter. 400+ sessions in milliseconds. Go TUI, one-liner install. github.com/wsaaaqqq/oos #golang #opencode #cli #tui |
| **掘金** | 我给 OpenCode 写了个 TUI 会话搜索工具 | 使用 OpenCode 一段时间后，会话列表越来越长，内置的 session list 只有翻页功能。我花了半天用 Go + bubbletea 写了个 TUI 工具 oos，解决这个痛点。文章包含了设计思路、数据库查询优化、TUI 列布局实现。 |

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
