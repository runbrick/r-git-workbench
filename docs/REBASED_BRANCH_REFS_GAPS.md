# Rebased Branch/Refs Parity 差异盘点

对应 `docs/ROADMAP.md` P6 任务。与 P1（`docs/REBASED_ACTION_INVENTORY.md`）、P2（`docs/REBASED_UI_PARITY_GAPS.md`）、P3（`docs/REBASED_SPLITTER_PERSISTENCE_GAPS.md`）、P4（`docs/REBASED_COMMIT_SHELF_GAPS.md`）、P5（`docs/REBASED_CHANGES_DIFF_GAPS.md`）配套，不重复已记录内容。

## 1. 分支弹出树与列表现状

`paintBranches`（`shell.go:976-1232`）是一个常驻页面（Branches 导航项点击后整页切换），不是弹出层/弹出树。内容：创建分支输入框+按钮（`shell.go:986-993`）、三个条件性预检确认条（删除分支/Checkout冲突/Force Push/Push预览，`shell.go:994-1118`）、多远端选择行（仅 `len(m.remotes) > 1` 时出现，`shell.go:1119-1139`）、一个按 Local/Remote/Tags 三段分组的滚动列表（`shell.go:1141-1219`）。

和 P1/P2 已记录的结论一致：这是"列表页"形态，不是 Rebased 基线"Ctrl+Shift+\` 弹出的分支快捷操作层"形态。本次确认列表本身的分组实现是存在的（三段式 Local/Remote/Tags，`shell.go:1142-1153`），比 P1 原先"未实现"的笔墨更细一点：**分组存在，弹出交互不存在**。

## 2. 搜索/Recent/Favorite

全文搜索 `paintBranches` 函数体，**没有发现任何搜索输入框**（对比 Changes 视图的 `f.Field`，Branches 列表没有等价物）、**没有"Recently Used"分组**（三段分组固定是 Local/Remote/Tags，不含时间维度）、**没有收藏/favorite 标记字段**（`git.Ref` 结构体本次未展开读取，但 `paintBranches` 渲染逻辑里找不到任何 favorite 相关的图标或点击切换代码）。规格 §4.5"分组显示 Local、Remote、Recently Used"中的 Recently Used 完全缺失，这是本次新确认的点（P1 原先的"分支弹出树分组/筛选"条目把搜索、Recent、分组混在一起描述为一项，这里拆清楚：分组的静态三段式有，搜索和 Recent 都没有）。

## 3. Checkout 安全检查核实

**结论：ROADMAP M3 记录的"计划绑定HEAD、远程目标SHA和工作区状态，执行前重新校验"准确无误，不是文档虚夸。**

`PlanCheckoutRemoteBranch`（`repo.go:1040-1083`）采集 `Head`、`Target`（远程分支 SHA）、`StatusHash`（工作区状态指纹）、`DirtyFiles` 列表，组装成 `RemoteCheckoutPlan`。`ExecuteCheckoutRemoteBranch`（`repo.go:1085-1116`）在加写锁后**重新跑一次 `PlanCheckoutRemoteBranch`**并逐字段比较（`repo.go:1091-1097`：`current.Remote != plan.Remote || ...`），任何字段不一致直接拒绝执行并要求"review a new plan"。三种模式（`CheckoutSafe`/`CheckoutStash`/`CheckoutForce`）在同一执行函数里分支处理，`CheckoutSafe` 遇到脏文件直接报错（`repo.go:1098-1100`），`CheckoutStash` 先 `git stash push --include-untracked`（`repo.go:1101-1106`），`CheckoutForce` 用 `git switch --discard-changes`（`repo.go:1108-1109`）。

UI 侧（`shell.go:1009-1061`）三个按钮（Stash and Checkout / Force Checkout 二次确认 / Cancel）对应规格 §4.5 要求的 Cancel/Stash/Force Checkout 三选项，Force Checkout 走的是"点第一次变成 ConfirmForceCheckout 文案，点第二次才真正执行"的二次确认模式（`shell.go:1038-1052`），符合规格"Force Checkout 需二次确认"要求。**这部分是目前盘点到的 P1-P6 范围内实现质量最高的功能之一**。

## 4. 分支 CRUD 与 Upstream 核实

- **创建**（`CreateBranch`，`repo.go:910-921`）：校验分支名（`checkBranchName` 调用 `git check-ref-format --branch`）后直接 `git branch <name>`，没有预检/确认流程（符合规格，创建分支不是有损操作，不需要预检）。测试：`repo_test.go:730 TestCreateBranchValidatesNameAndPreservesHead`。
- **重命名**（`RenameBranch`，`repo.go:923-937`）：`git branch -m oldName newName`，同样只做名称校验，没有"目标名已存在"的预检查（依赖 Git 原生报错）。测试：未找到专属的 `TestRenameBranch*`，只在 `TestBranchManagementAndDeletePreflight`（`repo_test.go:843`）里可能覆盖（本次未展开读取该测试体，标记为待确认而非直接判定缺失）。
- **Checkout（本地分支切换）**（`SwitchBranch`，`repo.go:1129` 起，本次未完整展开）：UI 侧 `shell.go:1172-1174` 的"Switch Branch"按钮直接调用 `m.write("switch-branch", ...)`，**没有看到预检流程**——对比远程分支 checkout 有完整的 Plan/Execute 两段式，本地分支切换看起来是直接执行，没有"切换会覆盖本地改动"的预检确认。规格 §4.5"Checkout 遇到未提交变更时先检查是否会覆盖文件"理论上应该也适用于本地分支间切换，不只是远程分支转本地。测试 `TestSwitchBranchPreservesDirtyWorktreeOnRefusal`（`repo_test.go:775`）从名字看验证的是"拒绝"行为（可能是 Git 原生拒绝并返回错误，而不是客户端自己预检），这意味着本地分支切换的"安全检查"可能完全依赖 Git 自身的拒绑切换机制，而不是像远程 checkout 一样有应用层的 Plan/DirtyFiles/Stash/Force 完整流程。**这是本地分支 checkout 和远程分支 checkout 功能完整度不对等的新发现，此前 P1/P2/P4/P5 都没有提到。**
- **Upstream**：`SetUpstream`（`repo.go:996-1013`）支持设置（`--set-upstream-to=`）和取消（`--unset-upstream`），UI 侧有对应的 Set/Unset 按钮（`shell.go:1186-1198`），这部分完整。

## 5. 删除未合并分支的预检核实

**结论：规格 §4.5"删除未合并分支时显示未合并提交和保留该分支的替代操作"没有实现，当前只是把 Git 原生的拒绝错误转发出去。**

`PlanDeleteBranch`（`repo.go:946-971`）只做三件事：验证分支名、确认分支存在、确认不是当前分支（`repo.go:965-968`）。**没有任何代码检测这个分支相对其他分支/HEAD 是否"已合并"**（没有调用 `git branch --merged` 或等价逐步检查）。`DeleteBranch`（`repo.go:973-994`）执行时用的是 `git branch -d`（`repo.go:989`，普通删除，不是 `-D` 强制删除），如果分支未合并，这条命令本身会被 Git 拒绝并返回非零退出码，错误信息走 `fmt.Errorf("delete branch: %w: %s", ...)`（`repo.go:991`）原样转发。

也就是说：**用户删除未合并分支时，客户端不会提前告知"这个分支有N个未合并提交"，而是让用户点击删除后才从 Git 的报错里看到失败**，这不符合规格"显示未合并提交"（应该在确认对话框里主动展示，不是失败后才展示）。而且规格还要求"保留该分支的替代操作"（比如提示用户转去 Merge 该分支），这完全没有，`DeleteBranch` 失败后 UI 侧只会把错误信息显示出来（通过通用错误展示机制），不会引导到任何替代操作。

测试：`repo_test.go:912 TestDeleteBranchRejectsStalePreflight` 从名字看测的是"预检过期后拒绝执行"（并发修改场景），不是"未合并分支删除时的用户提示内容"，`TestBranchManagementAndDeletePreflight`（`repo_test.go:843`）可能涉及基础删除流程但同样未验证未合并提示文案。**没有发现任何测试验证"未合并分支删除时用户能看到未合并提交列表"这个规格要求的具体行为**，因为这个行为本身在代码里不存在。

## 6. 远端选择 UI

多远端场景（`m.remotes` 长度大于1时，`shell.go:1119-1139`）：有一行按钮组，"Upstream Default"（默认跟随各分支自己的 upstream）+ 逐个远端名称按钮，点击切换 `m.selectedRemote`，用 `*` 后缀标记当前选中项（`shell.go:1124,1132`）。这个选择结果被 Fetch 使用（P1 已提到"Fetch 可指定所选远端"）。

**UI 形态问题**：这是一排始终平铺的按钮，不是下拉选择器，如果远端数量多（比如5个以上），会和 P2 已经指出的"顶栏横排 Pull 策略按钮密度过高"犯同样的错误——Rebased 基线大概率是下拉或列表选择，不是平铺按钮组，这是密度层面的新发现，但本次未看到 Rebased 截图里多远端场景的具体样式（screenshot.png 只有单远端的 `origin`），无法逐项对照基线，只能记录现状形态供后续截图比对参考。

## 7. Tag 功能边界

**结论：导航入口存在（`keymap.ShowTags`），但点击后落入通用占位分支，没有任何 Tag 专属内容；后端完全没有 Tag 相关函数。**

- `keymap.ShowTags` 在 `internal/keymap/actions.go` 里作为 6 个导航 `ActionID` 之一存在（第12、26行）。
- `paintContent`（`shell.go:947-974`）用一串 `if m.Active == keymap.XXX` 显式路由 `ShowRepositories`/`ShowChanges`/`ShowLog`/`ShowBranches` 到各自的绘制函数，**`ShowTags` 和 `ShowStash` 都没有专属分支**，都会落入最后的通用占位代码（`shell.go:965-973`）：只显示动作名称标题+分隔线+居中的"Select Repository"文字。这段占位文字本身就是错的（显示"请选择仓库"，但用户是在已打开仓库的情况下点进 Tags 的），是一个连空状态文案都没有正确适配的占位符。
- `internal/repo/repo.go` 全文函数名搜索（`grep "^func" ... | grep -i tag`）：**零匹配**，没有任何创建/删除/推送/列出 Tag 的独立函数。Tag 在当前代码里唯一的存在形式是 `paintBranches` 列表里的"Tags"分组（靠 `ref.Tag` 字段从 `for-each-ref` 解析结果里筛出来，`shell.go:1142-1153`，只读展示，和 P1 已确认的"`Git.Tag` 未实现"一致），以及 Log 提交图里的引用标签装饰（P2 已提到的 `refLabels`）。
- 这和 P1 的结论一致（`Git.Tag` 未实现，`Git.CreateNewTag` 未实现，`GitPushTagsActionGroup` 未实现），本节确认的是**具体的代码证据**：导航项是摆设，专属视图不存在，后端能力为零，现有的 Tag 展示只是 Branches 列表的只读旁支。

## 8. 从 Log 操作分支

全文搜索 `paintLog` 函数体（此前 P2 已部分读取过）和相关的点击处理代码：Log 的提交行点击只触发选中（展示详情/Diff），没有发现右键菜单或类似的上下文操作入口。P4 已确认整个应用没有右键菜单/快捷键系统，这里确认的是具体后果——从 Log 创建分支（`Git.CreateNewBranch.FromCommit`）、从 Log checkout 到某个历史提交、从 Log cherry-pick/revert 等操作**连一个占位按钮都没有**，不是"有入口但功能未接"，是入口本身不存在。这和 P1 标记的"未实现"一致，没有新发现，只是确认没有任何局部UI雏形。

## 9. Worktree 范围确认

全文搜索 `docs/GIT_CLIENT_PRODUCT_SPEC.md` 的 "worktree" 关键词：只有两处非Git-worktree功能性提及——§3"有 Tag/Worktree 时按需显示对应入口"（左侧导航条目的*可能*存在性，不是功能规格）和 §6"Git identity...来源（system/global/local/worktree）"（这里 worktree 是指 identity 配置的作用域，和 git-worktree 功能无关）。**规格文档没有任何章节系统描述 Git Worktree（`git worktree add/list/remove`）这个功能本身该怎么工作**，和 P1 已经做出的判断一致："规格完全没提及 worktree，需用户确认是否纳入范围"。

本次确认：`internal/repo/repo.go` 全文搜索 `worktree`，命中的全部是 Git 的"工作区"文件状态字段（`file.Worktree` 表示 porcelain 状态里的工作区列，和 `git worktree` 命令完全是两个不同概念，命名只是撞了同一个英文词），**没有任何 `git worktree add/list/remove/prune/lock` 的封装**。ROADMAP P6 原文提到的"worktree 间状态提示"这个验收点目前连功能本体都不存在，提示自然也不存在，这是待用户决策的范围问题，不是实现缺陷。

## 10. 测试覆盖现状

| 场景 | 是否有测试 | 测试文件/函数 |
|---|---|---|
| 列出本地/远程分支 | 有 | `internal/repo/repo_test.go:665 TestRefsListsLocalAndRemoteBranches` |
| 创建分支（校验名称、保留HEAD） | 有 | `repo_test.go:730 TestCreateBranchValidatesNameAndPreservesHead` |
| 本地分支切换，脏工作区拒绝 | 有 | `repo_test.go:775 TestSwitchBranchPreservesDirtyWorktreeOnRefusal`（需确认是客户端预检还是Git原生拒绝，详见第4节） |
| 分支管理与删除预检（基础） | 有 | `repo_test.go:843 TestBranchManagementAndDeletePreflight` |
| 删除分支预检过期拒绝（并发场景） | 有 | `repo_test.go:912 TestDeleteBranchRejectsStalePreflight` |
| **删除未合并分支时显示未合并提交内容** | **没有** | 功能本身不存在（第5节），无法测试 |
| 重命名分支（目标名冲突等边界） | 待确认 | 可能包含在 `TestBranchManagementAndDeletePreflight` 内，未展开读取确认，不直接判定缺失 |
| 远程分支 checkout：Cancel/Stash/Force 三路径 | 有 | `repo_test.go:951 TestRemoteCheckoutPlansCancelStashAndForce` |
| 远程 Fetch/Pull/Push 基础（针对裸仓库） | 有 | `repo_test.go:1047 TestRemoteFetchPullPushAgainstBareRepository` |
| Upstream 设置/取消 | 待确认 | 本次未找到明确的 `TestSetUpstream*` 命名测试，可能内含于其他测试，标记待确认 |
| 推送/拉取非当前选中分支（分支选择器直接操作） | 没有 | P1 已标记该功能"部分实现"（缺非当前分支操作），无专属测试 |
| Tag 创建/删除/推送 | 没有（功能不存在） | 无相应代码 |
| 分支搜索/Recent/Favorite | 没有（功能不存在） | 无相应代码 |
| 从 Log 操作分支（创建/checkout/cherry-pick等） | 没有（功能不存在） | 无相应代码 |
| Worktree 管理 | 没有（功能不存在，范围待决策） | 无相应代码 |
| Merge into Current / Rebase Current onto Selected | 没有（功能不存在） | 无相应代码，`repo.go` 里的 `--rebase`/`--no-rebase` 仅用于 Pull 策略，不是独立分支操作 |

## 汇总：P6 验收待办清单

区分"后端已支持只缺UI"、"纯UI缺失"、"两端都要做"、"需要用户先决策范围"四类。

**后端已支持只缺UI**
1. 无——本次核查没有发现"后端做了、UI没接"的孤岛（不同于 P4 的 Amend），Branch/Refs 领域的缺口基本是两端同步缺失或纯粹没做。

**纯UI缺失**
2. Tags 导航项点击后的占位文案错误（显示"Select Repository"，与实际上下文不符）——`shell.go:965-973`，即使 Tag 功能本身未实现，至少应该显示一个准确的"Tag 功能尚不可用"或空状态，不是复用错误的占位文字。
3. 分支列表缺搜索框——`paintBranches`（`shell.go:976-1232`）整个函数没有任何过滤/搜索输入，规格 §4.5 隐含要求"可搜索的完整列表"（Branches 视图），需要新增。
4. 分支列表缺 Recently Used 分组——当前固定三段式分组（Local/Remote/Tags），需要新增一个基于最近切换/操作时间的第四分组或排序提示。

**两端都要做**
5. 删除未合并分支的预检提示——`PlanDeleteBranch`（`repo.go:946-971`）需要新增"是否已合并"检测（比如对比 `git branch --merged` 的结果或 `git merge-base --is-ancestor`），UI 侧（`shell.go:994-1008` 的删除确认条）需要在确认文案里展示未合并提交数量/列表，并提供"改为 Merge 该分支"之类的替代操作入口。
6. 本地分支切换的安全检查链路缺失（对比远程分支 checkout 的完整 Plan/Execute 两段式）——`SwitchBranch`（`repo.go:1129`起）需要补上和 `PlanCheckoutRemoteBranch`/`ExecuteCheckoutRemoteBranch` 对等的预检-确认-执行流程（检测脏文件、提供 Stash/Force 选项），当前疑似完全依赖 Git 自身拒绝机制，没有应用层预检，需要先读完 `SwitchBranch` 完整实现确认具体缺口范围（本次受限于读取范围未展开到函数结尾之后的部分）。
7. Merge into Current / Rebase Current onto Selected——P1 已标记未实现，这里确认后端 `repo.go` 里没有任何独立的分支级 merge/rebase 封装（只有 Pull 策略里的 `--rebase` 参数），需要两端新增，属于 M6/M8 范围，不建议在 P6 单独处理，仅在此记录交叉依赖关系。
8. 推送/拉取非当前选中分支——分支选择器目前只能对当前分支做 Push/Pull（P1 已标记部分实现），需要在 `paintBranches` 的分支行里加 Push/Pull 按钮并接入对应的非当前分支远端操作后端逻辑。
9. Tag 完整管理（创建/删除/推送）——`repo.go` 零 Tag 函数，需要新增 `CreateTag`/`DeleteTag`/`PushTag` 等，UI 侧需要让 `ShowTags` 路由到专属绘制函数而不是落入通用占位。
10. 远端选择 UI 改造——多远端场景下的平铺按钮组（`shell.go:1119-1139`）在远端数量增多时密度会和 P2 已指出的顶栏按钮密度问题一样显得拥挤，建议改为下拉选择器，需先确认 Rebased 基线在多远端场景下的实际样式（当前截图只有单远端，无法直接比对）。

**需要用户先决策范围**
11. Worktree 管理与"worktree 间状态提示"——规格文档完全没有系统描述这个功能，P1 已标记"规格未提及，需用户决策"，本次确认后端和UI都是完全空白，不是被遗漏，是从未被纳入过范围讨论，建议作为单独的范围决策项提给用户，不要默认纳入或默认排除。
