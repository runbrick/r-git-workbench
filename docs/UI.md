# 界面层设计

覆盖 `internal/ui/*`、`internal/keymap`、`internal/i18n`。Viva 编程模型见 `.claude/skills/building-with-viva/SKILL.md`。

## 1. 编程模型约定

- 绘制函数就是界面：每帧根据 UI 状态和仓库快照声明画面，不保留布局缓存。
- 每个视图一个包，导出 `type View struct{...}`（视图自己的 UI 状态）和 `func (v *View) Paint(f *flex.Frame, ctx *Context)`。`Context` 提供当前快照、主题、i18n、`Submit(intent)`。
- 视图只产出 Intent（“提交这些文件”“切换到分支 X”），不直接调用 `git` 或 `repo`。
- 点击在控件声明处处理：`if f.Button(id, label).Clicked() { ctx.Submit(...) }`。
- 昂贵结果（diff 行切分、文本塑形、图布局）由拥有者缓存，按快照代数或请求键失效，不在每帧重新计算。

## 2. 标识树

全局一个 `ui.Tree`，每个视图一个 Feature，控件名在声明处固定：

```go
var (
	tree     = ui.Tree{}
	shell    = tree.Feature("shell")
	changes  = tree.Feature("changes")
	logView  = tree.Feature("log")
	branches = tree.Feature("branches")
	stash    = tree.Feature("stash")
	conflict = tree.Feature("conflict")
	dialog   = tree.Feature("dialog")
)
```

列表行使用 `ui.Scope` + 行序号/稳定键，禁止手工编号。标识名也是 headless 测试的寻址方式（`click id:changes/commit`），改名视同破坏测试接口。

## 3. 主窗口结构

对应规格 §3：

```text
shell.View
├─ TopBar     仓库切换 | 分支选择器 + ahead/behind | Fetch Pull Push | 搜索
├─ NavRail    Changes Log Branches Stash [Tags] [Worktrees]，窄窗口收为图标
├─ Content    当前视图（changes / log / branches / stash / conflicts）
├─ StatusBar  仓库路径 | Git 版本 | 后台操作 | 错误
└─ Overlays   对话框、命令面板、分支弹出层、上下文菜单、toast
```

| 视图 | 主要 Viva 构件 |
|---|---|
| Changes | `flex.Table`（虚拟化行）/ `components.Tree`（目录树模式）、`flex.Splitter`、`components.Textarea`（提交消息） |
| Diff | `components.CodeDiff`（左右并排）；上下排列、hunk 勾选需自建，见 §8 |
| Log | `flex.Table`（`TableSource` 只读取可见行）+ 自绘图列；`flex.Splitter` 分隔详情 |
| Branches | `components.Tree`（Local / Remote / Recent 分组）+ 搜索 `components.Input` |
| Stash | `flex.Table` + 详情 Diff |
| Conflicts | 冲突文件 `flex.Table` + 三方合并视图（自建，见 §8） |
| 命令面板 | `components.Command` |
| 菜单 | `components.ContextMenu`、`components.DropdownMenu` |
| 对话框 | `components.Dialog`、`components.AlertDialog`（有损确认） |
| 通知 | `components.Sonner` / `Toast`（非阻断）；需要用户处理的错误用对话框（规格 §3） |

## 4. 视图状态记忆

每个视图的可持久化状态（筛选、排序、选中项、面板尺寸、Diff 方向）是普通结构体，由 `workbench` 按仓库保存与恢复（规格 §3 视图切换）。选中项按稳定键（路径、SHA、引用全名）记录，快照刷新后重新定位，找不到则选中相邻项。

## 5. 快捷键（keymap）

- **动作注册表**：每个动作是一个常量 `ActionID`，带 i18n 名称、适用上下文、是否需要选中项。命令面板、菜单和快捷键都从同一张表读取（规格 §5）。
- **上下文**：`Global`、`Changes`、`Log`、`Diff`、`Conflict`、`TextInput`、`Dialog`。按键先交给焦点控件；焦点在文本输入时，标准编辑键（复制、粘贴、撤销、Home/End、方向键）永远不被动作抢走。
- **预设**：IntelliJ IDEA（默认）、VS Code、Default 三套完整映射。用户修改保存为对预设的差异。
- **冲突检测**：同一上下文链上同一组合键绑定两个动作时在 Keymap 设置中标出。
- **平台**：键位表中写 `Primary`（Windows/Linux 为 Ctrl，macOS 为 Cmd），显示时按平台渲染为 `Ctrl+K` 或 `⌘K`。
- **接入点**：`app.Config.OnShortcut` 接收带修饰键的可打印键，`OnKey` 接收功能键（F5、F7、Esc、Enter）。

## 6. 本地化（i18n）

- 首发中文、英文。键是 Go 常量（`i18n.CommitButton`），表是 `[...]string` 数组，按语言索引；查表无分配、无 map。
- 启动时按 `app.PreferredLanguages()` 选择，设置中可改。
- Git 命令、路径、引用名、原始错误输出不翻译（规格 §8）。

## 7. 无障碍

- 每个控件有可访问名称；图标按钮必须传文字标签。
- 状态不只靠颜色：文件状态带字母徽标（M/A/D/R/U/?），ahead/behind 带箭头与数字，Diff 行带 `+`/`-` 前缀列，Log 引用标签带图标区分本地分支/远程分支/Tag/HEAD（规格 §4.4、§8）。
- 焦点顺序与视觉顺序一致；用 headless `tree` 与 `expect named` 检查无名控件。
- 高对比模式读取 `app` 提供的系统高对比状态切换主题。

## 8. Viva 缺口

以下需求 Viva v0.2.4 没有直接提供。按 Viva 规则，可被第二个 Viva 应用复用的能力应提议给 Viva，而不是在应用中用平台调用绕过。

| 需求 | 现状 | 处理建议 |
|---|---|---|
| 递归文件监听 | Viva 不提供 | 已提问是否属于 Viva：[#133](https://github.com/vivaui/viva/issues/133)；等待答复期间在 `internal/fswatch` 实现，接口保持小而独立 |
| “Shift 两次”全局搜索 | 普通模式下单独的修饰键会被丢弃（`app/painter.go` 的 `painter.key`），只有设置了 `TextInput.KeyReleases` 的终端能收到 | 已提议：[#130](https://github.com/vivaui/viva/issues/130)；在此之前首发用 `Primary+Shift+A` 和可配置替代键 |
| 三方合并视图 | `CodeDiff` 只支持两栏、右侧可编辑 | 已提议：[#132](https://github.com/vivaui/viva/issues/132)；若维护者认为不属于框架则应用内自建 |
| Diff 上下排列、hunk 勾选、行内（字符级）改动高亮、改动导航 | `CodeDiff` 为左右并排，`CodeDiffRow` 无行内范围，行号栏无命中区域 | 已提议：[#131](https://github.com/vivaui/viva/issues/131)；hunk 暂存本身留在应用 |

Log 图列不算缺口：`flex.Table` 的行布局返回单元格矩形，可在图列矩形内自绘。

每个缺口在实现前单独确认方案，见 [ROADMAP.md](ROADMAP.md) 待决事项。
