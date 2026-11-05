# r-git-workbench 开发约定

基于 Viva（Go，GPU 自绘）的独立 Git 桌面客户端。

## 先读

- 产品范围：`docs/GIT_CLIENT_PRODUCT_SPEC.md`
- 架构：`docs/ARCHITECTURE.md`，细节在 `docs/GIT_BACKEND.md`、`docs/UI.md`、`docs/TESTING.md`
- 进度与待决事项：`docs/ROADMAP.md`
- Viva 编程模型：`.claude/skills/building-with-viva/SKILL.md`

## 必须遵守

- 只通过子进程调用系统 `git`；不链接 Git 库。
- 依赖方向向下：`ui → workbench → repo → git → gitexec`。`internal/git` 及以下不导入 Viva。
- 绘制函数不做 I/O、不等待；后台结果经收件箱 + `Invalidator` + `OnWake` 进入窗口线程。
- 每仓库同时只有一个写操作。有损操作必须走“预检 → 确认 → 执行”，执行时校验预检时的仓库状态。
- 空闲时没有轮询、心跳或定时器。
- 控件标识用 `ui.Tree` 命名，不手工编号；标识名是测试接口。
- 新增发布依赖前测量剥离后二进制大小、启动时间、空闲内存。
- 用户可见文字走 `internal/i18n`；Git 原始输出、路径、引用名不翻译。
- 不打开真实窗口做验证，除非用户同意；用 `app.Driver` 和 headless 脚本。

## 完成前

```sh
gofmt -l .
go vet ./...
go test ./... -count=1
```

架构或范围变化时同步更新 `docs/` 中对应文档和 `docs/ROADMAP.md`。
