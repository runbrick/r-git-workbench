# 里程碑与待决事项

每个里程碑结束时：规格 §10 对应验收项有自动化测试，`go vet` / `go test` 通过，性能预算中相关项已测量。

## 里程碑

| 编号 | 内容 | 规格章节 | 完成标志 |
|---|---|---|---|
| M0 | 工程骨架：`go.mod`、目录结构、`gitexec` 与版本检测、空主窗口（顶栏/导航/状态栏）、Driver 测试与 headless 入口 | §3、§4.1 末条 | 空窗口 headless `tree` 输出含全部导航项且 `expect named` 通过 |
| M1 | **垂直切片**：打开仓库、Changes 三分组、文件级 Stage/Unstage、文件 Diff、Commit；文件监听刷新 | §4.1、§4.2、§4.3 | 测量启动、刷新、二进制大小、无障碍，写入 `docs/benchmarks/` |
| M2 | Log：分批加载、图布局、筛选、提交详情与文件 Diff、Copy SHA | §4.4 | 基准仓库滚动无卡顿，图形增量连续 |
| M3 | 分支：选择器、Branches 视图、新建/切换/重命名/删除、upstream、Checkout 安全检查 | §4.5 | Checkout 覆盖文件时出现 Cancel/Stash/Force |
| M4 | 远端：Fetch/Pull/Push、进度与取消、错误分类、Force push with lease | §4.6、§7 | bare 仓库测试覆盖 §6 全部错误分类中的可构造项 |
| M5 | Changes 完整化：hunk 级暂存、Discard、Amend、提交并推送、提交消息历史、目录树模式 | §4.2 | 规格 §10 第 3、4 项 |
| M6 | Merge / Rebase / Cherry-pick / Revert / Reset 与预检确认 | §4.4、§4.7 | 有损操作全部经过预检 |
| M7 | 冲突解决：冲突清单、三方视图、Continue/Abort | §4.8 | 规格 §10 第 6 项 |
| M8 | Stash 全流程 | §4.9 | 规格 §10 第 8 项 |
| M9 | 快捷键：三套预设、Keymap 设置、冲突检测、命令面板 | §5 | 规格 §10 第 9 项 |
| M10 | 设置、i18n 补全、无障碍审查、诊断导出 | §6、§7、§8 | 规格 §10 剩余项与三平台人工验收 |
| M11 | Rebased parity 主窗口：工具栏、导航轨道、Commit/Shelf 面板、Log 编辑区、分支侧栏、右侧目录树/详情、可调分栏及窗口级设置；除品牌外按基线校准 | Rebased screenshot、README；§3 | 区域、相对比例、密度、控件、面板开关/拖动和窄窗口行为逐项通过截图检查表及 Driver/headless 测试 |
| M12 | Rebased Git 工作流 parity：Git 插件动作盘点和差距关闭，含 Staging Area、文件/块/行操作、分支操作菜单、Log 上下文操作、提交编辑及各类冲突/恢复流 | Git 插件动作注册、§4-§8 | 基线动作清单每项有实现/等价交互/明确差异；核心成功、取消、失败恢复均有临时仓库测试 |
| M13 | Rebased 默认随附集成 parity：GitHub/GitLab 工作流、Terminal、TextMate 基础高亮及默认设置盘点 | `BaseIdeaProperties.kt`、对应插件能力 | 每项按基线清单验收；Go/Viva 原生集成方案和依赖体积/启动/内存测量记录完成 |
| 后续 | 基线以外的平台集成及扩展功能 | §2 扩展 | 单独确定范围 |

快捷键和 i18n 的基础设施（动作注册表、字符串表）在 M0 建立，各里程碑新增动作与文案时直接登记，M9/M10 只做预设补全与设置界面。

## 待决事项

开发开始前或对应里程碑开始前需要确认：

| 编号 | 问题 | 当前倾向 | 何时决定 |
|---|---|---|---|
| D1 | Go module 路径 | M0 使用本地模块名 `r-git-workbench`；确定发布仓库 owner 后再改为正式地址 | 发布前 |
| D2 | 最低 Git 版本 | M0 暂定 2.34；较老版本显示警告，仅允许只读使用 | M1 验证 |
| D3 | 文件监听实现 | 自写三平台实现（`ReadDirectoryChangesW` / FSEvents / inotify），不引入 fsnotify：fsnotify 在 macOS 用 kqueue，每个文件占一个描述符，不适合大仓库；已在 [viva#133](https://github.com/vivaui/viva/issues/133) 询问是否由 Viva 提供 | M1 前 |
| D4 | Diff 视图：扩展 Viva `CodeDiff` 还是应用内自绘 | 保留 `CodeDiff` 并排只读比较；应用层从 Git patch 拆分 hunk 并提供独立 Stage/Unstage 操作，不自绘另一套 Diff 控件 | 已决定 |
| D5 | “Shift 两次”触发全局搜索 | 已提议 [viva#130](https://github.com/vivaui/viva/issues/130)；未支持前首发不提供此键位，规格偏差需记录 | M9 前 |
| D6 | 内置 askpass 对话框 | 首发不做，依赖 credential helper / SSH agent | M4 前 |
| D7 | Viva 性能预算与 parity 实现方案 | Go + Viva 已固定；M1 继续测量并对布局、Diff、无障碍和启动预算列出限制，不再作为更换技术栈的决策门 | M1 验收 |
| D8 | 打包与更新渠道 | `viva build` / `viva package`；自动更新暂不做 | M10 前 |

## M0 进度

已建立 Go 1.26 / Viva 工程、`gitexec` 子进程封装、Git 定位与版本能力检测、i18n 字符串表、导航动作注册表和可切换的空窗口。开发时通过 `go.mod` 的 `replace` 使用 `D:/viva` 工作树；发布前需移除本机路径并确定依赖版本。Git 检测发生在窗口创建前；后续仓库操作的异步结果已有 `workbench` 收件箱、`Invalidator`、`OnWake` 接口。

## M1 进度

已实现异步打开仓库（原生目录选择或路径输入）、porcelain v2 状态解析、Changes 分组、文件级 Stage/Unstage、并排只读 `CodeDiff` 和 Commit。比较两端分别取自 HEAD/index/工作区；二进制与超过 512 KiB 的文件回退到 Git patch。Windows 使用 `ReadDirectoryChangesW`、Linux 使用 inotify 监听工作树和 Git 目录，变更按 300 ms 合并刷新，切换仓库时取消旧监听；macOS 暂用手动刷新。写操作在仓库对象内串行，成功或失败后重新读取状态；提交失败保留输入，可展开原始错误。输出捕获有大小上限（一般命令 stdout 64 MiB、Diff 8 MiB、stderr 1 MiB）。Linux/macOS 子进程使用独立进程组，取消时终止整组；Windows 目前只终止直接 Git 进程，Job Object 仍待实现。临时仓库及 Driver 测试覆盖浏览打开、Diff、暂存、取消暂存、首次提交、外部编辑自动刷新、仓库切换、失败保留草稿、窄窗口布局；Linux inotify 的嵌套/新增目录在 WSL 运行测试通过。此纵切目前未实现 macOS 文件监听、大 Diff 的流式处理、Windows 子进程树取消与命令日志；Linux 运行中新目录监听注册失败仍须给出提示。大型仓库性能与三平台无障碍验收也未完成，不能据此决定 D7。

M0 headless 与二进制测量见 [benchmarks/M0_WINDOWS.md](benchmarks/M0_WINDOWS.md)。

原始 Viva v0.2.4 headless 协议仅有 `hits` / `layout`；当前 `D:/viva` 工作树已支持 `tree` / `expect named` / `expect layout clean`，M1 headless 构建执行三条命令均通过。发布时需锁定与之对应的 Viva 版本。

## M2 进度

Log 已接入按需加载的只读列表：首次进入加载 100 条，滚动到底部自动请求下一批，也可点击“加载更多”；提交消息、作者、路径、日期范围和分支/引用可由 Git 查询筛选并按筛选结果分页。选择提交后异步读取完整消息和文件路径，可选择文件查看与父提交对比的只读并排 Diff，并复制 SHA；合并提交与第一父提交比较。提交主题附带 HEAD、本地分支、远程分支和 Tag 的文本引用标签。主窗口参照 Rebased README 截图调整为深色紧凑工具栏、Lucide 图标导航轨道、常驻 Changes/多行 Commit 面板和中央 Log/右侧文件与提交详情；提交图按父 SHA 绘制可见行的轨道和节点，表格只声明可见行。真实仓库测试覆盖空历史、105 条提交的分页边界和滚动加载、消息/作者/路径/日期/引用筛选及新增、修改、删除和合并提交的文件 Diff；Driver 测试覆盖多行提交、筛选、选择、复制 SHA、文件 Diff、窄窗口布局及 10,000 条内存提交的可见行数量。提交图仅支持前六条可见轨道，复杂分支折叠和大仓库滚动基准尚未实现，M2 未完成。

## M3 进度

Branches 页面通过 `for-each-ref` 展示本地、远程分支及 upstream；支持新建、切换、重命名本地分支，设置/取消 upstream，以及从远程分支创建跟踪分支。删除本地分支采用预检、界面确认和执行时状态校验；Git 普通 `branch -d` 会拒绝未合并分支。远程 checkout 会预检并显示脏文件，提供 Cancel、先 Stash（包括未跟踪文件）再检出，以及二次确认后的 Force Checkout。计划绑定 HEAD、远程目标 SHA 和工作区状态，执行前重新校验。真实仓库测试覆盖正常流程、过期计划拒绝、取消保持原状、Stash 保留和 Force 覆盖。M3 功能实现完成，仍需按里程碑要求完成三平台验收。

## M4 进度

顶栏已接入 Fetch、Pull 和 Push，均在后台执行并在完成后刷新状态、引用与 Log；Git 的 CR/LF 进度消息逐行投递至窗口，远端操作可取消。Branches 页提供上游默认目标与多远端选择；Pull 支持 Merge、Rebase、Fast-forward only 三种策略，默认 Merge，选择仅保留在当前窗口会话中；当前分支相对已 fetch upstream 的 ahead/behind 计数从 porcelain v2 读取并在分支选择器中显示。常规 Push 先预览所选目标远端/分支、两侧 SHA 与待推提交，确认后重新校验计划再执行。Push/Pull 使用当前分支 upstream 的分支名映射所选远端；Fetch 可指定所选远端。无 upstream、网络、认证和 hook 错误保留原始 Git 输出。Force push with lease 使用预检计划、展示目标远端/分支和本地 SHA/预期远端 SHA、二次确认，并在执行前重新校验后运行显式 `--force-with-lease`。真实 bare 仓库测试覆盖普通同步、rebase、ff-only 拒绝分叉、ahead/behind 计数、多远端待推提交预览及过期计划/lease 拒绝。Pull 策略持久化及错误分类 UI 尚未实现，M4 未完成。首发继续依赖系统 credential helper/SSH agent，不提供内置 askpass。

## M5 进度

已实现从 Git unified diff 解析独立 hunk，并用 `git apply --cached` 及反向应用完成 hunk 级暂存/取消暂存；操作重新读取当前 diff 并在 hunk 已变化时拒绝执行。真实仓库测试验证 index 与工作区分别只包含预期改动。提交区已支持 Amend、最近 20 条提交消息选择，以及 Commit and Push；后者提交成功后进入已有推送目标/提交预览确认流程，提交失败保留草稿。Driver 测试覆盖提交面板在窄窗口布局下无重叠。M5 仍未完成：Discard 的预检/确认流程与目录树/扁平模式切换尚未实现；hunk UI 当前通过独立按钮操作，未提供 Diff 行内勾选及任意单行扩展交互。

## Rebased parity 任务基线（2026-11-06）

用户已确认目标为“除替换 logo 外，功能和样式必须与 Rebased 一模一样”，实现架构固定为 Go + Viva。基线仓库：`D:\privateProject\rebased`，`master` SHA `245042348669114adb8e24e5b0fc0ce73763f3c4`；基线证据为该提交的 README、`screenshot.png`、Git 插件动作注册与默认随附插件清单。更新上游或截图时须更新 SHA 和差异清单。

Rebased 是 IntelliJ Community 派生产品，Git 插件本身依赖 IntelliJ 平台；不能将复制其源码作为 Go/Viva 实现方案。应复现用户可见能力和交互语义，平台服务由本项目架构替代。Go/Viva、系统 Git CLI、单仓库写队列、破坏性操作预检确认执行、异步收件箱回传和 idle 无轮询仍是硬约束。

当前代码基线已有：仓库打开、Changes 基本分组、文件级 stage/unstage、只读并排 Diff、提交、多行消息、可分页 Log 和初步提交图、消息/作者/路径/日期/引用筛选、基础本地/远端分支列表、创建/切换、安全 checkout、Fetch/Pull/Push 和 force-with-lease 计划。M2、M3、M4 仍有各自章节标明的未完成项；已实现不等于与 Rebased 行为/UI parity。

### 功能与界面任务

| 任务 | 当前不足与目标 | 验收 |
|---|---|---|
| P1 基线清单 | 将 `git4idea`、`git-features-trainer`、`git-modal-commit`、GitHub、GitLab、Terminal、TextMate 的用户可见入口和默认配置逐项登记；区分 Git 客户端能力与 IntelliJ IDE 能力 | 每项有源码/界面证据、优先级、实现状态、Go/Viva 承载方案和测试；未实现项不从清单删除 |
| P2 主窗口视觉 | 当前界面只有大体区域借鉴；核对顶部品牌区、仓库/分支组合框、同步按钮、工具窗 rail、Commit/Shelf 标签、Changes 工具栏、中央 Log 筛选条和表格、Branches 侧栏、右侧文件树/提交详情、底部状态栏 | 以固定尺寸基线截图制作标注图和差异截图；分区、顺序、相对尺寸、行高、留白、边框、颜色层级及选中态逐项验收；只替换品牌资产 |
| P3 分栏和工具窗 | 当前面板组合、显示/隐藏和尺寸记忆未等同 Rebased；缺少完整命名视图和工具窗行为 | 面板可拖动/隐藏/恢复，状态按仓库持久化；窄尺寸下无重叠、裁切，headless `expect layout clean` 通过 |
| P4 Commit 与 Shelf | 复刻 Commit/Shelf 模式切换、暂存区浏览、提交选项、Amend、检查、Commit、Commit and Push、失败后草稿/选择保留 | Driver 覆盖切换模式、选项和焦点；临时 Git 仓库覆盖 hooks、identity、失败恢复及提交范围 |
| P5 Changes 与 Diff | 完成冲突置顶、树/扁平、分组动作、ignored 展示选项、行/hunk 暂存、局部撤销、三版本比较、外部打开、路径复制、差异导航和 Diff 显示选项 | Git patch/index 结果与系统 Git 对照；大/二进制文件可用；有损动作显示影响并二次确认 |
| P6 Branch/Refs | 完成分支弹出树与列表、搜索、Recent、favorite/group/filter、远端选择、checkout/create/rename/delete/upstream、tag、从 Log 操作分支，以及 worktree 间状态提示 | 每个动作来自基线动作表；重命名、未合并删除、覆盖和过期状态有仓库测试与确认 UI 测试 |
| P7 Log | 完成分支/ref 侧栏和筛选语义、引用装饰、作者/提交者/日期/路径列、图形布局、文件树、详情、Diff、比较、复制、检出/建分支/Cherry-pick/Revert/Reset/Push to commit 等上下文操作 | 构造复杂历史及大仓库测试；图随分页连续，所有危险操作计划绑定状态并验证 |
| P8 历史编辑与冲突 | 完成交互式 Rebase（reorder/squash/fixup/reword/drop）、cherry-pick 序列、revert、reset/uncommit、merge/rebase 冲突、三方视图、接受 ours/theirs/both、手改、continue/skip/abort | 真实临时仓库覆盖成功、冲突、取消、hook 失败与部分操作后的恢复；三方 UI 在 Viva 能力不足时按架构评估自绘/API |
| P9 Stash/Shelf | 明确 IntelliJ Shelf 与 Git Stash 的语义差别；实现 Rebased 用户可见的 Git stash 操作和 Shelf 面板等价行为，不将二者错误合并为同一 Git 对象 | save/apply/pop/drop/branch/show diff/冲突恢复以及面板持久化均有测试 |
| P10 设置与动作 | 完成默认设置盘点、Git 路径/配置/identity/remotes/pull 策略、提交检查、keymap/Find Action 等产品可承载的操作入口；呈现命令结果和脱敏诊断 | 设置持久化、键盘导航、动作查找、快捷键冲突、国际化和无障碍测试通过 |
| P11 Hosting 集成 | 盘点 GitHub/GitLab 插件包含的 clone/auth/status、PR/MR 浏览/创建/checkout/review 用户流程；区分 Git 核心不依赖账号与平台集成可选 | 按基线流程逐项实现等价入口；凭据交由安全凭据机制；无账号时核心 Git 流程完整 |
| P12 Terminal/TextMate | 评估 Rebased 默认 Terminal 和 TextMate 功能；Viva 若无通用终端组件，先登记能力差距和可复用 Viva API，不能在 Git 绘制层私造终端模拟器 | 用户可见能力与决定均在清单；新增发布依赖测量剥离体积、启动时间和空闲内存 |

P1 是所有 parity 工作的盘点门槛。P2-P4 是贯穿 M2-M13 的 UI parity 任务，在 M11 汇总验收；P5-P10 按现有 Git 能力阶段落入 M5-M10，并以 parity 标准重新打开验收；P11-P12 归入 M13。任务完成条件还包括 `gofmt -l .`、`go vet ./...`、`go test ./... -count=1`，UI/input 改动附 Driver/headless 验证，不打开真实窗口，除非用户同意。
