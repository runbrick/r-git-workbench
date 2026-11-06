# Rebased Terminal/TextMate Parity 差异盘点

对应 `docs/ROADMAP.md` P12 任务，这是 P1-P12 系列盘点的最后一项。与 P1（`docs/REBASED_ACTION_INVENTORY.md` 的"Terminal（默认随附）"/"TextMate（默认随附）"两节）配套，不重复已有的产品范围决策——"完整终端模拟器不做"和"TextMate bundle 不接入"这两条已经决定，不需要重新论证。本篇任务是确认两个遗留的小命令（外部终端、文件管理器）的实现现状和路径，并补上 P1-P11 都没专门盘点过的一个独立缺口（CodeDiff 组件自带的语法高亮回调未被使用）。

## 1. "在外部终端打开"功能现状与实现路径

**结论：确认零实现，这是 P2/P5 已发现过的缺失在 P12 的最终整合确认，不是重复发现。**

全文搜索 `internal/` 目录下 `exec.Command`/`os/exec` 的全部调用点（见本次核查附带的 grep 结果）：唯二两处使用标准库 `os/exec` 的位置是 `internal/git/version.go`（Git 版本探测）和 `internal/gitexec/exec.go`（系统 git 子进程封装本身）。**没有任何代码调用过系统终端程序**（`cmd.exe`、`powershell.exe`、`wt.exe`、`Terminal.app`、`gnome-terminal` 等），`internal/ui/shell/shell.go` 里也没有任何按钮或菜单项尝试触发这个命令。

跨平台实现路径评估（这是一个真实存在的平台碎片化问题，不是简单的"调一下系统命令"）：

| 平台 | 典型实现 | 复杂度 |
|---|---|---|
| Windows | `exec.Command("cmd", "/c", "start", "cmd", "/k", "cd /d "+path)` 或更现代的 `wt.exe -d <path>`（Windows Terminal，但不保证所有用户机器都装了） | 中——需要决定默认用传统 `cmd.exe` 还是尝试检测 `wt.exe` 存在再优先使用，两者用户体验差异大 |
| macOS | `exec.Command("open", "-a", "Terminal", path)`，或更通用的 `exec.Command("open", "-a", "Terminal.app", path)` | 低——`open` 是系统标准命令，行为稳定 |
| Linux | 没有统一命令，需要按发行版/桌面环境探测可用的终端模拟器（`gnome-terminal`、`konsole`、`xfce4-terminal`、`xterm` 等），通常做法是按优先级列表依次尝试 `exec.LookPath` 判断哪个存在 | 高——这是三个平台里唯一没有"标准答案"的，需要维护一个候选列表并做存在性探测，且不同终端模拟器的"在指定目录打开"参数语法不同（比如 `--working-directory=` vs 直接传路径） |

**技术判断**：Windows/macOS 路径相对直接，Linux 路径需要一个"按优先级探测可用终端模拉器"的小型适配层，这不是阻塞性难题，但工作量不是三个平台均摊的"一行代码乘三"，Linux 分支明显更重。

## 2. "在文件管理器打开"功能现状与实现路径（新增盘点项）

**结论：P1-P11 全部十一份文档都没有专门盘点这个命令，确认是遗漏项，本次补上。现状同样是零实现。**

规格 §2"明确不做"一节原文："通用文件浏览器或完整终端模拟器。首发仅提供'在文件管理器打开'和'在外部终端打开'命令。"——这两个命令是规格并列提出的，但 P1 的"Terminal（默认随附）"一节只登记了后者（"在外部终端打开"），前者（"在文件管理器打开"）没有被登记为任何插件盘点项的一部分（它本质上也不对应 Rebased 的某个具体插件功能，是规格自己提出的通用能力，可能因此被 P1 的"按插件盘点"方法论遗漏）。

当前代码：和第1节一样，全文搜索未发现任何调用 `explorer.exe`、`open -R`（macOS Finder 定位文件）、`xdg-open`（Linux 通用）等命令的代码，零实现。

跨平台实现路径：

| 平台 | 典型实现 | 复杂度 |
|---|---|---|
| Windows | `exec.Command("explorer", path)`（打开目录）或 `exec.Command("explorer", "/select,", filePath)`（定位到具体文件并选中） | 低——`explorer` 命令行为稳定，是系统标准组件 |
| macOS | `exec.Command("open", path)`（打开目录）或 `exec.Command("open", "-R", filePath)`（在 Finder 中显示并选中文件） | 低——同样是系统标准命令 |
| Linux | `exec.Command("xdg-open", path)`——这是桌面环境无关的标准方式（由 `xdg-utils` 提供，几乎所有主流发行版默认安装），比第1节的终端探测问题简单得多，因为"打开文件管理器"这个操作本身有跨桌面环境的标准化协议（`xdg-open` 会交给用户系统配置的默认文件管理器处理），不需要应用自己维护候选列表 | 低 |

**技术判断**：这个命令的跨平台实现明显比第1节"在外部终端打开"更简单——三个平台都有各自稳定的"标准答案"，Linux 尤其因为 `xdg-open` 的存在而不需要像终端那样做多候选探测。"定位到具体文件并选中"（而不只是打开目录）这个增强功能在 Windows/macOS 都有对应参数（`/select,`、`-R`），Linux 的 `xdg-open` 不支持"选中文件"语义，只能打开所在目录，这是 Linux 平台上一个功能略逊于其他两个平台的已知限制，建议在实现时明确记录而不是假装三平台行为完全一致。

## 3. Viva 是否已提供跨平台外部程序调用能力

**结论：没有发现任何证据表明 Viva 提供了"打开外部程序"或"在文件管理器中显示"这类 API，`.claude/skills/building-with-viva/SKILL.md` 通篇没有提及这类能力，`docs/UI.md` §8"Viva 缺口"清单（三方合并视图、Diff 上下排列、hunk 勾选、"Shift 两次"搜索、递归文件监听）里也没有列出这一项——这意味着目前没有人问过 Viva 是否打算提供这个能力，这个问题本身是悬而未决的，不是"已确认 Viva 不提供"。**

核查方法：读取 `.claude/skills/building-with-viva/SKILL.md` 全文，这是本项目开发 Viva 应用的权威参考文档，列出了程序模型规则、headless 测试协议、以及"Decide whose code it is"这条判断标准（第10行起）：*"A reusable desktop capability: windows, input, text and IME, focus, layout, components, rendering, accessibility, lifecycle, headless control. If a second Viva app would copy the same helper or wiring, it belongs in Viva, not in your app."* ——"打开系统默认程序处理某个路径"（无论是终端还是文件管理器）完全符合这条判断标准里"第二个 Viva 应用也会需要同样的逻辑"的描述，理论上应该先问 Viva 维护者是否已有或计划提供这类 API，而不是直接在应用里用 `os/exec` 绕过。

**这和 P8/P10 已经发现的模式是同一类问题**（P8 发现"没人问过 Viva 是否支持拖拽重排列表"，P10 发现动作注册表设计问题），都是"在没有向框架提需求的情况下，应用层默默地要自己实现一个本来可能属于框架职责范围的能力"。

**与 P8/P10 已发现问题的区别**：三方合并视图、拖拽重排序这些是复杂 UI 交互能力，Viva 不太可能悄悄支持却没人发现；但"调用系统默认程序打开一个路径"是一个相对底层、简单、所有桌面应用都需要的基础能力，这种级别的能力更有可能已经存在于 Viva 的 `app` 包里只是没在 SKILL.md 里被提及（SKILL.md 本身声明不是完整 API 参考，而是"读这个文件，再读它指向的指南，不要猜 API"）。本次核查受限于只读取了 SKILL.md 而没有深入 Viva 源码本身（`go.mod` 里 `replace github.com/vivaui/viva => D:/viva` 指向本机的 Viva 工作树，若要彻底确认需要直接搜索该目录的 `app` 包源码），**建议在实现前先搜索 `D:/viva` 的 `app` 包是否已有类似 `app.OpenPath`/`app.RevealInFileManager`/`app.OpenExternal` 这样的函数，如果有，直接用；如果没有，再考虑是否要先向 Viva 提 issue（参照三方视图、拖拽排序的处理方式）还是因为这个能力足够基础、跨平台实现足够简单（第1、2节已给出的 `os/exec` 方案），直接在应用层用标准库实现并不违反"先问框架"的精神**（因为调用系统命令本身就是 Go 标准库职责范围，不是"绕过 Viva 用平台调用workaround"的那种情况——第1、2节的方案全部基于 `os/exec` 调用系统自带程序，没有触碰 Viva 本该负责的渲染、输入、窗口等领域）。

## 4. 发布依赖测量要求的适用性评估

**结论：这两个命令大概率不会触发 `AGENTS.md`"新增发布依赖前测量剥离后二进制大小、启动时间、空闲内存"这条硬性约束，因为它们只使用标准库 `os/exec`，不引入任何新的第三方依赖。**

`AGENTS.md`"必须遵守"一节原文："新增发布依赖前测量剥离后二进制大小、启动时间、空闲内存"——这条约束的触发条件是"新增发布依赖"（即 `go.mod` 里新增一个外部包），第1、2节给出的实现路径全部是：
- 调用系统已存在的外部程序（`explorer`/`open`/`xdg-open`/`cmd` 等）——这些不是 Go 依赖，是运行时环境里的外部二进制，不计入"发布依赖"。
- 使用标准库 `os/exec` 发起调用——标准库不是需要测量的"新增依赖"，项目里已经在用（`internal/gitexec`、`internal/git/version.go`），这条能力的使用方式和现有代码模式一致，不是新引入的技术栈。

**可以豁免测量的结论成立，前提是实现时严格只用标准库**。如果未来发现需要引入第三方库（比如某个跨平台"打开系统默认程序"的封装包，类似 `github.com/pkg/browser` 这类社区库）来简化第1节 Linux 终端探测的复杂度，那时才需要重新评估是否触发测量要求——本次盘点基于"用标准库自己写"这个假设给出结论，不代表"无论怎么实现都不需要测量"。

## 5. CodeDiff 组件语法高亮能力的使用现状

**结论：这是本次核查的关键新发现——`components.CodeDiffSnapshot` 结构体本身提供了可插拔的语法高亮回调机制（`Language string` + `Highlight func(...)` 字段），这个能力完全独立于 TextMate bundle 决策，不需要接入 TextMate 就能使用，但 cetacean 当前两处调用都没有设置这两个字段，等于完全没用上这个现成能力。**

证据：直接读取 Viva 源码（`go.mod` 的 `replace github.com/vivaui/viva => D:/viva` 指向的本机工作树）`D:/viva/components/code_diff.go` 第8-17行：

```go
type CodeDiffSnapshot struct {
    Result CodeDiffResult
    ID       ui.ID
    Label    string
    Hint     string
    Value    string
    Language string
    // Highlight colors the changed rows, as for a CodeEditor.
    Highlight    func(language, line string, dst []CodeEditorRun) []CodeEditorRun
    AddedColor   ui.Color
    RemovedColor ui.Color
    ...
}
```

第343-344行的渲染逻辑确认这个回调真的会被调用（不是摆设字段）：

```go
if changed && diff.snapshot.Highlight != nil && len(value) <= CodeEditorMaxHighlightedLine {
    spec.highlight = cleanHighlight(value, diff.snapshot.Highlight(diff.snapshot.Language, value, diff.marks[:0]))
}
```

即：只要应用层提供一个 `Highlight` 函数（接收语言名和一行文本，返回着色区间 `[]CodeEditorRun`），`CodeDiff` 组件就会在渲染每一行改动时调用它做语法着色，这和 `components.CodeEditor` 的高亮机制是同一套接口（注释明确写"as for a CodeEditor"），说明 Viva 已经有一套通用的、组件间共享的高亮回调约定。

cetacean 当前的两处调用点（`internal/ui/shell/shell.go:828-832` 和 `835-840`）：

```go
m.codeDiff.Paint(canvas, theme, m.diffSlot.Rect(), components.CodeDiffSnapshot{
    Result: m.aligned, ID: diffID, Label: text(i18n.StagedDiff), Value: m.comparison.Right,
    ReadOnly: true, FontSize: 12, VerticalScrollbarID: diffVScrollID,
    HorizontalScrollbarID: diffHScrollID, ViewportID: diffViewportID,
})
```

**两处都没有设置 `Language` 和 `Highlight` 字段**，Go 的零值规则下 `Language` 是空字符串，`Highlight` 是 `nil` 函数——对照上面第343行的判断条件 `diff.snapshot.Highlight != nil`，这个条件永远为假，高亮逻辑完全不会被触发，当前所有 Diff 视图展示的都是纯无色文本。

**这和 TextMate 决策完全独立**：P1 排除的是"接入 TextMate bundle 解析引擎"这件事本身（引入一整套通用语言语法规则解析器，规格判定为超出"Git 客户端"范围的编辑器能力）。而这里发现的缺口是"Viva 自己的 `CodeDiff` 组件已经预留了一个简单的回调接口，哪怕只实现几种最常见语言（比如 Go、JSON、Markdown 这类项目自身可能最常遇到的文件类型）的基础关键字/字符串/注释着色，也不需要 TextMate，纯粹是应用层写一个小型高亮函数"。规格 §2 排除的是"语法高亮"这个能力类别本身，如果严格按规格字面意思，这个回调即使实现了也不该用在首发——**这里需要和用户确认的是：规格"不做语法高亮"这条排除决定，是否包含"Diff 视图里完全不给改动行做任何代码着色"，还是只排除"通用任意语言的完整 TextMate 级别高亮"，中间可能存在一个"只做几种常见语言的基础着色，提升 Diff 可读性"的折中地带，这是产品范围问题，本节只负责指出技术上这个能力现成可用，不代为决定是否要用**。

## 汇总：P12 待办清单

**不需要决策、技术路径明确的小功能（可直接排期）**
1. **"在外部终端打开"命令**——`os/exec` 调用系统终端，Windows/macOS 路径简单，Linux 需要候选列表探测（第1节），不触发依赖测量要求（第4节），建议先搜索 `D:/viva` 的 `app` 包确认 Viva 是否已有等价 API（第3节）再决定是否自己实现。
2. **"在文件管理器打开"命令**——P1-P11 遗漏的盘点项，本次补上，三平台实现路径都比终端命令简单（第2节），同样先确认 Viva 是否已提供。

**需要向 Viva 提需求或至少查证源码的点（参照 P8/P10 已发现的"先问框架"模式）**
3. 在实现第1、2条之前，先搜索 `D:/viva` 的 `app` 包（不只是 SKILL.md）确认是否已有 `OpenPath`/`RevealInFileManager` 一类跨平台能力，避免重复发明轮子；如果确认没有，这个请求的量级足够小、足够基础，可以直接用标准库 `os/exec` 实现，不需要像三方合并视图那样等待框架决策（第3节已给出判断依据）。

**独立于 TextMate 决策的新发现，需要产品范围确认**
4. `components.CodeDiffSnapshot` 的 `Language`/`Highlight` 字段完全未被使用（第5节），这个能力不依赖 TextMate、不引入新依赖（`Highlight` 是应用层自己写的纯函数），技术上随时可以给几种常见语言加基础着色。需要用户确认：规格"不做语法高亮"的排除范围是否包含这个更轻量的折中方案，还是可以认为这属于"Diff 可读性"的合理范畴而不算"语法高亮功能"。这是本次 P1-P12 全系列盘点里少数几个"做不做是产品范围问题、但技术上完全现成"的发现之一。

**里程碑状态**

P1-P12 全部盘点已完成。本篇是 P1-P12 系列的最后一项（P12），至此 Rebased parity 的基线动作清单（P1）、主窗口视觉（P2）、分栏与工具窗（P3）、Commit/Shelf（P4）、Changes/Diff（P5）、Branch/Refs（P6）、Log（P7）、历史编辑与冲突（P8）、Stash（P9）、设置与动作系统（P10）、Hosting 集成（P11）、Terminal/TextMate（P12）十二份差异盘点文档全部产出完毕，均已写入 `docs/` 目录并在 `docs/ROADMAP.md` 的对应任务行留有摘要与链接。
