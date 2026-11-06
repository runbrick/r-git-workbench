# Rebased Log Parity 差异盘点

对应 `docs/ROADMAP.md` P7 任务。与 P1（`docs/REBASED_ACTION_INVENTORY.md`）、P2（`docs/REBASED_UI_PARITY_GAPS.md`）、P3（`docs/REBASED_SPLITTER_PERSISTENCE_GAPS.md`）、P4（`docs/REBASED_COMMIT_SHELF_GAPS.md`）、P5（`docs/REBASED_CHANGES_DIFF_GAPS.md`）、P6（`docs/REBASED_BRANCH_REFS_GAPS.md`）配套，不重复已记录内容。

## 1. 分页加载机制

机制（`shell.go:328-334` `loadLog`，`repo.go:70-120` `LogQuery`）：`loadLog` 在 `m.repository != nil && !m.logLoading && m.logHasMore` 时才发起请求，`skip = len(m.commits)`、`limit` 由调用方给定（UI 侧固定按 100 一批，参照 M2 进度记录）。`LogQuery` 对 `skip`/`limit` 做了边界校验（`limit` 必须在 1-200 之间），防止异常分页请求。

重复加载防护：`m.logLoading` 布尔位在请求发出后立即置真（`shell.go:332`），收到结果前再次触发 `loadLog` 会被短路，这是基础的防抖，没有发现竞态漏洞。

分页边界：`repo_test.go:554 TestLogPages` 验证了 105 条提交下首页 100 条、次页 5 条、跨页无重复 SHA——这是功能正确性验证，不是性能验证（105 条提交对任何实现都是瞬时的）。**没有发现任何测试验证"大仓库"（比如 10,000+ 条真实提交历史）下分页的实际加载耗时**，见第9节。

`m.logHasMore` 的终止条件依赖 `LogQuery` 返回的结果数量是否等于请求的 `limit`（本次未展开读取判定具体代码行，但从 `TestLogPages` 行为推断次页返回 5 条后应停止"加载更多"，逻辑合理），没有发现明显 bug。

## 2. 图形布局（Commit Graph）限制

`internal/ui/shell/graph.go:11`：`const graphLanes = 6`，是**硬编码常量**，不是数据结构的天然限制——`graphRow` 结构体的 `before`/`after`/`merge` 三个数组固定长度 `[graphLanes]string`/`[graphLanes]bool`（`graph.go:15-17`），把并行分支轨道数硬限制在编译期常量。

根因明确：`buildGraph`（`graph.go:22-94`）在找不到空闲 lane 时的兜底逻辑是 `lane = graphLanes - 1`（`graph.go:42`），即**强行把新分支塞进最后一条轨道**，如果这条轨道当时被别的分支占用，会直接覆盖（`lanes[lane] = commit.SHA`，`graph.go:44`，无条件覆写）。

实际后果（复杂分支场景）：当同时存在超过 6 条"活跃"（尚未合并完的）分支线时，第 7 条及以后的分支会被挤到第 6 条轨道，与原本在那条轨道上的分支共享同一条视觉轨迹——**这不是单纯的"视觉简化"或"截断"，是真实的图形错误**：两条不相关的分支会被画成同一条连续的线，颜色也会复用 `graphColors[5]`（`graph.go:96-99`，颜色数组同样只有 6 个），用户会看到两条历史上不相关的提交被图形连在一起，构成误导性的视觉结果，而不是优雅降级（比如"第7条开始不画线，只留提交点"）。

测试覆盖：`internal/ui/shell/graph_test.go:10 TestGraphKeepsParentsAcrossPagesAndMerge`——从名字看验证"跨分页/合并场景下父提交关系保持"，但测试名不包含"7条以上分支"或"lane溢出"相关字样，本次未展开读取测试体确认具体构造的分支数，**没有发现专门针对"超过6条并行分支"场景的测试**，这个 bug 大概率没有被验证过。

## 3. 引用装饰（Ref Labels）

**结论：P2 的怀疑完全证实——`refLabels`（`graph.go:137-160`）是纯文字字符串拼接，没有任何图标、没有任何颜色区分，HEAD/本地分支/远程分支/Tag 四种引用只靠文字前缀区分。**

具体实现：
```go
if sha == head {
    labels = append(labels, "HEAD")
}
// ...
if ref.Tag {
    prefix = "tag: "
} else if ref.Remote {
    prefix = "remote: "
} else {
    prefix = "branch: "
}
labels = append(labels, prefix+ref.Name)
```
最终拼成 `"  [HEAD, branch: main, tag: v1.0]"` 这种形式的字符串，直接附加在提交消息文本后面（`shell.go:213` 一带，`Cell` 函数），渲染时就是普通的表格单元格文字，**没有图标、没有背景色块、没有字体颜色区分**，四种引用类型只能靠"HEAD"/"tag: "/"remote: "/"branch: " 这几个英文前缀词肉眼辨认。

这直接违反规格 §4.4"不能只依赖颜色区分"的更基础前提——这里连颜色区分都没有做到，是纯文字堆砌。另外还有国际化问题：这几个前缀词（`"tag: "`、`"remote: "`、`"branch: "`）是硬编码英文字符串字面量，没有走 `i18n.Text`，和项目"用户可见文字走 i18n"的约定不一致（但规格也要求"Git 原始输出、路径、引用名不翻译"——这几个前缀词是客户端自己拼的 UI 文案，不是 Git 原始输出，应该走 i18n，这是本次新发现的 i18n 遗漏点）。

`len(labels) == cap(labels)` 的提前退出（`graph.go:143`，cap 固定为4）意味着一个提交最多显示4个引用标签，超过的会被静默忽略，没有"还有N个"之类的提示。

## 4. 作者/提交者列区分

**结论：当前只取 author，完全没有 committer 信息。**

`repo.go:74`：`--format=%H%x00%P%x00%an%x00%aI%x00%s%x00`——`%an` 是 author name，`%aI` 是 author date（ISO格式）。全字段列表里**没有 `%cn`（committer name）或 `%cI`（committer date）**。`internal/git/log.go:9` 的 `Commit` 结构体（本次未展开全部字段，但从 `repo.go` 的解析逻辑看只填充了 author 相关字段）同样不含 committer。

这意味着：对于 rebase/cherry-pick/amend 产生的提交（author 和 committer 不同的常见场景），cetacean 当前完全无法区分展示"谁写的代码"和"谁提交/改写的这次提交"，规格 §4.4"作者、提交者"分列的要求连数据源都没有采集，是彻底的功能空白，不是 UI 没画出来的问题。

## 5. 比较任意两个提交/分支/Tag

全文搜索 `paintLog` 及 Log 相关的选择/点击逻辑：`m.logTableBounds.RowClicked()`（`shell.go:1693`）只支持**单行选中**（`m.selectCommit`，`shell.go:357`），表格状态里没有发现任何"多选"或"已选两行"的字段（对比 P5 已确认的 Diff 区域"多选 hunk"用的是独立的 `hunkScope` 选择状态，Log 表格没有等价物）。

没有发现任何"选中两个提交后出现 Compare 按钮"的代码雏形。这和 P1 已标记的 `GitCompareWithBranchAction`/`Git.Ref.Compare.With`/`Git.Ref.Diff.With.Local` 均"未实现"的结论一致——本节确认的是：**连多选交互本身都不存在**，这不是"有多选但没接比较功能"，是表格层面只支持单选，要实现比较功能需要先给 Log 表格加多选能力，这是比单纯"加一个比较按钮"更大的改动。

## 6. 复制功能

除 P2 已发现的 Copy SHA 按钮本身无 `.Clicked()` 处理（`shell.go:1722`，本节确认行号与 P2 一致，未发现新证据反驳）外，全文搜索 Log 相关代码未发现任何其他复制功能：
- 复制提交消息：无
- 复制作者信息：无
- 复制文件路径（详情区文件列表）：无，与 P5 第9节"路径复制全局零匹配"的结论一致

Log 详情区的复制能力目前是**完全空白**（连一个有效的复制按钮都没有，Copy SHA 按钮是唯一的尝试但也没接上逻辑）。

## 7. 文件树结构化（详情区）

数据结构：`m.commitFiles []string`（扁平字符串切片，存完整相对路径），渲染逐项直接 `f.Text(path)`（`shell.go:1726-1739`），**没有任何目录层级解析**——直接复用 Changes 区域的 `paintChangesTree` 不行，因为那个函数读的是 `m.status.Files []git.File`（带 staged/worktree 状态字段的结构），而这里只有裸路径字符串，没有文件状态可供着色，需要的是"纯路径折叠成树"这个更简单的子集逻辑（只做目录分组和缩进，不需要状态图标）。

工作量评估：**中等，不是简单复用，但也不需要从零设计**——`paintChangesTree` 里"按路径前缀逐层分组、`m.expandedPaths` 记录展开状态"（P5 第2节已描述，`shell.go:1513-1537`）这部分的路径分组算法可以抽出来复用，需要新写的是一个只接受 `[]string` 的轻量版本（或者给现有树形逻辑增加一个不需要文件状态的渲染分支），并给 `Model` 增加一个独立于 `m.expandedPaths`（Changes区域用）的新展开状态字段（避免两个树共享一份展开状态导致互相干扰）。

## 8. 危险操作与计划绑定验证

**结论：这条验收标准在 P7 阶段完全不适用，不是遗漏，是前提条件不存在。**

ROADMAP P7 验收标准原文"所有危险操作计划绑定状态并验证"——当前 Log 视图里**没有任何危险操作**：Reset/Revert/Cherry-pick/Push to commit 等（P1 已标记全部"未实现"）在 Log 的 UI 里连占位按钮都不存在（P6 第8节已确认"从 Log 操作分支"连占位入口都没有，本节确认这个结论同样适用于 Reset/Revert/Cherry-pick 这类历史编辑操作，不只是分支操作）。

这意味着"验证危险操作的计划绑定状态"这件事，目前没有任何验证对象——这条验收标准实质上依赖 P8（历史编辑与冲突）范围的功能先落地，P7 阶段只能记录"暂不适用"，不能标记为"未通过"或"已通过"，这是两个不同的状态。建议在 ROADMAP 里明确这条验收标准要推迟到 P8 完成后才能重新评估。

## 9. 大仓库性能测试现状

`docs/TESTING.md` 和 `.claude/skills/building-with-viva/SKILL.md` 描述的 headless 协议本次确认支持 `tree`/`expect named`/`expect layout clean` 等结构性断言，**没有发现任何内建的性能计时断言机制**（headless 协议本身不是为benchmark设计的）。

ROADMAP M2 记录提到的"10,000条内存提交的可见行数量"对应 `internal/ui/shell/shell_test.go:55 TestLogTableOnlyDeclaresVisibleRows`：
```go
m.commits = make([]git.Commit, 10000)
// ...
m.logSource.commits = m.commits
for _, size := range [][2]int{{1200, 760}, {720, 480}} {
    // resize driver, count Hits within logScope
    if visible < 1 || visible > 45 {
        t.Errorf(...)
    }
}
```

**这个测试验证的是：给定一个已经在内存里的 10,000 条 `git.Commit` 切片，表格虚拟化只声明 45 行以内的可见行**——这是纯粹的 UI 虚拟化正确性验证（证明不会因为数据量大就声明一万个 UI 节点），**完全不涉及**：
- 真实 git 仓库的 `git log` 命令执行耗时
- 10,000 条提交规模下分页请求的实际响应时间
- 滚动加载时连续触发 `loadLog` 的性能表现
- 图形计算（`buildGraph`）在大量提交下的 CPU 开销

`repo_test.go:554 TestLogPages` 用的测试仓库只有 105 条提交，远不构成"大仓库"量级。**结论：ROADMAP M2 记录和 P7 验收标准提到的"大仓库测试"目前完全没有被覆盖**，现有测试只证明了内存数据结构层面的虚拟化行为是对的，对真实大仓库（不管是提交数量级还是图形复杂度）的实际使用体验没有任何验证，这是本次核查最关键的发现之一。

## 10. 筛选语义完整度

当前 6 个筛选字段覆盖维度：消息（`logQuery`→`Message`）、作者（`logAuthor`→`Author`）、路径（`logPath`→`Path`）、起始日期（`logSince`→`Since`）、结束日期（`logUntil`→`Until`）、引用（`logRef`→`Ref`）。这覆盖了规格要求的"消息、作者、路径、日期范围"四个维度（SHA 筛选未见独立字段，但 Git 的 `--grep` 对 SHA 文本本身不生效，如果需要按 SHA 查找更可能是跳转到具体提交而非筛选列表，这里不计入缺口，是维度理解问题不是实现缺口）。

"当前分支/所有分支"这个维度，核查 `repo.go:74-82`：
```go
if filter.Ref == "" {
    args = append(args, "--all")
} else {
    // ... 校验 filter.Ref 后 append(args, filter.Ref)
}
```
**发现一个此前文档都没提到的具体行为**：`logRef` 为空时默认行为是 `--all`（显示所有分支的历史），不是"只看当前分支（HEAD）"。也就是说默认状态其实对应规格"所有分支"这个选项，但**没有反向的"只看当前分支"快捷开关**——`logRef` 字段的设计是"指定一个具体 ref 名进行精确筛选"，用户要看"仅当前分支"必须手动输入当前分支的名字（而且还得知道怎么拼 `refs/heads/` 前缀，虽然 `applyLogFilter` 会自动补全前缀，`shell.go:338-340`），不是一键切换"当前分支 ⇄ 所有分支"的开关交互。这和规格 §4.4"搜索/筛选：...当前分支/所有分支"要求的"二元快捷切换"语义不完全对应，是部分实现而非完全缺失。

## 11. 测试覆盖现状

| 场景 | 是否有测试 | 测试文件/函数 |
|---|---|---|
| 分页基础正确性（首页/次页/无重复） | 有 | `internal/repo/repo_test.go:554 TestLogPages` |
| 消息/作者/路径筛选（后端） | 有 | `repo_test.go:554`（同一测试内）、`repo_test.go:1271 TestLogFiltersPathAndAuthor` |
| 不安全路径筛选拒绝 | 有 | `repo_test.go:606-608`（`TestLogPages` 内） |
| 滚动触发加载下一页（UI Driver） | 有 | `internal/ui/shell/log_scroll_test.go:20 TestLogScrollLoadsNextPage` |
| 跨分页/合并场景图形连续性 | 有 | `internal/ui/shell/graph_test.go:10 TestGraphKeepsParentsAcrossPagesAndMerge` |
| 10,000条提交的表格虚拟化（可见行数量） | 有（但范围有限，见第9节） | `internal/ui/shell/shell_test.go:55 TestLogTableOnlyDeclaresVisibleRows` |
| 合并提交文件 Diff（父提交比较） | 有 | `repo_test.go:612 TestMergeCommitDetailsAndFileDiff` |
| **超过6条并行分支的图形正确性** | **没有** | 第2节已确认的硬编码限制，未找到专门测试覆盖溢出场景 |
| **大仓库真实性能（提交数量级/图形计算耗时）** | **没有** | 第9节已确认现有测试不构成大仓库场景 |
| 作者/提交者分列展示 | 没有（功能不存在） | 后端数据源未采集 committer，无法测试 |
| 引用标签图标/颜色区分 | 没有（功能不存在，纯文字） | 无相应代码 |
| 比较任意两提交/分支/Tag | 没有（功能不存在） | 无相应代码，Log表格不支持多选 |
| 复制提交消息/作者等（非SHA） | 没有（功能不存在） | 无相应代码 |
| Log详情区文件树形结构 | 没有（功能不存在，当前是扁平列表） | 无相应代码 |
| 危险操作计划绑定验证 | 不适用 | 依赖P8范围功能，第8节已说明 |
| "仅当前分支"快捷切换 | 没有（功能不完整，见第10节） | 无相应代码，只能手动输入ref名 |

## 汇总：P7 验收待办清单

按优先级列出，区分"后端已支持只缺UI"、"纯UI缺失"、"两端都要做"、"依赖P8范围暂不适用"四类。

**两端都要做（本次盘点优先级最高的新发现）**
1. **图形轨道硬编码6条，超出时会产生真实的视觉错误（不同分支被误画成同一条线）** —— `graph.go:11,41-44`，需要重新设计 `buildGraph` 的溢出策略（比如动态扩展轨道数，或超出时降级为"只画提交点不连线"而不是覆盖占用），这不是性能优化，是正确性 bug，涉及 `graphRow` 结构体定长数组的根本设计。
2. **作者/提交者数据源缺失** —— `repo.go:74` 的 `--format` 字符串需要加 `%cn`/`%cI`（committer name/date），`internal/git/log.go` 的 `Commit` 结构体需要新增字段，UI 侧需要在详情区和列表列里分别展示两者（rebase/amend 场景下两者经常不同，这是规格明确要求的区分）。
3. **大仓库真实性能从未被测试验证** —— 需要新增一个用真实大仓库（建议数千到上万条提交，脚本生成而非内存构造）跑分页加载、滚动、图形计算的集成测试/benchmark，当前的 `TestLogTableOnlyDeclaresVisibleRows` 不能替代这个验证，这是 ROADMAP P7/M2 验收标准里明确要求但实际空白的一项。
4. **比较任意两个提交/分支/Tag** —— 需要先给 Log 表格加多选能力（当前只支持单选，`shell.go:1693` 的 `RowClicked`），再接入比较功能，和 P1 已标记的 `GitCompareWithBranchAction` 等动作一起实现，工作量比单纯"加个按钮"大。

**纯UI缺失**
5. 引用标签图标化 —— `refLabels`（`graph.go:137-160`）改为返回结构化数据（而不是拼好的字符串），由渲染层画出 HEAD/本地分支/远程分支/Tag 四种图标和颜色区分，同时把 `"tag: "`/`"remote: "`/`"branch: "` 这几个硬编码英文前缀改为走 `i18n.Text`（本次新发现的 i18n 遗漏）。
6. Log 详情区文件列表改为树形结构 —— 可复用 `paintChangesTree` 的路径分组算法子集，但需要新写一个不依赖文件状态的轻量渲染分支和独立的展开状态字段，工作量中等（第7节已评估）。
7. Copy SHA 按钮补上实际的复制到剪贴板逻辑（P2 已记录，这里不重复计入新待办，仅在此提醒 P7 实施时顺带修复）。
8. 补充复制提交消息/作者信息等功能，Log 详情区当前复制能力完全空白。
9. "仅当前分支"快捷切换开关 —— 在现有 `logRef` 自由输入之外，加一个"当前分支/所有分支"二元切换控件，默认值对应现有的 `--all` 行为保持不变，只是增加一个显式切到"仅HEAD所在分支"的快捷方式。

**依赖P8范围暂不适用**
10. "所有危险操作计划绑定状态并验证"这条 P7 验收标准——第8节已说明，Reset/Revert/Cherry-pick/Push to commit 等危险操作本身都在 P8 范围，P7 阶段这条验收标准没有验证对象，建议 ROADMAP 明确标注"待P8完成后重新评估"而不是让它悬空。
