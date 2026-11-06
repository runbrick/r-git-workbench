# Rebased 分栏与工具窗行为 Parity 差异盘点

对应 `docs/ROADMAP.md` P3 任务。与 P1（`docs/REBASED_ACTION_INVENTORY.md`）、P2（`docs/REBASED_UI_PARITY_GAPS.md`）配套。

## 1. 持久化现状核查

**结论：架构文档里已经完整设计了持久化方案，但对应的包和代码一行都还没写。这是"设计完成、实现为零"，不是"未设计"。**

证据：

- `docs/ARCHITECTURE.md` §7"持久化"（第119-129行）明确写了设计：
  - 应用设置、键位、最近仓库 → 用户配置目录 `r-git-workbench/settings.json`，JSON，写临时文件后重命名
  - 每仓库视图状态（筛选、选中项、**面板尺寸**）→ 用户配置目录 `r-git-workbench/repos/<路径哈希>.json`
  - 最近提交消息 → 每仓库状态文件内
- §2 的分层表把 `internal/settings` 列为"旁路"包之一，和 `internal/keymap`、`internal/i18n`、`internal/fswatch` 并列。
- 实际检查：`internal/settings` 目录**不存在**，目前只有 `fswatch git gitexec i18n keymap repo ui workbench` 八个包。
- `internal/workbench/session.go` 全文：`Session` 结构体只有 `pending []Detection`、`updates []Update`、两个 `sync.Mutex`，是纯内存的异步结果收件箱（后台 goroutine 写、`DrainUpdates`/`Drain` 读），**没有任何文件 I/O，没有任何字段对应 UI 布局或设置**。
- `internal/repo/repo.go` 全文：`Repository` 结构体只有 `Root string`、`runner gitexec.Runner`、一个写锁，全部方法是对 `git` 子进程的封装，**没有仓库级配置文件路径的概念**。
- 全库搜索 `persist|Persist|SaveConfig|LoadConfig|os.UserConfigDir|settings.json|viper|yaml.Marshal`：只有 `docs/ARCHITECTURE.md` 这一处文字提到 `settings.json`，代码里零匹配。
- M4 进度记录的旁证："Pull 策略选择仅保留在当前窗口会话中"——这句话本身在承认持久化缺失，不是局部疏漏。

`changesSplit`/`logBranchesSplit`/`logDetailsSplit` 三个 `flex.SplitterState` 字段（`shell.go:193-195`）目前**不可能**跨会话记忆，因为它们是 `Model` 的瞬时字段，而 `Model` 本身也没有被任何地方序列化过。这不只是"这三个分栏没持久化"，是"整个应用目前没有任何跨会话状态保存机制"——包括设置、键位、最近仓库列表、每仓库筛选条件、选中项、提交消息历史的持久化，全部要靠 M10 待建的 `internal/settings` 包才能生效。

## 2. 三个可调分栏的工具窗行为对照

对照 IntelliJ/JetBrains 工具窗标准行为模式（固定展开、折叠为图标条、拖动改尺寸、记忆开关状态、按仓库隔离）：

| 分栏 | 可拖动 | 可折叠为图标条 | 记忆尺寸 | 按仓库隔离 | 差距 |
|---|---|---|---|---|---|
| `changesSplit`（Changes/Commit侧栏，`shell.go:815`） | 有，`f.Splitter(...).Mins(230, 300)` | 无——没有收起为图标轨道的交互，只能拖到最小宽度 | 无——瞬时字段，重启或切仓库后回到初始 `Pos: 300` | 无 | 比 Rebased"Commit/Shelf 侧栏可能支持折叠为图标"缺折叠态和持久化两项 |
| `logBranchesSplit`（Log内分支侧栏，`shell.go:1623`） | 有，`.Mins(105, 260)` | **用 `m.width >= 1000` 硬阈值代替**——窄窗口下整个侧栏消失，不是折叠为 Rebased 截图里那条"竖排文字 Branches"的窄条，是直接从布局中消失，没有保留入口 | 无 | 无 | 三者中差距最大：Rebased 基线明确有"折叠但保留可点击窄条"的状态，cetacean 目前是"显示/完全不显示"二值状态，缺中间态 |
| `logDetailsSplit`（Log提交详情侧栏，`shell.go:1705`） | 有，`.Mins(230, 250)` | 无——同样用 `m.width >= 1000` 阈值整体隐藏（`shell.go:1704,1709`） | 无 | 无 | 同上，缺折叠中间态和持久化 |

三个分栏共性问题：
1. **没有任何"记忆尺寸"能力**——这是 ROADMAP P3 验收标准"状态按仓库持久化"的直接缺口，根因是第1节确认的"整个应用没有持久化层"，不是分栏代码本身的问题。
2. **没有"按仓库隔离"**——即使将来补上持久化，目前 `Model` 结构里也没有任何按 `repo.Root` 区分状态的键，需要在设计持久化方案时一并考虑（对应 ARCHITECTURE.md §7 的"路径哈希文件名"方案）。
3. **折叠行为用宽度阈值代替用户手动折叠**——方向性问题：Rebased 的折叠状态应该是用户主动点击触发的持久化偏好（"我希望这个面板一直收起"），cetacean 现在是窗口宽度的被动函数（"窗口只要够宽就显示，不管用户想不想看"）。即使之后做了尺寸持久化，如果不改这个判断逻辑，用户也无法手动收起分支侧栏来腾空间（哪怕窗口够宽）。

## 3. 窄窗口响应式行为

当前响应式阈值：
- `m.width >= 900`（`shell.go:871`）：控制 Fetch/Pull/Push 按钮组和 Pull 策略横排按钮是否显示（对应 P2 盘点已记录的"顶栏信息架构"问题）。
- `m.width >= 1000`（`shell.go:1585,1704,1709`）：控制 Log 分支侧栏和提交详情侧栏是否显示。

两个阈值之间（900-999px）存在一个**未被任何现有测试覆盖的组合状态**：顶栏同步按钮可见，但 Log 的两个侧栏都被隐藏。静态读代码没有发现这个区间会产生重叠或裁切的直接证据（都是整段条件渲染，不是裁切单个控件），但也没有证据证明它经过验证。

实测结果：

```
go test ./internal/ui/shell/... -run 'TestNarrowWindowLayout|TestHeadlessNavigationAndLayout' -v
=== RUN   TestHeadlessNavigationAndLayout
--- PASS (0.01s)
=== RUN   TestNarrowWindowLayout
--- PASS (0.00s)
```

两个测试都通过，但覆盖的窗口尺寸只有：
- `720×480`——低于两个阈值，测的是"全部收起"状态。
- `1100×720`——高于两个阈值，测的是"全部展开"状态。

**没有任何现有测试落在 900-999px 或刚好跨越 900/1000 边界的尺寸**（比如 899 vs 901，或 950）。`TestNarrowWindowLayout` 这个名字暗示它是"窄窗口无重叠裁切"验收的覆盖点，但它只测了一个远低于阈值的尺寸，没有测阈值附近的过渡状态——这正是响应式布局最容易出 bug 的地方（条件判断差一个像素、子元素在隐藏前后抢空间）。

## 4. 命名视图缺口

`docs/UI.md` §4"视图状态记忆"设计：

> 每个视图的可持久化状态（筛选、排序、选中项、面板尺寸、Diff 方向）是普通结构体，由 `workbench` 按仓库保存与恢复（规格 §3 视图切换）。选中项按稳定键（路径、SHA、引用全名）记录，快照刷新后重新定位，找不到则选中相邻项。

实际代码核查：

- **筛选条件**：`Model` 里有 `logQuery`/`logAuthor`/`logPath`/`logSince`/`logUntil`/`logRef` 等 `flex.Field`，这些是 `Model` 的瞬时字段，没有看到切换视图或重启后恢复的逻辑。
- **选中项**：没有发现"按稳定键记录选中项、快照刷新后重新定位"的实现，`Model` 整体就是一份，没有"离开视图时序列化、进入时反序列化"的动作。
- **面板尺寸**：已在第1、2节确认不持久化。
- **Diff 方向**（左右/上下）：`docs/UI.md` §8 提到"Diff 上下排列"本身是 Viva 框架缺口（`CodeDiff` 只支持左右并排），这项当前连功能都没有，记忆与否不适用。

`workbench.Session` 当前完全不含"按仓库保存与恢复视图状态"的职责——它只是一个异步结果收件箱。UI.md 设计里说的"由 workbench 按仓库保存与恢复"，对应的代码实现在 `workbench` 包里完全找不到，和第1节的持久化缺口是同一个根因：**`internal/settings` 包和它与 `workbench`/`ui` 之间的读写集成都还没有建立**。

## 5. 汇总：P3 验收待办清单

按优先级列出需要新增的具体机制（基础设施，不是视觉调整）。

**P0（阻塞性基础设施缺口，P3 其余验收项都依赖它）**
1. 新建 `internal/settings` 包，落地 `docs/ARCHITECTURE.md` §7 已设计好的方案：应用级 `settings.json`（用户配置目录）+ 每仓库 `repos/<路径哈希>.json`，JSON 原子写入（写临时文件后 rename）。这是目前唯一缺失的"地基"，没有它，下面第2-4条都无法真正实现"持久化"，只能停留在内存状态。
2. 在 `internal/workbench` 里补上"按仓库保存/恢复视图状态"的职责（UI.md §4 已设计但未实现），需要定义一个可序列化的视图状态结构体（筛选条件、选中项稳定键、三个 `SplitterState` 的 `Pos`），在仓库切换/关闭时保存，打开时恢复。

**P1（在基础设施之上的具体对接点）**
3. 把 `changesSplit`/`logBranchesSplit`/`logDetailsSplit` 的 `Pos` 接入第2条的视图状态读写，做到"按仓库记忆分栏尺寸"（ROADMAP P3 验收标准原文）。
4. 把 Log 的 `logQuery`/`logAuthor`/`logPath`/`logSince`/`logUntil`/`logRef` 筛选字段接入同一机制。
5. 把 M4 进度记录里提到的"Pull 策略选择"从会话级提升为持久化设置（存入 `settings.json` 或仓库状态文件，需确定是全局设置还是按仓库，产品规格 §6 倾向全局默认+每次可临时覆盖）。
6. 把 Log 分支侧栏/详情侧栏从"`m.width >= N` 二值硬阈值隐藏"改造成"用户手动折叠为窄条（对照 Rebased 截图的竖排文字 Branches 条）+ 折叠状态持久化"，涉及 `shell.go:1585`、`1704`、`1709` 三处判断逻辑，需要新增一个布尔折叠状态字段并接入第2条的持久化机制。

**P2（测试覆盖缺口，验证用，不涉及新功能）**
7. 补一个 headless 测试覆盖 900-1000px 之间以及刚好跨越 900、1000 边界的窗口尺寸（如 899×720、901×720、999×720、1001×720），用 `expect layout clean` 验证这个响应式阈值过渡区间没有被现有测试覆盖到的重叠/裁切风险，位置建议加在 `internal/ui/shell/headless_test.go` 或 `shell_test.go` 旁边。
8. 待第1-6条基础设施落地后，补充"关闭并重新打开同一仓库后，筛选条件/选中项/分栏尺寸被还原"的集成测试，验证 ROADMAP P3 验收标准"状态按仓库持久化"真正生效，而不只是接口存在。
