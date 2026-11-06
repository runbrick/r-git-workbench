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

已实现从 Git unified diff 解析独立 hunk，支持多 hunk 选择后暂存/取消暂存，并校验补丁指纹。未暂存 hunk 可预检、确认后局部还原工作区，执行时重新校验状态和 Diff；真实仓库测试覆盖多 hunk 暂存、局部还原与过期计划拒绝。提交区支持 Amend、最近 20 条提交消息选择，以及 Commit and Push（提交成功后进入推送预览确认，失败保留草稿）。Discard 先展示目标文件并确认，计划绑定状态和工作区内容；tracked 文件只还原工作区，untracked 文件经确认后删除，index 保留。Changes 支持目录缩进树视图及折叠、扁平分组视图和 ignored 文件只读展示。仓库测试覆盖 ignored 路径查询；Driver 覆盖模式切换、Discard 确认、窄窗口布局。M5 仍有 parity 缺口：Diff 中用 hunk 按钮选择而不是逐行内联勾选；任意单行扩展、改动导航及 Diff 显示选项待实现。

## Rebased parity 任务基线（2026-11-06）

用户已确认目标为“除替换 logo 外，功能和样式必须与 Rebased 一模一样”，实现架构固定为 Go + Viva。基线仓库：`D:\privateProject\rebased`，`master` SHA `245042348669114adb8e24e5b0fc0ce73763f3c4`；基线证据为该提交的 README、`screenshot.png`、Git 插件动作注册与默认随附插件清单。更新上游或截图时须更新 SHA 和差异清单。

Rebased 是 IntelliJ Community 派生产品，Git 插件本身依赖 IntelliJ 平台；不能将复制其源码作为 Go/Viva 实现方案。应复现用户可见能力和交互语义，平台服务由本项目架构替代。Go/Viva、系统 Git CLI、单仓库写队列、破坏性操作预检确认执行、异步收件箱回传和 idle 无轮询仍是硬约束。

当前代码基线已有：仓库打开、Changes 基本分组、文件级 stage/unstage、只读并排 Diff、提交、多行消息、可分页 Log 和初步提交图、消息/作者/路径/日期/引用筛选、基础本地/远端分支列表、创建/切换、安全 checkout、Fetch/Pull/Push 和 force-with-lease 计划。M2、M3、M4 仍有各自章节标明的未完成项；已实现不等于与 Rebased 行为/UI parity。

### 功能与界面任务

| 任务 | 当前不足与目标 | 验收 |
|---|---|---|
| P1 基线清单 | 已完成，见 [REBASED_ACTION_INVENTORY.md](REBASED_ACTION_INVENTORY.md)：`git4idea` 约115个动作逐项登记（已实现≈18、部分实现≈12、未实现≈85），`git-features-trainer`/`git-modal-commit`/GitHub/GitLab/Terminal/TextMate 已登记并给出排除/延后建议 | 每项有源码/界面证据、优先级、实现状态、Go/Viva 承载方案和测试；未实现项不从清单删除 |
| P2 主窗口视觉 | 已完成差异盘点，见 [REBASED_UI_PARITY_GAPS.md](REBASED_UI_PARITY_GAPS.md)：逐区域对照基线截图，发现3项P0功能性缺陷（Copy SHA按钮无动作、Amend开关无UI入口、文件状态字母徽标缺失）及Log筛选条/文件树结构/顶栏信息架构等P1视觉结构性差异；实现尚未开始 | 以固定尺寸基线截图制作标注图和差异截图；分区、顺序、相对尺寸、行高、留白、边框、颜色层级及选中态逐项验收；只替换品牌资产 |
| P3 分栏和工具窗 | 已完成差异盘点，见 [REBASED_SPLITTER_PERSISTENCE_GAPS.md](REBASED_SPLITTER_PERSISTENCE_GAPS.md)：确认整个应用目前无任何跨会话持久化机制（`internal/settings` 包未建立，`ARCHITECTURE.md` §7 设计未落地）；三个可调分栏均不记忆尺寸/不按仓库隔离；Log两侧栏用宽度硬阈值隐藏而非用户可控折叠；900-1000px 过渡区间无测试覆盖 | 面板可拖动/隐藏/恢复，状态按仓库持久化；窄尺寸下无重叠、裁切，headless `expect layout clean` 通过 |
| P4 Commit 与 Shelf | 已完成差异盘点，见 [REBASED_COMMIT_SHELF_GAPS.md](REBASED_COMMIT_SHELF_GAPS.md)：确认 Amend 后端已支持但 UI 入口零引用（`amendID` 声明后从未被绘制代码使用）；Identity 检测两端均不存在；Commit and Push 组合流程无端到端测试；Shelf tab 路由已正确指向 Stash，只需改显示文案 | Driver 覆盖切换模式、选项和焦点；临时 Git 仓库覆盖 hooks、identity、失败恢复及提交范围 |
| P5 Changes 与 Diff | 已完成差异盘点，见 [REBASED_CHANGES_DIFF_GAPS.md](REBASED_CHANGES_DIFF_GAPS.md)：冲突置顶在树视图下失效（扁平视图正确）；分组级批量操作两端均不存在；hunk勾选UI独立于CodeDiff组件实现，Viva框架修复不会自动带来行级能力；二进制文件展示的是git diff原始文本而非结构化类型/大小信息，且该渲染路径无任何Diff工具栏能力；外部打开/路径复制/差异导航/三版本同屏均为彻底空白 | Git patch/index 结果与系统 Git 对照；大/二进制文件可用；有损动作显示影响并二次确认 |
| P6 Branch/Refs | 已完成差异盘点，见 [REBASED_BRANCH_REFS_GAPS.md](REBASED_BRANCH_REFS_GAPS.md)：远程分支checkout的Plan/Execute两段式安全检查实现质量高，但本地分支切换`SwitchBranch`疑似无对等的应用层预检，依赖Git原生拒绝；删除未合并分支无"未合并提交"预检提示；Tag后端零函数，导航项落入错误占位文案；分支列表缺搜索/Recent分组；Worktree范围规格从未提及，需用户决策 | 每个动作来自基线动作表；重命名、未合并删除、覆盖和过期状态有仓库测试与确认 UI 测试 |
| P7 Log | 已完成差异盘点，见 [REBASED_LOG_GAPS.md](REBASED_LOG_GAPS.md)：commit graph 轨道数硬编码6条且溢出时会把不相关分支误画成同一条线（正确性bug非性能问题）；作者/提交者数据源只采集author，committer完全未采集；引用标签是纯文字拼接无图标无颜色区分，且前缀硬编码英文未走i18n；Log表格不支持多选，比较任意两提交无从实现；"10000条提交"现有测试只验证内存虚拟化，大仓库真实加载/图形计算性能从未测试 | 构造复杂历史及大仓库测试；图随分页连续，所有危险操作计划绑定状态并验证（该项依赖P8范围功能，P7阶段暂不适用） |
| P8 历史编辑与冲突 | 已完成差异盘点（含技术路线评估），见 [REBASED_HISTORY_CONFLICT_GAPS.md](REBASED_HISTORY_CONFLICT_GAPS.md)：阻塞性地基缺失——仓库完全无法检测是否处于merge/rebase/cherry-pick进行中（不读`.git/MERGE_HEAD`等），几乎所有P8功能依赖它；三方合并视图建议首发走路线A（AcceptOurs/Theirs/Both简化操作，不等Viva修复#132）；交互式Rebase拖拽重排是全新发现的未提issue的Viva缺口，建议首发用上移/下移按钮替代；现有Plan/Execute安全模式可部分复用于Reset/Merge/Rebase | 真实临时仓库覆盖成功、冲突、取消、hook 失败与部分操作后的恢复；三方 UI 在 Viva 能力不足时按架构评估自绘/API |
| P9 Stash/Shelf | 已完成差异盘点（含从零设计评估），见 [REBASED_STASH_GAPS.md](REBASED_STASH_GAPS.md)：后端零实现（唯一stash调用是远程checkout流程内部借用，非独立功能）；`ShowStash`与`ShowTags`共享同一处占位文案bug（P6同一代码位置）；关键设计点——stash apply/pop冲突不产生`.git/MERGE_HEAD`等持久化标记，P8的仓库级操作状态检测地基无法覆盖，需P9自行维护会话级"当前冲突来自哪个stash"状态；建议面板布局裁剪复用paintLog的列表+详情结构，数据模型参考ParseLog模式但用新结构体 | save/apply/pop/drop/branch/show diff/冲突恢复以及面板持久化均有测试 |
| P10 设置与动作 | 已完成差异盘点，见 [REBASED_SETTINGS_ACTIONS_GAPS.md](REBASED_SETTINGS_ACTIONS_GAPS.md)：动作注册表仅6个导航项，无上下文/选中项字段，`OnShortcut`从未调用，`OnKey`硬编码绕开动作系统；待动作化操作约百位量级，`ActionID uint8`裸枚举值不能直接做持久化键（需稳定字符串标识符层）；设置项分三类（已有运行时字段只差持久化/全新两端都要做/框架限制不适用）；命令日志零记录但`gitexec.Runner.Run`是现成扩展点；Find Action命令面板不应在动作注册表内容充实前提前做 | 设置持久化、键盘导航、动作查找、快捷键冲突、国际化和无障碍测试通过 |
| P11 Hosting 集成 | 已完成差异盘点（含决策选项矩阵），见 [REBASED_HOSTING_GAPS.md](REBASED_HOSTING_GAPS.md)："无账号时核心Git流程完整"验收标准已满足（远程操作纯基于remote URL字符串，无平台耦合，有代码证据）；凭据处理零自建逻辑，符合规格；核心决策待用户裁定——是否引入GitHub/GitLab账号体系；折中方案：不登录也可用平台固定ref约定（`refs/pull/<N>/head`等）fetch已知PR/MR对应代码，工作量小且不引入新凭据逻辑 | 按基线流程逐项实现等价入口；凭据交由安全凭据机制；无账号时核心 Git 流程完整（已验证通过） |
| P12 Terminal/TextMate | 已完成差异盘点，见 [REBASED_TERMINAL_TEXTMATE_GAPS.md](REBASED_TERMINAL_TEXTMATE_GAPS.md)："在外部终端打开"/"在文件管理器打开"（后者是P1-P11全系列遗漏项，本次补上）均零实现，跨平台路径明确，不触发依赖测量义务；建议先查`D:/viva`的`app`包源码确认有无等价API再自行实现；关键独立新发现——Viva `CodeDiffSnapshot`已提供`Language`/`Highlight`可插拔语法高亮回调且与TextMate决策无关，当前两处调用点完全未设置，是否启用需用户界定"不做语法高亮"的排除边界 | 用户可见能力与决定均在清单；新增发布依赖测量剥离体积、启动时间和空闲内存 |

P1 是所有 parity 工作的盘点门槛。P2-P4 是贯穿 M2-M13 的 UI parity 任务，在 M11 汇总验收；P5-P10 按现有 Git 能力阶段落入 M5-M10，并以 parity 标准重新打开验收；P11-P12 归入 M13。任务完成条件还包括 `gofmt -l .`、`go vet ./...`、`go test ./... -count=1`，UI/input 改动附 Driver/headless 验证，不打开真实窗口，除非用户同意。

## 实施顺序建议（P1-P12 盘点完成后，2026-11-06）

P1-P12 十二份差异盘点文档已全部产出（见上表链接）。本节基于这些文档的发现，给出下一步开工的顺序建议，核心原则：地基先于功能、决策先于实现、技术确定项先于不确定项。

### 第零阶段：三块地基（互相独立，可并行开工，是后续多数工作的前提）

1. **新建 `internal/settings` 包**（P3）——落地 `docs/ARCHITECTURE.md` §7 已设计的方案（用户配置 `settings.json` + 每仓库 `repos/<路径哈希>.json`）。没有它，分栏尺寸记忆、Log 筛选记忆、Pull 策略持久化等都无法落地。
2. **仓库级操作状态检测**（P8）——新增读取 `.git/MERGE_HEAD`、`.git/rebase-merge/`、`.git/rebase-apply/`、`.git/CHERRY_PICK_HEAD`、`.git/REVERT_HEAD` 的函数。没有它，Merge/Rebase/Cherry-pick 的 Continue/Skip/Abort 和顶栏进行中状态指示都无法正确工作。
3. **重新设计动作注册表**（`internal/keymap`，P10）——`ActionID`/`Action` 结构体加"适用上下文"和"是否需要选中项"字段；持久化键改用稳定字符串标识符，不直接用 `ActionID uint8` 裸值（避免未来枚举重排导致存量配置错位）。

三项谁先谁后不重要，但不产出用户可见新功能，建议作为单独的地基投入排期，不要和功能开发混排。

### 第一阶段：独立的小缺陷修复（随时可做，不依赖地基，适合穿插在地基开发间隙完成）

- Amend 开关补 UI 入口（P4，`amendID` 已声明，后端已支持，只缺一个 Checkbox）
- Copy SHA 按钮补复制到剪贴板逻辑（P2，按钮已画出但无 `.Clicked()` 处理）
- 文件状态字母徽标（M/A/D/R/U/?）补齐（P2，同时是无障碍要求）
- `ShowTags`/`ShowStash` 共享的错误占位文案修复（P6、P9 发现是同一处代码 `shell.go:965-973`）
- `graph.go` 引用标签前缀硬编码英文改走 i18n（P7，确认是孤例不是系统性问题）
- 本地分支切换（`SwitchBranch`）补安全预检，对齐远程 checkout 已有的 Plan/Execute 高质量模式（P6）

### 第二阶段：需要先做产品范围决策的三个问题（技术工作量都不大，但决定了后面怎么排期，建议尽早定）

- GitHub/GitLab 账号体系要不要做（P11 给出选项矩阵；有一个工作量很小的折中方案——不登录，靠平台固定 ref 约定如 `refs/pull/<N>/head` 直接 fetch 已知 PR 对应代码）
- `CodeDiffSnapshot.Highlight` 语法高亮回调要不要启用（P12，技术上现成可用，纯粹是"不做语法高亮"这条规格排除范围怎么划）
- Worktree 管理要不要纳入范围（P6，规格从未提及，不是遗漏是未决）

### 第三阶段：向 Viva 框架提需求（在正式写相关功能前先确认，避免应用层绕过框架职责）

- 列表拖拽重排序是否支持（P8，交互式 Rebase 的 reorder 需要，此前没人问过，建议先查 `D:/viva` 源码或提 issue，若不支持用"上移/下移按钮"作为首发替代）
- 跨平台"打开外部程序/在文件管理器中显示"是否已有 API（P12，建议先查 `D:/viva` 的 `app` 包源码，若无则直接用标准库 `os/exec` 实现，不违反"先问框架"原则）

### 第四阶段：核心 Git 工作流功能开发（依赖第零阶段地基）

按 P8 文档的工作量分级：
1. 小工作量、技术确定：Reset(soft/mixed)、Merge/Rebase/CherryPick/Revert 的 Abort、冲突简化操作（Accept Ours/Theirs/Both，路线A，不等三方可编辑视图）、Merge/Rebase 发起对话框
2. 中等工作量、依赖地基：Continue/Skip、Reset(hard) 完整数据丢失预览、Cherry-pick/Revert 主流程
3. 大工作量、技术不确定（押后，先用简化方案上线不阻塞）：三方合并可编辑 Result 面板、交互式 Rebase 的拖拽排序本身（动作标记 pick/squash/fixup/reword/drop 不含拖拽部分可以先做）

### 第五阶段：Stash（P9，依赖地基，可与第四阶段并行）

注意 P9 的关键发现：Stash apply/pop 冲突不产生 `.git/MERGE_HEAD` 等持久化标记，第零阶段第2项的检测机制不能直接覆盖，需要新增一个会话级"当前冲突来自哪个 stash 操作"的临时状态。

### 第六阶段：动作注册表内容填满后才有意义的呈现层

命令面板（Find Action）、三套快捷键预设（IntelliJ/VSCode/Default）——这两者本质是"动作注册表的 UI 呈现层"，在第四、五阶段大部分操作完成动作化之前提前做，验证不出真实价值，建议排在后面。

### 第七阶段：收尾

Hosting 集成实现（若第二阶段决定要做）、"在外部终端打开"/"在文件管理器打开"命令、设置界面 UI 本身（依赖 `internal/settings` 和各设置项后端能力先就位）。
