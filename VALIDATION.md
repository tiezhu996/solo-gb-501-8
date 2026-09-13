# gb-501 验收记录

验收日期：2026-08-22（Asia/Shanghai）

## 结论

通过。项目符合原始提示词的领域、分层、规模、部署和主要业务约束，可通过 Docker Compose 正常启动。管理员主流程、操作员 RBAC、五个核心页面、跨页共享组件、审计快照及关键业务红线均已真实验证。

验收中发现并修复一处前端权限缺口：无 `audit:read` 权限的 `operator` 原本仍可进入 `/audit` 并触发 403。现已增加权限路由守卫，并同步隐藏无权访问的审计菜单。修复后使用新的内置 Browser 标签页复验，直达 `/audit` 被重定向至 `/lines`，控制台 error/warn 为 0。

## 原始要求逐项核验

| 原始要求 | 实际核验 | 结果 |
| --- | --- | --- |
| 制造质量控制领域，排除电商/订单/仓库/财务 | README、页面、API 和实体均围绕无菌包装产线、批次、检验与放行，未发现越界模块 | 通过 |
| 四个核心实体全链路贯穿 | `PackagingLine`、`ProductionBatch`、`InspectionSample`、`ReleaseDecision` 均有独立 model、DTO、repository、service、handler；前端有对应 API、store/page 或业务页入口 | 通过 |
| 五个核心页面 | `/lines`、`/batches`、`/inspections`、`/release`、`/audit` 均已在内置 Browser 中加载并检查真实数据 | 通过 |
| `BatchStatusBadge` 跨页共用 | 批次队列、批次详情和放行审批均引用；Browser 中可见草稿、生产中、暂停/隔离、已放行状态 | 通过 |
| `DecisionPanel` 跨页共用 | `ReleasePage.tsx` 与 `BatchDetailPage.tsx` 均引用；Browser 中分别检查可提交审批弹窗和已放行禁用态 | 通过 |
| JWT + RBAC 全链路 | 数据库用户角色、Go auth/RBAC 中间件、路由权限、前端 `User/Role/permissions` 类型及按钮权限均存在；`operator` 检验/放行按钮禁用，审计菜单隐藏，直达审计路由被拦截；后端审计接口返回 403 | 通过 |
| 写操作审计 | Browser 审计页检查到创建产线、创建/变更批次、创建/完成检验、提交放行决定；详情包含操作者、请求 ID、来源 IP、变更前后 JSON | 通过 |
| 全局错误、请求 ID、Redis 限流独立实现 | `middleware/error.go`、`middleware/rate_limit.go`、请求上下文中间件独立；响应含 `X-Request-ID`、限流响应头 | 通过 |
| 共享枚举一致 | Go 与 TypeScript 中 `BatchStatus` 为 `draft/running/hold/rework/released`，`DecisionType` 为 `release/quarantine/rework`；README 列出全部同步位置 | 通过 |
| 至少三个公共组件、两个 hooks | `StatusBadge`、`EntityTable`、`EmptyState` 及额外跨页组件齐全；`useAuth`、`usePagination` 齐全且被页面实际引用 | 通过 |
| React 18 + TypeScript + Vite + Ant Design | `package.json`、源码和生产构建结果一致 | 通过 |
| Go 1.22 + Gin + GORM + PostgreSQL + Redis | `go.mod`、后端实现、Compose 服务与健康检查一致 | 通过 |
| 3000-4200 行、30-42 个功能 Go 文件 | 排除 `_test.go` 后为 3009 行、42 个 `.go` 文件 | 通过 |
| 强制目录与实体独立文件 | 前端 `api/stores/types/components/common/hooks/pages/router/utils`、后端九类 `internal` 目录均存在；未发现单文件大杂烩 | 通过 |
| Compose、环境文件与反向代理 | 顶层 `name` 正确且无 `version`；`.env`/`.env.example` 均有项目名；端口为 18501/19501；前端只请求 `/api`，Nginx 转发 `backend:8080` | 通过 |
| PostgreSQL 命名卷、健康检查和依赖等待 | PostgreSQL/Redis 命名卷和 healthcheck 存在，后端等待二者 healthy，前端等待后端 healthy | 通过 |
| 真实 `/healthz`、README、Git | 后端与 Nginx 同源 `/healthz` 均返回 200 且报告 PostgreSQL/Redis 正常；README 完整；仓库为 `main`，保留提交 `6ad6801` | 通过 |

## 命令验证

以下命令均在最终修复后执行并返回退出码 0：

```bash
cd backend
go test ./...
go vet ./...
go build ./...

cd ../frontend
npm run build

cd ..
docker compose config --quiet
docker compose up -d --build
docker compose ps
```

实际运行时 `postgres`、`redis`、`backend`、`frontend` 四个容器均为 `healthy`。前端生产构建成功；Vite 仅提示主包超过 500 kB，这是性能建议，不阻断功能或部署。

规模统计命令与结果：

```text
find backend -name '*.go' ! -name '*_test.go' | wc -l  -> 42
功能 Go 文件 wc -l 汇总                              -> 3009
```

## API 与业务红线

- `GET http://localhost:19501/healthz`：`200 OK`，`status=ok`、`database=ok`、`redis=ok`。
- `GET http://localhost:18501/healthz`：经 Nginx 转发后同样为 `200 OK`。
- 使用 `operator` JWT 请求 `GET /api/audit-logs`：`403 Forbidden`，错误码 `FORBIDDEN`，且响应带请求 ID。
- 对存在待完成/待复测检验的 `B20260821-009` 尝试提交 `release`：`409 Conflict`，消息为“仍有待完成或待复测的检验”，未改变业务状态。
- 已放行验证批次 `QA-BATCH-002634` 在 Browser 中显示 1/1 检验合格、状态 `released`；对应审计链包含建线、建批次、开工、建样本、完成检验与放行决定。

## 内置 Browser 验证

只使用 Codex 内置 Browser，未使用外部 Chrome。

- `/lines`：产线统计、四条产线数据和查询控件正常。
- `/batches`：搜索 `QA-BATCH` 后只返回目标批次；进入 `/batches/5` 后批次信息、检验明细、`BatchStatusBadge` 和 `DecisionPanel` 正常。
- `/inspections`：列表、状态和复测信息正常；“录入结果”弹窗能打开并展示结果、测量值、说明字段；取消不产生写入。
- `/release`：待审批队列、待处理/不合格提示正常；审批弹窗包含放行/隔离/返工和理由；决定记录页显示已完成放行记录。
- `/audit`：审计列表及详情弹窗正常，详情展示请求 ID 和前后状态。
- `operator` 登录成功；“登记样本”“录入结果”“提交决定”均为禁用状态。
- 修复后在全新标签页以 `operator` 直达 `/audit`，实际 URL 变为 `/lines`，审计菜单数量为 0，控制台 error/warn 数量为 0。

## 清理状态

验收后已执行：

```bash
docker compose down -v --remove-orphans
```

`docker compose ps -a` 为空；按项目名过滤的容器和卷均为空。测试数据卷已按任务要求删除，不可从本次 Compose 环境恢复。

## 2026-08-22 企业级复核补充

- 所有产线、批次、检验与放行写操作现由统一事务上下文执行，业务变更和审计写入同提交、同回滚。
- 批次、产线及检验更新使用数据库行锁；放行判断、检验计数、状态更新、决定和审计均在同一事务内，避免并发重复放行或基于旧检验结果审批。
- JWT 鉴权以数据库实时用户状态和角色为准。实测把 `admin` 从数据库降为 `viewer` 后，旧管理员 token 的写请求立即返回 403，`/auth/me` 返回实时角色 `viewer`。
- 登录增加独立的 10 次/5 分钟 IP 限流；同一空卷实例连续错误登录后真实返回 429。全局限流仍独立保留。
- 外部 `X-Request-ID` 仅接受 1-64 位安全字符；非法值改用服务端 UUID，避免审计列溢出和日志注入。
- 前端启动时会向 `/auth/me` 复核缓存身份；损坏的 localStorage JSON 会被安全清除，不再导致白屏。
- 生产环境拒绝默认或短于 32 字符的 JWT 密钥；开发 Compose 通过显式 `APP_ENV=development` 保持开箱可运行。
- 新增鉴权实时降权、停用用户、非法 request ID 及事务边界回归测试。

本轮重新执行 `go test ./...`、`go test -race ./...`、`go vet ./...`、`go build ./...`、`npm run build`、`docker compose config --quiet`、`docker compose up -d --build` 和 `git diff --check`，均退出 0。真实 API 验证结果为：健康检查 200、当前用户 200、事务内创建批次 201、同 request ID 审计 1 条、旧高权限 token 降权后写入 403；完成后再次删除容器、网络和命名卷。
