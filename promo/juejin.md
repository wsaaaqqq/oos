# OpenCode 会话太多找不到？我写了 `oos` 一行命令搞定

## 一、场景

你有没有过这种经历——

周一早上坐下来，想起上周和一个 AI 结对编程讨论过一个 bug，聊了半天还挺深入。但问题是：**那个对话在哪个项目里？**

翻 `my-api` 项目的 session list……不是。翻 `payment-service`……也不是。再翻 `frontend-app`……翻了三页，终于找到了——里面又聊了 40 轮对话。

OpenCode 用多了就是这样的：几十个项目 × 几十个会话 = 几百条历史，内置的 `session list` 只能翻不能搜，找一条旧会话像大海捞针。

于是写了 **oos** — 跨所有项目，搜索全部会话，一键定位继续对话。

## 二、怎么用

```bash
oos
```

一个终端界面：

```
╭───────────────────────────────────────────────────────────────╮
│ bug fix                                              msgs ON   │
╰───────────────────────────────────────────────────────────────╯
> !p/my-api                 │ 帮我修复登录页面的 bug     │ 07-14 16:32
  !p/payment-service        │ Fix race condition in order       │ 07-13 15:20
  !w/frontend-app           │ 这个 bug 怎么定位的        │ 07-10 13:40
─────────────────────────────────────────────────────────────────
type to search  enter: resume  esc: quit              3 matches
```

**操作流程**：

1. 输关键字，比如 `bug fix`——实时过滤，跨所有项目命中
2. `↑` `↓` 上下键选到你要的会话
3. `Enter` 回车——直接回到那次对话，无缝继续

**常用快捷键**：

| 操作 | 快捷键 |
|---|---|
| 复制项目路径 | `Alt+Q` |
| 删除当前会话 | `Ctrl+D` 按两次确认 |
| 切换搜索模式 | `Alt+S`（全历史 / 仅首条问题） |
| 退出 | `Esc` |

**搜索技巧**：
- 多关键字 AND 逻辑：`bug fix` 匹配同时包含 bug **和** fix 的会话
- `!` 前缀排除：`bug !python` 排除包含 python 的
- 目录列自动缩略：`/home/user/projects/my-backend-service` → `!p/my-backend-service`
- 消息列智能匹配：有搜索关键字时，自动从全部历史消息中找最佳匹配的那条，视口切到关键字附近

## 三、安装

### 方式一：从 Gitee 下载（国内推荐）

去 Gitee Releases 页面下载对应平台的二进制：
https://gitee.com/haitao666/oos/releases

选择对应文件：
- Windows 64位：`oos_windows_amd64.exe`
- macOS Intel：`oos_darwin_amd64`
- macOS Apple Silicon：`oos_darwin_arm64`
- Linux 64位：`oos_linux_amd64`
- Linux ARM64：`oos_linux_arm64`

Linux/macOS 还需要 `chmod +x` 并放到 `PATH` 里：
```bash
chmod +x oos_linux_amd64
mv oos_linux_amd64 ~/.local/bin/oos
```

Windows 用户把 `.exe` 放到任意 `PATH` 目录下即可。

### 方式二：从 GitHub 下载

```bash
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/wsaaaqqq/oos/master/install.sh | bash

# Windows
iwr -useb https://raw.githubusercontent.com/wsaaaqqq/oos/master/install.ps1 | iex
```

> 国内用户如果 GitHub 访问慢，推荐用方式一。

### 使用

```bash
oos
```

进入 TUI 后输入关键字搜索，`↑↓` 选会话，`Enter` 继续对话。

## 四、总结

`oos` 实现跨项目查找opencode会话，并快速继续对话。

GitHub: https://github.com/wsaaaqqq/oos
Gitee: https://gitee.com/haitao666/oos
License: MIT

---

<!-- TODO: 插入终端录屏 gif -->
