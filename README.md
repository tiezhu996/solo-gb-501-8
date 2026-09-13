# 无菌包装生产放行协同

面向医疗器械包装工厂的生产质量控制系统。系统覆盖包装产线、生产批次、检验样本、复测和放行决定，所有关键写操作都会记录操作者、请求 ID 以及变更前后状态。项目不包含电商、订单、仓储或财务能力。

## 主要流程

1. 产线操作员在「产线总览」确认设备可用，在「批次队列」创建批次并开工。
2. 检验员在「检验工作台」登记抽样位置和检验项目，录入合格/不合格结果；不合格结果自动进入待复测状态。
3. 放行审批员在「放行审批」查看生产和检验依据，选择放行、隔离或返工。
4. 放行要求至少一项检验、无待检验、无待复测且无不合格结果；隔离和返工会同步更新批次状态。
5. 管理员在「审计记录」按操作者、实体或请求 ID 回溯操作。

首次启动会幂等写入 3 条产线、4 个批次和 5 条检验样本，便于直接验证完整流程；已有业务数据时不会覆盖。

## 技术结构

- 前端：React 18、TypeScript、Vite、Ant Design、React Router、Axios
- 后端：Go 1.22、Gin、GORM、JWT、RBAC
- 基础设施：PostgreSQL 16、Redis 7、Nginx、Docker Compose
- 默认端口：前端 `18501`，后端 `19501`

```text
.
├── frontend/
│   ├── src/api                 # 请求封装与实体 API
│   ├── src/stores              # 认证、产线、批次、检验、放行状态
│   ├── src/types               # 前端领域类型与共享枚举
│   ├── src/components/common   # 表格、状态、空状态、审批面板、应用外壳
│   ├── src/hooks               # useAuth、usePagination
│   ├── src/pages               # 产线、批次、检验、放行、审计页面
│   ├── src/router              # 登录守卫和业务路由
│   └── src/utils               # 格式化和权限工具
└── backend/
    ├── cmd/server              # 进程入口与优雅关闭
    └── internal/
        ├── model,dto           # 数据模型和输入输出契约
        ├── repository          # GORM 数据访问及质量总览聚合
        ├── service             # 状态机、检验和放行规则
        ├── handler,router      # HTTP 处理器与路由装配
        ├── middleware          # JWT、RBAC、全局错误、Redis 限流
        ├── constants           # 共享枚举
        └── util                # 数据库、安全、HTTP 响应
```

## 快速启动

需要 Docker 24+ 和 Docker Compose v2。根目录名称包含中文也可以直接启动。

```bash
cp .env.example .env
# 修改 JWT_SECRET 和 POSTGRES_PASSWORD
docker compose up -d --build
docker compose ps
curl http://localhost:19501/healthz
```

浏览器访问 <http://localhost:18501>。本地演示账号：

| 账号 | 密码 | 角色 | 主要权限 |
| --- | --- | --- | --- |
| `admin` | `admin123` | 质量管理员 | 全部权限 |
| `operator` | `operate123` | 产线操作员 | 产线、批次写入 |
| `inspector` | `inspect123` | 检验员 | 检验写入、审计读取 |
| `approver` | `approve123` | 放行审批员 | 放行决定、审计读取 |

演示密码只用于本地验证，部署到共享环境前必须替换或接入组织身份系统。

停止服务但保留数据：

```bash
docker compose down
```

同时删除本项目命名卷中的演示数据：

```bash
docker compose down -v
```

## 本地开发与校验

后端依赖 PostgreSQL；可以先用 Compose 启动 `postgres` 和 `redis`，再运行后端：

```bash
docker compose up -d postgres redis
cd backend
go mod download
DATABASE_URL='postgres://sterile:sterile_dev_password@localhost:5432/sterile_release?sslmode=disable' \
REDIS_ADDR='localhost:6379' JWT_SECRET='local-development-secret-change-me' go run ./cmd/server
```

前端开发服务器会将 `/api` 和 `/healthz` 代理到 `localhost:19501`：

```bash
cd frontend
npm ci
npm run dev
```

静态检查与构建：

```bash
cd backend && gofmt -w . && go test ./... && go build ./cmd/server
cd ../frontend && npm ci && npm run build
docker compose config --quiet
```

## API 概览

除登录和健康检查外，接口均要求 `Authorization: Bearer <token>`。

| 方法与路径 | 功能 | 写权限 |
| --- | --- | --- |
| `GET /healthz` | PostgreSQL、Redis 和服务健康状态 | 公开 |
| `POST /api/auth/login` | 登录并签发 JWT | 公开 |
| `GET /api/overview` | 产线、批次、检验与风险聚合 | 已登录 |
| `GET/POST/PATCH /api/lines` | 产线查询、创建、更新 | `line:write` |
| `GET/POST/PATCH /api/batches` | 批次查询、创建、更新 | `batch:write` |
| `POST /api/batches/:id/transition` | 开工、暂停、恢复或进入返工 | `batch:write` |
| `GET/POST /api/inspections` | 检验查询和登记 | `inspection:write` |
| `POST /api/inspections/:id/complete` | 录入检验或复测结果 | `inspection:write` |
| `GET/POST /api/release-decisions` | 查看和提交放行决定 | `release:write` |
| `GET /api/audit-logs` | 查询不可变更的审计事件 | `audit:read` |

登录和创建产线示例：

```bash
curl -s http://localhost:19501/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}'

curl -s http://localhost:19501/api/lines \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"code":"PKG-D04","name":"验证包装线","team":"验证班","equipmentStatus":"ready","location":"验证区"}'
```

## 枚举同步位置

`BatchStatus` 的值固定为 `draft`、`running`、`hold`、`rework`、`released`：

- 后端定义：`backend/internal/constants/batch_status.go`
- 后端持久化：`backend/internal/model/production_batch.go`
- 后端状态机：`backend/internal/service/batch_service.go`
- 后端放行联动：`backend/internal/service/release_service.go`
- 前端类型：`frontend/src/types/domain.ts`
- 前端 API：`frontend/src/api/index.ts`
- 前端公共显示：`frontend/src/components/common/BatchStatusBadge.tsx`
- 前端使用页面：`BatchesPage.tsx`、`BatchDetailPage.tsx`、`InspectionsPage.tsx`、`ReleasePage.tsx`

`DecisionType` 的值固定为 `release`、`quarantine`、`rework`：

- 后端定义和角色定义：`backend/internal/constants/decision_type.go`
- 后端持久化：`backend/internal/model/release_decision.go`
- 后端规则：`backend/internal/service/release_service.go`
- 前端类型：`frontend/src/types/domain.ts`
- 前端 API：`frontend/src/api/index.ts`
- 前端公共面板：`frontend/src/components/common/DecisionPanel.tsx`
- 前端使用页面：`ReleasePage.tsx`、`BatchDetailPage.tsx`

修改枚举时必须同时更新以上位置、数据库兼容策略和 README，不应只修改某一层。

## 安全与运维

- JWT 使用 HS256 签名并包含角色；生产环境必须使用独立高熵密钥。
- RBAC 同时约束后端路由和前端按钮。前端限制只改善体验，后端中间件才是权限边界。
- Redis 限流不可用时，单实例会退化为内存限流，`/healthz` 返回 `redis: fallback`。
- 每个响应包含 `X-Request-ID`，写操作的审计记录保存同一请求 ID、操作者及前后状态。
- PostgreSQL 和 Redis 使用以 `sterile-packaging-release-control` 为 Compose 项目名的命名卷，避免与其他项目冲突。

