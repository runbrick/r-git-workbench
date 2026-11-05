# 测试与性能预算

## 1. 测试分层

| 层 | 对象 | 方式 | 依赖 |
|---|---|---|---|
| 解析器 | `internal/git` 中的 `parseX` | 表驱动 + `go test -fuzz`；样例取自真实 Git 输出，存于 `testdata/` | 无 |
| Git 操作 | `internal/git` 的命令函数 | 在 `t.TempDir()` 中建真实仓库，用系统 `git` 执行，再用 `git` 命令核对结果 | 系统 git |
| 远端流程 | Fetch / Pull / Push / Force push | 本地 bare 仓库充当远端（`file://`），不访问网络（规格 §9） | 系统 git |
| 仓库服务 | `internal/repo` 写队列、刷新、预检一致性校验 | 真实仓库 + 可注入的假 `fswatch` 事件源 | 系统 git |
| 图布局 | Log 泳道算法 | 构造的小历史（线性、分叉、八爪合并、多根）表驱动 | 无 |
| 界面行为 | 各视图 | `app.Driver`：`Click`、`Key`、`Type` 后读 `Snapshot` 与 `Session.Expect` | 无窗口 |
| 端到端脚本 | 完整工作流 | `-tags vivahead` 构建，`VIVA_HEADLESS=1` + 临时 `HOME`，stdin 发送命令，脚本以 `expect` 行结尾 | 无窗口 |
| 人工验收 | IME、真实键盘、高 DPI、凭据委派、窗口管理器 | 每个平台按规格 §10 清单执行，记录未覆盖项 | 真实窗口 |

规则：

- 测试用户依赖的行为（文件出现在正确分组、提交后 Log 出现新提交），不测像素偏移或绘制原语数量。
- 测试仓库的 Git 配置隔离：设置 `GIT_CONFIG_GLOBAL`、`GIT_CONFIG_SYSTEM` 指向临时文件，`HOME` 指向临时目录，固定 `user.name`/`user.email` 与提交时间（`GIT_AUTHOR_DATE`、`GIT_COMMITTER_DATE`）。
- 失败恢复必须有测试：hook 失败保留消息、identity 缺失、index.lock 存在、取消长操作后状态重读。
- headless 自动化入口放在 `//go:build vivahead` 文件中，发布构建不包含。

## 2. 规格 §10 验收映射

| 验收项 | 自动化层 |
|---|---|
| 打开/克隆仓库，辨认分支与同步状态 | Git 操作 + Driver |
| 文件分组正确，Diff 与 CLI 一致 | Git 操作（与 `git diff` 输出逐字比较）+ Driver |
| 文件级 / hunk 级 Stage | Git 操作（比较 `diff --cached` 与 `diff`） |
| 提交失败不丢消息 | 仓库服务 + Driver |
| 分支、Fetch、Merge/Rebase Pull、Push、upstream 引导 | 远端流程 + Driver |
| 冲突三方解决、Continue/Abort | Git 操作 + Driver |
| Log 搜索、Cherry-pick、Revert、受保护 Reset | Git 操作 + Driver |
| Stash 全流程与冲突 | Git 操作 + Driver |
| 快捷键预设、重映射、冲突、焦点不抢键 | `keymap` 单元测试 + Driver |
| 有损操作展示影响并确认 | 仓库服务（预检结果）+ Driver（确认框内容） |
| 三平台人工验收 | 人工 |

## 3. 性能预算

以下为初始目标，在 M1 垂直切片中测量后修订。数字必须附测量方法与机器，未测量前不对外宣称（Viva 规则 §2）。

| 指标 | 目标 | 测量方法 |
|---|---|---|
| 冷启动到显示最近仓库 | < 300 ms | 进程启动到首帧，`docs/benchmarks/` 记录脚本 |
| 打开中型仓库到 Changes 可交互 | < 500 ms | 基准仓库：约 5 万文件、1 万次提交 |
| 文件变化到 Changes 刷新 | < 去抖 300 ms + 一次 `git status` | 同上 |
| Log 首批显示 | < 300 ms | 基准仓库，首批 1000 条 |
| 空闲 CPU | 0（无唤醒） | 平台工具观察 1 分钟 |
| 空闲内存 | 待测量后定 | 打开基准仓库后 |
| 每帧分配 | Changes / Log 滚动稳态为 0 | `go test -bench` + `-benchmem` |
| 剥离后二进制 | 待测量后定，任何依赖变化前后对比 | `go build -ldflags="-s -w"` |

## 4. 完成前检查

每次改动结束前运行：

```sh
gofmt -l .
go vet ./...
go test ./... -count=1
```

涉及绘制或输入路径时附 `go test -bench` 的每帧分配与稳态耗时；涉及链接的改动附剥离后二进制大小变化。
