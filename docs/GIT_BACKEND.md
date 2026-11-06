# Git 后端设计

覆盖 `internal/gitexec`、`internal/git`、`internal/repo`、`internal/fswatch`。总体分层见 [ARCHITECTURE.md](ARCHITECTURE.md)。

## 1. Git 可执行文件

- 查找顺序：设置中的路径 → `PATH` 中的 `git` → 平台常见位置（Windows：`Program Files\Git\cmd\git.exe`；macOS：`/usr/bin/git`、Homebrew 路径）。
- 启动时运行 `git version` 解析版本。**最低版本暂定 2.34**（M0 已采用，M1 验证；见 ROADMAP 待决事项），低于时提示但允许只读使用。
- 版本相关的功能差异集中在 `git.Capabilities` 结构里判断，不在调用点散落版本比较。

## 2. 子进程执行（gitexec）

每次调用都是一个 `Command` 值：参数列表、工作目录、stdin、是否写操作、超时、取消上下文。

**固定环境与参数**，保证输出可解析，同时不改变用户语义：

| 设置 | 原因 |
|---|---|
| `LC_ALL=C`、`LANGUAGE=` | 错误分类依赖英文消息；展示给用户时仍原样显示 stderr |
| `GIT_TERMINAL_PROMPT=0` | 没有终端，避免挂起等待输入 |
| `GIT_OPTIONAL_LOCKS=0`（仅读命令） | 后台 `status` 不抢 index 锁，不与用户在终端的操作冲突 |
| `-c core.quotepath=false` | 路径原样输出（同时使用 `-z`） |
| `-c color.ui=false` | 无 ANSI 颜色 |
| 不加 `--no-verify` | 规格要求正常执行 hooks |

**取消**：Windows 下把子进程放进 Job Object，取消时终止整个 Job（git 会派生 `ssh`、`git-remote-https`、hooks）；Unix 下设置独立进程组，取消时向进程组发 `SIGTERM`，超时后 `SIGKILL`。

**输出**：stdout 流式读取供大输出逐批解析（log、大 diff）；stderr 完整捕获，单条上限 1 MiB，超出截断并标注。进度信息来自 stderr 中的 `--progress` 行（`\r` 分隔），解析成阶段与百分比。

**命令日志**：内存环形缓冲区保留最近 500 条，每条记录参数、工作目录、开始/结束时间、退出码、stderr。写入前对 URL 中的 `user:password@`、`Authorization:` 头和常见 token 形式做遮蔽（规格 §7）。

## 3. 凭据

- HTTPS：交给 Git 的 credential helper（Windows 上通常是 Git Credential Manager，它自带图形窗口）。
- SSH：交给 SSH agent；Windows 上使用 Git for Windows 自带或系统 OpenSSH 的配置。
- 客户端不保存密码，不实现 askpass 输入框（首发）。凭据缺失导致的认证失败按 §6 分类展示，并说明如何配置 helper。
- 是否在后续版本提供内置 askpass 对话框，见 ROADMAP 待决事项。

## 4. 读命令与解析

所有解析器是纯函数 `func parseX([]byte) (X, error)`，配有来自真实 Git 输出的样例测试和模糊测试。

| 数据 | 命令 |
|---|---|
| 工作区状态 | `status --porcelain=v2 -z --branch --untracked-files=all` |
| 引用 | `for-each-ref -z --format=...`（含 upstream、`upstream:track`、objectname、committerdate） |
| Log | `log -z --format=<固定字段> --topo-order` + 范围/筛选参数；`--parents` 用于图 |
| 提交详情 | `show -z --format=... --name-status --find-renames <sha>` |
| 文件 Diff | `diff --no-color --no-ext-diff -U3 [--cached] [-w] -- <path>`；比较提交时 `diff <a> <b> -- <path>` |
| 文件内容 | `cat-file --batch`（长驻进程，Log/Diff 浏览期间复用，空闲 30 秒后关闭） |

当前 M2 增量：Log 按 100 条分页，提交消息以 `--fixed-strings --regexp-ignore-case --grep=<消息>`、作者以 `--author=<名称>`、路径以 `-- <路径>` 在 Git 中筛选后分页；提交文件通过 `ls-tree` 确认父提交/当前提交是否包含该路径，再读取两侧对象，根提交以空内容为基准；合并提交的文件列表与 Diff 相对第一父提交。大文件和二进制文件回退显示 Git patch。图形目前按父 SHA 在 UI 绘制至多六条轨道；日期/分支筛选、复杂图折叠和持久 `cat-file --batch` 尚待实现。
当前 M3 增量：`for-each-ref` 读取本地/远程引用及 upstream；支持新建、重命名、删除本地分支和 upstream 管理。远程 checkout 预检绑定远程目标 SHA、HEAD 与 porcelain 状态；确认后再次校验，干净工作区安全检出，脏工作区可取消、先以 `stash push --include-untracked` 保存再检出，或经二次确认后 `switch --discard-changes` 强制检出。所有写操作由仓库互斥锁串行；变更状态后的过期计划会被拒绝。
当前 M4 增量：Fetch 使用 `fetch --progress --prune`，Pull 使用 `pull --progress --no-rebase`，Push 使用 `push --progress`，后两者依赖当前分支 upstream。`gitexec.Command.OnProgress` 解析 stderr 中 CR/LF 分隔的进度行，同时保留完整有界 stderr；调用方通过 context 取消，平台进程终止逻辑负责停止 Git。完成后统一刷新仓库状态。
Force push 通过计划读取本地 HEAD 与 upstream SHA，确认后重新计算并要求计划完全一致，再执行 `push --force-with-lease=refs/heads/<branch>:<expected>`。远端更新发生在预检之后时，lease 会拒绝覆盖。
| 冲突三方 | `show :1:<path>` / `:2:` / `:3:`，或从 `ls-files -u -z` 读取 stage |
| Stash | `stash list -z --format=...`；`stash show -p --include-untracked` |
| 配置 | `config --list --show-origin --show-scope -z` |
| 进行中状态 | 直接读取 `$GIT_DIR` 下 `MERGE_HEAD`、`CHERRY_PICK_HEAD`、`REVERT_HEAD`、`rebase-merge/`、`rebase-apply/` |

`$GIT_DIR` 由 `rev-parse --git-dir --show-toplevel --git-common-dir` 得出，支持 `.git` 文件形式的工作树与子模块（规格 §4.1）。

二进制判断沿用 Git：diff 输出 `Binary files ... differ` 时只展示类型、大小（`cat-file -s`）和版本信息（规格 §4.3）。

## 5. 写操作

| 规格功能 | 实现 |
|---|---|
| 文件级 Stage / Unstage | `add -- <paths>` / `restore --staged -- <paths>`（初始提交前用 `rm --cached`） |
| Hunk 级 Stage / Unstage | 由已解析的 diff 生成只含所选 hunk 的补丁，`apply --cached [--reverse] --recount -`；单行选择扩展到完整 hunk（规格 §4.2） |
| Discard | `restore --worktree -- <paths>`；未跟踪文件用 `clean -f -- <paths>`，预检列出具体文件 |
| Commit / Amend | `commit -F -`（消息走 stdin，避免转义问题）`[--amend]`；hook 失败时保留消息 |
| 分支 | `switch [-c]`、`branch -m/-d/-D`、`branch --set-upstream-to` |
| Checkout 安全检查 | 预检 porcelain 状态并列出受影响文件；给 Cancel / Stash / Force，Force 二次确认且执行前校验计划状态 |
| Fetch / Pull / Push | `fetch --progress [--prune]`、`pull --progress --no-rebase / --rebase / --ff-only`、`push --progress [--force-with-lease=<ref>:<sha>] [-u]` |
| Merge / Rebase / Cherry-pick / Revert | 对应命令；`--continue` / `--abort`；`GIT_EDITOR=true` 让 continue 使用默认消息，消息需要编辑时由客户端先写入再继续 |
| 冲突标记解决 | 写入结果文件后 `add -- <path>` |
| Stash | `stash push [-u] -m`、`apply`、`pop`、`drop`、`branch` |
| Reset | `reset --soft/--mixed/--hard <sha>`；hard 必须走预检 |

需要编辑器的命令一律通过环境变量（`GIT_EDITOR`、`GIT_SEQUENCE_EDITOR`）禁用交互，避免子进程等待一个不存在的编辑器。

## 6. 错误分类

`git.Error` 保留原始命令、退出码、完整 stderr，并附带一个分类，供 UI 选择呈现方式与下一步操作（规格 §4.6、§7）：

| 分类 | 识别依据（`LC_ALL=C` 下） | 建议操作 |
|---|---|---|
| `Network` | `Could not resolve host`、`Connection timed out`、`unable to access` | 重试 |
| `Auth` | `Authentication failed`、`Permission denied (publickey)`、`could not read Username` | 说明凭据配置 |
| `HookRejected` | `pre-receive hook declined`、本地 hook 非零退出 | 展示 hook 输出 |
| `NonFastForward` | `[rejected] ... (non-fast-forward)`、`fetch first` | Fetch / Pull |
| `Conflict` | 退出码 + 存在未合并路径 | 进入冲突视图 |
| `WouldOverwrite` | `would be overwritten by` | 列出文件，Stash / Cancel |
| `Locked` | `index.lock': File exists` | 说明另有 Git 进程，提供重试 |
| `NoUpstream` | `has no upstream branch` | 设置 upstream 流程 |
| `Identity` | `Please tell me who you are` | 设置 user.name / user.email |
| `Unknown` | 其他 | 展示原文 |

分类只决定 UI 的建议操作，任何时候都能展开原始输出。

## 7. 文件监听与刷新（fswatch + repo）

- 监听工作树根目录（递归）和 `$GIT_DIR`。Windows：`ReadDirectoryChangesW`；macOS：FSEvents；Linux：inotify 按目录添加，目录数超过系统上限时退化为只监听 `$GIT_DIR` 并提示手动刷新。
- 事件经 300 ms 去抖后合并为一次刷新请求，按路径决定刷新哪些快照部分：工作树文件 → `Status`；`index` → `Status`；`refs/`、`HEAD`、`packed-refs` → `Refs` + `Status`；`MERGE_HEAD` 等 → `InProgress`。
- 被 `.gitignore` 忽略的路径产生的事件不触发刷新（用 `check-ignore --stdin` 批量判断并缓存目录结论）。
- 客户端自己的写操作期间暂停事件驱动刷新，结束后统一刷新一次，避免操作过程中的中间状态闪现。
- 窗口不可见或最小化时继续收集事件但不刷新，恢复可见时刷新一次。
- 去抖计时器只在有事件时存在，空闲时没有定时器（Viva 规则）。

M1 当前实现：Windows 使用阻塞的 `ReadDirectoryChangesW`，Linux 使用 inotify + epoll 监听工作树和 Git 目录；事件经 300 ms 合并后刷新整个 Status。切换仓库取消旧监听，监听就绪时再读一次状态。macOS 目录监听、按路径部分刷新、忽略路径过滤以及 Linux 运行中新目录注册失败的通知尚未实现；初次注册失败会在界面显示错误，仍可手动刷新。Git 子进程在 Linux/macOS 使用独立进程组，取消时终止该组；Windows 当前仅终止直接进程，Job Object 仍待实现。

## 8. Log 图布局

- 输入：按 `--topo-order` 到达的 `(sha, parents)` 流。
- 输出：每行的节点列、穿过该行的边、合并/分叉连线。采用常见的“活动泳道”算法：维护当前每列期待的 sha，新提交占据期待它的最左列，父提交依次分配列。
- 增量：每批到达后只计算新行，泳道状态在批次之间延续，保证滚动和追加加载时图形连续（规格 §4.4）。
- 布局结果是紧凑的整数数组，绘制时按可见行读取，不为每行分配对象。
- 纯函数、无 UI 依赖，用小型构造历史做表驱动测试。
