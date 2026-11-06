# Rebased 基线动作与能力盘点

对应 `docs/ROADMAP.md` 的 P1 任务。基线来源：`D:\privateProject\rebased`（IntelliJ Community fork），核心插件目录 `plugins/git4idea`、`plugins/git-features-trainer`、`plugins/git-modal-commit`、`plugins/github`、`plugins/gitlab`、`plugins/terminal`、`plugins/textmate`。基线 SHA 见 `docs/ROADMAP.md` "Rebased parity 任务基线"一节，升级基线需同步更新本文件。

状态列定义：**已实现**（cetacean 当前代码已覆盖且行为等价）／**部分实现**（功能存在但缺子能力）／**未实现**（代码中不存在）。依据 `internal/keymap/actions.go`（仅 6 个导航动作，无具体 Git 操作注册）、`internal/ui/shell/shell.go`、`internal/repo/repo.go`、`internal/workbench/session.go`，以及 `docs/ROADMAP.md` M0-M5 进度记录。

优先级：**P0**（核心 Git 工作流，首发必需）／**P1**（常用，M6-M10 范围内应完成）／**P2**（边缘，可延后或排除）。

未实现项不从本清单删除；新增实现后更新对应行的"当前状态"列，不要整行移除。

---

## git4idea（核心 Git 插件）

证据来源：`backend/resources/intellij.vcs.git.backend.xml`（1116行，约105个 action/group）、`frontend/resources/intellij.vcs.git.frontend.xml`（49行，4个）、`shared/resources/intellij.vcs.git.shared.xml`（77行，6个）。总计约115个顶层 `<action id="...">` 条目（不含仅用于分组/占位的 `<group>` 和 `<reference>`）。

### 仓库初始化与克隆

| 动作 ID | 用户可见名称/功能 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|---|
| `Git.Init` | 初始化仓库（在未纳入版本控制的目录执行 `git init`） | 未实现 | P1 | 原生实现：打开非仓库目录时提供"初始化仓库"按钮，调用 `git init` | 临时空目录仓库测试 |
| `Git.Clone` | 克隆仓库（URL + 目标目录 + 可选分支） | 未实现 | P0 | 原生实现：规格 §4.1 已定义克隆表单 | 临时裸仓库作为克隆源测试 |
| `Git.Configure.Remotes` | 配置远端（增删改 remote URL） | 未实现 | P1 | 原生实现：设置界面的 remotes 管理 | 临时仓库测试增删 remote |
| `Git.Unshallow` | 将浅克隆转换为完整仓库 | 未实现 | P2 | 原生实现，优先级低 | 浅克隆临时仓库测试 |

### Stage / 暂存区面板（独立 Staging Area 模式，`Git.Stage.*`）

IntelliJ 有两种提交模式：默认"非模态提交"（Changes 面板直接勾选文件提交）和"Staging Area"模式（类似传统 `git add` 两步式暂存区浏览面板，通过 `Git.Stage.Enable`/`Git.Show.Stage` 开启）。cetacean 当前的文件级/hunk级 stage/unstage（M1/M5 已完成）对应的是非模态提交模式下的操作语义，不是这个独立 Staging Area 面板。

| 动作 ID | 用户可见名称/功能 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|---|
| `Git.Show.Stage` | 显示独立 Staging Area 面板 | 未实现 | P2 | 等价交互：cetacean 的 Changes 三分组已覆盖同等心智模型，不复刻这个可切换的第二套面板，作为差异记录并建议排除 | — |
| `Git.Stage.Enable`/`Git.Stage.Disable` | 启用/禁用 Staging Area 模式 | 未实现 | P2 | 跟随上一条排除决定 | — |
| `Git.Commit.Stage` | 在 Staging Area 模式下提交 | 未实现 | P2 | 同上，排除 | — |
| `Git.Stage.Add` | 暂存选中文件（含内容） | 已实现（非模态等价） | — | 已有 Stage 动作覆盖 | 已有测试 |
| `Git.Stage.Add.No.Content` | 仅跟踪文件但不暂存内容（`git add -N`） | 未实现 | P2 | 原生实现：intent-to-add，使用场景少 | 临时仓库测试新文件 add -N 后 diff 行为 |
| `Git.Stage.Add.All`/`Git.Stage.Add.Tracked` | 暂存全部 / 暂存全部已跟踪文件 | 未实现 | P1 | 原生实现：工具栏"全部暂存"按钮 | 临时仓库多文件测试 |
| `Git.Stage.Reset`/`Git.Stage.Revert` | 取消暂存 / 还原（Stage 面板语境下的 unstage 和 discard） | 已实现（非模态等价） | — | 已有 Unstage/Discard 覆盖 | 已有测试 |
| `Git.Stage.AcceptTheirs`/`Git.Stage.AcceptYours` | 冲突文件在 Stage 面板直接接受 theirs/yours | 未实现 | P0（冲突解决必需） | 原生实现，属于 M7 冲突解决范围 | 真实合并冲突仓库测试 |
| `Git.Stage.Merge` | 冲突文件打开合并视图 | 未实现 | P0 | 同上，M7 范围 | 同上 |
| `Git.Stage.ThreeSideDiff` | 三方 Diff 查看（不应用，只看） | 未实现 | P1 | 原生实现，复用 M7 的三方视图能力 | 同上 |
| `Git.Stage.ToggleIgnored` | 切换显示/隐藏 ignored 文件 | 已实现 | — | M5 已做"ignored 文件只读展示" | 已有测试 |
| `Git.Stage.ToggleCommitAll` | 无暂存内容时自动提交全部改动的开关 | 部分实现 | P1 | 规格 §4.2 已提到相关设置项，需要补成可配置项 | Driver 测试设置开关 + 提交流程 |
| `Git.Stage.Show.Staged`/`Git.Stage.Show.Local` | 查看暂存版本/本地版本单独内容 | 部分实现 | P2 | Diff 面板已有并排对比，独立"仅看一侧"入口可选 | — |
| `Git.Stage.Compare.Local.Staged`/`Compare.Staged.Local`/`Compare.Staged.Head`/`Compare.Three.Versions` | HEAD/暂存区/工作区两两对比及三版本对比 | 部分实现 | P1 | M1 已有 HEAD↔工作区、HEAD↔暂存区对比；缺三版本同屏对比 | 临时仓库三状态差异测试 |
| `Git.Stage.Stash.Files` | 从 Stage 面板直接 stash 选中文件 | 未实现 | P2 | 待 M8 Stash 完成后补充入口 | — |

### Commit / 提交动作

| 动作 ID | 用户可见名称/功能 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|---|
| `CheckinProject`（引用） | 提交（标准非模态提交） | 已实现 | — | M1 已完成 | 已有测试 |
| `Git.Commit.And.Push.Executor` | Commit and Push（Ctrl+Alt+K） | 已实现 | — | M5 已完成 | 已有测试 |
| `Git.Uncommit` | 撤销上一次提交但保留改动（类似 `reset --soft HEAD~1`，面向 Log） | 未实现 | P1 | 原生实现，M6 Reset 范围 | 临时仓库提交后 uncommit 测试 |
| `Git.Reword.Commit` | 修改某次提交的提交消息（非 HEAD 也可，走 interactive rebase） | 未实现 | P1 | 原生实现，M6/M8 范围（依赖交互式 rebase 基础设施） | 临时仓库多提交 reword 测试 |
| `Git.PushUpToCommit` | 推送到某个历史提交为止（不含之后的提交） | 未实现 | P2 | 原生实现，相对少用 | 临时裸仓库测试部分推送 |
| `Git.Add`（Schedule for Addition） | 添加未跟踪文件到版本控制 | 已实现 | — | 文件级 stage 已覆盖 untracked → staged | 已有测试 |
| `Git.CompareWithBranch` | 文件与指定分支对比 | 未实现 | P1 | 原生实现：规格 §4.3 的分支对比部分 | 临时仓库跨分支文件对比测试 |

### Log 右键菜单 / 历史编辑

此分组是 M6（Merge/Rebase/Cherry-pick/Revert/Reset）和 M8（交互式历史编辑）的主要落点，当前完全未实现。

| 动作 ID | 用户可见名称/功能 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|---|
| `Git.Reset.In.Log` | Reset Current Branch to Here（`--soft`/`--mixed`/`--hard` 三选） | 未实现 | P0 | 原生实现，规格 §4.4 已明确要求显示三种模式效果并对 hard reset 二次确认 | 临时仓库三种 reset 模式测试 + 确认 UI 测试 |
| `Git.Revert.In.Log` | Revert Commit（生成反向提交） | 未实现 | P0 | 原生实现 | 临时仓库 revert 成功/冲突测试 |
| `Git.Fixup.To.Commit` | Fixup 到指定提交（squash 不保留消息） | 未实现 | P1 | 原生实现，依赖交互式 rebase 引擎 | 临时仓库 fixup 测试 |
| `Git.Squash.Into.Commit` | Squash 到指定提交（保留消息合并） | 未实现 | P1 | 同上 | 同上 |
| `Git.Squash.Commits` | 多选提交合并为一个 | 未实现 | P1 | 同上 | 多提交选择 squash 测试 |
| `Git.Drop.Commits` | 删除选中提交 | 未实现 | P1 | 同上 | 删除中间提交测试（含冲突情形） |
| `Git.Interactive.Rebase` | 打开交互式 Rebase 编辑器（reorder/squash/fixup/reword/drop） | 未实现 | P0 | 原生实现：M8 核心，需评估 Viva 承载序列编辑 UI 的能力 | 真实仓库构造多提交历史，覆盖 reorder/成功/冲突/abort |
| `Git.Drop.Selected.Changes` | 从某次提交中丢弃选中的文件改动（变更浏览器内） | 未实现 | P2 | 原生实现，依赖交互式 rebase 基础设施 | 同上场景的子集测试 |
| `Git.InMemory.Extract.Selected.Changes` | 将选中改动提取为独立提交（内存优化版） | 未实现 | P2 | 原生实现是性能优化路径，Go+Viva 下用普通实现代替即可 | 同上 |
| `Git.CreateNewBranch.FromCommit` | 从 Log 中选中的提交创建新分支 | 未实现 | P0 | 原生实现，规格 §4.5 已要求 | 临时仓库从历史提交建分支测试 |
| `Git.CreateNewTag` | 创建 Tag（可从 Log 选中提交） | 未实现 | P1 | 原生实现 | 临时仓库建 Tag 测试 |
| `GitLogBranchOperationsActionGroup` | Log 右键的分支操作子菜单（Checkout/New Branch/Cherry-pick 等） | 未实现 | P0 | 原生实现，是规格 §4.4"右键操作"列表的容器 | 随各子项测试覆盖 |
| `GitCheckoutActionGroup` | Checkout 选中的提交/分支/Tag | 未实现 | P0 | 原生实现，M3 已有分支级 checkout，缺"从 Log 任意提交 checkout" | 临时仓库从历史提交 checkout 测试（含 detached HEAD 提示） |
| `Git.CheckoutRevision` | Checkout 到指定版本 | 未实现 | P0 | 同上 | 同上 |
| `Git.BrowseRepoAtRevision` | 浏览某个历史版本的完整文件树 | 未实现 | P1 | 原生实现：只读文件树 + 该版本文件内容查看 | 临时仓库历史版本浏览测试 |
| `Git.AddCommitToRemoteBranch`（internal） | 内部调试功能 | 未实现 | P2（排除） | JetBrains 内部调试用，不进入产品范围 | — |
| `Git.Log.HighlightCherryPickedCommits` | 高亮已被 cherry-pick 过的提交 | 未实现 | P2 | 原生实现，视觉辅助功能 | 真实仓库 cherry-pick 后高亮验证 |
| `Git.Log.Show.Command`（internal） | 显示生成该 Log 查询的底层 git 命令（调试） | 未实现 | P2（可选） | 可作为诊断功能延伸（规格 §6 命令日志），否则排除 | — |

### Branches（分支管理）

| 动作 ID | 用户可见名称/功能 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|---|
| `Git.Branches` | 打开分支弹出树（Ctrl+Shift+\`） | 部分实现 | P0 | M3 已有 Branches 页面列表；缺专门快捷键触发的弹出树 | Driver 测试快捷键打开弹出层 |
| `Git.ShowBranches`（顶栏分支组合框） | 顶栏分支选择器 | 已实现 | — | 规格 §3 已实现 | 已有测试 |
| `Git.CreateNewBranch` | 新建分支 | 已实现 | — | M3 完成 | 已有测试 |
| `Git.Rename.Local.Branch` | 重命名本地分支 | 已实现 | — | M3 完成 | 已有测试 |
| `GitDeleteRefAction` | 删除分支/Tag/引用（通用删除） | 部分实现 | P0 | M3 已有删除本地分支预检确认；缺统一 `$Delete` 快捷键语义，且未覆盖删除 Tag、删除远程分支 | 临时仓库删除各类 ref 测试 |
| `GitRebaseBranchAction` | Rebase Current onto Selected | 未实现 | P0 | 原生实现，规格 §4.5 已列出 | 临时仓库 rebase 成功/冲突测试 |
| `GitMergeRefAction` | Merge into Current | 未实现 | P0 | 原生实现，规格 §4.5 已列出 | 临时仓库 merge 成功/冲突测试 |
| `GitCompareWithBranchAction`/`GitShowDiffWithRefAction` | Compare With（分支/Tag 对比当前） | 未实现 | P1 | 原生实现 | 临时仓库跨分支对比测试 |
| `GitUpdateSelectedBranchAction` | 更新选中分支（fast-forward 到远端） | 未实现 | P1 | 原生实现 | 临时仓库 ff-update 测试 |
| `GitPushBranchAction` | 推送选中分支 | 部分实现 | P1 | M4 已有当前分支 Push；缺从分支选择器直接推送非当前分支 | 临时裸仓库推送非当前分支测试 |
| `GitPullBranchAction$WithRebase`/`WithMerge` | 用 Rebase/Merge 策略拉取选中分支 | 部分实现 | P1 | M4 已有当前分支 Pull 策略选择；缺对非当前分支执行 | 同上场景扩展 |
| `GitPushTagsActionGroup`/`GitPushTagAction` | 推送 Tag（单个/全部） | 未实现 | P1 | 原生实现 | 临时裸仓库推送 Tag 测试 |
| `GitCheckoutFromInputAction` | 输入任意 ref 名直接 checkout | 未实现 | P2 | 原生实现，便捷入口 | 临时仓库输入 ref checkout 测试 |
| `GitCheckoutAsNewBranch` | Checkout 远程分支为新本地分支（改名） | 部分实现 | P1 | M3 已有创建跟踪分支；缺自定义新分支名 | 临时仓库自定义命名测试 |
| `GitCheckoutWithRebaseAction` | Checkout 并立即 Rebase | 未实现 | P2 | 原生实现，组合操作 | — |
| `Git.Checkout.Update` | Checkout 并更新（pull） | 未实现 | P2 | 原生实现 | — |
| `Git.Tag` | Tag 管理入口 | 未实现 | P1 | 原生实现：创建/删除/推送 Tag 的统一入口 | 临时仓库 Tag 增删测试 |
| `Git.Branches.Popup.Fetch` | 分支弹出层内直接 Fetch | 部分实现 | P2 | M4 已有顶栏 Fetch；缺弹出层内联触发 | Driver 测试 |
| `Git.Cleanup.Branches`（internal） | 清理过期分支引用（调试） | 未实现 | P2（排除） | internal 标记，调试功能 | — |
| `Git.FindMergedLocalBranches`（internal） | 查找已合并的本地分支 | 未实现 | P2（作为实现细节覆盖） | 规格 §4.5"删除已合并分支快速确认"的实现细节，不作独立入口 | 随删除分支测试覆盖 |
| 分支弹出树分组/筛选（按目录/仓库分组、显示最近使用、显示 Tag、按动作/仓库筛选、弹出层可调整大小） | — | 未实现 | P1 | 原生实现，规格 §4.5 的 Local/Remote/Recently Used 分组已部分覆盖，按目录分组和弹出层尺寸记忆缺失 | Driver 测试分组切换 + 窄窗口测试 |
| `GitTrackedBranchActionGroup` | 针对已设置 upstream 分支的子菜单 | 部分实现 | P1 | M3/M4 已有 upstream 设置和 ahead/behind；缺统一子菜单呈现 | — |

### Working Trees（Git worktree 管理）

git4idea 新增的 Worktree 管理面板，cetacean 完全未涉及，ROADMAP 里也没提到 worktree。

| 动作 ID | 用户可见名称/功能 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|---|
| `Git.Show.WorkingTrees` | 显示 Worktree 列表面板 | 未实现 | P2 | 原生实现：规格完全没提及 worktree，需用户确认是否纳入范围 | — |
| `Git.CreateNewWorkingTree` | 新建 Worktree | 未实现 | P2 | 同上，待范围决策 | 临时仓库建 worktree 测试 |
| `Git.WorkingTrees.Remove`/`Open`/`Refresh`/`Prune`/`ToggleLock` | 删除/打开/刷新/清理/锁定 Worktree | 未实现 | P2 | 同上 | 同上 |

### Merge / Rebase / Cherry-pick / Revert 主流程与进行中状态

| 动作 ID | 用户可见名称/功能 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|---|
| `Git.Merge` | 发起 Merge（选择来源分支） | 未实现 | P0 | 原生实现，规格 §4.7 已完整定义 | 临时仓库 merge 成功/冲突测试 |
| `Git.Rebase` | 发起 Rebase（选择目标） | 未实现 | P0 | 同上 | 临时仓库 rebase 成功/冲突测试 |
| `Git.Merge.Commit` | 冲突解决后完成 Merge 提交 | 未实现 | P0 | 原生实现 | 同上 |
| `Git.Merge.Abort` | 中止 Merge | 未实现 | P0 | 原生实现，规格 §4.7 已要求 | 同上 |
| `Git.Rebase.Continue`/`Git.Rebase.Skip`/`Git.Rebase.Abort` | Rebase 继续/跳过当前提交/中止 | 未实现 | P0 | 原生实现 | 临时仓库 rebase 冲突后三种后续操作测试 |
| `Git.CherryPick.Continue`/`Git.CherryPick.Abort` | Cherry-pick 继续/中止 | 未实现 | P0 | 原生实现 | 临时仓库 cherry-pick 冲突测试 |
| `Git.Revert.Abort` | Revert 中止 | 未实现 | P0 | 原生实现 | 临时仓库 revert 冲突中止测试 |
| `GitMergeRebaseWidget`/`GitMergeRebaseWidgetGroup` | 顶栏常驻"进行中操作"状态与快捷操作 | 未实现 | P1 | 原生实现：规格 §4.7"保留正在进行的 Git 操作状态"的 UI 承载 | Driver 测试进行中状态显示 |
| `Git.ResolveConflicts` | 打开冲突解决视图 | 未实现 | P0 | 原生实现，M7 核心 | 真实冲突仓库测试 |
| `Git.RevertResolved` | 冲突标记为已解决后撤销标记，重新编辑 | 未实现 | P1 | 原生实现，规格 §4.8 | 同上 |
| `Git.ChangesView.Merge`/`AcceptTheirs`/`AcceptYours` | Changes 视图里对冲突文件的直接操作 | 未实现 | P0 | 原生实现，是三方视图的快捷替代路径 | 同上 |

### Stash

| 动作 ID | 用户可见名称/功能 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|---|
| `Git.Stash` | 保存 Stash（可选消息、是否含未跟踪文件） | 未实现 | P0 | 原生实现，M8 核心，规格 §4.9 | 临时仓库 stash save 含/不含 untracked 测试 |
| `Git.Stash.Silently` | 静默 Stash（不弹确认框） | 未实现 | P1 | 原生实现，便捷变体 | 同上 |
| `Git.Unstash` | 恢复 Stash（走 Apply/Pop 选择对话框） | 未实现 | P0 | 原生实现 | 临时仓库 unstash 测试 |
| `Git.Stash.Apply` | Apply（保留 Stash 条目） | 未实现 | P0 | 原生实现，规格 §4.9 | 同上 |
| `Git.Stash.Pop` | Pop（成功后删除该 Stash） | 未实现 | P0 | 原生实现 | 同上 |
| `Git.Stash.UnstashAs` | 恢复为新分支 | 未实现 | P1 | 原生实现，规格 §4.9 | 同上 |
| `Git.Stash.Drop` | 删除单个 Stash（需确认） | 未实现 | P0 | 原生实现 | 同上 |
| `Git.Stash.Clear` | 清空全部 Stash（需确认） | 未实现 | P1 | 原生实现 | 同上 |
| `Git.Show.Stash` | 显示 Stash 列表面板 | 未实现 | P0 | 原生实现，对应规格 Stash 主视图 | Driver 测试面板显示 |
| `Git.Stash.Refresh` | 手动刷新 Stash 列表 | 未实现 | P2 | 需与"空闲不轮询"原则协调，优先走文件监听触发 | — |
| `Git.Stash.Toggle.Split.Preview` | 切换 Stash 列表与 Diff 预览的分栏/独立显示 | 未实现 | P2 | 原生实现，视觉布局细节 | — |
| Stash 冲突恢复流程 | 应用 Stash 产生冲突时进入冲突解决 | 未实现 | P0 | 原生实现，规格 §4.9"发生冲突时保留 Stash 并进入冲突流程" | 临时仓库构造 stash apply 冲突测试 |

### 冲突解决三方视图与导航

| 能力 | 用户可见名称/功能 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|---|
| 三方合并视图（由 `Git.Stage.Merge`/`Git.ChangesView.Merge` 触发） | Current/Incoming/Result 三栏视图 | 未实现 | P0 | D4 已决定 Diff 保留 CodeDiff 并排只读比较，但三方合并需要可编辑的 Result 面板，超出 D4 范围，需单独评估 Viva 能否承载三路合并编辑，或走自绘/外部编辑器方案 | 真实合并冲突仓库测试，覆盖 accept current/incoming/both/手工编辑 |
| Diff 中"下一个/上一个冲突"导航 | F7/Shift+F7 | 未实现 | P1 | 原生实现 | Driver 键盘导航测试 |

### 文件/路径相关

| 动作 ID | 用户可见名称/功能 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|---|
| `CopyPathFromRepositoryRootProvider` | 复制相对仓库根的路径 | 未实现 | P1 | 原生实现，规格 §4.3 工具栏"复制路径"的变体 | — |
| `Git.OpenExcludeFile`/Ignore 分组 | 打开 `.git/info/exclude`，添加文件到 `.gitignore`/exclude | 未实现 | P1 | 原生实现 | 临时仓库测试忽略文件写入 |
| GitIgnore/Exclude 文件语言支持（语法高亮等） | — | 未实现 | P2（排除） | 规格 §2 已排除"IDE 编辑器、语法高亮" | — |

### 其他

| 动作 ID | 用户可见名称/功能 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|---|
| `gpg.agent.configuration` | 配置 GPG Agent 路径用于提交签名 | 未实现 | P2 | 原生实现，规格未提及签名提交，需用户决策是否纳入 M10 设置范围 | — |

### 不属于本项目范围的 IDE 平台能力

以下 git4idea 的 `<extensions>` 注册项依赖 IntelliJ 平台框架本身，不是"Git 客户端用户可见能力"，不建议实现：
- VFS 相关（`GitIndexFileSystem` 等）—— Go+Viva 下用普通文件 I/O 替代。
- 统计采集（`GitStatisticsCollector` 等）—— JetBrains 内部遥测。
- `searchEverywhereContributor`、`searchScopesProvider` —— 对应规格"全局搜索"由 D5 待决事项单独跟踪（Viva 尚不支持"Shift 两次"）。
- `history.activityPresentationProvider`、`vcs.taskHandler` —— 依赖 IntelliJ Tasks/Local History 平台功能，规格未提及，建议排除并记录。
- `codeInsight.linkHandler`、提交消息自动补全（fixup/squash 前缀） —— 编辑器智能感知能力；提交消息自动补全严格说服务于 Git 工作流本身，建议降级为 P2 可选项而非直接排除。
- `jbProtocolRevisionResolver` —— IDE 间协议，不适用于独立桌面应用。
- `warmupConfigurator` —— 平台启动优化机制，不适用。

---

## git-features-trainer

插件性质：IntelliJ 内置交互式教程框架（`training` 模块）在 Git 场景的课程集，不是 Git 操作能力本身。课程覆盖：Git 入门、提交教程、功能分支工作流、交互式 Rebase 教程、Changelist 与 Shelve、Annotate/Blame、项目历史查看。

| 能力 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|
| 交互式分步教程框架 | 未实现 | P2 | 依赖 IntelliJ `training` 平台（高亮覆盖层、步骤校验引擎），属于 IDE 平台能力。规格范围内未提及新手引导教程，建议明确排除；若需要引导，走更轻量的空状态提示文案，不复刻完整教程引擎 | — |
| Changelist/Shelve 课程涉及的概念 | 未实现 | P2（需用户决策） | Changelist 是 IntelliJ 把工作区改动分组管理的平台概念，Shelve 是基于 Changelist 的搁置机制，均非 Git 原生概念。ROADMAP P9 已提醒区分 Shelf 与 Git Stash 的语义差别；本项目只做 Git Stash，不复刻 Changelist/Shelve | — |

## git-modal-commit

插件性质：极小的设置开关插件，通过 `advancedSetting id="git.non.modal.commit"` 把默认"非模态提交"切换回"模态提交"对话框，没有独立 Git 操作逻辑。

| 能力 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|
| 模态/非模态提交界面切换开关 | 未实现（cetacean 只有非模态形态） | P2 | 不建议实现第二套提交 UI。cetacean 的 Commit 面板本身是常驻可见的，对应非模态默认行为；模态对话框是历史兼容选项，用户量小，建议排除并在差异清单记录 | — |

## GitHub 插件

插件性质：依赖 Git4Idea，提供账号登录、仓库克隆、PR 浏览/创建/review、Gist 创建。

| 能力 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|
| GitHub 账号登录（OAuth/Token） | 未实现 | P2 | 规格已明确"不把远程托管平台帐号作为使用基本 Git 功能的前置条件"，归入 ROADMAP P11/M13 | — |
| 从 GitHub 浏览仓库列表再克隆 | 未实现 | P2 | 核心克隆能力（`Git.Clone`）已在 P0；"浏览我的仓库列表"是增值能力，归入 P11/M13 | — |
| PR 浏览（列表、筛选状态/作者/标签） | 未实现 | P2 | 归入 P11/M13 | — |
| PR 创建、Checkout PR 分支到本地 | 未实现 | P2 | "Checkout PR 分支"本质是 fetch+checkout，核心能力已覆盖，缺的是从 PR 列表跳转的便捷集成 | — |
| PR Review（Diff、评论、Approve/Request Changes、Merge/Close） | 未实现 | P2 | 归入 M13 | — |
| Gist 创建 | 未实现 | P2 | 规格范围外的 GitHub 平台特定功能，建议排除或归入 P11 决定 | — |

## GitLab 插件

插件性质：与 GitHub 插件对等，提供账号登录、克隆、MR 筛选浏览（state/author/assignee/reviewer/label）/timeline/review（Diff、评论、建议修改）/Merge/Close。额外有 `.gitlab-ci.yml` 语法支持模块。

| 能力 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|
| GitLab 账号登录、多账号切换 | 未实现 | P2 | 同 GitHub，归入 P11/M13 | — |
| 克隆 GitLab 项目 | 未实现 | P2 | 核心克隆能力已在 P0，项目浏览器是增值集成 | — |
| Merge Request 浏览/筛选/timeline | 未实现 | P2 | 归入 P11/M13 | — |
| MR Review（Diff、评论、建议、Merge/Close） | 未实现 | P2 | 同上 | — |
| `.gitlab-ci.yml` 语法高亮支持 | 未实现 | P2（排除） | 属于 IDE 编辑器语法高亮能力，规格已排除 | — |

## Terminal（默认随附）

插件性质：IntelliJ 内置终端模拟器，多 Shell/多标签/分屏/自定义 prompt，随 IDE 默认携带而非可选安装。

| 能力 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|
| 完整终端模拟器 | 未实现（按规格已决策排除） | P2（已决策） | 规格 §2 已明确排除，只提供"在外部终端打开"命令；ROADMAP P12 要求不在 Git 绘制层私造终端模拟器。此项不需要新决策 | — |
| "在外部终端打开"命令 | 未实现 | P1 | 规格要求的简单命令，与完整终端模拟器无关，需补上 | Driver 测试命令触发（平台调用可 mock） |

## TextMate（默认随附）

插件性质：基于 TextMate bundle 语法规则的通用语言高亮支持。

| 能力 | 当前状态 | 优先级 | Go/Viva 方案 | 测试方式 |
|---|---|---|---|---|
| 任意文件类型的 TextMate 语法高亮 | 未实现 | P2（排除） | 规格已排除"IDE 编辑器、语法高亮"。Diff 视图只做只读比较，若需要语法着色，属于 Viva `CodeDiff` 组件自身能力（D4 已决定复用），不接入 TextMate bundle 解析 | — |

---

## 汇总与建议

### 规模统计

git4idea 核心动作约115项：**已实现 ≈18项**（Stage基本操作、Commit/Amend/Commit and Push、分支CRUD基础、Fetch/Pull/Push基础、ignored展示），**部分实现 ≈12项**（分支弹出层筛选、三版本比较、CheckoutAsNewBranch等），**未实现 ≈85项**（Merge/Rebase/Cherry-pick/Revert全流程、交互式Rebase、Stash全部、冲突三方视图、Worktree、Log右键历史编辑、Tag管理等）。这与 ROADMAP.md 记录的里程碑进度吻合：M0-M5 对应"已实现+部分实现"，M6-M10（未开始）对应"未实现"的大头。

### 需要用户决策排除或降级的项（不计入首发 P0/P1 范围）

1. **独立 Staging Area 面板模式**（`Git.Show.Stage` 及子动作族）——与非模态 Changes 面板并存的两套 UI，建议只实现一套。
2. **模态提交对话框**（git-modal-commit 整体）——同理，只提供一种提交面板形态。
3. **IntelliJ Changelist/Shelve**——与 Git Stash 是不同概念，本项目只做 Git Stash。
4. **交互式教程框架**（git-features-trainer 整体）——依赖 IntelliJ training 平台，建议用轻量空状态引导文案代替。
5. **GitHub/GitLab 账号登录与 PR/MR 全流程**——规格已明确非前置条件，且归入 ROADMAP P11/M13。
6. **TextMate 语法高亮、GitIgnore/Exclude 文件语言智能**——规格已排除语法高亮类编辑器能力。
7. **完整终端模拟器**——规格已排除，只做"在外部终端打开"命令。
8. **GPG 签名提交配置**、`Git.Log.Show.Command` 调试面板、各类 `internal="true"` 标记的动作——低频或 JetBrains 内部调试功能，归入 P2 待定。
9. **提交消息自动补全**（fixup/squash前缀智能提示）——轻微跨越"不做代码补全"边界，建议降级为可选项而非直接排除。

### 对后续 P2-P12 的影响

- **P5（Changes 与 Diff）**：补齐三版本对比、分支/Tag 对比文件、冲突时 AcceptTheirs/AcceptYours/三方视图快捷操作。
- **P6（Branch/Refs）**：补齐分支弹出树分组/筛选/最近使用/尺寸记忆、Rebase/Merge into Current、从 Log 任意提交 checkout/建分支、Tag 完整管理、推送非当前分支。
- **P7（Log）**：补齐几乎整个"Log 右键操作"分类（Reset/Revert/Cherry-pick/Uncommit/Reword/Fixup/Squash/Interactive Rebase/BrowseRepoAtRevision）——当前差距最大的单一类别。
- **P8（历史编辑与冲突）**：补齐 Merge/Rebase/Cherry-pick/Revert 主流程及 Continue/Skip/Abort、交互式 Rebase 编辑器、三方合并视图（唯一标记"需要架构评估"而非直接"原生实现"的项）。
- **P9（Stash）**：从零实现整个分类，当前完全未覆盖。
- **P10（设置与动作）**：补齐 Tag/GPG 签名等设置项，以及"在外部终端打开"命令。
- **P11/P12**：已有明确排除/延后建议，只需在决策记录里确认。
