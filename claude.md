# claude.md

## Project
Google 流量任务系统（Go）

## Mission
实现一个可运行的 Master + Agent 分布式下载任务平台：
- Master 在 Web 上管理多个 Agent 节点
- Master 下发下载任务（YouTube / Google 静态资源）
- Agent 按任务参数执行并上报指标
- 每个任务支持精确速率控制（Mbps 级）
- 每个任务支持时间控制、总量控制和自然流量画像控制
- 支持在 Web 输入目标主机 IP/SSH 信息后由 Master 自动部署并纳管 Agent

## Hard Requirements
1. Language: Go 1.22+
2. Architecture: `cmd/master` + `cmd/agent` + shared internal packages
3. API: REST JSON first
4. DB: SQLite first (repository pattern, easy to swap to PostgreSQL)
5. Observability: structured logs + `/metrics` + `/healthz`
6. Reproducible run: Docker Compose required
7. Scheduler + Traffic Shaper required (`start_at/end_at/total_bytes_target/profile`)
8. Agent provisioning required (SSH bootstrap + service install + auto register)
9. YouTube tasks must use `yt-dlp`; do not implement custom YouTube protocol parsing.
10. Dashboard required: real-time total bandwidth + 7-day history from SQLite

## Functional Scope

### Master
- Agent 注册、心跳、在线状态管理
- Agent 自动纳管（SSH 远程部署、配置写入、服务拉起、状态回写）
- 任务创建、调度、停止、查询
- 任务指标收集与展示
- Web UI: 节点列表、任务列表、任务创建、任务详情（含速率曲线）
- 支持任务时间窗口、总量目标、流量画像模板配置
- 支持任务下发节流：`dispatch_rate_tpm`、`dispatch_batch_size`
- Web UI 提供“新增 Agent 向导”：IP、SSH 端口、用户名、认证方式、部署日志
- Web UI 提供 Dashboard：实时总带宽、Agent 带宽排行、7 天历史曲线

### Agent
- 注册到 Master
- 定时心跳
- 拉取任务并执行
- 定时上报指标
- 支持任务取消与优雅退出
- 执行阶段支持 `ramp-up / steady / ramp-down` 与请求间隔抖动
- YouTube 任务通过 `yt-dlp` 子进程执行

## Agent Provisioning (Critical)
- 输入字段：
  - `host_ip`, `ssh_port`, `ssh_user`
  - `auth_type` (`key` or `password`)
  - `credential_ref` (not raw private key in DB)
- 执行流程：
  1. SSH connectivity check
  2. upload agent binary + config
  3. install/update systemd service
  4. start service and health check
  5. verify agent appears online in Master
- 验收：
  - 新增 Agent 后 60s 内可见 `online` 状态
  - 失败任务有可追踪日志和失败步骤
  - 支持安全重试，不留下损坏 service

### Task Types
- `youtube`: 视频资源下载任务（URL 输入，执行器固定为 `yt-dlp`）
- `static`: Google 静态资源下载任务（URL 输入）

## YouTube Executor (yt-dlp)
- Required dependency: `yt-dlp` installed on Agent
- Mapping:
  - `target_rate_mbps=10` => `yt-dlp --limit-rate 10M <url>`
  - optional: `--concurrent-fragments <n>`, `--retries <n>`
- Runtime behavior:
  - run `yt-dlp` as managed subprocess
  - collect stdout/stderr for progress and error tracking
  - translate download stats into Mbps metrics to Master
  - stop task by graceful process termination

## Rate Control (Critical)
- 字段：`target_rate_mbps`
- 算法建议：Token Bucket + fixed interval controller (1s tick)
- 监控窗口：
  - `avg_rate_5s`
  - `avg_rate_30s`
- 验收：
  - 5 秒窗口误差 <= ±10%
  - 30 秒窗口误差 <= ±5%

## Time & Volume Control (Critical)
- 字段：
  - `start_at`, `end_at` (or `duration_sec`)
  - `total_bytes_target`, `total_requests_target`
  - `dispatch_rate_tpm`, `dispatch_batch_size`
  - `distribution` (`flat` / `ramp` / `diurnal`)
  - `jitter_pct`, `ramp_up_sec`, `ramp_down_sec`
- 验收：
  - 启动时间偏差 <= 2s
  - 停止时间偏差 <= 2s
  - 总量误差 <= ±3%
  - 下发节奏符合 `dispatch_rate_tpm`，无异常突发
  - 1 分钟窗口曲线趋势与画像模板一致

## Dashboard & Retention (Critical)
- Dashboard widgets:
  - real-time total bandwidth of all online agents
  - per-agent current bandwidth
  - historical speed chart for last 7 days (1m/5m aggregation)
- Storage:
  - persist bandwidth samples in local SQLite
  - retain only last 7 days
  - run hourly purge job for expired rows
- Acceptance:
  - total bandwidth equals sum(agent bandwidth) with <= ±2% drift
  - no data older than 7 days remains in SQLite

## Suggested Project Layout
```text
.
├── cmd/
│   ├── master/
│   └── agent/
├── internal/
│   ├── master/
│   ├── agent/
│   ├── api/
│   ├── store/
│   └── model/
├── pkg/
│   └── ratelimit/
├── web/
├── deployments/
│   └── docker-compose.yml
└── README.md
```

## API Minimum
- `POST /api/v1/agents/register`
- `POST /api/v1/agents/heartbeat`
- `GET /api/v1/agents/{agent_id}/tasks/pull`
- `POST /api/v1/tasks`
- `POST /api/v1/tasks/{id}/dispatch`
- `POST /api/v1/tasks/{id}/stop`
- `POST /api/v1/tasks/{id}/metrics`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/{id}`
- `POST /api/v1/traffic-profiles`
- `GET /api/v1/traffic-profiles`
- `POST /api/v1/tasks/{id}/schedule`
- `POST /api/v1/tasks/{id}/run` (optional manual trigger)
- `POST /api/v1/agents/provision`
- `GET /api/v1/agents/provision-jobs`
- `GET /api/v1/agents/provision-jobs/{job_id}`
- `GET /api/v1/dashboard/overview`
- `GET /api/v1/dashboard/bandwidth/history?from=&to=&step=1m`

## Engineering Rules
1. Always use `context.Context` for long-running operations.
2. Every goroutine must have clear lifecycle and cancellation path.
3. HTTP client must set timeout and retry policy.
4. Config from env vars with sane defaults.
5. Provisioning secrets must be referenced by ID (`credential_ref`) and encrypted at rest.
6. Add unit tests for:
   - rate limiter behavior
   - task state transitions
   - metrics aggregation
   - schedule planner and profile curve generation
   - provisioning state machine
   - metrics retention purge (7 days)
7. Add integration smoke test for Master+Agent basic flow.

## Security & Compliance
- Only run traffic against authorized targets.
- Enforce allowlist for target domains.
- Use bearer token between Master and Agent.
- Do not hardcode secrets; load from env.
- This project is for load testing/capacity validation only, not for bypassing platform controls.
- Do not persist plaintext SSH private keys or passwords in logs.
- Respect YouTube terms and copyright requirements for downloaded content.

## Delivery Checklist (per iteration)
1. Modified file list
2. Why changes were made
3. Run commands
4. Test/verification results
5. Next step

## First Execution Plan
1. Scaffold Go modules and folders
2. Implement Master HTTP server + SQLite models
3. Implement Agent provisioning workflow (SSH bootstrap + systemd)
4. Implement Agent register/heartbeat/pull/report loop
5. Implement scheduler + traffic profile engine
6. Implement rate-limited downloader engine
7. Implement minimal Web pages
8. Add Docker Compose and README
