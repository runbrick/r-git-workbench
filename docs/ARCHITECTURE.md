# 架构总览

状态：草案 0.1，对应 [产品规格](GIT_CLIENT_PRODUCT_SPEC.md) 草案 0.1
技术栈：Go 1.26、Viva v0.2.4（GPU 自绘，无 WebView）、系统 Git CLI

细节分文档：

- [GIT_BACKEND.md](GIT_BACKEND.md)：Git 调用、解析、操作、错误分类、数据安全
- [UI.md](UI.md)：界面结构、标识树、快捷键、本地化、无障碍、Viva 缺口
- [TESTING.md](TESTING.md)：测试分层与性能预算
- [ROADMAP.md](ROADMAP.md)：里程碑与待决事项

## 1. 设计约束

规格中直接决定架构的要求：

| 规格要求 | 架构上的落点 |
|---|---|
| Git CLI 是最终语义来源（§1） | 只通过子进程调用系统 `git`，不链接 libgit2/go-git |
| 每仓库同时只能有一个写操作（§7） | 每个仓库一个写队列；读操作并行 |
| Log 加载不得冻结界面（§9） | 所有 Git 调用在后台 goroutine，结果经 `Invalidator` 回到窗口线程 |
| 状态刷新事件驱动、批量（§9） | 文件监听 + 去抖，一次 `git status` 覆盖全仓库 |
| 有损操作先展示影响再确认（§1、§7） | 有损操作统一走“预检 → 确认 → 执行”三段式 |
| 失败不丢上下文（§4.2、§7） | 用户输入（提交消息、选择、筛选）只存在 UI 状态，不随操作结果重建 |
| 空闲不耗资源（Viva 规则） | 没有轮询与心跳；后台 goroutine 只在等待 OS 事件或子进程时存在 |

## 2. 分层

```text
cmd/workbench          main：组装各层，启动窗口
   │
internal/ui/...        Viva 绘制函数与界面状态（只在窗口线程）
   │  读取快照 / 提交 Intent
internal/workbench     会话：已打开仓库、视图状态记忆、操作调度、通知 UI
   │
internal/repo          单仓库服务：快照、写队列、刷新调度、文件监听
   │
internal/git           Git 领域层：类型化命令、输出解析、错误分类
   │
internal/gitexec       子进程执行：环境、取消、输出捕获、命令日志

旁路：internal/keymap  internal/i18n  internal/settings  internal/fswatch
```

依赖只能向下。`internal/git` 及以下不导入 Viva，可在没有窗口的环境里单独测试。

### 包职责

| 包 | 职责 | 不负责 |
|---|---|---|
| `gitexec` | 启动 `git`，设置环境变量，捕获 stdout/stderr，取消进程树，记录命令日志 | 理解输出含义 |
| `git` | 每种 Git 操作的类型化函数与解析器（status、log、refs、diff、stash …）；错误分类 | 并发、缓存 |
| `repo` | 一个仓库的当前快照；写操作排队；监听文件变化后刷新；仓库进行中状态（merge/rebase …） | 绘制 |
| `workbench` | 多仓库会话、最近项目、每仓库视图记忆、后台结果到 UI 的投递 | Git 细节 |
| `ui/*` | 每个视图一个包：shell、changes、diff、log、branches、stash、conflicts、dialogs、settings | 直接启动子进程 |
| `keymap` | 动作注册表、预设键位、上下文匹配、冲突检测 | 执行动作 |
| `i18n` | 中英文字符串表，编译期常量键 | 翻译 Git 原始输出 |
| `settings` | 应用设置与每仓库 UI 状态的读写（JSON，原子写入） | Git config（由 `git` 包读写） |
| `fswatch` | 递归目录变化通知，按平台实现 | 判断是否需要刷新 |

## 3. 线程与数据流

Viva 的绘制函数在窗口线程上每帧运行。规则：

1. **窗口线程**拥有全部 UI 状态和当前显示的仓库快照指针。绘制函数只读快照，不等待 I/O。
2. **后台工作**（Git 子进程、解析、diff 计算）在 goroutine 中运行，完成后把结果放进 `workbench` 的收件箱（互斥锁保护的切片），然后调用 `Invalidator.Invalidate()`。
3. **`Config.OnWake`** 在窗口线程上取出收件箱，替换快照指针、更新操作状态，然后 Viva 重绘一次。
4. 快照是**不可变值**：后台生成新快照，窗口线程整体替换，不对共享结构加锁读写。

```text
用户点击 Commit
  → ui 构造 Intent{Commit, message, amend}
  → workbench.Submit(intent)                    窗口线程，立即返回
  → repo 写队列执行 git commit                  后台
  → 结果入收件箱 + Invalidate()
  → OnWake：标记操作完成，触发 status/log 刷新  窗口线程
  → 刷新结果入收件箱 + Invalidate()
  → OnWake：替换快照 → 重绘
```

空闲时没有 goroutine 在运行：写队列为空时不保留工作协程，文件监听协程阻塞在 OS 调用上。动画（进度条、旋转图标）只在有操作运行时安排重绘。

## 4. 仓库快照

`repo.Snapshot` 是界面读取的唯一数据源，按数据来源分成可独立刷新的部分，每部分带代数（generation）号：

| 部分 | 来源 | 触发刷新 |
|---|---|---|
| `Status` | `git status --porcelain=v2 -z --branch` | 工作区或 index 变化 |
| `Refs` | `git for-each-ref` | `.git/refs`、`packed-refs`、`HEAD` 变化 |
| `InProgress` | 读 `.git` 中的 `MERGE_HEAD`、`rebase-merge/` 等 | `.git` 变化 |
| `Stashes` | `git stash list` | `refs/stash` 变化 |
| `Config` | `git config --list --show-origin -z` | `config` 文件变化、设置页修改 |

Log 和 Diff 体量大，不放进快照：

- **Log** 由 `repo.LogCursor` 分批加载（每批 1000 条以内），图形布局随批次增量计算，界面只绘制可见行。筛选条件改变时创建新游标，旧游标取消。
- **Diff** 按需请求，结果按 `(左版本, 右版本, 路径, 选项)` 缓存，最多保留少量最近项；大文件请求可取消。

## 5. 写操作调度

每个仓库一个写队列，任一时刻最多执行一个写操作（规格 §7）。

- 操作对象 `Operation` 记录：类型、目标描述（用于状态栏与确认框）、是否可取消、开始时间、阶段文字。
- 写操作执行期间，读刷新照常进行；刷新结果代数落后于写操作结束时，以写操作结束后的那次刷新为准。
- 取消会终止整个子进程树（见 GIT_BACKEND §2），之后必定重新读取仓库状态，不假设已回滚。
- `Esc` 只关闭对话框，不取消运行中的操作（规格 §5）。

## 6. 有损操作：预检 → 确认 → 执行

Discard、Hard reset、Force checkout、Force push、删除未合并分支、删除 Stash 等操作统一分三步：

1. **预检**（后台）：计算影响，例如将丢弃的文件列表、将被覆盖的远端引用、未合并的提交。
2. **确认**（UI）：对话框展示预检结果中的对象名、SHA、文件数和 refspec。
3. **执行**：只接受预检返回的 `Plan` 值，执行前校验仓库状态与预检时一致（HEAD、index 代数、目标引用值），不一致则作废并重新预检。

这样确认框里展示的内容就是实际执行的内容，Force push 的 `--force-with-lease=<ref>:<预期 SHA>` 直接取自 `Plan`。

## 7. 持久化

| 数据 | 位置 | 格式 |
|---|---|---|
| 应用设置、键位、最近仓库 | 用户配置目录 `r-git-workbench/settings.json` | JSON，写临时文件后重命名 |
| 每仓库视图状态（筛选、选中项、面板尺寸） | 用户配置目录 `r-git-workbench/repos/<路径哈希>.json` | 同上 |
| 最近提交消息 | 每仓库状态文件内 | 同上 |
| Git 配置 | 仓库与用户的 Git 配置文件 | 只通过 `git config` 读写 |
| 命令日志 | 内存环形缓冲区；用户主动导出时写文件 | 导出前遮蔽凭据 |

不向仓库内写入任何客户端私有文件。不保存密码。

## 8. 依赖策略

发布二进制只依赖 Go 标准库、`golang.org/x/sys`（平台调用）和 Viva。新增依赖前需测量剥离后二进制大小、启动时间和空闲内存（Viva 规则 §2）。测试代码可使用标准库以外的工具，但不进入发布构建。
