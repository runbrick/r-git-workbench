# Rebased Commit/Shelf Parity 差异盘点

对应 `docs/ROADMAP.md` P4 任务。与 P1（`docs/REBASED_ACTION_INVENTORY.md`）、P2（`docs/REBASED_UI_PARITY_GAPS.md`）、P3（`docs/REBASED_SPLITTER_PERSISTENCE_GAPS.md`）配套，不重复已记录内容。

## 1. Amend 功能核实

**历史盘点结论已过期：Amend 控件现已接入 Commit 面板。**

- 后端：`internal/repo/repo.go:1166-1185`。`Commit(ctx, message)` 是 `CommitWithOptions(ctx, message, false)` 的薄封装；`CommitWithOptions` 在 `amend == true` 时正确拼接 `git commit -F - --amend`（`repo.go:1177-1179`）。后端逻辑正确、完整。
- UI 状态字段：`m.amend bool`（`shell.go:114`），在提交点击处理中被正确读取——`shell.go:1296-1301`（Commit 按钮）和 `1303-1309`（Commit and Push 按钮）都会判断 `if m.amend { kind = "amend" }`，然后调用 `m.write(kind, "", m.message.Text())`。提交成功后 `m.amend` 会被重置为 `false`（`shell.go:769`）。这条链路本身是对的、闭环的。
- UI 入口：`paintChangesSidebar` 使用 `f.Checkbox(amendID, m.amend)` 绘制可访问复选框，点击时更新 `m.amend`。Commit 和 Commit and Push 都读取该状态；Driver 测试检查控件存在并覆盖开关两种状态。
- 当前差异：基线还包含下拉和时钟图标，本应用目前提供等价的 amend 开关语义，视觉控件尚未完全匹配。

## 2. 提交失败恢复核实

**结论：草稿保留确实做到了，但是"失败路径没有清空消息"带来的保留，不是专门的失败处理分支。**

失败路径代码（`shell.go:767-779`）：
```go
if (update.Kind == "commit" || update.Kind == "amend") && update.Err == nil {
    m.message.SetText("")   // 仅成功时清空
    m.amend = false
    ...
} else if update.Kind == "commit" || update.Kind == "amend" {
    m.commitAndPush = false  // 失败时只重置 commitAndPush，消息文本不动
}
```

`m.message` 是独立字段，不在失败分支里被触碰，所以文本框内容自然保留——效果等价于规格要求，但没有显式的"恢复草稿"逻辑。

`m.amend` 在失败分支里**没有被重置**，由于没有 UI 能设置它，实际效果上这个字段永远是 `false`，这条路径目前测不出意义；将来接上 Amend UI 后，"失败重试时是否保持 amend 模式"需要重新验证，现状代码路径恰好会保留，但这是偶然结果，不是设计验证过的。

测试覆盖：`internal/repo/repo_test.go:171-191` 通过临时仓库 pre-commit hook 验证了"hook 拒绝提交后 HEAD 不变、暂存区文件不变"，这是后端层面的失败恢复验证，但**没有任何测试覆盖 UI 层"提交失败后消息文本框内容是否还在界面上"**——现有验证只到 `repo.Commit()` 返回 error 为止，没有延伸到 `shell.Model` 层面断言 `m.message.Text()` 未被清空。

## 3. Commit and Push 流程核实

**结论：状态机基本正确，但"提交成功、推送失败"这个常见场景完全没有测试验证，也没有专门的用户反馈。**

- 流程（`shell.go:773-776`）：提交成功（`update.Err == nil`）且 `m.commitAndPush == true` 时，重置 `commitAndPush` 标志并调用 `m.planPush()` 进入推送预检流程（和常规 Push 共用同一个预检/确认/执行管线，规格 §4.6 要求的预览-确认流程）。
- 提交失败（`update.Err != nil`）时，`commitAndPush` 被重置为 `false`（`shell.go:778`），不会触发推送——这是对的，提交失败不该尝试推送。
- 缺口：`m.planPush()` 之后的推送本身如果失败（网络、非快进等），用户体验上应该看到"提交已经成功，但推送失败，需要你决定下一步"这种明确提示，而不是和"从头推送失败"一样的通用错误展示。当前代码没有为"commit-and-push 组合里的推送失败"设计单独的错误文案/状态区分。这不是功能缺失（底层流程能跑通），是用户反馈粒度不足。
- 测试覆盖：`internal/ui/shell/changes_test.go:21 TestDriverOpenStageAndCommit` 只测基础 Stage+Commit 流程；`repo_test.go` 里的 push 相关测试都是独立 Push 测试，不是从 Commit and Push 组合入口触发的。**"提交成功后自动触发的推送"这条路径，从 UI 层到后端层都没有专门测试**，只能靠两段独立测试的逻辑推理它应该能工作。

## 4. 提交检查（hooks/identity）

**结论：hooks 这一半做对了（原样转发错误，符合规格"保留原始 Git 错误详情"），identity 检测完全不存在。**

- Hooks：后端完全不绕过 hooks——`CommitWithOptions` 直接调用系统 `git commit`，hook 的拒绝行为由 Git 自身处理。有真实测试验证（`repo_test.go:171-191`，写一个 exit 1 的 pre-commit hook，确认提交被拒绝且 HEAD/暂存区不变，错误信息含 Git 原始 "rejected" 文案）。错误处理是把 Git 的 stderr 原样包进 Go error（`repo.go:1182` `fmt.Errorf("commit: %w: %s", err, result.Stderr)`），没有专门识别"这是 hook 拒绝"还是"其他提交错误"——符合规格要求，不算缺口。
- Identity：全库搜索 `user.name`/`user.email`/`identity` 相关的**生产代码**（不含测试文件里为搭建测试仓库而配置的 `git config user.name`），**零匹配**。规格 §4.1"克隆完成后打开该仓库并检查 Git identity；缺失时提示设置 user.name 和 user.email"、规格 §6"Git identity：显示其来源（system/global/local/worktree）"均未实现，没有任何代码读取或校验 `user.name`/`user.email` 配置。这不是"UI 没画出来"的问题，是后端完全没有这个能力，属于两端都要做的缺口。

## 5. 提交范围与自动暂存设置

**结论：默认提交范围的核心语义是对的，但"没有暂存内容自动暂存"的设置项完全不存在，空暂存区边界情况未处理。**

- 当前 Commit 按钮点击后调用 `m.write(kind, "", m.message.Text())`（`shell.go:1301`，路径参数为空字符串）。后端 `git commit -F -`（`repo.go:1176`）**不带任何路径参数、不带 `-a`**，只提交当前已经在 index 里的内容（真正的"已暂存变更"），不会提交未暂存的已跟踪文件改动。这个默认行为符合规格 §4.2"默认提交所有已暂存变更"的字面要求，是正确的。
- 规格同一段还要求"若当前没有暂存内容，可按产品设置自动暂存所选文件，必须清楚展示实际提交范围"——这是条件触发的设置项，本次核查确认：
  - cetacean **没有任何设置项或开关**能控制这个行为，也没有"用户没有暂存任何内容、点击 Commit 时会发生什么"的专门处理。Commit 按钮在消息非空且 `CanWrite` 为真时就是可点击的（`shell.go:1296` 的 `Disabled` 条件不检查是否有暂存内容）。如果用户没有暂存任何文件、消息框写了字就点 Commit，会触发 Git 原生"nothing to commit"报错，这个边界情况没有专门测试验证错误提示是否清楚呈现给用户。
  - 没有自动暂存逻辑，也没有"清楚展示实际提交范围"的 UI（比如提交前显示"将提交 N 个已暂存文件"的统计）。P2 已指出"选中更改统计"这块 UI 缺失，这里进一步确认连后端触发自动暂存的能力都不存在，是两端都缺。

## 6. Shelf 语义决策建议

**明确建议：现有"Shelf" tab 的目标指向不需要改变（代码已经做对了），只需要把标签文字从"Shelf"改成"Stash"相关文案。**

推理链：
1. P1 盘点已经决定"不复刻 IntelliJ Changelist/Shelve"（因为它是 IntelliJ 平台概念，不是 Git 原生概念，复刻它违反"客户端是 Git CLI 的忠实呈现"这个产品原则）。
2. P2 盘点发现现状代码里"Shelf" tab 点击后跳转到 `keymap.ShowStash`——也就是说代码作者本人已经做出了和 P1 决策一致的选择：不建一个新的 Shelve 概念，而是把这个 tab 复用为 Stash 入口。这不是代码的错误，是提前对齐了后续会在 P1 里正式写下来的决策。
3. 矛盾点在于：Rebased 截图上这个 tab 字面文字是"Shelf"，而 cetacean 决定它实际指向的功能是 Git Stash。两个产品概念的用户心智模型不同（Shelf 是"搁置、可能永远不恢复的补丁"，Stash 是"临时收起工作区改动，几乎总是会恢复"）——继续用"Shelf"这个词但功能换成 Stash，会让熟悉 Rebased/IntelliJ 的用户发生误解。
4. 建议：**把这个 tab 的标签文字改为"Stash"相关文案**（对应 i18n key 需要新增或调整），保留现有"点击跳到 Stash 功能"这个已经做对的路由逻辑。不建议保留"Shelf"字样再让它指向语义不符的功能，也不建议真的实现一套独立的 Shelve 存储机制（违反 P1 已定的排除决策，且 Git 原生没有对应概念，实现成本不值得）。
5. 这个决策目前完全依赖 Stash 功能本身（P1 标记"完全未实现"）才能验证闭环，Tab 改名可以先做（纯文案），但"点击后看到实际 Stash 列表"要等 M8/P9 落地。

## 7. 快捷键 Ctrl/Cmd+K

**结论：完全未注册，且这不是 Commit 功能单独的缺口——整个应用目前没有任何快捷键系统。**

- `internal/keymap/actions.go` 全文只有 6 个 `ActionID` 常量（`ShowChanges`/`ShowLog`/`ShowBranches`/`ShowStash`/`ShowTags`/`ShowRepositories`），对应左侧导航的 6 个视图切换，**没有 Commit、Amend、Stage、Push 等任何具体操作的 ActionID**，自然也不存在"提交"动作的快捷键绑定。
- 全库搜索 `OnShortcut`（`docs/UI.md` §5 设计的接入点）：**零匹配**，`shell.go` 里没有调用这个 API。
- 规格 §5 整张快捷键表（Ctrl+K 提交、Ctrl+Alt+K 提交并推送、Ctrl+Shift+K Push、F7/Shift+F7 等约20个键位）都还没有接入任何实际处理代码，不是"Ctrl+K 这一个键缺失"，是"快捷键基础设施本身是 M9 待建项，目前是空的"。
- 补 Commit 快捷键不能只在 `shell.go` 加一行判断，需要先有 `keymap` 的动作注册和按键分发机制，这个基础设施缺口应该在 M9 或更早一次性解决，不建议为了 P4 单独给 Commit 功能打一个局部补丁快捷键。

## 8. 测试覆盖现状

| 场景 | 是否有测试 | 测试文件/函数 |
|---|---|---|
| 正常提交（Stage后Commit，状态刷新） | 有 | `internal/repo/repo_test.go:17 TestStageUnstageCommitAndStatus`；`internal/ui/shell/changes_test.go:21 TestDriverOpenStageAndCommit` |
| Amend 提交（后端 `--amend` 参数正确性） | 没有 | 全库搜索未发现任何针对 `CommitWithOptions(..., amend=true)` 的测试，后端能力存在但零测试覆盖 |
| 提交失败后草稿保留（UI 层，`m.message` 未被清空） | 没有 | 现有测试只验证到 `repo.Commit()` 返回 error，没有延伸到 `shell.Model` 断言消息文本框内容 |
| Hook 拒绝提交（后端） | 有 | `internal/repo/repo_test.go:171-191`（pre-commit hook exit 1，验证 HEAD/暂存区不变、错误信息含"rejected"） |
| Identity 缺失检测 | 没有（功能本身不存在） | 无相应代码，无法测试 |
| Commit and Push 组合流程（提交成功→自动推送） | 没有 | 独立的 Commit 测试和独立的 Push 测试都存在，但没有端到端覆盖"提交成功后自动触发推送"这条组合路径 |
| Commit and Push 组合流程里推送失败的处理 | 没有 | 同上，且代码本身也没有为这种情况设计专门反馈 |
| 空暂存区点击 Commit 的边界情况 | 没有 | 代码未做前置检查，行为依赖 Git 原生报错，未验证报错是否清楚呈现给用户 |
| 提交消息历史记录与选择 | 待确认 | `RecentCommitMessages`（`repo.go:1187`）有独立实现，本次未逐一核对是否有专属测试，标记为待确认而非直接判定缺失 |

## 汇总：P4 验收待办清单

区分"后端已支持只缺UI"和"后端也不支持需要两端都做"，按优先级列出。

**P0（功能性缺陷/缺口，直接影响用户能否用到已声明的功能）**
1. Amend 开关已接入并由 Driver 覆盖；剩余的是对齐基线的下拉/时钟视觉，以及独立验证 amend 后 HEAD 更新、未暂存内容不被带入和 hook 失败时模式/草稿保留。
2. **Identity 检测完全不存在** — 后端也不支持，需要新增读取 `git config user.name`/`user.email`（含来源 system/global/local/worktree 的判定）的逻辑，以及克隆/打开仓库后检测缺失并提示的 UI 流程，两端都要做。
3. **空暂存区点击 Commit 的边界情况未处理** — 需要在 Commit 按钮的 `Disabled` 条件里增加"是否有已暂存内容"的判断，或明确设计这种情况下的错误展示，目前完全依赖 Git 原生报错，用户体验未经验证。

**P1（功能基本可用，但完整性/体验有缺口）**
4. **"没有暂存内容时自动暂存"的设置项不存在** — 规格 §4.2 要求的可配置行为，需要在设置里补一个开关，并在 Commit 触发时按开关状态决定是否自动暂存，两端都要做（涉及 M10 设置范围）。
5. **提交范围展示缺失** — 提交前"清楚展示实际提交范围"（比如统计将提交的文件数）目前没有对应 UI，P2 已指出"选中更改统计"缺失，这里确认后端也没有现成的统计接口可直接复用，需要补。
6. **Commit and Push 组合流程里推送失败缺少专门反馈** — 当前复用通用推送错误展示，建议为"提交已成功但推送失败"设计区分于"从未提交成功"的提示文案，涉及 `shell.go:773-779` 附近状态机。
7. **Shelf tab 标签文字需要改名为 Stash 相关文案** — 保留现有"跳转到 Stash 功能"的路由逻辑（已经做对），只改显示文字，避免误导熟悉 Rebased 的用户；真正验证要等 Stash 功能（M8/P9）落地。

**P2（测试覆盖缺口，用于验证用，不是新功能）**
8. 补 Amend 提交的后端测试（验证 `git commit --amend` 参数、修改上一条提交消息/内容的效果）。
9. 补 UI 层"提交失败后消息文本框内容保留"的 Driver 测试。
10. 补 Commit and Push 端到端集成测试（临时裸仓库，覆盖"提交成功+推送成功"和"提交成功+推送失败"两种组合）。
11. 确认/补充提交消息历史（`RecentCommitMessages`）的专属测试覆盖。

**超出 P4 范畴但在核查中发现、需要记录的关联缺口**
12. Ctrl/Cmd+K 等全部快捷键未接入——根因是 `internal/keymap` 目前只有 6 个导航 ActionID，没有 Commit/Stage/Push 等操作级动作注册，也没有任何 `OnShortcut` 调用。这是 M9（快捷键基础设施）的范畴，不建议在 P4 里单独打补丁，但记录于此以提醒 M9 规划时需要覆盖 Commit 相关的全部键位（Ctrl+K / Ctrl+Alt+K / Ctrl+Shift+K 等）。
