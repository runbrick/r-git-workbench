# M0 Windows baseline

测量时间：2026-11-05；系统：Windows/amd64；CPU：AMD Ryzen 7 7735H；Go 1.26.3；Git 2.53.0.windows.1；Viva v0.2.4。测试机上不打开真实窗口。

| 指标 | 结果 | 方法 |
|---|---|---|
| 剥离后发布二进制 | 4,418,048 字节 | `go build -ldflags="-s -w" -o <temp> ./cmd/workbench`；从零建立项目，无变更前二进制 |
| headless 启动到 ready / 首帧 | 热缓存五次中位数 82.2 ms；首次 3,199.9 ms | `go build -tags vivahead` 后启动独立进程，设 `VIVA_HEADLESS=1`，计时至协议输出 `ok ready`；五次样本 3199.9 / 82.2 / 81.6 / 85.1 / 76.8 ms；包括 Git 版本探测，不代表真实 GPU 窗口启动 |
| 空闲工作集 / 私有内存 | 12,574,720 / 17,137,664 字节 | headless 收到 `ok ready` 后等待 500 ms，再读 `Process.WorkingSet64` 与 `PrivateMemorySize64`；单次样本，无仓库 |
| 导航点击并重绘 | 45,754 ns/op；0 allocs/op | `go test ./internal/ui/shell -run '^$' -bench BenchmarkNavigationPaint -benchmem -benchtime=100x`；100 次点击相同导航项，24 B/op 是基准统计的非分配开销 |

M1 的性能验收需要使用真实窗口和中型仓库重新测量启动、状态刷新、空闲资源与跨平台行为；此处仅为 M0 headless 基线。
