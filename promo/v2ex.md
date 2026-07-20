# V2EX 帖子

## 标题
OpenCode 会话太多找不到？写了个跨项目全局搜索工具 oos

## 正文

用 OpenCode 一段时间后的通病：会话堆积了上百条，分散在不同项目里，想找回三周前聊过的某个话题——完全想不起来在哪个项目里。

OpenCode 内置的 session list 只能翻不能搜，更没法跨项目全局搜索。于是用 Go + bubbletea 写了个 TUI 工具——

**核心就一件事**：输关键字，跨所有项目搜索全部会话，回车直接继续对话。

```
oos              # 启动
输入 "bug fix"    # 实时过滤，跨所有项目的会话全出来
↑↓               # 上下键选会话
Enter            # 回到那次对话，无缝继续
```

常用：`Alt+Q` 复制项目路径 · `Ctrl+D` 按两次删除会话 · `Esc` 退出

特性：
- 输入关键字实时过滤
- 跨项目全局搜索：一次搜索命中所有项目
- 智能消息匹配：不只是搜首条问题，自动从全部历史消息找最匹配的那条
- 多关键字 AND 逻辑 + `!` 排除
- 目录列自动缩略、消息列上下文居中

MIT 开源，一条命令安装：

```bash
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/wsaaaqqq/oos/master/install.sh | bash

# Windows
iwr -useb https://raw.githubusercontent.com/wsaaaqqq/oos/master/install.ps1 | iex
```

GitHub: https://github.com/wsaaaqqq/oos
Gitee（国内镜像）: https://gitee.com/haitao666/oos

<!-- TODO: 插入终端录屏 gif -->
