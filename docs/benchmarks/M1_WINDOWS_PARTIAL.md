# M1 Windows partial measurements

Date: 2026-11-05; OS: Windows/amd64; CPU: Intel Core i7-14700; Go 1.26.3; local Viva checkout `D:/viva`. No native window was opened.

| Metric | Result | Method |
|---|---|---|
| Stripped executable | 4,736,000 bytes | `go build -ldflags='-s -w' -o <temp> ./cmd/workbench`; M0 baseline on a different CPU and Viva v0.2.4: 4,418,048 bytes, a 317,952-byte increase. Earlier M1 shell without `CodeDiff`/watcher: 4,617,728 bytes. |
| Navigation paint | 22,426 ns/op, 0 allocs/op | `go test ./internal/ui/shell -run '^$' -bench BenchmarkNavigationPaint -benchmem -benchtime=100x`; 100 clicks. The benchmark measures navigation, not Changes with a populated repository. |
| Changes layout at 720 x 480 | No layout problems in a temporary repository with one file | `TestDriverOpenStageAndCommit` uses `app.Driver` and `headless.CheckLayout`; the side-by-side comparison also passes layout checks at 1100 x 720. |
| Windows filesystem event refresh | External edit appears in Changes, including after switching repositories | `go test ./internal/ui/shell -run TestDriverOpenStageAndCommit -count=5` uses real temporary repositories and an app.Driver window. |
| Headless startup to ready | 84.4 / 75.5 / 73.3 / 78.7 / 79.4 ms; median 78.7 ms | `go build -tags vivahead`, launch with `VIVA_HEADLESS=1`, time from process start until `ok ready`; five warm-cache launches with Git detection, no repository. |
| Headless idle memory | Working set 13,692,928 bytes; private memory 17,707,008 bytes | After fifth ready reply, wait 500 ms and read `Process.WorkingSet64` and `PrivateMemorySize64`; one sample without a repository. |

Real-window startup, status refresh time and idle memory on a representative 50,000-file repository, and accessibility on the three platforms have not been measured. The headless figures above cannot replace those checks. This is not sufficient to complete M1 or decide whether to retain Viva.

M2 incremental measurement (same machine and Go version, local `D:/viva`): after adding paged Log, message search and commit file Diff, `go build -ldflags='-s -w' -o <temp> ./cmd/workbench` produced a 4,785,152-byte executable. `go test ./internal/ui/shell -run '^$' -bench BenchmarkNavigationPaint -benchmem -benchtime=100x` measured 23,470 ns/op and 0 allocs/op for navigation. This benchmark does not measure a populated Log or large-repository scrolling.

M3 partial measurement (2026-11-06, same machine and local Viva): with read-only refs, branch creation/switching and the redesigned shell, the stripped executable is 4,871,680 bytes. `BenchmarkNavigationPaint` (100 iterations) measured 56,293 ns/op, 0 allocs/op; it does not cover a populated branch view. A headless build started at 1200 x 760 and passed `expect named` and `expect layout clean` both without a repository and with the current repository. A headless screenshot of the repository view was reviewed; no real window was opened. Startup time and idle memory were not remeasured for this increment.
