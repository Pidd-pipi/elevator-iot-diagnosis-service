# 电梯物联网状态诊断与困人告警服务（elevator-iot-diagnosis-service）

## 一、项目概述

基于 Go 实现的电梯物联网 Web 项目，一款后端服务，完成电梯运行状态采集、故障码诊断、困人事件告警、处置闭环与运行健康评分。

项目类型：**全栈 Web 应用**（Go 后端服务 + `go:embed` 内嵌前端页面）。

## 二、业务背景与领域规则

电梯物联网终端实时采集电梯运行状态：轿厢位置、运行方向、开关门状态、平层信号、故障码。系统根据故障码与状态组合做诊断，区分普通故障与困人事件（轿厢停在非平层且门关闭、有人按了警铃）。困人事件要立即告警并生成处置任务，维保人员到场处置后回填结果。系统按故障频率与处置时效给每台电梯算健康评分。

关键领域规则（这些规则是后续埋 bug 验证跨文件改动的核心约束，必须真实实现）：

1. 状态采集：终端按 5 秒周期上报（elevator_id + 位置 + 方向 + 门状态 + 平层信号 + 故障码），服务端做状态机合法性校验（如运行中禁止开门）。
2. 困人判定：满足「非平层 + 门关闭 + 存在乘客信号（警铃/红外）+ 持续超过 30 秒」即生成困人事件；同一电梯未关闭的困人事件不重复生成。
3. 困人处置状态机：已告警(alerted) → 已接单(accepted) → 处置中(processing) → 已解除(released) / 已升级(escalated)；接单超 10 分钟未处理自动升级并二次告警。
4. 故障码诊断：故障码表映射到诊断结论（如 E01 门锁回路故障 → 建议检查门锁触点），未知故障码必须记录并提示人工确认，不得静默丢弃。
5. 健康评分：score = 100 - 近 30 天故障次数×系数 - 未按时处置次数×系数；评分 ≤60 的电梯进入「重点关注」名单。
6. 维保记录：处置完成后必须填写处置人、处理措施、恢复时间；未填完整不允许关闭处置任务。

## 三、核心实体（≥3 个，必须贯穿全栈）

每个实体必须贯穿「数据库/存储表 → domain model → repository → service → handler → 前端 API 层 → 前端页面/组件」全链路。

| 实体 | 关键字段 | 业务动作 |
|---|---|---|
| 电梯 Elevator | id、楼栋、型号、健康评分、重点关注标记 | 台账、评分 |
| 状态上报 StateReport | id、电梯id、位置、方向、门状态、平层信号、故障码、时间戳 | 采集、合法性校验 |
| 困人事件 EntrapmentEvent | id、电梯id、开始时间、结束时间、状态、升级标记 | 判定、处置 |
| 处置任务 DisposalRecord | id、事件id、接单时间、处理人、措施、恢复时间、状态 | 接单、处置、关闭 |
| 故障码记录 FaultCodeLog | id、电梯id、故障码、诊断结论、是否已知、时间 | 诊断、未知登记 |

## 四、核心页面与 API

### 前端页面（≥4 个路由，至少 2 个页面共用同一个业务组件）

| 项目 | 说明 |
|---|---|
| / 电梯总览 | 电梯状态卡片 + 健康评分 + 进行中困人事件 | Elevator、EntrapmentEvent |
| /elevators/{id} 电梯详情 | 实时状态 + 故障码时间线 + 评分明细 | Elevator、StateReport |
| /events 困人事件 | 事件列表 + 处置流转 | EntrapmentEvent |
| /diagnosis 故障诊断 | 故障码诊断映射表 + 未知故障码 | FaultCodeLog |
| /watchlist 重点关注 | 低评分电梯列表 | Elevator |

### 后端 REST API（与页面一一对应，命中真实业务链路）

| 项目 | 说明 |
|---|---|
| POST /api/elevators/{id}/states | 状态上报（合法性校验 + 困人判定） |
| GET /api/events | 困人事件列表 |
| POST /api/events/{id}/accept | 接单处置 |
| POST /api/events/{id}/resolve | 处置完成（校验必填字段） |
| POST /api/events/{id}/escalate | 升级处置 |
| GET /api/elevators/{id}/faults | 故障码记录 |
| GET /api/elevators/{id}/score | 健康评分明细 |
| GET /api/watchlist | 重点关注名单 |
| GET /api/overview | 总览聚合 |
| GET /api/healthz | 健康检查 |

## 五、横切关注点（≥2 个）

1. 操作审计日志：接单、处置、升级全部留痕；触达 handler → service → audit store。
2. 困人超时扫描定时任务：每 30 秒扫描接单超时事件并升级；触达 service → store → 事件页。
3. 全局错误处理与统一响应格式。

## 六、共享枚举/常量（≥2 组）

枚举/常量要求前后端各自定义且保持一致，README 中列出所有出现位置。

1. 困人事件状态 EventStatus：alerted / accepted / processing / released / escalated。
2. 门状态 DoorStatus：open / closed；运行方向 Direction：up / down / idle。
3. 故障类型 FaultType：known / unknown。

## 七、共享前端组件与 hooks（组件 ≥3 个、hooks ≥2 个）

### 共享组件（放 `web/components/`）

1. ElevatorCard：电梯状态卡片，被总览与重点关注共用。
2. EventTable：困人事件表格，被总览与事件页共用。
3. FaultTimeline：故障码时间线，被电梯详情与诊断页共用。

### 自定义 hooks（放 `web/hooks/`）

1. useElevators(poll)：电梯列表轮询，被总览与重点关注共用。
2. useEvents(filter)：事件列表，被事件页与总览共用。

## 八、后端中间件（≥2 个）

1. auditLogger：审计日志中间件。
2. errorHandler：统一错误/panic 处理中间件。
3. requestID：trace id 注入中间件。

## 九、技术要求

- 语言：**Go 1.23**（go.mod 声明 `go 1.23`，module 路径 `example.com/elevator-iot-diagnosis-service`）
- 运行：`go run .` 默认监听 `8080`，支持 `PORT` 环境变量覆盖
- 存储：SQLite（`modernc.org/sqlite` 纯 Go 驱动，CGO 关闭）或内置内存仓储 + JSON 文件持久化，二选一，必须可重复构建、无外部服务依赖
- 前端：纯原生 HTML/CSS/JS，`go:embed` 内嵌 `web/` 静态资源，禁止引入外部 CDN 依赖（离线可跑）
- 服务入口：`GET /healthz` 返回 200；页面 `GET /` 可访问
- 根目录必须包含 `runtime_smoke.json`：`mode: service` + `start: go run .` + `ready_url: /healthz`；`project_intro` 一句话简介必须包含项目类型（如「基于 Go 实现的XXX Web 项目，一款后端服务，完成……」）
- 根目录必须包含 `README.md`：项目说明、目录结构、运行与测试命令、环境变量说明
- 构建：`go build ./...` 与 `go test ./...` 必须全部通过（基线干净、无 bug）

## 十、文件结构强制清单（规模目标：≥2000 行 Go 功能代码、≥20 个 `.go` 文件）

```
backend/
├── go.mod
├── main.go
├── config/
│   └── config.go            # 上报周期、困人判定阈值、超时时限
├── domain/
│   ├── elevator.go          # 电梯实体 + 健康评分
│   ├── report.go            # 状态上报 + 合法性校验
│   ├── event.go             # 困人事件 + 判定
│   ├── disposal.go          # 处置任务状态机
│   └── fault.go             # 故障码诊断
├── store/
│   ├── elevator_store.go
│   ├── report_store.go
│   ├── event_store.go
│   ├── disposal_store.go
│   ├── fault_store.go
│   └── audit_store.go
├── service/
│   ├── ingest_service.go    # 采集 + 合法性 + 困人判定
│   ├── event_service.go     # 处置流转
│   ├── diagnose_service.go  # 故障码诊断
│   ├── scoring_service.go   # 健康评分
│   ├── overdue_sweeper.go   # 超时升级
│   └── audit_service.go
├── httpapi/
│   ├── router.go
│   ├── elevator_handler.go
│   ├── report_handler.go
│   ├── event_handler.go
│   ├── diagnose_handler.go
│   └── health_handler.go
├── middleware/
│   ├── audit.go
│   ├── error_handler.go
│   └── request_id.go
└── web/
    ├── index.html
    ├── app.js
    ├── style.css
    ├── components/
    └── hooks/
```

**严禁合并职责到单一文件**：handler、service、repository、domain 必须分层；禁止把所有逻辑塞进 `main.go` 或一个 `handlers.go`。目标规模下限 2000 行 / 20 个 `.go` 文件，实际建议做到 3000 行以上 / 30 个文件以上，保证每个业务模块（实体、状态机、联动、报表）都有独立文件。

## 十一、运行、测试与交付要求

1. `go build ./...` 通过；`go test ./...` 全绿（含各业务模块的单元测试，测试文件不计入规模）。
2. `go run .` 后 `GET /healthz` 返回 200，前端页面 `GET /` 可打开且核心接口可用。
3. 每个核心业务动作都要有可复现的输入（API 请求/页面操作），方便后续构造缺陷与验证命令。
4. 代码中不得出现任何「故意埋错」「TODO bug」类注释；交付为干净基线。

## 十二、质量红线

1. **天然多文件、多层耦合**：任何一个小改动（如给某状态新增一个合法迁移）都应触达 3-5 个文件（domain + repository + service + handler + 前端组件 + 枚举定义）。
2. 业务规则必须具体、可验证：状态机迁移表、联动逻辑、校验边界、生命周期管理必须真实存在，禁止空壳 CRUD。
3. 本项目用于评测跨文件协同改动能力，禁止做成本目录、对账/财务、库存盘点、电商订单、预约挂号、工单客服、数据可视化报表类业务。
4. 前端页面必须真实消费后端接口，禁止纯静态假页面。

---
*生成说明：本提示词面向 Go 标注数据流水线 2000 行档位，主题已对照禁选题材清单核验。*
