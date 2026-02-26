# ngoogle — Google 流量任务系统

Master + Agent 分布式下载任务平台，支持 YouTube 和静态资源下载，具备精确速率控制、时间窗口调度和自然流量画像。

## 快速启动

### 使用 Docker Compose

```bash
# 构建并启动 Master + Agent
cd deployments
docker compose up --build

# 访问 Web UI
open http://localhost:8080
```

### 本地开发

```bash
# 启动 Master（先构建前端）
cd web && npm install && npm run build && cd ..
go run ./cmd/master

# 启动 Agent（另一个终端）
MASTER_URL=http://localhost:8080 go run ./cmd/agent
```

### 前端开发模式

```bash
cd web
npm run dev   # 启动 Vite dev server（:5173），自动代理到 Master
```

## 架构

```
┌──────────────────────────────────────────────────────┐
│                     Master (:8080)                    │
│  ┌─────────┐  ┌──────────┐  ┌─────────────────────┐  │
│  │  REST   │  │Scheduler │  │  Dashboard/Metrics  │  │
│  │  API    │  │ (5s tick)│  │  /metrics /healthz  │  │
│  └────┬────┘  └─────┬────┘  └─────────────────────┘  │
│       │             │                                  │
│  ┌────▼─────────────▼──────┐                          │
│  │     SQLite Store         │                          │
│  │  agents|tasks|metrics    │                          │
│  │  bandwidth|credentials   │                          │
│  └─────────────────────────┘                          │
└───────────────────┬──────────────────────────────────┘
                    │ HTTP REST
         ┌──────────▼──────────┐
         │    Agent (n 个)      │
         │  register/heartbeat  │
         │  pull tasks          │
         │  execute (yt-dlp/   │
         │   static download)  │
         │  report metrics      │
         └─────────────────────┘
```

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/agents/register` | Agent 注册 |
| POST | `/api/v1/agents/heartbeat` | Agent 心跳 |
| GET  | `/api/v1/agents/{id}/tasks/pull` | 拉取任务 |
| POST | `/api/v1/agents/provision` | SSH 自动部署 Agent |
| GET  | `/api/v1/agents/provision-jobs/{id}` | 查看部署进度 |
| POST | `/api/v1/tasks` | 创建任务 |
| POST | `/api/v1/tasks/{id}/dispatch` | 下发任务 |
| POST | `/api/v1/tasks/{id}/stop` | 停止任务 |
| POST | `/api/v1/tasks/{id}/metrics` | 上报指标 |
| GET  | `/api/v1/dashboard/overview` | Dashboard 概览 |
| GET  | `/api/v1/dashboard/bandwidth/history` | 带宽历史 |
| GET  | `/healthz` | 健康检查 |
| GET  | `/metrics` | Prometheus 指标 |

## 任务示例

```bash
# 创建 10 Mbps 静态下载任务，运行 60 秒
curl -X POST http://localhost:8080/api/v1/tasks \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "test-download",
    "type": "static",
    "target_url": "https://example.com/large-file.zip",
    "agent_id": "<agent-id>",
    "target_rate_mbps": 10,
    "duration_sec": 60,
    "distribution": "ramp",
    "ramp_up_sec": 10,
    "ramp_down_sec": 10
  }'

# 下发任务
curl -X POST http://localhost:8080/api/v1/tasks/<task-id>/dispatch

# 创建 YouTube 下载任务
curl -X POST http://localhost:8080/api/v1/tasks \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "yt-test",
    "type": "youtube",
    "target_url": "https://www.youtube.com/watch?v=XXXX",
    "agent_id": "<agent-id>",
    "target_rate_mbps": 5,
    "concurrent_fragments": 4
  }'
```

## 环境变量

### Master
| 变量 | 默认值 | 说明 |
|------|--------|------|
| `MASTER_ADDR` | `:8080` | HTTP 监听地址 |
| `SQLITE_DSN` | `file:master.db?...` | SQLite 连接 |
| `MASTER_URL` | `http://localhost:8080` | 对 Agent 暴露的 Master URL |
| `AGENT_BIN_PATH` | `` | Agent 二进制路径（SSH 部署用） |

### Agent
| 变量 | 默认值 | 说明 |
|------|--------|------|
| `MASTER_URL` | `http://localhost:8080` | Master 地址 |
| `AGENT_HOST_IP` | 自动检测 | Agent IP（上报给 Master） |

## 运行测试

```bash
go test ./... -timeout 60s
```

## 项目结构

```
.
├── cmd/master/          # Master 入口
├── cmd/agent/           # Agent 入口
├── internal/
│   ├── model/           # 领域模型
│   ├── store/           # SQLite 仓储
│   ├── master/          # Master 业务逻辑
│   │   ├── handler/     # HTTP handlers
│   │   ├── service/     # 服务层
│   │   ├── scheduler/   # 任务调度器
│   │   └── provision/   # SSH 自动部署
│   └── agent/           # Agent 逻辑
│       ├── client/      # Master HTTP 客户端
│       ├── executor/    # yt-dlp + static 执行器
│       └── reporter/    # 指标上报
├── pkg/ratelimit/       # Token Bucket 限速器
├── web/                 # React + Vite 前端
└── deployments/         # Docker Compose + Dockerfiles
```
