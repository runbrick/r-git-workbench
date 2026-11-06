# Rebased Changes/Diff Parity 差异盘点

对应 `docs/ROADMAP.md` P5 任务。与 P1（`docs/REBASED_ACTION_INVENTORY.md`）、P2（`docs/REBASED_UI_PARITY_GAPS.md`）、P3（`docs/REBASED_SPLITTER_PERSISTENCE_GAPS.md`）、P4（`docs/REBASED_COMMIT_SHELF_GAPS.md`）配套，不重复已记录内容。`docs/UI.md` §8"Viva 缺口"已记录的框架限制（Diff 上下排列、hunk 勾选、行内字符级改动高亮、改动导航，均因 `CodeDiff` 组件能力不足，已提给 Viva [#131](https://github.com/vivaui/viva/issues/131)）直接引用，不重复列为新发现。

## 1. 冲突置顶

**结论：没有做到。冲突分组不是物理置顶，而是按文件路径字母顺序散落在列表中。**

`paintChangesTree`（`shell.go:1485-1567`）先把 `m.status.Files` 的索引按 `Path` 字母顺序排序（`shell.go:1490`），再遍历时才判断 `file.Conflict` 给出 `group = -1`（`shell.go:1502-1503`）。排序键是路径字符串，不是分组序号，所以冲突文件只是在遍历到它字母位置时临时插入一个"Conflicted"小标题，不会被提到列表最前面。扁平分组视图（`paintGroup` 系列，`shell.go:1279`）倒是先调用 `m.paintGroup(f, i18n.ConflictedFiles, ...)` 再调用其余分组（`shell.go:1279-1282`），这条路径下冲突确实置顶。

**两种视图模式对"冲突置顶"的实现不一致**：扁平视图做到了，树视图没做到。这是本次核查发现的新问题，不在 P1/P2/P4 任何一份文档里。

## 2. 树/扁平视图

树视图实现（`paintChangesTree`，`shell.go:1485-1567`）：按路径排序后，逐层检测目录前缀（`shell.go:1513-1537`），用 `m.expandedPaths map[string]bool` 记录展开状态，默认展开（`shell.go:1527-1529`：第一次见到某前缀时直接置 `true`）。文件行缩进用 `depth*12` 像素（`shell.go:1541,1558`）。

已知问题：
- 除第1节提到的冲突不置顶外，目录折叠状态 `m.expandedPaths` 是 `Model` 瞬时字段，没有持久化（和 P3 盘点的结论一致，不是本节新发现，只是再次确认具体受影响的字段）。
- 文件数统计：树视图标题行（`shell.go:1510`）只画分组文字 `title`，不带文件计数，和 P2 已指出的"Changes N files 没有复选框"是同一处代码但不同侧面——这里额外确认**树视图模式下连文件数字都没有**，扁平视图的 `paintGroup`（不在本次读取范围，但 P2 已引用其输出格式为 `"%s  %d"`）才有数字。两种模式的分组标题信息密度不一致。
- 深层嵌套：代码逐级创建 `change-folder-<prefix>` 按钮 ID（`shell.go:1526`），理论上支持任意深度，未发现硬编码深度上限，这部分没有发现 bug。

## 3. 分组动作

**结论：不存在任何分组级批量操作。**

全文搜索未发现"暂存本组全部文件"之类的处理函数——`paintGroup` 和 `paintChangesTree` 都只提供逐文件选中后在 `paintChangesDiff` 里对单个 `m.selected` 文件执行 Stage/Unstage/Discard（`shell.go:1347-1422`）。P1 已经把 `Git.Stage.Add.All`/`Git.Stage.Add.Tracked`（全部暂存/全部暂存已跟踪）标记"未实现"，P2 指出分组标题缺复选框；这里确认的是：**即使将来补上复选框，当前也没有任何后端调用路径能接收"批量文件列表"**——`r.Stage`/`r.Unstage`（`repo.go`）的调用点全部是单路径调用（`shell.go:1348,1351`），没有循环批量调用的封装函数，是纯粹的功能空白，不是"只缺UI勾选"这么简单。

## 4. Ignored 展示

`paintIgnored`（`shell.go:1569-1579`）：只在 `m.restoreIgnored == true` 且 `len(m.ignoredPaths) > 0` 时渲染一个只读列表（`f.Text(ignored).Muted()`），每行用 `Hit` 注册点击区域但只在 `shell.go:1575` 看到 `Hit(id, label)`，**没有看到对应的 `.Clicked()` 处理**（搜索 `ignored-paths` 相关点击逻辑未找到）。即点击 ignored 文件目前没有任何动作，是纯展示。

规格或 Rebased 基线里"从 ignored 恢复为 tracked"、"取消 ignore（编辑 .gitignore/exclude）"等交互（P1 已标记 `Git.OpenExcludeFile` 未实现）在这里确认：ignored 列表连"点击查看这个文件为什么被忽略"这类基础交互都没有，是完全只读、不可操作的列表。

## 5. Hunk 级暂存与行级暂存边界

**结论：ROADMAP M5 记录的"用 hunk 按钮选择而不是逐行内联勾选"现在仍然成立，且边界清晰——不是 Viva 框架限制，是应用层确实只做到 hunk 粒度，没有写行级逻辑。**

实际代码（`paintChangesDiff`，`shell.go:1366-1414`）：
- `m.hunks`（类型来自 `git.ParseHunks`，`shell.go:593`）是整块 hunk 的列表，每个 hunk 对应 Diff 里的一个 `@@ ... @@` 块。
- UI 只提供"整块 hunk 勾选"按钮（`shell.go:1376`，`hunkScope.At(index)`），勾选后通过 `applyHunksID`/`restoreHunksID` 对选中的若干整块 hunk 调用 `r.ApplyHunks`（`repo.go` 层，签名见 `session.go:169`：`r.ApplyHunks(ctx, path, selections, ..., true)`）。
- 全文搜索没有发现任何"单行"粒度的选择状态字段或按�ウ（没有类似 `m.selectedLines` 的结构），确认当前**连选择单行的 UI 都不存在**，不是"选了单行会自动扩展为整块 hunk"这种有行为但不精确的实现，是行级交互入口完全没写。
- 这确实部分撞上 `docs/UI.md` §8 记录的 Viva 框架限制（`CodeDiffRow` 无行内范围，行号栏无命中区域），但当前的 hunk 勾选 UI（`shell.go:1366-1414`）本身并不是用 `CodeDiff` 组件画的，而是独立的按钮列表（`f.Button(hunkScope.At(index), ...)`），和 `CodeDiff`/`m.codeDiff` 组件是分开的两套东西（对照第11节）。这意味着即使 Viva 修复了 `CodeDiffRow` 的行内命中问题，cetacean 现在的 hunk 按钮 UI 也不会自动获得行级能力，需要应用层重新设计这部分交互，不能完全归咎于框架限制。

## 6. 局部撤销

`restoreHunksID` 对应的是"取消选中的若干个 hunk 的工作区改动"（`shell.go:1407-1412`，调用 `PlanRestoreHunks` → 预检 → 确认对话框 `shell.go:1462-1481` → `RestoreHunks` 执行），走的是规格要求的"预检→确认→执行"流程，且执行前用补丁指纹校验（`Expected: fmt.Sprintf("%x", sha256.Sum256(...))`，`shell.go:1384`，防止预检和执行之间内容已变化）。

测试覆盖：`internal/repo/repo_test.go:368 TestApplyMultipleHunksAndRestoreSelectedHunk` 真实验证了多 hunk 暂存和局部还原，`repo_test.go:205 TestApplyHunkStagesAndUnstagesOnlySelectedChange` 验证了单 hunk 场景。**这部分实现完整、测试覆盖到位**，是 P5 列表里相对成熟的一项，和第5节"没有行级能力"是两个不同粒度的结论，不要混淆。

## 7. 三版本比较

当前支持的两两比较：
- HEAD↔暂存区（`staged == true` 时，`Compare` 函数 `repo.go:679-703` 用 `HEAD:old` 和 `:path`（index 版本）两侧）
- HEAD↔工作区（`staged == false` 时，`repo.go:704-734` 用 `:path`（index，当 untracked 时跳过）和磁盘文件内容两侧——注意这条路径实际比较的是"index 版本 ↔ 磁盘工作区"，当文件已经被 staged 过又在工作区改动时才是真正的三者关系中的一段，当文件完全未 staged 时 `comparison.Left` 来自 index（等同 HEAD，如果 HEAD 和 index 一致），逻辑上覆盖了规格要求的"HEAD↔工作区或HEAD↔暂存区，可切换查看 staged 与 unstaged 差异"）
- `toggleDiffID` 按钮（`shell.go:1358-1363`）在两种对比间切换，这是规格 §4.3"可切换查看 staged 与 unstaged 差异"的实现。

未支持：
- **工作区↔暂存区**直接对比（不经过 HEAD）——当前两条路径都以 HEAD 或 index 为一侧基准，没有发现"直接比较 index 内容和磁盘工作区内容"这一第三种组合的独立入口（虽然 unstaged 对比路径事实上部分覆盖了这个语义，但没有作为显式选项呈现）。
- **三版本同屏**：全文搜索没有发现任何"三栏布局同时显示 HEAD/Index/Worktree"的代码雏形，和 P1 已标记 `Git.Stage.Compare.Three.Versions`"未实现"的结论一致，这里确认连局部雏形都不存在，是彻底的空白而非部分实现。

## 8. 外部打开

全文搜索"外部"/"External"相关功能：**零匹配**。规格 §4.2"双击在外部编辑器打开"、规格 §2"在外部终端打开"（P2 已指出左侧轨道缺终端图标）均未实现。双击文件当前没有看到任何特殊处理（`paintChangesTree`/`paintGroup` 的文件行只注册了单击 `Hit`/`.Clicked()`，`shell.go:1562-1564`），没有双击区分逻辑，这意味着"双击打开外部编辑器"这个交互在当前实现里连双击事件本身都没有被区分处理。

## 9. 路径复制

全文搜索"复制路径"/"CopyPath"：**零匹配**。P1 已标记 `CopyPathFromRepositoryRootProvider` 未实现，这里确认 Changes/Diff 界面没有任何形式的路径复制入口（不是按钮缺失，是连右键菜单、快捷键或其他替代形式都不存在——因为 P4 已确认整个应用没有右键菜单和快捷键系统）。

## 10. 差异导航

F7/Shift+F7"下一处/上一处改动"：和 P4 已确认的"整个应用没有快捷键系统"结论一致，这里额外确认**没有非快捷键的替代按钮**——`paintChangesDiff` 的工具栏（`shell.go:1326-1337`、`1346-1365`）里没有任何"上一处/下一处"图标按钮。这个功能在当前 UI 里完全没有任何形式的入口，键盘和鼠标都不行。

## 11. Diff 显示选项（空白忽略/方向切换等）

实际 Diff 工具栏内容盘点（`paintChangesDiff`，`shell.go:1326-1365`）：
- 有：显示/隐藏 ignored 文件开关（`ignoredToggleID`，`shell.go:1328`，这是 Changes 列表级开关，不是 Diff 本身的开关，但物理上画在同一行工具栏里）、Stage/Unstage 按钮、Staged/Working Diff 切换（`toggleDiffID`）。
- 没有：忽略空白差异开关（规格 §4.3 明确要求）、复制路径（第9节已确认）、外部打开（第8节已确认）、面板方向切换（左右/上下，这部分属于 `docs/UI.md` §8 已记录的 Viva 框架限制，`CodeDiff` 只支持左右并排，不是本节新发现）。

两套渲染路径的工具栏能力不对等：`m.showComparison == true` 时走 `components.CodeDiff` 渲染（`shell.go:828` 一带的 `m.codeDiff.Paint`），`== false` 时走纯文本逐行渲染（`shell.go:1427-1435`，`f.Text(line).Mono()`）。本次核查确认**纯文本路径没有任何工具栏能力**（不能忽略空白、不能复制、不能切换方向，因为它根本不是 `CodeDiff` 组件），而这条路径恰恰是二进制/大文件会走的路径（见第12节），等于大文件/二进制文件的 Diff 体验比普通文本文件更简陋，没有任何工具栏功能。

## 12. 大/二进制文件处理

机制（`repo.go:673-737`，`Compare` 函数）：
- 阈值统一为 512 KiB（`repo.go:658,725,749` 三处 `512<<10`），超过阈值或检测到 NUL 字节（`strings.IndexByte(..., 0) >= 0`，`repo.go:736`）即标记 `comparison.Binary = true`。
- **关键发现**：无论最终是否判定为 Binary，`Compare` 都会先无条件调用 `r.Diff`（`repo.go:674`，内部执行 `git diff --binary`，`repo.go:631`）得到 `patch` 字符串，赋给 `comparison.Patch`（`repo.go:678`）。这个 `patch` 字符串后续原样作为 `update.Diff`/`m.diff` 显示给用户（`session.go:144`→`shell.go:591`）。
- 对真正的二进制文件，`git diff --binary` 的标准输出是类似 `Binary files a/x and b/x differ` 这一行文字（或者 `--binary` 模式下的 base64 补丁块，取决于 Git 版本和配置），这段文字会被 `paintChangesDiff` 的纯文本分支（`shell.go:1427-1435`）原样逐行显示。
- **结论**：ROADMAP M1 进度记录"二进制与超过 512 KiB 的文件回退到 Git patch"字面准确——但这不等于规格 §4.3"二进制文件顯示類型、大小和版本信息"的要求。当前展示的是 Git diff 命令的原始输出文本，**不是结构化的"文件类型/大小/版本"信息**（没有代码读取文件的 MIME 类型或大小再拼成专门的展示文案）。这是一个未被此前三份文档记录的新缺口：规格要求的二进制文件展示和实际展示的内容形式不同。
- 超过阈值但本身不是二进制的大文本文件（比如一个 600 KiB 的日志文件被修改）同样会被这套逻辑标记为 `Binary = true`，然后用户看到的也是 `git diff --binary` 的输出（对于文本文件，`--binary` 模式下 Git 仍然会给出正常的文本 diff，不会强制转成二进制格式），所以"大文件"分支实际上退化成了"仍然显示 patch 文本，只是不走 CodeDiff 并排渲染"，这一点和 M1 记录一致，不算新发现，但第11节已指出这条路径没有任何工具栏能力，是组合起来的新问题。

## 13. 测试覆盖现状

| 场景 | 是否有测试 | 测试文件/函数 |
|---|---|---|
| Hunk 暂存/取消暂存（单hunk） | 有 | `internal/repo/repo_test.go:205 TestApplyHunkStagesAndUnstagesOnlySelectedChange` |
| 多 Hunk 暂存 + 局部还原 | 有 | `internal/repo/repo_test.go:368 TestApplyMultipleHunksAndRestoreSelectedHunk` |
| Discard 预检与执行 | 有 | `internal/repo/repo_test.go:279 TestDiscardPreflightAndExecution` |
| 合并提交的文件 Diff（父提交比较） | 有 | `internal/repo/repo_test.go:612 TestMergeCommitDetailsAndFileDiff` |
| Stage/Commit 基础流程（UI Driver） | 有 | `internal/ui/shell/changes_test.go:21 TestDriverOpenStageAndCommit` |
| 冲突文件分组展示顺序（树视图 vs 扁平视图） | 没有 | 本次核查第1节发现的不一致，未见任何测试验证冲突置顶行为 |
| 树视图深层嵌套目录展开/折叠 | 没有 | 未找到针对 `paintChangesTree` 折叠交互的专门测试 |
| Ignored 文件点击行为 | 没有（因为功能本身是纯展示） | 无相应代码，无法测试 |
| 行级暂存 | 没有（功能不存在） | 无相应代码 |
| 三版本比较/工作区↔暂存区直接对比 | 没有（功能不存在） | 无相应代码 |
| 外部打开、路径复制、差异导航 | 没有（功能不存在） | 无相应代码 |
| 二进制文件 Diff 展示内容 | 没有 | 未找到验证二进制文件时 `m.diff` 实际内容格式的测试 |
| 大文件（>512KiB 非二进制）Diff 展示 | 没有 | 同上，阈值触发后的用户可见效果未被测试断言 |
| Diff 工具栏空白忽略/方向切换 | 没有（功能不存在） | 无相应代码 |

## 汇总：P5 验收待办清单

区分"后端已支持只缺UI"、"纯UI缺失"、"两端都要做"、"Viva框架限制（已知，不是新发现）"四类。

**后端已支持只缺UI**
1. 树视图模式下冲突分组不置顶（扁平视图已做对）——需要把 `paintChangesTree` 的遍历顺序改成先过一遍 conflict 分组再遍历其余文件，而不是纯路径排序后在遍历中临时插入标题，涉及 `shell.go:1485-1512`。

**纯UI缺失（后端有能力支撑或不需要新后端能力）**
2. Ignored 文件列表点击无任何动作——`shell.go:1575`，`Hit` 注册了但没有 `.Clicked()` 处理，至少应该支持点击查看或跳转。
3. 外部打开（双击文件）——`paintChangesTree`/`paintGroup` 文件行完全没有双击区分逻辑，需要先确认 Viva 是否支持双击事件，再接入系统默认程序打开（规格 §4.2）。
4. 路径复制——需要补一个复制到剪贴板的按钮/菜单项，`paintChangesDiff` 工具栏是最直接的位置。
5. Diff 忽略空白差异开关——`paintChangesDiff` 工具栏（`shell.go:1326-1337`）需要补一个 Checkbox，并在请求 Diff 时带上 `git diff -w` 之类的参数。

**两端都要做**
6. 分组级批量操作（全部暂存/全部取消暂存本组）——后端 `r.Stage`/`r.Unstage` 目前只接受单路径，需要新增批量接口（或在应用层循环调用并合并错误），UI 侧需要在分组标题行加复选框触发批量动作，涉及 `repo.go` Stage/Unstage 和 `shell.go` `paintGroup`/`paintChangesTree`。
7. 工作区↔暂存区的独立三版本/两两对比入口——`repo.go:673-737` 的 `Compare` 需要新增一个显式的"index vs worktree"比较模式，不依赖当前隐含在 unstaged 路径里的近似语义。
8. 三版本同屏比较——当前完全空白，需要新设计一个三栏或分段展示的 UI，和冲突三方视图（P1 已标记"需要架构评估"）在技术选型上可能有共用之处，但产品语义不同（这个是查看三个版本内容，不是合并冲突），建议单独评估。
9. 差异导航（上一处/下一处改动）——需要先有基础的"改动块"概念能在 UI 和键盘/鼠标交互间传递，目前连按钮都没有，这不只是接个按钮的问题，需要确认 `CodeDiff` 组件是否暴露了可供应用层调用的"跳转到第N处改动"接口。
10. 二进制文件的结构化展示（类型/大小/版本信息）——需要新增读取文件 MIME 类型/大小的逻辑（后端），以及一个专门的二进制文件展示 UI（替代当前把 `git diff --binary` 原始文本直接显示的做法），涉及 `repo.go` Compare 函数和 `shell.go:1427-1435`。
11. 纯文本 Diff 渲染路径（二进制/大文件走的那条）完全没有工具栏能力——这个问题随第10条一起解决会更自然，如果二进制文件改成专门展示 UI，大文件（非二进制）这条路径是否要补全部分 `CodeDiff` 能力需要重新评估，可能涉及放宽/调整 512 KiB 阈值的判断逻辑。

**Viva 框架限制（已知，不是新发现，引用 `docs/UI.md` §8）**
12. Diff 上下排列、hunk 勾选（行级）、行内字符级改动高亮、改动导航——`CodeDiff`/`CodeDiffRow` 组件能力不足，已提给 Viva [#131](https://github.com/vivaui/viva/issues/131)。注意：第5节已指出当前 hunk 勾选 UI 不是用 `CodeDiff` 组件实现的，是独立按钮列表，所以"等 Viva 修好 CodeDiffRow"不会自动解决 cetacean 现在的 hunk 交互问题，应用层仍需重新设计这部分。
