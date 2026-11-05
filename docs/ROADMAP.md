# 里程碑与待决事项

每个里程碑结束时：规格 §10 对应验收项有自动化测试，`go vet` / `go test` 通过，性能预算中相关项已测量。

## 里程碑

| 编号 | 内容 | 规格章节 | 完成标志 |
|---|---|---|---|
| M0 | 工程骨架：`go.mod`、目录结构、`gitexec` 与版本检测、空主窗口（顶栏/导航/状态栏）、Driver 测试与 headless 入口 | §3、§4.1 末条 | 空窗口 headless `tree` 输出含全部导航项且 `expect named` 通过 |
| M1 | **垂直切片**：打开仓库、Changes 三分组、文件级 Stage/Unstage、文件 Diff、Commit；文件监听刷新 | §4.1、§4.2、§4.3、§11 | 按规格 §11 测量启动、刷新、二进制大小、无障碍，写入 `docs/benchmarks/`；**决定是否继续使用 Viva** |
| M2 | Log：分批加载、图布局、筛选、提交详情与文件 Diff、Copy SHA | §4.4 | 基准仓库滚动无卡顿，图形增量连续 |
| M3 | 分支：选择器、Branches 视图、新建/切换/重命名/删除、upstream、Checkout 安全检查 | §4.5 | Checkout 覆盖文件时出现 Cancel/Stash/Force |
| M4 | 远端：Fetch/Pull/Push、进度与取消、错误分类、Force push with lease | §4.6、§7 | bare 仓库测试覆盖 §6 全部错误分类中的可构造项 |
| M5 | Changes 完整化：hunk 级暂存、Discard、Amend、提交并推送、提交消息历史、目录树模式 | §4.2 | 规格 §10 第 3、4 项 |
| M6 | Merge / Rebase / Cherry-pick / Revert / Reset 与预检确认 | §4.4、§4.7 | 有损操作全部经过预检 |
| M7 | 冲突解决：冲突清单、三方视图、Continue/Abort | §4.8 | 规格 §10 第 6 项 |
| M8 | Stash 全流程 | §4.9 | 规格 §10 第 8 项 |
| M9 | 快捷键：三套预设、Keymap 设置、冲突检测、命令面板 | §5 | 规格 §10 第 9 项 |
| M10 | 设置、i18n 补全、无障碍审查、诊断导出 | §6、§7、§8 | 规格 §10 剩余项与三平台人工验收 |
| 后续 | Worktree、子模块、交互式 Rebase、Tag/远端管理、托管平台集成 | §2 后续 | — |

快捷键和 i18n 的基础设施（动作注册表、字符串表）在 M0 建立，各里程碑新增动作与文案时直接登记，M9/M10 只做预设补全与设置界面。

## 待决事项

开发开始前或对应里程碑开始前需要确认：

| 编号 | 问题 | 当前倾向 | 何时决定 |
|---|---|---|---|
| D1 | Go module 路径 | M0 使用本地模块名 `r-git-workbench`；确定发布仓库 owner 后再改为正式地址 | 发布前 |
| D2 | 最低 Git 版本 | M0 暂定 2.34；较老版本显示警告，仅允许只读使用 | M1 验证 |
| D3 | 文件监听实现 | 自写三平台实现（`ReadDirectoryChangesW` / FSEvents / inotify），不引入 fsnotify：fsnotify 在 macOS 用 kqueue，每个文件占一个描述符，不适合大仓库；已在 [viva#133](https://github.com/vivaui/viva/issues/133) 询问是否由 Viva 提供 | M1 前 |
| D4 | Diff 视图：扩展 Viva `CodeDiff` 还是应用内自绘 | M1 先用 `CodeDiff` 做并排只读，hunk 勾选与上下排列看 [viva#131](https://github.com/vivaui/viva/issues/131) 进展，M5 前定 | M5 前 |
| D5 | “Shift 两次”触发全局搜索 | 已提议 [viva#130](https://github.com/vivaui/viva/issues/130)；未支持前首发不提供此键位，规格偏差需记录 | M9 前 |
| D6 | 内置 askpass 对话框 | 首发不做，依赖 credential helper / SSH agent | M4 前 |
| D7 | 是否保留 Viva | 以 M1 测量结果为准（规格 §11 要求以垂直切片决定技术栈） | M1 结束 |
| D8 | 打包与更新渠道 | `viva build` / `viva package`；自动更新暂不做 | M10 前 |

## M0 进度

已建立 Go 1.26 / Viva v0.2.4 工程、`gitexec` 子进程封装、Git 定位与版本能力检测、i18n 字符串表、导航动作注册表和可切换的空窗口。Git 检测发生在窗口创建前；后续仓库操作的异步结果已有 `workbench` 收件箱、`Invalidator`、`OnWake` 接口。M1 需要把首次仓库加载放到后台，并补齐写操作排队、子进程树取消、Git 日志和输出上限。

M0 headless 与二进制测量见 [benchmarks/M0_WINDOWS.md](benchmarks/M0_WINDOWS.md)。

本机缓存的 Viva v0.2.4 headless 协议使用 `hits` / `layout`，没有文档中所写的 `tree` / `expect named`。M0 用 Driver 和真实 headless 行协议断言全部导航标识与可访问名称、布局零问题；升级到支持新协议的 Viva 版本时补做原命令验收。
