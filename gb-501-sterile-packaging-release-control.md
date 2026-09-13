请生成一个名为 `sterile-packaging-release-control` 的 Go 全栈 Web 项目「无菌包装生产放行协同」。它面向医疗器械包装工厂，用于管理包装线、生产批次、检验结果和放行决定；这是制造质量控制项目，不做电商、订单、仓库或财务功能。

## 项目主要需求

复杂度下限：核心实体不少于 3 个、核心页面不少于 4 个、横切关注点不少于 2 个、共享前端组件不少于 3 个、自定义 hooks/utils 不少于 2 个、后端中间件不少于 2 个。

### 核心实体

- `PackagingLine`：产线、班组、设备状态；贯穿数据库、Go model/service/handler、前端 API/store/page。
- `ProductionBatch`：批次号、规格、当前状态、责任班组；贯穿全链路。
- `InspectionSample`：抽样位置、检验项、结果、复测状态；贯穿全链路。
- `ReleaseDecision`：放行/隔离/返工决定、审批人、理由；贯穿全链路。

### 核心页面

`/lines` 产线总览（产线+批次）；`/batches` 批次队列（批次+检验）；`/inspections` 检验工作台（检验+批次）；`/release` 放行审批（批次+决定）；`/audit` 审计记录。`BatchStatusBadge` 同时被批次队列和放行审批使用，`DecisionPanel` 同时被放行审批和批次详情使用。

### 横切关注点

JWT + RBAC：数据库角色、Go auth/rbac 中间件、路由守卫、前端按钮权限和类型定义必须联动；操作审计：写入批次、检验、放行决定时记录操作者、前后状态和请求 ID；全局错误处理和限流也要有独立中间件。

### 共享枚举/组件

前后端各自定义并保持一致：`BatchStatus`（draft/running/hold/rework/released）、`DecisionType`（release/quarantine/rework）。共享组件至少有 `StatusBadge`、`EntityTable`、`EmptyState`；hooks 至少有 `useAuth`、`usePagination`。

### 技术与规模要求

前端 React 18 + TypeScript + Vite + Ant Design；后端 Go 1.22 + Gin + GORM；数据库 PostgreSQL；JWT、Redis 限流。目标 Go 功能代码 3000–4200 行、30–42 个 `.go` 文件（不含 `_test.go`、vendor），禁止把职责合并到单文件。

### 文件结构强制清单

`frontend/src/api`、`stores`、`types`、`components/common`、`hooks`、`pages`、`router`、`utils`；后端 `backend/internal/{model,dto,repository,service,handler,router,middleware,constants,util}`，每个实体至少各有独立文件。README 必须列出共享枚举在前后端的全部出现位置。

### 结构红线

严禁合并职责到单一文件；后续任何功能都要沿数据库、Go 后端、前端 API/store/page 多层修改。

### 部署与交付

根目录必须提供 `docker-compose.yml`（顶层 `name: sterile-packaging-release-control`，且不写 `version:`）、`.env` 和 `.env.example`（均含 `COMPOSE_PROJECT_NAME=sterile-packaging-release-control`）、`README.md`、`frontend/Dockerfile`、`backend/Dockerfile` 和 `frontend/nginx.conf`。前端端口 `18501`、后端端口 `19501`；前端只调用 `/api`，Nginx 在 `frontend/nginx.conf` 中转发到 `backend:8080`；数据库使用命名卷和 healthcheck，后端等待 `condition: service_healthy`。提供真实 `/healthz`、Docker 启动命令 `docker compose up -d`、初始化 Git 和适配技术栈的 `.gitignore`。
