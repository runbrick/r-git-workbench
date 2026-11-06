# Rebased 历史编辑与冲突 Parity 差异盘点

对应 `docs/ROADMAP.md` P8 任务。与 P1（`docs/REBASED_ACTION_INVENTORY.md`）、P2（`docs/REBASED_UI_PARITY_GAPS.md`）、P3（`docs/REBASED_SPLITTER_PERSISTENCE_GAPS.md`）、P4（`docs/REBASED_COMMIT_SHELF_GAPS.md`）、P5（`docs/REBASED_CHANGES_DIFF_GAPS.md`）、P6（`docs/REBASED_BRANCH_REFS_GAPS.md`）、P7（`docs/REBASED_LOG_GAPS.md`）配套，不重复已记录内容。本篇重点是技术可行性评估，不重复确认 P1 已经逐项标记的"未实现"状态。

## 1. 现有安全基础设施的复用评估

**结论：有一套清晰、可直接抽象的"Plan→指纹→Execute时重新校验"模式，三处独立实现但结构高度一致，值得抽成通用基础设施再用于 Merge/Rebase/Cherry-pick/Revert/Reset。**

三个现有实现逐一核对：

- `PlanDeleteBranch`/`DeleteBranch`（`repo.go:946-994`）：Plan 阶段采集 `Head`（`rev-parse HEAD`）、`Target`（`rev-parse refs/heads/<name>`）、`StatusHash`（`sha256.Sum256(status.Stdout)`，对 `git status --porcelain=v2 -z --branch --untracked-files=all` 的原始字节做哈希）。Execute 阶段加写锁后**重新调用同一个 Plan 函数**，把新结果和传入的 plan 逐字段 `!=` 比较（`repo.go:986`，`current != plan`，因为 `BranchDeletePlan` 是纯值结构体，可以直接用 `!=` 整体比较），不一致就拒绝执行。
- `PlanCheckoutRemoteBranch`/`ExecuteCheckoutRemoteBranch`（`repo.go:1040-1116`）：同样的 Plan→StatusHash→Execute重新Plan→逐字段比较模式，多了 `DirtyFiles []string` 字段用于 UI 展示脏文件列表，以及一个 `CheckoutMode`（Safe/Stash/Force）参数控制执行分支。
- `PlanRestoreHunks`/`RestoreHunks`（`repo.go:860-?`）：模式相同，但指纹对象不是整个仓库状态，是**具体某个文件的 hunk 补丁内容**（`sha256.Sum256(hunks[selection.Index].Patch)`，`repo.go:883`），这是更细粒度的指纹校验，说明这套模式不要求"状态指纹"必须是整仓库范围，可以按操作对象灵活选择指纹范围。

**可抽象的通用骨架**（三处实现目前是手写三份，没有抽出公共代码，这是具体的技术判断，不是"看起来能抽象"的空话）：
1. 一个通用的 `Plan[T]` 概念：采集"操作前状态快照"（HEAD、目标 ref/commit、相关文件状态指纹），返回一个值类型的 Plan 结构体。
2. 一个通用的 `Execute` 包装：加 `r.write` 锁 → 用相同参数重新跑一次 Plan 函数 → 和传入的 plan 做字段级比较 → 不一致则返回"plan 已过期，请重新生成"的错误 → 一致则执行真正的 git 命令。
3. Go 没有运算符重载和简单的泛型值比较复用机制，三个 Plan 结构体字段不同（`BranchDeletePlan` 4 个字段，`RemoteCheckoutPlan` 6 个字段+slice，`RestoreHunkPlan` 含 slice 不能用 `!=`），**不能直接用泛型做到"一行代码判断是否过期"**，`RemoteCheckoutPlan`/`RestoreHunkPlan` 含 slice 字段的比较已经是手写逐字段展开（不是整体 `!=`）。实际可抽的是"写锁+重新执行某个函数+比较结果"这个**调用模式**（可以用一个高阶函数包装：`func executeWithRevalidation[T comparable](r *Repository, plan T, replan func() (T, error), apply func() error) error`，但 `comparable` 约束排除了含 slice 的 plan 类型，这部分仍需各自手写比较逻辑），不是"消除重复代码"意义上的完全复用。

对 Merge/Rebase/Cherry-pick/Revert/Reset 的具体应用：
- **Reset**（soft/mixed/hard）：Plan 阶段采集当前 HEAD、目标 commit SHA、工作区状态指纹（hard 模式需要，soft/mixed 不触碰工作区可以不采集或仍采集用于展示），这和 `PlanDeleteBranch` 的字段结构几乎一样，**复用成本最低**。
- **Merge/Rebase 发起**：Plan 阶段采集当前 HEAD、目标分支/commit、工作区状态指纹、**是否已有进行中的 merge/rebase**（这是新增的检查维度，见第2节），比 `PlanCheckoutRemoteBranch` 多一个"仓库级操作状态"字段，但整体结构仍可沿用。
- **Cherry-pick/Revert**：类似，Plan 阶段多了"选中的 commit 列表"（Cherry-pick 可多选且有顺序要求，见 P1 已记录），Plan 结构体需要一个 `[]string` 字段记录 SHA 序列，执行时重新校验这些 commit 是否还存在/未变化。
- **Continue/Skip/Abort**（merge/rebase/cherry-pick 进行中的后续操作）：这类操作不太适合"Plan→Execute"模式，因为它们操作的对象是"当前进行中的 Git 操作状态"本身（读取 `.git/MERGE_HEAD`、`.git/rebase-merge/` 等），更接近"检查当前是否真的处于该状态，若是则执行对应 git 命令"，不需要用户给出的 plan 和重新计算的 plan 做比较，是更简单的前置检查模式，不需要套用完整的指纹校验基础设施。

## 2. 冲突检测现状与可扩展性

**结论：当前的冲突识别是纯粹的文件状态行解析，没有任何仓库级"操作进行中"状态的识别能力，这是本次评估里最关键的新发现——不是"冲突检测需要加强"，是"目前完全没有能力知道仓库是否处于 merge/rebase 进行中"。**

当前机制（`internal/git/status.go:102-118`）：`git status --porcelain=v2` 的每条文件记录第一个字符如果是 `u`，直接置 `file.Conflict = true`（`status.go:118`，`file.Conflict = item[0] == 'u'`）。这是 porcelain v2 格式本身对"unmerged"文件的标记，解析逻辑只有一行判断，纯粹是读取 Git 已经告诉你的文件级事实，不是客户端自己计算出来的。

这个机制**可以直接复用**于 Merge/Rebase 产生的新冲突——因为不管冲突是来自已经存在的历史遗留（当前测试场景），还是用户刚执行 `git merge` 失败产生的，`git status --porcelain=v2` 的 `u` 标记行为是一致的，cetacean 现有的状态刷新机制（文件监听触发重新读取 status）天然就能捕捉到新产生的冲突文件，不需要新写冲突检测代码，这部分是**现成的、可以直接复用的**。

但有一个结构性空白：`git.Status` 结构体（`status.go:22-30`：`Root/Branch/Head/Ahead/Behind/HasTrack/Files`）**没有任何字段表达"仓库当前处于 merge/rebase/cherry-pick/revert 进行中"这种仓库级状态**。Git 用以下方式记录这类状态，当前代码完全没有读取：
- `.git/MERGE_HEAD` 存在 → 处于 merge 冲突中
- `.git/MERGE_MSG` → merge 提交消息草稿
- `.git/rebase-merge/` 或 `.git/rebase-apply/` 目录存在 → 处于 interactive/普通 rebase 中，且这些目录下有 `done`/`git-rebase-todo` 等文件记录 rebase 的具体进度
- `.git/CHERRY_PICK_HEAD` → 处于 cherry-pick 中
- `.git/REVERT_HEAD` → 处于 revert 中
- `.git/sequencer/` → interactive rebase 的更详细状态（如 todo 列表）

**这是规格 §4.7"保留正在进行的 Git 操作状态"和 P1 已标记的 `GitMergeRebaseWidget`（顶栏常驻进行中状态指示）的前提条件，目前这个前提条件完全不存在。** 没有这层检测，客户端无法在用户重新打开应用、或者在另一个终端手动执行了 `git merge` 后失败时，向用户展示"你当前有一个未完成的 merge，要继续还是中止"。这不是小补丁，是需要新增一个 `OperationState`（或类似命名）的读取函数，检查上述文件/目录是否存在并解析出"当前处于哪种操作、涉及哪些 commit"，这是 Merge/Rebase/Cherry-pick/Revert 四个功能能否正确工作的地基，比任何单个操作的 UI 都更优先。

## 3. Viva 三方合并视图技术路线评估

`docs/UI.md` §8 记录的现状：`CodeDiff` 只支持两栏、右侧可编辑，三方视图已提 [#132](https://github.com/vivaui/viva/issues/132) 但维护者是否会实现、何时实现均未知。给出四条现实技术路线评估：

**路线A：只做"接受 ours/接受 theirs/接受 both"的简化操作，不提供真正可编辑的 Result 面板**
- 做法：冲突文件用现有的 `CodeDiff` 两栏并排展示 ours（当前分支版本）和 theirs（合并来源版本），提供三个按钮：Accept Ours（`git checkout --ours <path>`）、Accept Theirs（`git checkout --theirs <path>`）、Accept Both（把两个版本的内容都写入，通常是简单拼接，需要明确告知用户这可能产生语法错误需要手动再编辑）。
- 工作量：**小**，复用现有 `CodeDiff` 组件，不需要新的 Viva 能力，只需要新的 git 命令调用和三个按钮。这对应 P1 已标记的 `Git.Stage.AcceptTheirs`/`AcceptYours`（标记 P0），技术上完全可行。
- 限制：不支持"保留 ours 的这一段、theirs 的那一段"这种混合编辑，复杂冲突（比如双方都改了同一行但各自正确）无法在应用内完成，用户必须退出去用外部编辑器手动改冲突标记（`<<<<<<<`/`=======`/`>>>>>>>`），然后回来标记为已解决。这不是功能残缺，Git 原生工作流本身也支持"手动编辑 + `git add`"，只是客户端没有提供编辑器。

**路线B：调用外部合并工具**
- 做法：读取 `git config merge.tool`（如果用户配置了 kdiff3/meld/vscode 等），调用 `git mergetool` 或直接启动配置的工具处理冲突文件，客户端只负责触发和等待结果。
- 工作量：**中**，不需要自建 UI，但需要处理"用户没配置 mergetool"的情况（规格要求的默认体验不能依赖用户额外配置），以及等待外部进程的异步状态管理（符合项目"绘制不等待"的约束，需要走后台收件箱模式）。
- 限制：依赖用户系统上有可用的外部工具，不是"开箱即用"，和产品规格"除 logo 外功能和样式一模一样"的 parity 目标有冲突——Rebased 基线明确有内置三方视图，不依赖外部工具，这条路线会产生明显的功能缺口而不只是视觉差异。

**路线C：用 CodeDiff 两栏 + 额外的只读 Result 预览（三个只读区域，不可编辑）**
- 做法：三个并排的只读区域（Current/Incoming/Result-预览），Result 区域实时显示"如果应用当前选择会得到什么内容"，但不允许直接在 Result 区域打字编辑，编辑仍需外部编辑器或路线A的三个按钮操作后刷新预览。
- 工作量：**中大**，需要自己布局三栏（`CodeDiff` 本身是两栏组件，第三栏需要应用层额外画一个只读文本展示区，不是直接扩展 `CodeDiff`），但不需要实现"可编辑的第三栏"这个 Viva 缺口本身，规避了 issue #132 的核心限制。
- 限制：用户仍然无法在应用内直接完成复杂冲突的手工编辑，只是比路线A多了一个"预览最终效果"的辅助视图，用户体验介于A和D之间。

**路线D：完全自绘一套三方 diff 组件（包括可编辑的第三栏）**
- 做法：不使用 `CodeDiff`，从 Git 的冲突标记文本（`<<<<<<<`/`=======`/`>>>>>>>`）自己解析出 ours/theirs/base 三个版本，自绘三栏布局，中间栏是一个可编辑文本区（类似 `components.Textarea` 但需要支持语法高亮冲突标记、支持从左右两栏点击"插入这段"到中间栏）。
- 工作量：**大，且有显著技术不确定性**。这是唯一需要"自绘可编辑多栏内容区"的路线，相当于半自己实现一个迷你代码编辑器组件（至少要支持多行文本编辑、选择、插入），这不是简单的布局问题，是需要评估 Viva 的 `Textarea`/文本输入基础能力能否撑起这种交互密度（比如大文件冲突时的性能、撤销/重做、和外部编辑器的本地化编辑习惯是否冲突）。如果 Viva 的文本编辑原语本身就不够用，这条路线的风险会从"UI 工作量大"升级为"需要先给 Viva 提更多 issue、等待框架能力"。

**建议**：短期（首发）采用路线A，这是 P1 已经标记为 P0 优先级的 `AcceptTheirs`/`AcceptYours`/`AcceptBoth`，技术确定、工作量小、能覆盖多数真实冲突场景（很多冲突是"整个文件某一侧版本是对的"，不需要逐行混合）。路线C可以作为路线A之后的增量改进，在用户反馈"经常需要混合编辑"后再评估。路线D不建议现在投入，技术不确定性太高，且可能随着 Viva 自己修复 issue #132 而变得不必要——这正是"先问框架维护者，不要在应用里绕过"这条项目约定应该适用的场景。路线B不建议作为默认路径，但可以作为"高级选项"补充（用户如果已经配置了喜欢的 mergetool，让他们继续用）。

## 4. Reset 操作的数据丢失边界评估

**结论：已有的 Diff 比较能力（`Compare` 函数）可以直接复用计算"hard reset 会丢失什么"，但还需要新增 HEAD↔目标commit 之间的 diff 统计，这部分数据源目前不存在。**

Hard reset 会丢失两类内容：
1. **工作区和暂存区里尚未提交的改动**——这正是 P5 已经分析过的 `Compare` 函数（`repo.go:673-737`）已经在算的内容：HEAD↔工作区、HEAD↔暂存区的差异。`r.Status()` 返回的 `git.File` 列表本身就是"当前有哪些文件被改动"的清单，**这部分数据完全现成，不需要新开发**，Reset 确认对话框可以直接复用 `Status()` 的结果展示"将丢弃以下 N 个文件的改动"。
2. **当前 HEAD 和目标 commit 之间，已经提交但会变成"从分支历史上消失"（虽然还在 reflog 里）的那些 commit**——这是 soft/mixed/hard 都应该展示的信息（不只是 hard 才需要，soft/mixed 虽然不丢工作区内容，但同样会让当前分支"回退"，用户需要知道会倒回哪些提交），当前代码**没有任何函数计算"HEAD 到目标 commit 之间有哪些 commit"**（这需要类似 `git log target..HEAD` 的调用，`internal/repo/repo.go` 里现有的 Log 查询函数（P7 已分析的 `LogQuery`）理论上可以复用来查询这个区间，但没有专门封装成"Reset 预览"的接口）。

技术判断：工作区/暂存区差异（丢失类型1）零新增成本，直接复用 `Status()`/`Compare()`；commit 区间差异（丢失类型2）可以**复用 P7 已分析的 Log 查询底层能力**（`git log <target>..HEAD` 和现有 `git log` 的 `LogQuery` 参数化方式接近，不是全新的 git 命令调用方式），但需要新写一个"Reset 预检"函数把这两部分数据拼起来，这个函数本身不存在，是中等工作量而不是"数据都现成了"。

## 5. 交互式 Rebase 的 UI 承载方式评估

**结论：当前代码里没有发现任何形式的拖拽重排实现，`docs/UI.md` 列出的 Viva 组件清单（`flex.Table`、`components.Tree` 等）里没有一个明确标注支持"拖拽重新排序行"能力，这是本次核查新确认的一个 Viva 缺口，此前三份文档（UI.md §8、P1、P7）都没有把它列为已知/已提交 issue 的缺口，和三方合并视图、Diff布局等已知缺口不是同一批。**

交互式 Rebase 需要的具体交互：
- 展示一个提交列表（最新在上或在下，取决于约定）
- 用户可以**拖动**某一行到列表中的另一个位置（reorder）
- 每行可以标记一个动作（pick/reword/edit/squash/fixup/drop），通常通过下拉菜单或快捷键切换，不一定需要拖拽
- squash/fixup 会把一行"合并"进相邻的另一行，视觉上需要体现"这几行会变成一个"

全局搜索当前代码：没有发现 `Drag`/`drag`/`Reorder`/`reorder` 相关的任何 Viva API 调用或自建逻辑，`docs/UI.md` 的 Viva 构件表（§3）列出的组件（`flex.Table`、`components.Tree`、`components.CodeDiff` 等）里，`flex.Table` 是 Log 视图在用的表格组件（P7 已分析过，只读虚拟化滚动），**没有看到关于它是否支持行拖拽排序的任何说明**，`docs/UI.md` §8"Viva 缺口"表格里没有列出"拖拽排序"这一项——这意味着目前谁都没有去确认过 Viva 是否支持这个能力，是一个尚未被提出的问题，不是"已知但没修"。

**三条可能路线**：
1. **如果 Viva 确实不支持拖拽排序**：需要先向 Viva 提 issue（遵循项目约定"可被第二个 Viva 应用复用的能力应提议给 Viva"），这和三方合并视图是同一类流程。等待期间可以用"上移/下移"按钮代替拖拽（每行一个↑↓按钮，点击与相邻行交换位置），这是功能等价但交互体验更笨重的替代方案，工作量小，纯按钮+列表重排逻辑，不需要新 Viva 能力。
2. **动作标记（pick/squash/fixup/reword/drop）**：这部分不需要拖拽，用下拉选择器或按钮组即可，`components` 包里应该有现成的选择控件（`f.Select` 在 Log 筛选等场景已经用到，P7提到过），工作量相对小。
3. **squash/fixup 的视觉合并提示**：需要在列表渲染时识别"这一行的动作是 squash/fixup"并调整其视觉呈现（比如缩进或连线表示"并入上一行"），这是纯应用层的渲染逻辑，不依赖新 Viva 能力。

**建议**：这个点需要和三方合并视图一样走"架构评估"流程，但优先级判断不同——三方合并视图在 P1 已经被标记为 P0（AcceptTheirs/Yours 可以先用路线A绕过，不阻塞冲突解决功能上线），而交互式 Rebase 本身就是 P1 标记的 P0 功能（`Git.Interactive.Rebase`），如果 Viva 确认不支持拖拽，"上移/下移按钮"的替代方案是能够不阻塞首发的可行路径，建议作为首发方案，拖拽留作后续体验改进。

## 6. 测试基础设施评估

**结论：构造"真实冲突场景"的测试辅助函数目前不存在，但现有的测试模式（临时仓库+系统 git 子进程）可以直接扩展，不需要新的测试框架或测试策略，只是缺少具体的辅助函数。**

现有可复用的模式（`internal/repo/repo_test.go:17-33`）：
- 每个测试用 `t.TempDir()` 建临时目录，`exec.Command(gitPath, args...)` 直接调系统 git（不是 mock），设置 `GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM` 指向临时路径隔离全局配置（避免读到开发机真实的 git 全局配置），`command()` 辅助闭包封装"跑 git 命令，失败就 `t.Fatalf`"。
- Hook 失败测试模式（`repo_test.go:171-191`）：直接往 `.git/hooks/pre-commit` 写一个 shell 脚本（`#!/bin/sh\necho rejected >&2\nexit 1\n`），设可执行权限，验证提交被拒绝。这个模式证明了"用真实文件系统操作模拟 Git 边缘场景"是这个项目既有的测试哲学，不是什么要重新设计的新东西。

构造合并冲突的测试辅助函数（目前不存在，但基于上述模式可以直接写）：
```
1. 建仓库，建文件 a.txt 内容 "line1\n"，commit（得到 commit X）
2. 建分支 branch-a，修改 a.txt 第一行为 "line1-a\n"，commit
3. 切回 X，建分支 branch-b，修改 a.txt 第一行为 "line1-b\n"，commit
4. 切到 branch-a，执行 git merge branch-b → 预期非零退出码，.git/MERGE_HEAD 存在，git status 里 a.txt 标记为 u (unmerged)
```
这是标准的"双分支改同一行"冲突构造手法，大约 10-15 行测试辅助代码（和现有 hook 测试的代码量级相当），**不是技术难点，是此前没人写而已**，工作量判断为小。Rebase 冲突、Cherry-pick 冲突的构造手法类似（用 rebase/cherry-pick 代替 merge，产生 `.git/rebase-merge`/`.git/CHERRY_PICK_HEAD` 而不是 `.git/MERGE_HEAD`），可以用同一套"双分支改同一行"的基础数据，只是触发冲突的命令不同，建议写一个共享的"构造冲突仓库"辅助函数，三种场景复用大部分逻辑，只在最后一步换成 merge/rebase/cherry-pick 命令。

## 7. 工作量分级与排期建议

**小工作量、技术确定（可以直接排期，不需要等待架构决策）：**
- Merge/Abort、Rebase.Abort、CherryPick.Abort、Revert.Abort：本质是"检测当前操作状态存在 → 执行对应的 `git merge/rebase/cherry-pick/revert --abort`"，不需要 Plan/Execute 复杂校验（中止操作本身风险较低，Git 自己会处理一致性），但**前提依赖第2节指出的"仓库级操作状态检测"先做出来**，这是唯一的前置阻塞点，不是 Abort 本身复杂。
- Reset（soft/mixed）：复用第1节的 Plan 模式（和 `PlanDeleteBranch` 结构几乎一样）+ 第4节已确认现成的 Status/Compare 数据源，只是 hard 模式需要额外接入 commit 区间查询（中等工作量的那部分）。
- AcceptOurs/AcceptTheirs/AcceptBoth（路线A的冲突简化操作）：复用现有 `CodeDiff` 两栏展示 + 三个新按钮，技术路径明确。
- Merge/Rebase 发起对话框（不含交互式 Rebase 本身）：复用 Plan/Execute 模式，和 `PlanCheckoutRemoteBranch` 结构接近。

**中等工作量，依赖其他前置项先完成：**
- Rebase.Continue/Skip、CherryPick.Continue：依赖仓库级操作状态检测（第2节）先做出来，之后的实现本身不复杂。
- Reset（hard）的完整数据丢失预览：依赖新的"commit 区间查询"函数（第4节）。
- Cherry-pick/Revert 主流程（非交互式部分）：Plan 阶段需要记录 commit 列表和顺序，比单 commit 操作的 Plan 结构复杂一些，但没有技术不确定性。

**大工作量，有技术不确定性，需要先做架构决策（不建议直接排期，需要用户先决定技术路线）：**
- 三方合并视图的可编辑 Result 面板（第3节路线C/D）：技术路线未定，建议先用路线A上线冲突解决能力，这部分作为后续迭代。
- 交互式 Rebase 的拖拽重排（第5节）：需要先确认 Viva 是否支持拖拽，如不支持是否接受"上移/下移按钮"替代方案；这个决策会影响整个交互式 Rebase 功能的设计起点，建议先做这个决策再开始写代码，不要边写边赌 Viva 的能力。

**阻塞性地基（必须最先做，没有先后商量空间）：**
- 仓库级操作状态检测（第2节：读取 `.git/MERGE_HEAD`/`rebase-merge`/`CHERRY_PICK_HEAD`/`REVERT_HEAD` 等）——几乎所有 P8 功能都直接或间接依赖这个状态才能正确工作（顶栏常驻状态指示、Continue/Abort 操作的前置检查、规格 §4.7"保留正在进行的 Git 操作状态"）。这应该是 P8 实际开工时的第一个任务，不是和其他功能并行的普通一项。

## 汇总：P8 待办清单（按工作量和技术不确定性分组）

**地基（最先做，阻塞后续一切）**
1. 新增仓库级操作状态检测：读取 `.git/MERGE_HEAD`、`.git/rebase-merge/`、`.git/rebase-apply/`、`.git/CHERRY_PICK_HEAD`、`.git/REVERT_HEAD`，解析出"当前处于哪种操作、涉及哪些 commit/进度"，扩展 `git.Status` 结构体或新增独立结构体承载这个信息，涉及 `internal/git/status.go` 新增函数，`internal/repo/repo.go` 接入。

**小工作量、技术确定**
2. Reset（soft/mixed）：复用 `PlanDeleteBranch` 式的 Plan/Execute 模式 + 现成的 `Status()`/`Compare()` 数据源。
3. Merge.Abort/Rebase.Abort/CherryPick.Abort/Revert.Abort：依赖第1条完成后，是简单的状态检查+执行。
4. AcceptOurs/AcceptTheirs/AcceptBoth（三方合并简化路线A）：复用现有 `CodeDiff` 两栏组件+新按钮+新 git 命令调用。
5. Merge/Rebase 发起对话框：复用 Plan/Execute 模式。

**中等工作量，有明确前置依赖**
6. Rebase.Continue/Skip、CherryPick.Continue：依赖第1条。
7. Reset（hard）完整数据丢失预览：需要新增 commit 区间查询函数（可复用 P7 已分析的 Log 查询底层能力）。
8. Cherry-pick/Revert 主流程：Plan 阶段需要记录 commit 列表和顺序，比单 commit 复杂但无技术不确定性。
9. 测试辅助函数：构造"双分支改同一行"的冲突仓库辅助函数（merge/rebase/cherry-pick 三种场景共享大部分逻辑），工作量小但本节单独列出因为是所有 P8 测试的前提。

**大工作量，技术不确定，需要先做架构决策**
10. 三方合并视图可编辑 Result 面板——建议先用条目4的路线A上线，不阻塞冲突解决功能；路线C/D作为后续迭代，决策权交给用户。
11. 交互式 Rebase 的 reorder 拖拽交互——需要先确认 Viva 是否支持拖拽排序（目前完全没人问过这个问题，不是已知缺口），如不支持，"上移/下移按钮"是功能等价的小工作量替代方案，建议作为首发方案而不是等待 Viva。squash/fixup/reword/drop 的动作标记本身（不含拖拽）是小工作量，可以和交互式 Rebase 的其他部分分开排期。
