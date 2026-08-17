# Bug Reproduction

## 包的性质

当前 test_model_fix 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。要复现原始缺陷，必须检出下面固定的 parent SHA；不要在当前修复结果源码上期待重新出现修复前失败。生成系统使用的可信验证补丁和完整验证日志仅在本地留存，不提交到结果分支。

## 问题现象

异常上报超时告警完全对不上，先不要修改代码，帮我定位原因。

线上实测（后台监控每 2 秒跑一轮，上报时限用环境变量 ANOMALY_REPORT_DEADLINE 配置）：

  - 把上报时限设成 1 分钟：巡检工单检测到异常之后 8 秒，GET /api/notifications 里已经堆了 4 条 anomaly_deadline_expired（“anomaly report deadline expired for order WO-1”），而且每轮监控都再加一条。巡护员还在现场取证、时限根本没到，值班台就被刷屏了。
  - 把上报时限设成 3 秒、检测后一直不上报：等了 10 秒，通知里只有 1 条，之后再也不涨 —— 真正该报的超时反倒没有持续告警。
  - 一旦正式提交了异常报告，告警就停了。

期望是：时限内不告警，超过时限还没上报才告警。请说明是哪个 Go 文件、哪个符号的什么行为导致上面这两个相反的现象，以及它如何一路影响到通知列表。请给出实际证据。这一轮只要诊断结论，请先不要修改仓库里的代码。

## 含 Bug 版本

- 仓库：11DingKing/go-eb409f-t020-03
- 仓库地址：https://github.com/11DingKing/go-eb409f-t020-03.git
- parent SHA：c122fa03b0cd18abdec1d2ca64b02c802a4715ed

## 复现步骤

```bash
git clone -- https://github.com/11DingKing/go-eb409f-t020-03.git bug-repro
cd bug-repro
git checkout --detach c122fa03b0cd18abdec1d2ca64b02c802a4715ed
go test ./internal/dispatch -run "^TestAnomalyDeadlineMonitorTiming$" -count=1 -v
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/dispatch -run "^TestAnomalyDeadlineMonitorTiming$" -count=1 -v
=== RUN   TestAnomalyDeadlineMonitorTiming
    anomaly_deadline_test.go:30: orders [WO-1] reported overdue only 0s after detection, still inside the reporting window
--- FAIL: TestAnomalyDeadlineMonitorTiming (0.00s)
FAIL
FAIL	microgrid-ops/internal/dispatch	0.040s
FAIL

```

stderr：

```text
(empty)
```

### linux/arm64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./internal/dispatch -run "^TestAnomalyDeadlineMonitorTiming$" -count=1 -v
=== RUN   TestAnomalyDeadlineMonitorTiming
    anomaly_deadline_test.go:30: orders [WO-1] reported overdue only 0s after detection, still inside the reporting window
--- FAIL: TestAnomalyDeadlineMonitorTiming (0.00s)
FAIL
FAIL	microgrid-ops/internal/dispatch	0.004s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

准确定位 internal/workorder/workorder.go 的 (*AnomalyInfo).IsExpired：把“当前时间晚于上报截止时间”写成了“当前时间早于上报截止时间”
解释 dispatch.(*Orchestrator).DetectAnomaly 写入的 AnomalyInfo.ReportDeadline = DetectedAt + 上报时限，(*Orchestrator).CheckAnomalyDeadlines（由 scheduler 每个 interval 调用）对 in_progress 且带 Anomaly 的工单调用 IsExpired(now)；判定方向反了以后，窗口内每一轮都判为超时并追加一条 anomaly_deadline_expired 通知（因此 8 秒内出现 4 条并持续增长），窗口过后判定恒为 false 而不再告警（3 秒窗口等 10 秒只留下窗口内产生的那 1 条）；并说明 ReportAnomaly 写入 ReportedAt 后工单转入 anomaly_reported，既被 IsReported 也被状态过滤挡掉，所以上报后告警停止
结论有代码阅读或定向复现证据（如 go test ./internal/dispatch -run '^TestAnomalyDeadlineMonitorTiming$' -count=1 -v，或按题面用 HTTP 观察通知列表）；目标仓库保持零改动
