# 里程碑与待决事项

每个里程碑结束时：规格 §10 对应验收项有自动化测试，`go vet` / `go test` 通过，性能预算中相关项已测量。

## 里程碑

| 编号 | 内容 | 规格章节 | 完成标志 |
|---|---|---|---|
| M0 | 工程骨架：`go.mod`、目录结构、`gitexec` 与版本检测、空主窗口（顶栏/导航/状态栏）、Driver 测试与 headless 入口 | §3、§4.1 末条 | 空窗口 headless `tree` 输出含全部导航项且 `expect named` 通过 |
| M1 | **垂直切片**：打开仓库、Changes 三分组、文件级 Stage/Unstage、文件 Diff、Commit；文件监听刷新 | §4.1、§4.2、§4.3、§11 | 按规格 §11 测量启动、刷新、二进制大小、无障碍，写入 `docs/benchmarks/`；**决定是否继续使用 Viva** |
| M2 | Log：分批加载、图布局、筛选、提交详情与文件 Diff、Copy SHA | §4.4 | 基准仓库滚动无卡顿，图形增量连续 |
| M3 | 分支：选择器、Branches 视图、新建/切换/重命名/删除、upstream、Checkout 安全检查 | §4.5 | Checkout 覆盖文件时出现 Cancel/Stash/Force |
| M4 | 远端：Fetch/Pull/Push、进度与取消、错误分类、Force push with lease | §4.6、§7 | bare 仓库测试覆盖 §6 全部错误分类中的可构造项 |
| M5 | Changes 完整化：hunk 级暂存、Discard、Amend、提交并推送、提交消息历史、目录树模式 | §4.2 | 规格 §10 第 3、4 项 |
| M6 | Merge / Rebase / Cherry-pick / Revert / Reset 与预检确认 | §4.4、§4.7 | 有损操作全部经过预检 |
| M7 | 冲突解决：冲突清单、三方视图、Continue/Abort | §4.8 | 规格 §10 第 6 项 |
| M8 | Stash 全流程 | §4.9 | 规格 §10 第 8 项 |
| M9 | 快捷键：三套预设、Keymap 设置、冲突检测、命令面板 | §5 | 规格 §10 第 9 项 |
| M10 | 设置、i18n 补全、无障碍审查、诊断导出 | §6、§7、§8 | 规格 §10 剩余项与三平台人工验收 |
| 后续 | Worktree、子模块、交互式 Rebase、Tag/远端管理、托管平台集成 | §2 后续 | — |

快捷键和 i18n 的基础设施（动作注册表、字符串表）在 M0 建立，各里程碑新增动作与文案时直接登记，M9/M10 只做预设补全与设置界面。

## 待决事项

开发开始前或对应里程碑开始前需要确认：

| 编号 | 问题 | 当前倾向 | 何时决定 |
|---|---|---|---|
| D1 | Go module 路径 | M0 使用本地模块名 `r-git-workbench`；确定发布仓库 owner 后再改为正式地址 | 发布前 |
| D2 | 最低 Git 版本 | M0 暂定 2.34；较老版本显示警告，仅允许只读使用 | M1 验证 |
| D3 | 文件监听实现 | 自写三平台实现（`ReadDirectoryChangesW` / FSEvents / inotify），不引入 fsnotify：fsnotify 在 macOS 用 kqueue，每个文件占一个描述符，不适合大仓库；已在 [viva#133](https://github.com/vivaui/viva/issues/133) 询问是否由 Viva 提供 | M1 前 |
| D4 | Diff 视图：扩展 Viva `CodeDiff` 还是应用内自绘 | M1 先用 `CodeDiff` 做并排只读，hunk 勾选与上下排列看 [viva#131](https://github.com/vivaui/viva/issues/131) 进展，M5 前定 | M5 前 |
| D5 | “Shift 两次”触发全局搜索 | 已提议 [viva#130](https://github.com/vivaui/viva/issues/130)；未支持前首发不提供此键位，规格偏差需记录 | M9 前 |
| D6 | 内置 askpass 对话框 | 首发不做，依赖 credential helper / SSH agent | M4 前 |
| D7 | 是否保留 Viva | 以 M1 测量结果为准（规格 §11 要求以垂直切片决定技术栈） | M1 结束 |
| D8 | 打包与更新渠道 | `viva build` / `viva package`；自动更新暂不做 | M10 前 |

## M0 进度

已建立 Go 1.26 / Viva 工程、`gitexec` 子进程封装、Git 定位与版本能力检测、i18n 字符串表、导航动作注册表和可切换的空窗口。开发时通过 `go.mod` 的 `replace` 使用 `D:/viva` 工作树；发布前需移除本机路径并确定依赖版本。Git 检测发生在窗口创建前；后续仓库操作的异步结果已有 `workbench` 收件箱、`Invalidator`、`OnWake` 接口。

## M1 进度

已实现异步打开仓库（原生目录选择或路径输入）、porcelain v2 状态解析、Changes 分组、文件级 Stage/Unstage、并排只读 `CodeDiff` 和 Commit。比较两端分别取自 HEAD/index/工作区；二进制与超过 512 KiB 的文件回退到 Git patch。Windows 使用 `ReadDirectoryChangesW`、Linux 使用 inotify 监听工作树和 Git 目录，变更按 300 ms 合并刷新，切换仓库时取消旧监听；macOS 暂用手动刷新。写操作在仓库对象内串行，成功或失败后重新读取状态；提交失败保留输入，可展开原始错误。输出捕获有大小上限（一般命令 stdout 64 MiB、Diff 8 MiB、stderr 1 MiB）。Linux/macOS 子进程使用独立进程组，取消时终止整组；Windows 目前只终止直接 Git 进程，Job Object 仍待实现。临时仓库及 Driver 测试覆盖浏览打开、Diff、暂存、取消暂存、首次提交、外部编辑自动刷新、仓库切换、失败保留草稿、窄窗口布局；Linux inotify 的嵌套/新增目录在 WSL 运行测试通过。此纵切目前未实现 macOS 文件监听、大 Diff 的流式处理、Windows 子进程树取消与命令日志；Linux 运行中新目录监听注册失败仍须给出提示。大型仓库性能与三平台无障碍验收也未完成，不能据此决定 D7。

M0 headless 与二进制测量见 [benchmarks/M0_WINDOWS.md](benchmarks/M0_WINDOWS.md)。

原始 Viva v0.2.4 headless 协议仅有 `hits` / `layout`；当前 `D:/viva` 工作树已支持 `tree` / `expect named` / `expect layout clean`，M1 headless 构建执行三条命令均通过。发布时需锁定与之对应的 Viva 版本。

## M2 进度

Log 已接入按需加载的只读列表：首次进入加载 100 条，滚动到底部自动请求下一批，也可点击“加载更多”；提交消息、作者、路径可由 Git 查询筛选并按筛选结果分页。选择提交后异步读取完整消息和文件路径，可选择文件查看与父提交对比的只读并排 Diff，并复制 SHA；合并提交与第一父提交比较。主窗口参照 Rebased README 截图调整为深色紧凑工具栏、Lucide 图标导航轨道、常驻 Changes/多行 Commit 面板和中央 Log/右侧文件与提交详情；提交图按父 SHA 绘制可见行的轨道和节点，表格只声明可见行。真实仓库测试覆盖空历史、105 条提交的分页边界和滚动加载、消息/作者/路径筛选及新增、修改、删除和合并提交的文件 Diff；Driver 测试覆盖多行提交、筛选、选择、复制 SHA、文件 Diff、窄窗口布局及 10,000 条内存提交的可见行数量。提交图仅支持前六条可见轨道，复杂分支折叠、引用标签、日期/分支筛选与大仓库滚动基准尚未实现，M2 未完成。

## M3 进度

Branches 页面已通过 `for-each-ref` 展示本地、远程分支及本地分支的 upstream；可用完整引用筛选 Log、新建本地分支和切换已有本地分支。切换使用普通 `git switch`，Git 检测到会覆盖工作区修改时拒绝操作，界面保留原始错误；成功后更新当前分支、状态、引用和日志。真实仓库测试覆盖合法/非法/重复分支名、正常切换及脏工作区拒绝，Driver 覆盖创建、切换和窄窗口布局。重命名、删除、upstream 管理、远程分支 checkout 及覆盖时的 Cancel/Stash/Force 流程未实现，M3 未完成。

## Rebased 对照（2026-11-06）

参考来源：`https://github.com/DetachHead/rebased` 的 README 和 `screenshot.png`。Rebased 基于 IntelliJ Community，包含其 Git 集成和部分平台能力。当前工程已具备打开仓库、文件级暂存/取消暂存、多行提交、文件 Diff、初步提交图与按消息/作者/路径搜索 Log，以及本地分支列表、新建和安全切换；主工作区按参考截图重排。对照截图仍缺完整的引用标签、右侧目录树与可调整分栏；功能还缺完整分支管理、远程同步、stash、merge/rebase/cherry-pick/revert/reset、冲突解决、hunk 级操作、设置与键盘工作流。它们对应 M2-M10，不能视为已实现；目前无计划复制 IntelliJ 的编辑器和插件生态，与产品规格 §2 的明确范围一致。
