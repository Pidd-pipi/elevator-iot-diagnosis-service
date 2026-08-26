# 电梯物联网状态诊断与困人告警服务（elevator-iot-diagnosis-service）

基于 Go 实现的电梯物联网 Web 项目，一款后端服务，完成电梯运行状态采集、故障码诊断、
困人事件告警、处置闭环与运行健康评分。前端为 `go:embed` 内嵌的原生 HTML/CSS/JS，
无任何外部 CDN 依赖，离线可运行；后端采用内存仓储 + JSON 文件原子持久化，
无需外部数据库即可重复构建、一键部署。

## 一、功能概览

- **状态采集**：终端按 5 秒周期上报（电梯 + 位置 + 方向 + 门状态 + 平层信号 + 故障码 + 乘客信号），
  服务端做状态机合法性校验（如「运行中禁止开门」）。
- **困人判定**：「非平层 + 门关闭 + 存在乘客信号」持续超过 30 秒生成困人事件；
  同一电梯未关闭事件不重复生成。
- **处置闭环**：已告警 → 已接单 → 处置中 → 已解除 / 已升级；接单超 10 分钟自动升级并二次告警。
- **故障诊断**：内置 E01-E12 故障码诊断映射表；未知故障码必须登记并提示人工确认。
- **健康评分**：`score = 100 - 近30天故障次数×2 - 未按时处置次数×5`，评分 ≤ 60 进入重点关注名单。
- **横切关注点**：操作审计日志、困人超时扫描定时任务、全局错误处理与统一响应格式。

## 二、架构与分层

```
httpapi（REST API + 静态资源路由 + 输入校验 + 分页）
   │
   ▼
service（业务用例：采集/处置/诊断/评分/超时扫描/审计）
   │
   ├──────────────► domain（实体、状态机、枚举、领域错误）
   │
   ▼
store（内存仓储 + JSON 原子持久化；读接口返回深拷贝，杜绝并发竞态）
```

依赖严格单向：`main → httpapi → service → store/domain`；`middleware → store`。
各层职责不合并：handler 只做协议与参数校验，service 承载业务规则，
repository 只负责存取，domain 只描述模型与状态迁移。

## 三、目录结构

```
elevator-iot-diagnosis-service/
├── go.mod                  # module example.com/elevator-iot-diagnosis-service，go 1.23
├── main.go                 # 入口：配置校验 + 装配 + 全超时 HTTP + 优雅关闭 + go:embed web
├── config/                 # 配置加载/Validate（环境变量 > 默认值）
├── domain/                 # 领域模型与业务规则
│   ├── enums.go            # EventStatus / DoorStatus / Direction / FaultType / PassengerSignal
│   ├── elevator.go         # 电梯台账 + 健康评分应用
│   ├── report.go           # 状态上报 + 合法性校验 + 困人条件
│   ├── event.go            # 困人事件状态机（accept/processing/release/escalate）
│   ├── disposal.go         # 处置任务 + 关闭校验 + 按时判定
│   ├── fault.go            # 故障码诊断映射表 + 未知故障登记
│   ├── audit.go            # 审计日志
│   ├── score.go            # 健康评分计算
│   ├── clone.go            # 实体深拷贝（仓储读接口防竞态）
│   └── errors.go           # 统一领域错误
├── store/                  # 内存仓储 + JSON 文件原子持久化
│   ├── store.go            # 仓储聚合 + 持久化写锁
│   ├── persistence.go      # 快照原子落盘/恢复（临时文件→fsync→rename；损坏备份 .bak）
│   ├── clone.go            # 观测记录深拷贝
│   ├── elevator_store.go / report_store.go / event_store.go /
│   ├── disposal_store.go / fault_store.go / audit_store.go / observation_store.go
│   └── ids.go              # ID 生成
├── service/                # 业务用例层
│   ├── ingest_service.go   # 采集 + 合法性 + 故障诊断 + 困人判定
│   ├── event_service.go    # 接单/处置/解除/升级
│   ├── diagnose_service.go # 故障码诊断
│   ├── scoring_service.go  # 健康评分
│   ├── overdue_sweeper.go  # 接单超时自动升级（30 秒扫描）
│   ├── audit_service.go    # 审计留痕
│   ├── seed.go             # 演示基线数据
│   └── services.go         # 服务装配
├── httpapi/                # REST API + 静态资源路由
│   ├── router.go           # 路由与中间件链
│   ├── response.go         # 统一响应格式与错误映射
│   ├── pagination.go       # limit/offset 分页解析（默认 20、上限 100）
│   ├── request.go          # JSON 请求体解析（体积限制/单一对象校验）
│   └── *_handler.go        # health/elevator/report/event/diagnose/score/watchlist/overview/audit
├── middleware/             # requestID / securityHeaders / auditLogger / errorHandler(Recoverer)
├── web/                    # go:embed 前端（原生 HTML/CSS/JS）
│   ├── index.html / app.js / style.css / constants.js / api.js
│   ├── components/         # ElevatorCard / EventTable / FaultTimeline
│   ├── hooks/              # useElevators / useEvents
│   └── pages/              # 总览 / 电梯详情 / 困人事件 / 故障诊断 / 重点关注
├── Dockerfile              # 多阶段构建（golang:1.23-alpine → alpine:3.20，非 root）
├── .dockerignore
├── Makefile
├── runtime_smoke.json      # 冒烟配置
└── README.md
```

## 四、运行与测试

```bash
# 构建
make build            # 等价于 CGO_ENABLED=0 go build -trimpath -o bin/elevator-iot-diagnosis-service .
go build ./...

# 测试（含竞态检测）
make test
make race
go test -race ./...

# 静态检查与格式化
make vet
make fmt

# 运行（默认监听 8080）
make run
go run .

# 指定端口
PORT=19007 go run .
```

启动后：

- 健康检查：`GET /healthz` → 200
- 就绪检查：`GET /readyz` → 200
- 前端页面：`GET /` 可访问（SPA 路由 /events、/diagnosis、/watchlist 等由后端回退 index.html）
- 演示数据：首次启动自动写入 6 台电梯、历史故障、已闭环/进行中困人事件

## 五、Docker 部署

```bash
# 构建镜像
make docker-build
# 或
docker build -t elevator-iot-diagnosis-service:latest .

# 运行（默认 8080，数据卷持久化）
docker run -d --name elevator \
  -p 8080:8080 \
  -v elevator-data:/app/data \
  elevator-iot-diagnosis-service:latest

# 覆盖端口运行
docker run -d --name elevator -p 19007:19007 -e PORT=19007 \
  -v elevator-data:/app/data \
  elevator-iot-diagnosis-service:latest
```

镜像特性：多阶段构建、`CGO_ENABLED=0` 静态编译、运行于 `alpine:3.20`、
非 root 用户、`EXPOSE 8080`、尊重 `PORT` 环境变量、内置 `HEALTHCHECK`。

## 六、环境变量

| 环境变量 | 默认值 | 说明 |
|---|---|---|
| `PORT` | `8080` | HTTP 监听端口（1-65535） |
| `DATA_FILE` | `data/elevator-state.json` | JSON 持久化文件路径（空字符串则仅内存） |
| `AUTO_PERSIST` | `true` | 是否随超时扫描周期自动落盘 |
| `LOG_LEVEL` | `info` | 日志级别：debug/info/warn/error |
| `READ_TIMEOUT_SEC` | `15` | HTTP 读超时（秒） |
| `WRITE_TIMEOUT_SEC` | `30` | HTTP 写超时（秒） |
| `IDLE_TIMEOUT_SEC` | `60` | HTTP keep-alive 空闲超时（秒） |
| `SHUTDOWN_TIMEOUT_SEC` | `8` | 优雅关闭等待在途请求的最长秒数 |
| `REPORT_PERIOD_SEC` | `5` | 终端状态上报周期（秒） |
| `ENTRAPMENT_THRESHOLD_SEC` | `30` | 困人判定持续阈值（秒） |
| `ACCEPT_DEADLINE_MIN` | `10` | 接单后处置时限（分钟） |
| `SWEEP_INTERVAL_SEC` | `30` | 超时扫描周期（秒） |
| `FAULT_SCORE_WEIGHT` | `2` | 每次故障扣分 |
| `UNTIMELY_SCORE_WEIGHT` | `5` | 每次未按时处置扣分 |
| `WATCHLIST_THRESHOLD` | `60` | 重点关注评分阈值（0-100） |

配置在 `config/config.go` 中加载并统一 `Validate()`，非法配置会阻止启动。

## 七、核心 API

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/elevators/{id}/states` | 状态上报（合法性校验 + 故障诊断 + 困人判定） |
| GET | `/api/elevators` | 电梯列表（`q`/`limit`/`offset`，返回 `total`） |
| GET | `/api/elevators/{id}` | 电梯详情（含最近上报与评分） |
| GET | `/api/elevators/{id}/score` | 健康评分明细 |
| GET | `/api/elevators/{id}/faults` | 故障码时间线（`limit`/`offset`，返回 `total`） |
| GET | `/api/events` | 困人事件列表（`status`/`elevator_id`/`limit`/`offset`，返回 `total`） |
| GET | `/api/events/{id}` | 事件详情（含处置任务与审计轨迹） |
| POST | `/api/events/{id}/accept` | 接单处置 |
| POST | `/api/events/{id}/resolve` | 处置完成（校验处置人/措施/恢复时间） |
| POST | `/api/events/{id}/escalate` | 升级处置（二次告警） |
| GET | `/api/watchlist` | 重点关注名单（`limit`/`offset`，返回 `total`） |
| GET | `/api/diagnosis` | 故障码诊断映射表 + 未知故障码（未知记录支持分页） |
| GET | `/api/overview` | 总览聚合 |
| GET | `/api/audit-logs` | 审计日志（`limit`/`offset`，返回 `total`） |
| GET | `/healthz` | 存活健康检查 |
| GET | `/readyz` | 就绪检查（与 healthz 等价，可独立配置） |
| GET | `/api/healthz` | API 健康检查 |

### 统一响应格式

```json
{ "code": 0, "message": "ok", "request_id": "...", "data": { ... } }
```

列表接口的 `data` 内统一包含：当前页数组、`total`、`limit`、`offset`。

### 分页

- `limit`：默认 20，上限 100；`offset`：默认 0。
- 非法（非整数或负数）返回 `422`。
- 例如：`GET /api/events?status=alerted&limit=10&offset=20`。

### 业务错误码

`40400` 资源不存在、`40900/40901` 冲突/重复、`40902` 非法状态迁移、
`42200/42201` 参数校验失败、`50000` 内部错误。

## 八、状态机与业务规则

**困人事件状态机**（`domain/enums.go` `EventStatus`）：

```
alerted → accepted → processing → released
   │         │            │
   └─────────┴────────────┴→ escalated（二次告警）
```

| 规则 | 说明 | 代码位置 |
|---|---|---|
| 运行中禁止开门 | 方向非 idle 且门 open → 拒绝上报 | `domain/report.go ValidateState` |
| 困人判定 | 非平层+关门+乘客信号，持续 >30s | `service/ingest_service.go trackEntrapment` |
| 不重复告警 | 同一电梯存在未关闭事件不重复生成 | `service/ingest_service.go` + `store/event_store.go OpenByElevator` |
| 接单超时升级 | 接单 >10 分钟未闭环自动升级 | `service/overdue_sweeper.go Sweep` |
| 处置必填 | 处置人/措施/恢复时间缺一不可 | `domain/disposal.go ValidateCompletion` |
| 未知故障登记 | 未知故障码必须记录并提示人工确认 | `service/diagnose_service.go` + `domain/fault.go` |
| 健康评分 | 100 - 故障×2 - 未按时×5，≤60 重点关注 | `domain/score.go` + `service/scoring_service.go` |

## 九、枚举/常量前后端位置对照

| 枚举/常量 | Go 侧定义 | JS 侧定义 |
|---|---|---|
| 困人事件状态 EventStatus（alerted/accepted/processing/released/escalated） | `domain/enums.go EventStatus` | `web/constants.js EVENT_STATUS` |
| 门状态 DoorStatus（open/closed） | `domain/enums.go DoorStatus` | `web/constants.js DOOR_STATUS` |
| 运行方向 Direction（up/down/idle） | `domain/enums.go Direction` | `web/constants.js DIRECTION` |
| 故障类型 FaultType（known/unknown） | `domain/enums.go FaultType` | `web/constants.js FAULT_TYPE` |
| 乘客信号 PassengerSignal（none/alarm/infrared/both） | `domain/enums.go PassengerSignal` | `web/constants.js PASSENGER_SIGNAL` |
| 严重度 Severity（low/medium/high） | `domain/fault.go FaultCodeRule.Severity` | `web/constants.js SEVERITY` |
| 评分阈值/权重 | `config/config.go` | 页面展示文案 `web/pages/*.js` |

## 十、共享组件与 hooks

- **组件**（`web/components/`）：
  - `ElevatorCard`：电梯状态卡片，被「电梯总览」与「重点关注」共用。
  - `EventTable`：困人事件表格，被「电梯总览」与「困人事件」共用。
  - `FaultTimeline`：故障码时间线，被「电梯详情」与「故障诊断」共用。
- **hooks**（`web/hooks/`）：
  - `useElevators(poll, query)`：电梯列表轮询，被「电梯总览」与「重点关注」共用。
  - `useEvents(filter)`：事件列表，被「困人事件」与「电梯总览」共用，支持 `setFilter` 动态过滤。

## 十一、健康检查

- `GET /healthz`：存活检查，返回 200 与 `uptime_sec`。
- `GET /readyz`：就绪检查，返回 200（当前无外部依赖，与 healthz 等价）。
- `GET /api/healthz`：API 兼容健康检查。
- Docker 镜像内置 `HEALTHCHECK`，每 30 秒探测 `/healthz`。

## 十二、持久化与数据安全

- 运行期数据默认写入 `data/elevator-state.json`。
- **原子写**：同目录临时文件 → 写数据 → fsync → `rename` 原子替换 → fsync 目录，
  进程崩溃不会留下半截文件。
- **损坏恢复**：启动时若 JSON 解析失败，将损坏文件备份为 `<path>.bak`，
  记录告警日志后降级为空库启动（随后由种子数据兜底，不会带病运行）。
- **并发写**：`Store.Save/Load` 通过持久化互斥锁串行化；各子仓储用 `RWMutex` 保护；
  读接口返回深拷贝，杜绝并发读写下修改共享对象的数据竞态。
- 删除 `data/` 目录即可重置为种子基线。

## 十三、故障排查

| 现象 | 排查方向 |
|---|---|
| 启动失败并提示「配置校验失败」 | 检查环境变量取值是否符合 README 第六节约束 |
| 启动日志出现 `load snapshot failed` | 检查 `data/elevator-state.json` 是否损坏，损坏文件已备份为 `.bak` |
| `GET /healthz` 非 200 | 查看 stdout 日志确认监听地址与端口 |
| 列表接口返回 `422` | 检查 `limit`/`offset` 是否为非负整数 |
| 前端页面空白 | 确认以 `GET /` 访问，SPA 路由由后端回退 index.html |
| 镜像健康检查失败 | 确认容器内 `PORT` 与 `-p` 映射一致，且 wget 可达 `127.0.0.1` |

## 十四、质量说明

- `go build ./...`、`go test ./...`、`go test -race ./...`、`go vet ./...` 全部通过。
- 测试覆盖：状态机迁移、合法性校验、困人判定（阈值/去重/中断重置）、
  处置闭环（必填校验/按时判定）、评分（扣分/重点关注）、超时扫描、持久化往返与损坏备份、
  API 集成链路、分页与输入校验、中间件安全头。
- 交付为干净基线，无任何「故意埋错 / TODO bug」类注释。
