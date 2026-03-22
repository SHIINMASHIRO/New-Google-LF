# ngoogle — Google 流量任务系统

Master + Agent 分布式下载任务平台，支持 YouTube 和静态资源下载，具备精确速率控制、时间窗口调度和自然流量画像。

## 快速启动

### 使用 Docker Compose（PostgreSQL）

```bash
cd deployments
docker compose up --build

# 访问 Web UI
open http://localhost:8080
```

### 本地开发（SQLite）

```bash
# 构建前端
cd web && npm install && npm run build && cd ..

# 启动 Master（默认 SQLite）
go run ./cmd/master

# 启动 Agent（另一个终端）
MASTER_URL=http://localhost:8080 go run ./cmd/agent
```

### 使用 PostgreSQL

```bash
# 启动 Master 连接 PostgreSQL
DB_DRIVER=postgres \
PG_DSN="postgres://ngoogle:ngoogle@localhost:5432/ngoogle?sslmode=disable" \
go run ./cmd/master
```

### 前端开发模式

```bash
cd web
npm run dev   # Vite dev server (:5173)，自动代理到 Master
```

## 架构

```
┌────────────────────────────────────────────────────────────┐
│                      Master (:8080)                         │
│  ┌─────────┐  ┌──────────┐  ┌───────────┐  ┌───────────┐  │
│  │  REST   │  │Scheduler │  │ Dashboard │  │ Provision │  │
│  │  API    │  │  Diurnal  │  │  Realtime │  │  SSH Auto │  │
│  └────┬────┘  └─────┬────┘  └─────┬─────┘  └─────┬─────┘  │
│       │             │             │               │         │
│  ┌────▼─────────────▼─────────────▼───────────────▼──────┐  │
│  │           PostgreSQL / SQLite Store                    │  │
│  │  agents │ tasks │ task_groups │ bandwidth │ credentials│  │
│  └───────────────────────────────────────────────────────┘  │
└───────────────────┬────────────────────────────────────────┘
                    │ HTTP REST
         ┌──────────▼──────────┐
         │    Agent (n 个)      │
         │  register/heartbeat  │
         │  pull tasks          │
         │  execute (yt-dlp /   │
         │   static download)   │
         │  report metrics      │
         │  token bucket 限速   │
         └─────────────────────┘
```

## 核心功能

### 数据库

- **SQLite** — 本地开发默认，零依赖
- **PostgreSQL** — 生产环境推荐，20 并发连接，无单写入者瓶颈
- 通过 `DB_DRIVER` 环境变量切换，Repository Pattern 保证接口一致

### Dashboard

- **Live 模式** — 实时滚动监控，3 秒刷新，从 overview API 推入 60 点滑动窗口
- **3 天 / 7 天** — 历史带宽曲线，两级聚合（先 AVG per agent，再 SUM across agents）
- Agent 带宽排行榜，实时总带宽统计
- Overview 由后台 goroutine 每 2 秒刷新到内存缓存，API 读取零延迟

### 流量画像（Diurnal S-Curve）

任务设置 `distribution: "diurnal"` 后，速率乘数按 wall clock 自动调整：

```
带宽
100% ━━━━━━━╸                                          ╺━━━━
            ╲                                         ╱
             ╲                                      ╱
 75%          ╲                                   ╱
               ╲                                ╱
                ╲                            ╱
 50%             ╲__________________________╱
     ├──────┬──────┬──────┬──────┬──────┬──────┬──────┤
     0      3      6      9     12     15     18     23
                         时间（小时）
```

| 时段 | 带宽 | 说明 |
|------|------|------|
| 23:00 - 00:00 | 100% | 峰值持续 |
| 00:00 - 06:00 | 100% → 50% | S 型余弦平滑下降 |
| 06:00 - 23:00 | 50% → 100% | S 型余弦平滑上升 |

### 速率控制

- **Token Bucket** — 每个任务独立限速，支持突发
- **Sliding Window Meter** — 5s / 30s 窗口监控实际速率
- Executor 每秒调用 `scheduler.RateForTask()` 动态调整
- YouTube 任务通过 `yt-dlp --limit-rate` 控速

### 任务管理

- **Task Group** — 多 URL Pool 组合，自动拆分速率和流量目标
- **Execution Scope** — `single_agent`（指定节点）或 `global`（自动分配到所有在线节点）
- 支持 `start_at / end_at / duration_sec` 时间窗口
- Ramp up / Ramp down 线性斜坡

### Agent 自动部署

- Web UI 输入 SSH 信息，Master 自动：上传二进制 → 安装 systemd 服务 → 启动 → 健康检查
- 部署日志实时追踪，失败步骤可追溯，支持安全重试

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/agents/register` | Agent 注册 |
| POST | `/api/v1/agents/heartbeat` | Agent 心跳 |
| GET  | `/api/v1/agents/{id}/tasks/pull` | 拉取任务 |
| POST | `/api/v1/agents/provision` | SSH 自动部署 Agent |
| GET  | `/api/v1/agents/provision-jobs/{id}` | 查看部署进度 |
| POST | `/api/v1/task-groups` | 创建任务组 |
| POST | `/api/v1/task-groups/{id}/dispatch` | 下发任务组 |
| POST | `/api/v1/task-groups/{id}/stop` | 停止任务组 |
| GET  | `/api/v1/task-groups/{id}/metrics` | 任务组指标 |
| POST | `/api/v1/tasks/{id}/metrics` | 上报指标 |
| GET  | `/api/v1/dashboard/overview` | Dashboard 概览（内存缓存） |
| GET  | `/api/v1/dashboard/bandwidth/history` | 带宽历史（支持 1m/5m/15m/30m/1h step） |
| GET  | `/api/v1/url-pools` | URL 池列表 |
| GET  | `/healthz` | 健康检查 |
| GET  | `/metrics` | Prometheus 指标 |

## 环境变量

### Master

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `MASTER_ADDR` | `:8080` | HTTP 监听地址 |
| `DB_DRIVER` | `sqlite` | 数据库驱动（`sqlite` / `postgres`） |
| `SQLITE_DSN` | `file:master.db?...` | SQLite 连接串 |
| `PG_DSN` | `postgres://ngoogle:ngoogle@localhost:5432/ngoogle?sslmode=disable` | PostgreSQL 连接串 |
| `MASTER_URL` | `http://localhost:8080` | 对 Agent 暴露的 Master URL |
| `AGENT_DOWNLOAD_URL` | `` | Agent 二进制下载地址（SSH 部署用） |

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
├── cmd/master/                # Master 入口
├── cmd/agent/                 # Agent 入口
├── internal/
│   ├── model/                 # 领域模型
│   ├── store/
│   │   ├── iface.go           # Store 接口定义
│   │   ├── sqlite/            # SQLite 实现
│   │   └── postgres/          # PostgreSQL 实现
│   ├── master/
│   │   ├── handler/           # HTTP handlers
│   │   ├── service/           # 服务层（Dashboard 内存缓存）
│   │   ├── scheduler/         # 任务调度 + 流量画像（Diurnal S-Curve）
│   │   └── provision/         # SSH 自动部署
│   └── agent/
│       ├── client/            # Master HTTP 客户端
│       ├── executor/          # yt-dlp + static + mixed 执行器
│       └── reporter/          # 指标上报
├── pkg/ratelimit/             # Token Bucket + Sliding Window Meter
├── web/                       # React 18 + Vite + TailwindCSS + Recharts
└── deployments/               # Docker Compose + Dockerfiles
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.22+ |
| 数据库 | PostgreSQL 16（生产）/ SQLite（开发） |
| 前端 | React 18 + Vite + TailwindCSS + Recharts |
| 限速 | Token Bucket + Sliding Window |
| YouTube | yt-dlp 子进程 |
| SSH 部署 | golang.org/x/crypto/ssh |
| 容器化 | Docker Compose |
