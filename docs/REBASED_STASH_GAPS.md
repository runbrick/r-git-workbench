# Rebased Stash Parity 差异盘点

对应 `docs/ROADMAP.md` P9 任务。与 P1（`docs/REBASED_ACTION_INVENTORY.md`）、P2（`docs/REBASED_UI_PARITY_GAPS.md`）、P3（`docs/REBASED_SPLITTER_PERSISTENCE_GAPS.md`）、P4（`docs/REBASED_COMMIT_SHELF_GAPS.md`）、P5（`docs/REBASED_CHANGES_DIFF_GAPS.md`）、P6（`docs/REBASED_BRANCH_REFS_GAPS.md`）、P7（`docs/REBASED_LOG_GAPS.md`）、P8（`docs/REBASED_HISTORY_CONFLICT_GAPS.md`）配套，不重复已记录内容。本篇重点是从零设计评估——Stash 和 P1-P8 大多数"已有雏形但不完整"的情况不同，是真正的空白功能，需要更多设计性思考而不是重复确认"未实现"。

## 1. 后端能力现状（代码证据）

**结论：后端没有任何面向用户的 Stash 操作函数，唯一出现的 `stash` 字符串是远程 checkout 安全流程的内部实现细节，不是独立功能。**

全文搜索 `internal/repo/repo.go` 和 `internal/git/*.go` 的 `stash`/`Stash` 关键词，生产代码里只有三处命中，全部集中在 `ExecuteCheckoutRemoteBranch`（`repo.go:1085-1116`，P6 已分析过的远程分支安全 checkout 流程）：

```go
// repo.go:1101-1104
if mode == CheckoutStash {
    result, stashErr := r.runner.Run(ctx, gitexec.Command{Dir: r.Root, Args: []string{"stash", "push", "--include-untracked", "-m", "r-git-workbench: before remote branch checkout"}, Write: true})
    ...
}
```

这是"Checkout 前临时存放脏文件"的内部手段，调用了 `git stash push`，但：
- 没有把这次 stash 操作的结果（stash 引用、消息）返回给调用方或展示给用户；
- 没有任何后续的"查看/恢复这个 stash"入口——执行完 checkout 后，这条 stash 记录就留在仓库里，用户只能自己用命令行 `git stash list`/`git stash pop` 处理；
- 没有 `StashList`/`StashShow`/`StashApply`/`StashPop`/`StashDrop`/`StashBranch` 等任何独立封装函数，`grep "^func" internal/repo/repo.go | grep -i stash` 零匹配。

测试代码里 `repo_test.go:1011-1012`（`TestRemoteCheckoutPlansCancelStashAndForce`）用 `run("stash", "list")` 验证"确实生成了一条 stash 记录"，这只是验证 checkout 流程本身的副作用，不是对 Stash 功能的测试。

**后端现状：零。** 这和 P1 的断言一致，本节补充的是具体代码位置证据：唯一相关的代码是别的功能（远程 checkout）借用 `git stash push` 当临时工具，不构成任何可复用的 Stash API 雏形。

## 2. UI 层路由现状

`keymap.ShowStash` 在 `paintContent`（`shell.go:947-974`）里**没有专属的 `if` 分支**。对比其余四个导航目标（`ShowRepositories`/`ShowChanges`/`ShowLog`/`ShowBranches` 各自在 `shell.go:949,953,957,961` 有显式路由），`ShowStash` 和 P6 已发现的 `ShowTags` 一样，直接落入第 965-973 行的通用占位代码：

```go
f.Column().Fill().Pad(16).Gap(10).Children(func() {
    f.Text(text(keymap.Actions[m.Active].Name)).FontSize(15).Bold()
    f.Divider()
    f.Spacer()
    f.Row().FillWidth().Justify(flex.Center).Children(func() {
        f.Text(text(i18n.SelectRepository)).Muted()
    })
    f.Spacer()
})
```

**确认：这和 P6 第7节发现的 Tags 占位文案错误是完全同一处代码（`shell.go:965-973`），不是两个独立 bug，是一个代码位置同时服务于 `ShowStash` 和 `ShowTags` 两个导航项的共同缺口。** 标题行会显示对应的动作名称（"Stash"或"Tags"，来自 `keymap.Actions[m.Active].Name`），但下方主体内容固定显示 `i18n.SelectRepository`（"请选择仓库"），这段文字在用户已经打开仓库、主动点击 Stash 导航项时出现是错误的上下文提示。修复这个占位文案错误本身不需要等 Stash 功能实现，可以独立先做（但意义不大，因为很快就要被真正的 Stash 面板替换）。

另外一处：`m.Active = keymap.ShowStash` 的赋值发生在 `shell.go:1251`（Commit/Shelf 侧栏的"Shelf" tab 点击处理），这是 P4 已经分析确认的路由逻辑（正确，只是标签文案待改）。

## 3. Stash 面板设计空间评估（布局/数据模型）

### 3.1 布局：可以复用 Log 视图的"列表+详情"模式

规格 §4.9 要求的信息（消息、建立时间、分支、涉及文件、Diff 查看、创建分支、删除）和 P7 已分析过的 `paintLog` 结构（`shell.go:1581` 起：左侧提交表格 + 右侧详情分栏，`logDetailsSplit` 分隔）在形态上高度重合——Stash 列表的每一行本质上和 Log 的每一行一样，是"一个带消息、时间、相关文件的条目"，右侧详情区需要展示 Diff（P7 已确认 Log 详情区的 Diff 展示用 `CodeDiff` 组件，这部分可以直接复用）。

建议的 Stash 面板结构：
- 左侧：一个简化的表格/列表（不需要 Log 那样的 commit graph 图形列，Stash 条目之间没有父子关系，只有线性的 `stash@{N}` 索引顺序），列：消息、建立分支、相对时间（"2 hours ago"风格，规格要求"建立时间"）。
- 右侧：选中条目的详情——涉及文件列表（P5 已分析过 `paintChangesTree`/`paintGroup` 的文件列表渲染逻辑，这里需要的是只读版本）+ Diff（复用 `CodeDiff`）。
- 操作按钮：Apply / Pop / Drop / Branch（创建分支）/ Save（新建，需要弹出消息输入+untracked勾选的小表单，而不是常驻输入框，因为 Stash 的"保存"动作发生在 Changes 视图触发，不是在 Stash 视图本身触发——这是一个需要确认的产品细节：规格§4.9说"保存时可选择包含未跟踪文件，输入消息"，但没说这个"保存"按钮应该出现在 Stash 视图还是 Changes 视图，Rebased 基线截图（P2已分析）没有展示Stash相关UI，这是一个需要后续补截图或用户决策的空白点，本次盘点不替用户做这个决定）。

**技术判断**：不需要从零设计一套新的列表+详情布局范式，可以把 `paintLog` 的整体结构（表格 `flex.TableState` + `flex.Splitter` 分隔详情）作为起点裁剪，去掉图形列和分页逻辑（Stash 条目数量通常很少，不需要 P7 分析过的分批加载机制），这比"参考 Changes 视图的树形文件列表"更贴近需求，因为 Stash 列表的每一行是独立条目而不是文件树节点。

### 3.2 数据模型：和 `git.Commit` 有结构相似性，但需要新结构体而非直接复用

`git stash list --format=...` 的典型输出（每条 stash entry 底层确实是一个特殊的 commit 对象，有 SHA、消息、作者、日期），和现有 `git.Commit` 结构体（`internal/git/log.go:9-15`：`SHA/Parents/Author/Date/Subject`）字段高度重叠。`ParseLog` 的 NUL 分隔字段解析逻辑（`log.go:17` 起）大概率可以直接复用于解析 `git stash list` 的输出（只需要换一下 Git 命令和 `--format` 字符串）。

但 Stash entry 比普通 commit 多出的信息需要额外字段或额外查询：
- **stash 引用本身**（`stash@{0}`、`stash@{1}`...）——这是操作时真正需要的标识符（apply/pop/drop 都要指定 `stash@{N}`），普通 `git.Commit` 没有这个概念，需要新增字段。
- **建立时的分支名**——`git stash list` 的默认消息格式是 `WIP on <branch>: <subject>` 或 `On <branch>: <message>`，分支名是消息文本的一部分，不是独立字段，需要解析消息文本提取（或用 `git stash list --format` 的其他占位符，需要查证 Git 版本支持情况）。
- **涉及文件**——需要对每条 stash 额外调用 `git stash show <stash@{N}> --name-only`（或 `--stat`），这是和 `git log` 不同的地方：`git log` 配合 `--name-only` 可以在同一次调用里拿到每个提交的文件列表，但 `git stash list` 本身不直接带文件列表，需要 N+1 次调用（1次list + N次show），或者用 `git stash list -p` 一次性拿到所有 patch（规格要求"支持查看Stash Diff"，这个倒是可以复用 P7/P5 已经分析过的 patch 解析逻辑）。

**技术判断**：建议新增一个独立的 `git.StashEntry` 结构体（不是复用 `git.Commit`），字段包括 `Ref`（stash@{N}）、`Message`、`Branch`、`SHA`、`Date`，解析逻辑可以参考 `ParseLog` 的实现模式（NUL分隔、Go字符串处理方式）但不是直接类型复用，因为字段集合不同。涉及文件和 Diff 的获取建议采用"选中某条 stash 时才按需查询"的懒加载模式（和 P7 已分析的 Log 详情区"选中提交后异步读取完整消息和文件路径"是同一个既有模式，可以直接套用这个异步收件箱模式）。

## 4. Apply/Pop 冲突与 P8 地基的关系

**关键发现：Stash apply/pop 冲突和 P8 分析的 merge/rebase 冲突走的是不同的仓库状态标记机制，P8 的"仓库级操作状态检测"地基不能直接覆盖 Stash 场景，需要专门处理。**

`git stash apply`/`git stash pop` 如果产生冲突：
- **不会**创建 `.git/MERGE_HEAD`、`.git/rebase-merge/` 等 P8 第2节列出的那些标记文件/目录（`git stash apply` 内部用的是三方合并算法，但它不是通过"真正的 git merge 命令"触发的，所以不会留下 merge 进行中的标准标记）。
- **会**把冲突文件标记为 unmerged 状态（porcelain v2 的 `u` 前缀），这部分和 P5 已确认的 `file.Conflict = item[0] == 'u'` 识别机制是通用的，**可以直接复用**——这是 Git 冲突标记体系里跨场景一致的部分。
- **但是**：`git stash pop` 失败（冲突）时，Git 的实际行为是**不会自动 drop 这个 stash 条目**（文档行为：如果 apply 阶段失败，pop 会停在"冲突已标记，stash 条目仍保留在列表里"的状态，不会像成功时那样自动 `git stash drop`）。这意味着客户端不需要额外记录"哪个 stash 引发了当前冲突"——`git stash list` 在冲突发生后仍然能看到那条未被删除的记录，用户解决冲突后可以手动 `git stash drop <ref>` 清理。

**客户端需要解决的具体问题**：用户看到"当前有冲突文件"时，如何知道"这是因为我刚才点了 Apply/Pop 某个 stash，而不是其他原因产生的冲突"？因为没有 `.git/MERGE_HEAD` 这种持久化标记，这个关联关系**不能靠读取 .git 目录状态恢复**（如果用户关掉应用重新打开，没有任何文件告诉客户端"上次是 stash pop 失败"）。可行方案：
1. 客户端在触发 Apply/Pop 操作时，自己在内存/会话状态里记录"当前这次操作是 stash apply，操作对象是 stash@{N}"，操作返回非零退出码时，把这个上下文带到冲突提示里（"应用 Stash 'xxx' 时发生冲突，解决后可以..."）。这是会话级记忆，不是读取 Git 持久化状态，和 P8 地基的"读取 .git/MERGE_HEAD 判断进行中状态"是完全不同的技术路径。
2. 如果应用重启或关闭后再打开，没有"进行中的 stash 操作"这个概念可恢复（不像 merge/rebase 那样可以在重新打开应用后看到"有一个未完成的 merge"），客户端只能看到"有冲突文件"这个事实，但无法准确知道这是 stash 冲突还是其他原因留下的冲突（比如用户手动 `git stash apply` 后开着终端没解决就关闭了应用）——**这是一个产品行为上需要明确的点**：规格§4.9"發生衝突時保留 Stash 並進入衝突流程"，这句话暗示的是"同一次会话内，操作触发后立刻进入冲突流程"，不要求跨会话持久化这个关联，本次盘点按这个理解处理，但建议在实现前和用户确认一次这个边界（如果重启应用后有残留冲突文件，客户端应该展示"有未解决的冲突"的通用提示，不强行猜测是不是stash引起的）。

**技术判断**：Apply/Pop 的冲突检测本身可以复用 P5 已有的 `file.Conflict` 机制（不需要新代码），但"关联到具体是哪个 stash 操作触发的"需要客户端自己维护会话级状态，这部分不在 P8 地基的覆盖范围内，是 P9 需要单独设计的小块状态（一个 "pendingStashOperation" 之类的临时字段，操作成功或用户确认已解决后清除）。

## 5. Untracked 文件选项的现有模式参考

`git.File` 结构体已经有独立的 `Untracked bool` 字段（`shell.go:1281-1282` 等多处使用，比如 `paintGroup(f, i18n.UntrackedFiles, false, func(file git.File) bool { return file.Untracked })`），这和 `ignored`（`m.ignoredPaths []string`，P5 第4节已分析过的只读列表，来自单独的 `git status --ignored` 查询）是两个独立概念，**代码里没有混淆风险**——`Untracked` 是 `git.File` 的字段（来自常规 `git status --porcelain` 的输出），`ignored` 是完全独立的查询结果（需要显式加 `--ignored` 参数才会返回，默认不显示）。

对 Stash Save 的"是否包含未跟踪文件"选项：`git stash push --include-untracked` 这个参数本身的语义和客户端已经掌握的 `file.Untracked` 概念是直接对应的——保存时的勾选框可以直接基于"当前 Untracked 分组里有多少文件"来提示用户（比如"包含 N 个未跟踪文件"），技术上不需要新的数据源，只需要在触发 `git stash push` 时根据勾选状态决定是否带 `--include-untracked` 参数。P6 已分析的 `ExecuteCheckoutRemoteBranch` 里的 stash 调用已经硬编码带了这个参数（`--include-untracked`，`repo.go:1102`），证明这条命令路径本身是通的，Stash Save 功能可以复用相同的参数模式，只是需要做成用户可选而不是硬编码。

## 6. 测试场景设计

参考 P8 第6节已经评估过的"构造合并冲突"测试辅助函数模式（临时仓库+真实git子进程，不mock），构造 Stash apply 冲突的测试思路：

```
1. 建仓库，建文件 a.txt 内容 "line1\n"，commit
2. 修改 a.txt 第一行为 "line1-stashed\n"，不commit，执行 git stash push（得到 stash@{0}）
3. 工作区此时恢复到commit时的内容（干净）
4. 再修改 a.txt 第一行为 "line1-conflict\n"，commit（新的commit Y）
5. 执行 git stash pop（或 apply）→ 预期非零退出码，a.txt 标记为 u (unmerged)，stash@{0} 仍保留在 stash list 里（因为是pop失败的场景）
```

**和 P8 的冲突构造手法的相同点**：都是"两处改动落在同一行"的经典冲突构造模式，都用真实临时仓库+系统git子进程，不需要mock。

**不同点**：P8 的 merge/rebase 冲突需要构造两个分支分叉再合并；Stash 冲突只需要单分支上的"先stash、再在原位置继续改、再pop回来"这个时序，不需要建分支，构造步骤更少（大约8-10行，比P8估算的merge冲突构造10-15行略简单）。另外需要补充验证"pop失败后stash条目未被删除"这个Git特有行为（`git stash list` 仍能看到该条目），这是Stash测试场景独有的断言，merge/rebase测试不需要验证类似的"保留"行为。

## 7. 工作量分级与依赖关系

**地基（必须先做，Stash功能自身的前提）**
- 后端 `git.StashEntry` 数据结构 + `StashList`/`StashShow` 解析函数（第3.2节）——从零设计，参考 `ParseLog` 的实现模式但是新结构体，工作量中等。
- 修复 `ShowStash`/`ShowTags` 共享的占位文案 bug（第2节，和P6是同一处代码）——工作量极小，但建议和真正的Stash面板一起做，不单独提交一个"从错误占位文案改成另一个占位文案"的中间状态。

**小工作量、技术确定（不依赖P8地基）**
- Stash Drop/Clear——简单的 `git stash drop`/`git stash clear` 调用+确认对话框，不涉及冲突。
- Stash Save（不含untracked选项的最简版本）——`git stash push -m <message>`，技术路径明确。
- Save时的untracked勾选——复用现有 `file.Untracked` 概念（第5节），工作量小。

**中等工作量，依赖地基**
- Stash 列表+详情面板——依赖第3.2节的数据模型先做出来，布局可参考 `paintLog` 裁剪（第3.1节），比从零设计省工作量但仍需要新UI代码。
- Stash Branch（从stash创建分支，`git stash branch <name> <stash>`）——技术简单，但依赖列表面板先有选中态。

**中等工作量，需要和P8协调但不是直接依赖**
- Apply/Pop 冲突流程——冲突检测本身复用P5现成机制（不依赖P8），但需要新增第4节分析的"会话级stash操作上下文记忆"，这是P9独有的状态，不在P8地基范围内，建议P9和P8的冲突解决UI（如果P8已经做出通用的冲突清单/三方视图）共享展示层，但触发来源的记忆逻辑要P9自己做。

**需要用户决策的产品细节（本次盘点不代为决定）**
- Stash Save 的触发入口应该在 Changes 视图还是 Stash 视图（第3.1节已提出，Rebased 基线截图没有展示这部分UI，无法直接对照）。
- 跨会话重启后残留冲突文件的提示文案是否需要明确"不确定是否由stash引起"（第4节已提出的边界场景）。

## 汇总：P9 待办清单

1. **（地基）新增 `git.StashEntry` 结构体和 `StashList`/`StashShow` 解析函数** —— `internal/git/` 新文件，参考 `ParseLog` 实现模式但字段不同（第3.2节）。
2. **（地基）后端新增 `StashSave`/`StashApply`/`StashPop`/`StashDrop`/`StashClear`/`StashBranch` 函数封装** —— `internal/repo/repo.go`，目前零实现（第1节），`StashSave` 需要支持untracked可选参数。
3. **Stash 列表+详情 UI 面板** —— 参考 `paintLog` 结构裁剪（去图形列、去分页），`internal/ui/shell/shell.go` 新增 `paintStash` 函数并接入 `paintContent` 路由（替代当前第965-973行的错误占位，同时修复Tags的占位——但Tags的真正面板是P6/P10范围，这里只需要让Stash有专属路由，不强行把Tags也一起修）。
4. **Stash Save 触发入口和消息/untracked表单** —— 需要先向用户确认入口位置（Changes视图 vs Stash视图），见第3.1节、第7节"需要用户决策"部分。
5. **Apply/Pop 冲突的会话级上下文记忆** —— 新增一个临时状态字段记录"当前冲突是否来自stash操作、具体是哪个stash"，第4节已分析这不能复用P8的持久化状态检测机制，需要P9单独设计。
6. **测试**：补充 Stash save/apply/pop/drop/branch 的真实仓库测试，以及 Stash apply 冲突场景的测试（第6节已给出构造步骤），验证"pop失败后stash条目保留"这个Git特有行为。
7. **UI细节**：确认"Shelf"标签改"Stash"的文案改动（P4已建议，待P9一起落地时顺带执行，不需要单独提交）。
