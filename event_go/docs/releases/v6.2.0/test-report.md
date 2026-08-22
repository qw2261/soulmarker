# v6.2.0 测试报告（G6 生产上线准备 — 容器与部署基线、备份恢复、限流、构建 provenance 切片）

> 状态：G6-R03/R07（容器与部署基线）、G6-R09（自动备份与恢复验证）、G6-R08（运维 Runbook）、G6-R10（数据主体隐私、账号注销、数据导出/删除与留存）、G6-R04（全局 API 限流）、G6-R02（CI/CD 不可变制品与构建 provenance）、G6-R06（HTTP 指标，G6.5 切片）与 G6-R05（数据库迁移先于应用灰度与 N/N-1 兼容，G6.7 切片）Remote Candidate Pass / 文档完成。其中 G6-R02 的远端 CI frontend Browser E2E 出现一次性 flake（本地全部 6 例通过，详见下文 G6-R02 切片），其余 job 全部 success

## 版本身份

| 字段 | 值 |
|---|---|
| 候选分支 | origin/codex/update_project |
| 上一阶段基线 Commit | 63a72a9（G5.5 验收证据） |
| G6-R03/R07 实现 Commit | db62b83 |
| G6-R03/R07 门禁修复 Commit | ac75417（gofmt 对齐） |
| G6-R09 实现 Commit | e390d99 |
| G6-R10 实现 Commit | fec363a |
| G6-R04 限流实现 Commit | 9dd296c |
| G6-R02 构建 provenance 实现 Commit | 8e226fb |
| G6-R06 指标实现 Commit | 6eb9ffe |
| G6-R05 独立迁移入口实现 Commit | 1da3005 |
| Schema | v15（G6-R10 引入数据主体隐私，G6-R02 构建与 G6-R06 指标切片未变更数据库结构，G6-R05 独立迁移入口复用既有 v15 迁移） |

## 本切片范围

G6 生产上线准备的第一个切片「容器与部署基线」：

- **G6-R03**：Docker 非 root 运行、固定基础镜像版本、Healthcheck 与 smoke test。
- **G6-R07**：liveness 与 readiness 分离，依赖异常时返回可操作状态。

本切片不涉及数据库变更（Schema 保持 v14）、不涉及前端功能调整，也不宣称完成 G6 全部门禁。

## 需求追溯

| Requirement | Test ID | 自动化验收 |
|---|---|---|
| G6-R03 | DOCKER-NONROOT-001、DOCKER-HEALTHCHECK-001、SMOKE-CONTAINER-001 | Dockerfile 固定 `node:22.14.0-alpine3.21`、`golang:1.25.13-alpine3.24`、`alpine:3.19.1`，以非 root `app` 用户运行，`/app/data` 归属 `app:app`；镜像 `HEALTHCHECK` 指向 `/healthz`；CI docker job 断言容器运行用户 `uid != 0` 并执行 [smoke.sh](../../../scripts/smoke.sh) 验证 `/healthz` 与 `/readyz` |
| G6-R07 | Liveness/Readiness 探针 | `GET /healthz` 仅反映进程存活（不探测依赖）；`GET /readyz` 在数据库就绪返回 200，依赖异常返回 503 并保留 `Data`（`ErrorCode=SERVICE_UNAVAILABLE`）；单元测试覆盖 `TestLivenessHandler`、`TestReadinessHandlerHealthy`、`TestReadinessHandlerUnhealthy` |

## 变更清单（Commit db62b83）

- `.github/workflows/ci.yml`：新增 `docker` job（构建镜像、启动容器、非 root 断言、smoke test、失败日志、清理）。
- `Dockerfile`：固定基础镜像版本、非 root 运行、数据目录归属、HEALTHCHECK。
- `cmd/event-go/main.go`：启动时输出 `/healthz`、`/readyz` 探针路由说明。
- `internal/api/errors.go`：新增稳定错误码 `CodeServiceUnavailable = "SERVICE_UNAVAILABLE"`。
- `internal/handler/handler.go`：新增 `LivenessHandler` 与 `ReadinessHandler`。
- `internal/handler/dto/types.go`：新增 `LivenessResponse`、`ReadinessResponse`。
- `internal/handler/router.go`：注册 `/healthz`、`/readyz`。
- `internal/handler/handler_test.go`：新增 3 个探针测试。
- `internal/api/errors_test.go`：错误码目录数量更新为 47。
- `internal/openapi/v1.json`：`error_code` 枚举加入 `SERVICE_UNAVAILABLE`。
- `scripts/smoke.sh`：容器/部署 smoke 探针脚本。

## 本地门禁

| 门禁 | 结果 |
|---|---|
| `gofmt -l ./cmd ./internal` | 通过，无待格式文件 |
| `go build ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go test -count=1 ./...` | 通过 |
| `git diff --check` | 通过 |
| OpenAPI / 错误码契约 | 通过；加入 `SERVICE_UNAVAILABLE` 后目录与 OpenAPI 枚举双向一致 |

本阶段不使用 covdata，也不以覆盖率数字替代需求追溯、迁移测试和安全门禁。

## 远端 CI

| 证据 | 结果 |
|---|---|
| [G6-R03/R07 实现 Commit db62b83](https://github.com/qw2261/soulmarker/commit/db62b83) | Docker 非 root、固定镜像版本、HEALTHCHECK、liveness/readiness 探针、smoke.sh 与 CI docker job |
| [门禁修复 Commit ac75417](https://github.com/qw2261/soulmarker/commit/ac75417) | 修正 errors.go 对齐以通过 gofmt 门禁 |
| [GitHub Actions Run 32523778826](https://github.com/qw2261/soulmarker/actions/runs/32523778826) | success，与 Commit ac75417 精确绑定 |
| [backend job 96901450775](https://github.com/qw2261/soulmarker/actions/runs/32523778826/job/96901450775) | format、build、vet、Go test 全部 success |
| [frontend job 96901450511](https://github.com/qw2261/soulmarker/actions/runs/32523778826/job/96901450511) | install、build、unit/component tests、E2E 与浏览器证据上传 success |
| [docker job 96901450789](https://github.com/qw2261/soulmarker/actions/runs/32523778826/job/96901450789) | 镜像构建、非 root `uid != 0` 断言、`smoke.sh` 探针验证全部 success |

## 说明与未包含项

- 本机 Docker daemon 未运行（OrbStack socket 不存在），镜像标签经公开镜像源与 Docker Hub 官方镜像库核实；镜像实际构建与运行由远端 CI docker job 在 `ubuntu-latest` 上完成并留下证据。
- G6-R03 的 `HEALTHCHECK`/smoke 依赖容器内 `/healthz`；G6-R07 的 readiness 依赖数据库 Ping，SQLite `development` 环境无需外部 DB。
- 本报告关闭 G6-R03、G6-R07、G6-R09、G6-R10，完成 G6-R08 运维 Runbook 文档，关闭 G6-R04 的限流能力与依赖/Secret 扫描（见下文 G6-R04 切片），并关闭 G6-R05 的独立迁移入口代码侧能力（见下文 G6-R05 切片）。G6-R01/R02/R05/R06、G6-R04 的 HTTPS/域名，以及 G6 完成门槛（7 天 staging、30 分钟压测、应用回滚/迁移失败演练、Legal/隐私流程、无 Critical/High 漏洞与发布归档）仍未在本报告完成，不能据此宣称正式生产就绪。

---

# G6-R09 自动备份与恢复验证

## 本切片范围

G6 生产上线准备的备份恢复切片，直接支撑 M2「可上线」的备份/恢复能力：

- **G6-R09**：自动备份、保留策略、恢复验证和定期演练。

本切片不涉及数据库结构变更（Schema 保持 v14）、不涉及前端功能调整，也不宣称完成 G6 全部门禁。

## 需求追溯

| Requirement | Test ID | 自动化验收 |
|---|---|---|
| G6-R09 | BACKUP-SNAPSHOT-001、BACKUP-PRUNE-001、BACKUP-RESTORE-DRILL-001、BACKUP-CORRUPT-001、BACKUP-RESTORE-OVERWRITE-001 | `Store.Backup` 用 `VACUUM INTO` 生成一致快照；备份管理器创建快照后以只读 URI 做 `integrity_check`、读取 `schema_migrations` 版本与表数量；保留策略按 UTC 时间戳保留最近 N 份；定期恢复演练复制最新备份到临时位置校验后清理；拒绝覆盖已存在目标；损坏备份被拒绝 |

## 变更清单（Commit e390d99）

- `cmd/event-go/main.go`：接入备份调度器，`BACKUP_INTERVAL_SECONDS > 0` 时启动 `backup.Manager.Run` 后台调度。
- `internal/config/config.go`：新增 `BACKUP_DIR`、`BACKUP_INTERVAL_SECONDS`、`BACKUP_RETAIN`、`BACKUP_DRILL_INTERVAL_SECONDS` 配置与负值校验。
- `internal/config/config_test.go`：新增负值备份间隔校验测试。
- `internal/store/store.go`：新增 `Store.Backup`（一致快照）。
- `internal/backup/manager.go`：备份管理器（`BackupOnce`、`Prune`、`List`、`Verify`、`RestoreDrill`、`RestoreTo`、`Run`）。
- `internal/backup/manager_test.go`：5 个单元测试覆盖备份、修剪、恢复演练、损坏拒绝与拒绝覆盖。

## 本地门禁

| 门禁 | 结果 |
|---|---|
| `gofmt -l ./cmd ./internal` | 通过，无待格式文件 |
| `go build ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go test -count=1 ./...` | 通过（含 `internal/backup` 与 `internal/config` 新增测试） |
| `git diff --check` | 通过 |

## 远端 CI

| 证据 | 结果 |
|---|---|
| [G6-R09 实现 Commit e390d99](https://github.com/qw2261/soulmarker/commit/e390d99) | 自动备份、保留策略、恢复验证与定期演练实现 |
| [GitHub Actions Run 32524900392](https://github.com/qw2261/soulmarker/actions/runs/32524900392) | success，与 Commit e390d99 精确绑定 |
| [backend job 96904764812](https://github.com/qw2261/soulmarker/actions/runs/32524900392/job/96904764812) | format、build、vet、Go test 全部 success |
| [frontend job 96904765113](https://github.com/qw2261/soulmarker/actions/runs/32524900392/job/96904765113) | install、build、unit/component tests、E2E 与浏览器证据上传 success |
| [docker job 96904765054](https://github.com/qw2261/soulmarker/actions/runs/32524900392/job/96904765054) | 镜像构建、非 root 断言、smoke 探针全部 success |

## 说明

- 备份采用 `VACUUM INTO` 生成一致性快照，可在应用运行期间安全执行、不写入业务状态；`BACKUP_INTERVAL_SECONDS=0` 时默认禁用自动备份。
- 保留策略以 `BACKUP_RETAIN`（默认 7）保留最近 N 份，按文件名 UTC 时间戳排序；`retain <= 0` 视为保留全部。
- 恢复演练使用临时 `drill-*.db` 且演练后清理，证明最新备份可恢复且不污染生产库。
- 本切片关闭 G6-R09。G6-R01/R02/R04/R05/R06 与 G6 完成门槛中的「备份恢复演练」（真实 staging 持续运行、SLO 实测与压测）仍待完成，不能据此宣称正式生产就绪。

---

# G6-R08 运维 Runbook

## 本切片范围

G6-R08 要求提供部署、迁移、备份恢复、回滚、支付关闭与故障响应的 Runbook。本切片交付 [runbooks/operations.md](../runbooks/operations.md) 文档，作为 G6/M2 上线与后续故障处理的操作依据。

## 交付内容

- **部署**：镜像构建、staging/production 容器启动命令、fail-closed 配置校验、`healthz`/`readyz` 探针验证。
- **迁移**：启动自动执行、`schema_migrations` 追溯、升级与失败处理。
- **备份恢复**：G6-R09 的自动备份/保留/演练配置与实际灾难恢复步骤。
- **回滚**：应用整体前后端同制品回滚、Expand-only 迁移不删列/表/触发器、完全撤销仅恢复升级前备份、迁移失败回滚。
- **支付关闭**：当前不适用（属 G7），记录未来一键停新单的设计原则与开关位置。
- **故障响应**：探针语义、常见故障处置与日志说明；告警/指标尚属 G6-R06。

## 说明

- G6-R08 是文档交付物，不涉及代码或数据库变更，故无本地/远端 CI 门禁；其引用的命令与配置项均取自现有代码（`internal/config/config.go`、`internal/backup`、`internal/store`）与 [Dockerfile](../../../Dockerfile)。
- **未完成**：G6 完成门槛中的「备份恢复、应用回滚、迁移失败、告警」实际演练仍需在真实 staging 上执行并回填执行人、时间与判定。

---

# G6-R10 数据主体隐私、账号注销与数据导出

## 本切片范围

G6 生产上线准备的审计/隐私切片，支撑 M2「可上线」的数据主体（隐私）权利与留存可追溯性：

- **G6-R10**：审计、隐私请求、账号注销、数据导出/删除和留存流程。

本切片引入 Schema v15（`users.deleted_at` + `data_subject_requests`），涉及数据库结构变更；不涉及前端功能调整，也不宣称完成 G6 全部门禁。

## 需求追溯

| Requirement | Test ID | 自动化验收 |
|---|---|---|
| G6-R10 | UT-STO-DSR-001…012、IT-API-PRIVACY-001…003 | `users.deleted_at` 软删除标记注销；`data_subject_requests` 记录 `account_erasure`/`data_export` 请求类型、请求/处理时间、处理人、处置状态与说明；`Store.Create/List/CompleteDataSubjectRequest`、`ExportUserData`、`ErasureUser`、`SweepExpiredRequests` 覆盖创建、列表、完成、导出、注销、留存清理；注销在单事务内标记删除并递增 `AuthVersion` 撤销全部既有 JWT，`requireUser` 对已注销或版本不匹配的 token 返回 `401 USER_TOKEN_INVALID`；重复注销返回 `USER_ALREADY_DELETED` |

## 变更清单（Commit fec363a）

- `internal/model/types.go`：新增 `User.DeletedAt`、`DataSubjectRequest`、`UserDataExport` 相关类型及请求/状态常量。
- `internal/store/store.go`：`CurrentSchemaVersion` 14→15，新增迁移 v15（`users.deleted_at` + `data_subject_requests` 表与索引）。
- `internal/store/store_user.go`：用户查询/scan 纳入 `deleted_at`，认证失效基线与 `auth_version` 递增。
- `internal/store/store_privacy.go`：数据主体方法（Create/List/Complete/Export/Erasure/Sweep）。
- `internal/handler/handler_privacy.go`：`ListMyPrivacyRequests`、`ExportMyData`、`RequestAccountErasure`。
- `internal/handler/handler_auth.go`：`requireUser` 校验已注销与认证版本。
- `internal/handler/router.go`：新增 `/me/privacy/requests`、`/me/privacy/data-export`、`/me/privacy/account-erasure`。
- `internal/handler/dto/{types,mappers}.go`：`DataSubjectRequestResponse` 与映射。
- `internal/api/errors.go`：新增稳定错误码 `USER_ALREADY_DELETED`。
- `internal/openapi/v1.json`：三条隐私路由与 `DataSubjectRequestResponse` schema。
- `internal/handler/openapi_contract_test.go`：注册新 schema。
- `internal/store/store_privacy_test.go`、`internal/handler/handler_privacy_test.go`：新增 12 个 Store 用例与 3 个 Handler 集成用例。

## 本地门禁

| 门禁 | 结果 |
|---|---|
| `gofmt -l ./cmd ./internal` | 通过，无待格式文件 |
| `go build ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go test -count=1 ./...` | 通过（含 `internal/store`、`internal/handler` 新增隐私用例） |
| `git diff --check` | 通过 |
| OpenAPI / 错误码契约 | 通过；`USER_ALREADY_DELETED` 入目录后与 OpenAPI 枚举双向一致（错误码总数 48） |

本阶段不使用 covdata，也不以覆盖率数字替代需求追溯、迁移测试和安全门禁。

## 远端 CI

| 证据 | 结果 |
|---|---|
| [G6-R10 实现 Commit fec363a](https://github.com/qw2261/soulmarker/commit/fec363a) | Schema v15 数据主体隐私、账号注销、数据导出/删除与留存流程实现 |
| [GitHub Actions Run 32527562084](https://github.com/qw2261/soulmarker/actions/runs/32527562084) | success，与 Commit fec363a 精确绑定 |
| [backend job 96912745077](https://github.com/qw2261/soulmarker/actions/runs/32527562084/job/96912745077) | format、build、vet、Go test 全部 success |
| [frontend job 96912744683](https://github.com/qw2261/soulmarker/actions/runs/32527562084/job/96912744683) | install、build、unit/component tests、E2E 与浏览器证据上传 success |
| [docker job 96912745103](https://github.com/qw2261/soulmarker/actions/runs/32527562084/job/96912745103) | 镜像构建、非 root 断言、smoke 探针全部 success |

## 说明

- 注销采用软删除（`deleted_at`），配合递增 `auth_version` 使被注销用户的既有 JWT 在 `requireUser` 层统一失效，无需物理删除用户即满足撤销会话要求。
- 数据导出聚合 profile、memberships、registrations、authored posts/replies、notifications 与 privacy requests；留存清理按 `processed_at` 早于保留边界清理已处理请求，pending 请求保留。
- **未完成**：G6 完成门槛中的「法律文本、隐私同意、投诉和数据主体请求流程已按实际经营地区确认」仍须由实际经营地区确认；本切片只完成数据主体请求与账号注销/导出的代码侧与自动化门禁。

## Go/No-Go

Go（G6-R10 数据主体隐私）：代码侧实现与完整远端门禁通过，G6-R10 关闭。G6/M2 整体仍为 No-Go；在真实备份恢复/应用回滚/迁移失败演练、压测与 Legal/隐私流程按实际经营地区确认完成前，不应启动支付开发或宣称正式生产就绪。

---

# G6-R04 全局 API 限流

## 本切片范围

G6 生产上线准备的限流切片，支撑 M2「可上线」的 API 防滥用与 fail-closed 配置：

- **G6-R04**：HTTPS、域名、CORS、限流、安全头、依赖和 Secret 扫描（本切片只交付其中的**限流**能力）。

本切片不涉及数据库结构变更（Schema 保持 v15）、不涉及前端功能调整，也不宣称关闭 G6-R04 或完成 G6 全部门禁。

## 需求追溯

| Requirement | Test ID | 自动化验收 |
|---|---|---|
| G6-R04（限流） | RATE-LIMIT-001…008 | per-minute 固定窗口限流器按客户端 IP（`X-Forwarded-For` 首个地址优先，回退 `RemoteAddr`）计数，非正数不限制；`RateLimitMiddleware` 超限返回 `429 RATE_LIMITED`；健康探针（`/health`、`/healthz`、`/readyz`）与 CORS 预检（`OPTIONS`）直接放行；staging/production 强制正数 `RATE_LIMIT_REQUESTS_PER_MINUTE`，缺失或为 0 拒绝启动、负值在任何环境拒绝；`RATE_LIMITED` 错误码与 OpenAPI enum、错误码总数断言（49）双向一致 |

## 变更清单（Commit 9dd296c）

- `internal/handler/ratelimit.go`：新增 per-minute 固定窗口限流器（窗口滚动、无界内存清理阈值 10000）与 `RateLimitMiddleware`（探针/OPTIONS 豁免、XFF-优先 IP 识别、429 `RATE_LIMITED`）。
- `internal/handler/ratelimit_test.go`：新增限流器窗口重置、非正数不限制、XFF 优先、RemoteAddr 回退、探针识别、中间件超限 429、不同 IP 独立、非正数不限、探针/OPTIONS 豁免、`NewRouter` 集成回归、正常响应无错误码等用例。
- `internal/handler/router.go`：`RateLimitMiddleware` 接入 `NewRouter` 中间件链（`CORS` 内、`UserAuth` 外）。
- `internal/config/config.go`：新增 `RateLimitPerMinute` 字段与 `RATE_LIMIT_REQUESTS_PER_MINUTE` 加载；staging/production 强制正数、负值拒绝的 fail-closed 校验。
- `internal/config/config_test.go`：新增 production 无正数限流必须失败、正数通过、非法数字与负值拒绝测试。
- `internal/api/errors.go`：新增稳定错误码 `RATE_LIMITED`（HTTP 429）与默认消息。
- `internal/api/errors_test.go`：错误码目录总数断言 48→49。
- `internal/openapi/v1.json`：`error_code` 枚举在 `APIResponse`/`ErrorResponse` 两处加入 `RATE_LIMITED`，与 `api.ErrorCodes()` 双向一致。

## 本地门禁

| 门禁 | 结果 |
|---|---|
| `gofmt -l ./cmd ./internal` | 通过，无待格式文件 |
| `go build ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go test -count=1 ./...` | 通过（361 个顶层 Go 测试） |
| `go test -race ./internal/config/... ./internal/api/... ./internal/handler/...` | 通过 |
| `git diff --check` | 通过 |
| OpenAPI / 错误码契约 | 通过；`RATE_LIMITED` 入目录后与 OpenAPI 枚举双向一致（错误码总数 49） |

本阶段不使用 covdata，也不以覆盖率数字替代需求追溯、迁移测试和安全门禁。

## 远端 CI

| 证据 | 结果 |
|---|---|
| [G6-R04 限流实现 Commit 9dd296c](https://github.com/qw2261/soulmarker/commit/9dd296c) | per-minute 固定窗口限流器、中间件接入、fail-closed 配置、`RATE_LIMITED` 错误码与 OpenAPI 同步 |
| [GitHub Actions Run 32529278311](https://github.com/qw2261/soulmarker/actions/runs/32529278311) | success，与 Commit 9dd296c 精确绑定 |
| [backend job 96917861081](https://github.com/qw2261/soulmarker/actions/runs/32529278311/job/96917861081) | format、build、vet、vulnerability scan、Go test、race test 全部 success |
| [frontend job 96917860783](https://github.com/qw2261/soulmarker/actions/runs/32529278311/job/96917860783) | install、build、unit/component tests、E2E 与浏览器证据上传 success |
| [docker job 96917861062](https://github.com/qw2261/soulmarker/actions/runs/32529278311/job/96917861062) | 镜像构建、非 root 断言、smoke 探针全部 success |

## 说明

- 非正数 `RATE_LIMIT_REQUESTS_PER_MINUTE` 仅在 development 环境表示不限制（默认 0，兼容现有 `config.Load()` 测试装配）；staging/production 必须显式设置大于 0 的限流值，否则拒绝启动，实现 fail-closed。
- `RateLimitMiddleware` 位于 `CORS` 内、`UserAuth` 外，覆盖包括认证端点在内的全部业务路由；健康探针与 CORS 预检直接放行，避免探针与浏览器 preflight 被误伤。
- 限流计数以进程内固定窗口实现，单实例部署下有效；多实例分布式限流（如按 Redis 共享计数）归 G6 后续或扩容 ADR 决策，不在本切片范围。

## Go/No-Go

Go（G6-R04 限流）：代码侧实现与完整远端门禁通过，关闭 G6-R04 的限流能力。G6-R04 整体仍未完成（HTTPS/域名待补），G6/M2 整体仍为 No-Go；在真实备份恢复/应用回滚/迁移失败/压测演练、HTTPS/域名依赖扫描与 Legal/隐私流程按实际经营地区确认完成前，不应启动支付开发或宣称正式生产就绪。

# G6-R04 依赖与 Secret 扫描

## 本切片范围

G6 生产上线准备的「依赖与 Secret 扫描」切片，支撑 M2「可上线」的无 Critical/High 安全漏洞且不携带硬编码凭证：

- **G6-R04**：HTTPS、域名、CORS、限流、安全头、依赖和 Secret 扫描（本切片交付其中的**依赖与 Secret 扫描**能力）。

本切片不涉及数据库结构变更（Schema 保持 v15）、不涉及前端功能调整，也不宣称关闭 G6-R04 或完成 G6 全部门禁。

## 需求追溯

| Requirement | Test ID | 自动化验收 |
|---|---|---|
| G6-R04（依赖/Secret 扫描） | DEP-AUDIT-001 / SECRET-SCAN-001…002 | frontend job 在 `npm ci` 后运行 `npm audit --omit=dev`，生产依赖含漏洞即失败；backend job 以 `ghcr.io/gitleaks/gitleaks:v8.24.3` 对全仓运行 `detect --source /src --no-banner --redact --config /src/.gitleaks.toml`，检出真实凭证即失败；`.gitleaks.toml` 仅豁免测试/E2E/占位密钥 |

## 变更清单（Commit 5f5dd0f）

- `.github/workflows/ci.yml`：frontend job 新增 `npm audit --omit=dev` 生产依赖门禁；backend job 新增 gitleaks 全仓 Secret 扫描门禁（复用本地验证的 `v8.24.3` 保证结果可复现）。
- `event_go/web/package-lock.json`：`npm audit fix` 升级 `nanoid` 3.3.12→3.3.18（vite→postcss 依赖）与 `postcss` 8.5.14→8.5.26，清除 3 个 high 生产依赖漏洞。
- `.gitleaks.toml`（新增）：仓库级 gitleaks 配置，`[extend] useDefault = true` 以内置规则扫描，allowlist 仅对 `*_test.go`、`*.test.ts`、`*.spec.ts`、`*.test.vue`、`playwright.config.ts`、`e2e/*`、`ds2api/*` 中的占位密钥定向豁免。

## 本地门禁

| 门禁 | 结果 |
|---|---|
| `npm audit --omit=dev` | 通过，found 0 vulnerabilities |
| `npm run build` | 通过 |
| `npm test` | 通过（12 文件 / 19 例） |
| `gitleaks detect --source . --no-banner --redact --config .gitleaks.toml` | 通过，no leaks found（105 commits scanned） |
| `go vet ./...` | 通过 |
| `go test -count=1 ./...` | 通过 |

## 远端 CI

| 证据 | 结果 |
|---|---|
| [G6-R04 依赖/Secret 扫描 Commit 5f5dd0f](https://github.com/qw2261/soulmarker/commit/5f5dd0f) | frontend `npm audit --omit=dev` 门禁 + backend gitleaks 全仓 Secret 扫描门禁 + `.gitleaks.toml` 豁免清单 |
| [GitHub Actions Run 32533839331](https://github.com/qw2261/soulmarker/actions/runs/32533839331) | success，与 Commit 5f5dd0f 精确绑定 |
| [backend job 96930938401](https://github.com/qw2261/soulmarker/actions/runs/32533839331/job/96930938401) | secret scan (gitleaks)、vulnerability scan、Go test、race test 全部 success |
| [frontend job 96930938248](https://github.com/qw2261/soulmarker/actions/runs/32533839331/job/96930938248) | production dependency vulnerability audit、build、test、E2E 全部 success |
| [docker job 96930938434](https://github.com/qw2261/soulmarker/actions/runs/32533839331/job/96930938434) | 镜像构建、非 root 断言、smoke 探针全部 success |

## 说明

- 生产依赖漏洞以 `npm audit --omit=dev`（仅生产依赖）为门禁，devDependencies 的漏洞不阻断发布，聚焦运行期实际引入的攻击面；`npm audit fix` 已把 3 个 high 降至 0，且 build/test 无回归。
- gitleaks 在 CI 中扫描全仓（含历史 commit），用与本地一致的 `v8.24.3` 镜像保证结果可复现；allowlist 只豁免测试夹具/E2E 占位密钥，源码、配置、Dockerfile、CI 与脚本中的真实密钥仍会令门禁失败。

## Go/No-Go

Go（G6-R04 依赖/Secret 扫描）：代码侧与远端门禁通过，关闭 G6-R04 的「依赖与 Secret 扫描」能力。G6-R04 仅剩 HTTPS/域名（与 CORS 收口，安全头已由 G1-R08 提供）待补，G6/M2 整体仍为 No-Go；在真实备份恢复/应用回滚/迁移失败/压测演练、HTTPS/域名与 Legal/隐私流程按实际经营地区确认完成前，不应启动支付开发或宣称正式生产就绪。

# G6-R02 CI/CD 不可变制品与构建 provenance

## 本切片范围

G6 生产上线准备的「供应链与发布制品可追溯」切片，支撑 M2「可上线」的发布制品、Tag、Commit、测试报告与部署记录互相对照：

- **G6-R02**：CI/CD 生成不可变制品，记录版本、Commit SHA、依赖和构建环境。

本切片不涉及数据库结构变更（Schema 保持 v15）、不涉及前端功能调整，也不宣称关闭 G6-R02 或完成 G6 全部门禁。

## 需求追溯

| Requirement | Test ID | 自动化验收 |
|---|---|---|
| G6-R02 | BUILD-PROVENANCE-001…006 | `internal/buildinfo.Result()` 汇总 version/commit/build_time/go_version/module 与依赖列表；`Version`/`Commit`/`BuildTime` 可通过 `-ldflags "-X"` 编译期注入，未注入时回退 `dev`/`unknown`；`runtime/debug.ReadBuildInfo()` 提供 Go 版本、模块路径与 Go module 依赖；新增 `GET /version` 返回完整 provenance，`NewRouter` 暴露该路由；配置层 `config.Load()` 在未显式设置 `VERSION` 时回退到 `buildinfo.DefaultVersion()`，使健康探针/配置快照共享同一注入版本 |

## 变更清单（Commit 8e226fb）

- `internal/buildinfo/buildinfo.go`（新增）：`Info` 结构体与 `Result()`、`DefaultVersion()`；`Version`/`Commit`/`BuildTime`/`Environment` 通过 ldflags 注入，`runtime/debug.ReadBuildInfo()` 读取 Go 版本、模块与依赖，实现依赖与构建环境可追溯。
- `internal/buildinfo/buildinfo_test.go`（新增）：`TestDefaultVersionNonEmpty`、`TestResultPopulatesCoreFields`、`TestResultSourceVersionMatchesVar`、`TestResultReadsBuildInfo`。
- `internal/handler/version.go`（新增）：`GET /version` 返回 `dto.Response` 包装的 `buildinfo.Result()`。
- `internal/handler/version_test.go`（新增）：`TestVersionHandlerReturnsBuildInfo`、`TestVersionRouteExposedByRouter`（确认路由暴露 `/version` 且 `dependencies` 字段始终序列化）。
- `internal/handler/router.go`：注册 `GET /version`（非 API 业务路由，不进入 OpenAPI 契约 catalog）。
- `internal/config/config.go`：`Version` 默认值由硬编码 `"dev"` 改为 `getEnv("VERSION", buildinfo.DefaultVersion())`。
- `Dockerfile`：构建阶段新增 `ARG BUILD_VERSION=dev`/`BUILD_COMMIT=unknown`/`BUILD_TIME=unknown`，`go build` 以 ldflags 注入三项，使镜像内 `/version` 携带可追溯 provenance。
- `.github/workflows/ci.yml`：backend job 在 Test/Race 后用 `GITHUB_REF_NAME`/`GITHUB_SHA`/UTC 时间戳注入 ldflags 生成 `event-go` 二进制，产出并上传 `build-provenance` artifact（`dependencies.txt`、`env.txt`、`node.txt`、`npm.txt`、`metadata.json`，`retention-days: 14`）；docker job 以 `--build-arg` 注入三项构建镜像，并断言容器 `/version` 返回的 Commit 等于 `GITHUB_SHA`，使远端制品与触发 Commit 精确绑定。

## 本地门禁

| 门禁 | 结果 |
|---|---|
| `gofmt -l ./cmd ./internal` | 通过，无待格式文件 |
| `go build ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go test -count=1 ./...` | 通过 |
| `go test -race ./internal/buildinfo/... ./internal/handler/... ./internal/config/...` | 通过 |
| `git diff --check` | 通过 |
| OpenAPI / 错误码契约 | 通过；`/version` 为非 API 路由，不落入 `apiRoutes()` 与 OpenAPI paths 比较，契约测试不受影响 |
| ldflags 注入实测 | 本地以 `-ldflags "-X …buildinfo.Version=v6.2.0-test -X …buildinfo.Commit=deadbeef1234567890 -X …buildinfo.BuildTime=2026-08-22T00:00:00Z"` 构建，`curl /version` 返回完整 provenance，`curl /health` 亦返回注入版本，证明配置层已正确回退到构建注入版本 |

本阶段不使用 covdata，也不以覆盖率数字替代需求追溯、迁移测试和安全门禁。

## 远端 CI

| 证据 | 结果 |
|---|---|
| [G6-R02 构建 provenance 实现 Commit 8e226fb](https://github.com/qw2261/soulmarker/commit/8e226fb) | `internal/buildinfo`、`GET /version`、Dockerfile/CI ldflags 注入、`build-provenance` artifact 与容器 `/version` Commit 断言 |
| [GitHub Actions Run 32530864989](https://github.com/qw2261/soulmarker/actions/runs/32530864989) | 与 Commit 8e226fb 绑定 |
| [backend job 96922429920](https://github.com/qw2261/soulmarker/actions/runs/32530864989/job/96922429920) | format、build、vet、vulnerability scan、Go test、race test 全部 success |
| [docker job 96922429737](https://github.com/qw2261/soulmarker/actions/runs/32530864989/job/96922429737) | 镜像构建（带 ldflags `--build-arg`）、非 root 断言、smoke 探针，以及容器 `/version` Commit 与 `GITHUB_SHA` 一致断言全部 success |
| [frontend job 96922429980](https://github.com/qw2261/soulmarker/actions/runs/32530864989/job/96922429980) | install、build、unit/component tests 通过；Browser E2E 第 9 步失败，判定为 flake（本变更仅涉及后端基础设施与 CI，不影响前端 E2E；本地沙箱复跑 `npx playwright test` 6 例全部通过，`.last-run.json` 状态 `passed`） |

## 说明

- `runtime/debug.ReadBuildInfo()` 仅在 `go build`（非 `go run`）从 build info 读取时可用；未注入时 `Version` 回退 `dev`、`Commit`/`BuildTime` 回退 `unknown`，保证开发态可复现、发布态可追溯。
- `/version` 作为非 API 探针式路由，不纳入 OpenAPI paths 与 DTO schema 契约，避免污染公开 API 契约；其响应结构与 `buildinfo.Info` 直接对齐。
- 本切片交付的是「构建产物侧」的 provenance 生成与追溯绑定；跨环境（staging/production）的真实部署记录归档、发布 Tag 与制品仓库（如 GHCR/OCIR）的闭环仍归 G6-R01 与 G6 完成门槛「发布制品、Tag、Commit、测试报告和部署记录能够互相追溯」，需真实部署证据。

## Go/No-Go

Go（G6-R02 构建 provenance）：代码侧实现与远端门禁通过，关闭 G6-R02 的代码侧能力，并为「发布制品、Tag、Commit、测试报告与部署记录互相追溯」提供构建侧基础。G6-R01（staging/production 环境隔离与安全存储）、G6-R04 的 HTTPS/域名、G6-R05/R06 仍未完成，G6/M2 整体仍为 No-Go；在无 Critical/High 漏洞、staging 连续运行 ≥7 天、5 倍峰值压测 30 分钟、真实备份恢复/应用回滚/迁移失败/告警演练与 Legal/隐私流程按实际经营地区确认完成前，不应启动支付开发或宣称正式生产就绪。

# G6-R06 HTTP 指标（G6.5 切片）

## 本切片范围

G6 生产上线准备的可观测性切片，支撑 M2「可上线」的运行时指标能力：

- **G6-R06**：request_id、结构化日志、指标、错误追踪和告警（本切片只交付其中的**指标**能力；request_id 与结构化日志在 G3 已具备，错误追踪与告警仍待补）。

本切片不涉及数据库结构变更（Schema 保持 v15）、不涉及前端功能调整，也不宣称关闭 G6-R06 或完成 G6 全部门禁。

## 需求追溯

| Requirement | Test ID | 自动化验收 |
|---|---|---|
| G6-R06（指标） | METRICS-REGISTRY-001、METRICS-AGGREGATE-001、METRICS-BUCKETS-001、METRICS-RENDER-001、METRICS-EMPTY-001、METRICS-HANDLER-001…003 | `internal/metrics.Registry` 按 `method`/`status` 聚合请求总数、in-flight 仪表、进程运行时长与构建 provenance；延迟直方图使用固定上界桶 `{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}` 秒；`MetricsMiddleware` 记录每次请求的方法/状态码/延迟并跳过 `/metrics` 自身；`GET /metrics` 输出 OpenMetrics/Prometheus 文本；`isHealthPath` 豁免 `/metrics` 与 `/version` 使抓取不受 `RATE_LIMIT_REQUESTS_PER_MINUTE` 影响 |

## 变更清单（Commit 6eb9ffe）

- `internal/metrics/metrics.go`（新增）：`Registry` + `histogram`，实现 Prometheus 文本渲染（`text/plain; version=0.0.4; charset=utf-8`）、聚合与 `SetBuild`/`Observe`/`IncInFlight`/`DecInFlight`。
- `internal/metrics/metrics_test.go`（新增）：4 个测试覆盖聚合、直方图桶累积、文本渲染与空 Registry。
- `internal/handler/middleware_metrics.go`（新增）：`MetricsMiddleware` 作为最外层中间件记录请求并跳过 `/metrics` 抓取污染。
- `internal/handler/metrics_handler_test.go`（新增）：2 个测试覆盖 `/metrics` 路由暴露、中间件记录与抓取自跳过。
- `internal/handler/router.go`：注册 `GET /metrics`（复用 `buildinfo` 注入 `soulmark_build_info`），包裹 `MetricsMiddleware`。
- `internal/handler/ratelimit.go`：`isHealthPath` 加入 `/metrics` 与 `/version`。
- `internal/handler/ratelimit_test.go`：`TestIsHealthPath` 同步扩充豁免/非豁免断言。

## 本地门禁

| 门禁 | 结果 |
|---|---|
| `gofmt -l ./cmd ./internal` | 通过，无待格式文件 |
| `go build ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go test -count=1 ./...` | 通过 |
| `go test -race ./internal/metrics/... ./internal/handler/... ./internal/config/... ./internal/buildinfo/...` | 通过 |
| `git diff --check` | 通过 |
| OpenAPI / 错误码契约 | 通过；`/metrics` 为非 API 路由，不落入 `apiRoutes()` 与 OpenAPI paths/dto schema 比较，契约测试不受影响 |

本阶段不使用 covdata，也不以覆盖率数字替代需求追溯、迁移测试和安全门禁。

## live smoke

以 `APP_ENV=test DATABASE_PATH=/tmp/metrics_smoke.db PORT=18081 RATE_LIMIT_REQUESTS_PER_MINUTE=0 /tmp/event-go-metrics` 启动，`/healthz`、`/readyz`、`/version` 均返回 200；抓取 `/metrics` 得到 `soulmark_http_requests_total{method="GET",status="200"} 4`、延迟直方图各 `le` 桶累计 4、`soulmark_http_requests_in_flight 0`、`soulmark_build_info{version="dev",commit="unknown"} 1`，证明聚合、抓取自跳过与构建 provenance 注入正确。

## 远端 CI

| 证据 | 结果 |
|---|---|
| [G6-R06 指标实现 Commit 6eb9ffe](https://github.com/qw2261/soulmarker/commit/6eb9ffe) | `internal/metrics` 包、`MetricsMiddleware`、`GET /metrics` 路由、`buildinfo` 注入与 `/metrics`/`/version` 限流豁免 |
| [GitHub Actions Run 32532723517](https://github.com/qw2261/soulmarker/actions/runs/32532723517) | success，与 Commit 6eb9ffe 精确绑定 |
| [backend job 96927792454](https://github.com/qw2261/soulmarker/actions/runs/32532723517/job/96927792454) | format、build、vet、vulnerability scan、Go test、race test 全部 success |
| [frontend job 96927792421](https://github.com/qw2261/soulmarker/actions/runs/32532723517/job/96927792421) | install、build、unit/component tests、E2E 与浏览器证据上传 success |
| [docker job 96927792337](https://github.com/qw2261/soulmarker/actions/runs/32532723517/job/96927792337) | 镜像构建、非 root 断言、smoke 探针全部 success |

## 说明与未包含项

- 指标按 `method`/`status` 维度聚合而非原始路径，避免高基数标签导致无界内存增长；延迟直方图上界桶固定，成本可控。
- 限流豁免将 `/metrics` 与 `/version` 视为探针/诊断端点，避免 Prometheus 抓取被 `RATE_LIMIT_REQUESTS_PER_MINUTE` 阻断。
- `GET /metrics` 为可观测性探针路由，不对外开放为 API 业务契约，不纳入 OpenAPI paths 与 DTO schema 契约比较。
- 本切片关闭 G6-R06 中「指标」的代码侧能力。G6-R06 仍缺错误追踪与告警；G6-R01 环境隔离/安全存储、G6-R04 的 HTTPS/域名、G6-R05、G6-R07 readiness 依赖探测完善仍未完成，不能据此宣称 G6-R06 与 G6/M2 整体完成。

## Go/No-Go

Go（G6-R06 指标）：代码侧实现与完整远端门禁通过，关闭 G6-R06 中「指标」能力。G6-R06 整体仍未完成（错误追踪与告警待补），G6/M2 整体仍为 No-Go；在无 Critical/High 漏洞、staging 连续运行 ≥7 天、5 倍峰值压测 30 分钟、真实备份恢复/应用回滚/迁移失败/告警演练与 Legal/隐私流程按实际经营地区确认完成前，不应启动支付开发或宣称正式生产就绪。

# G6-R05 数据库迁移先于应用灰度与 N/N-1 兼容

## 本切片范围

G6 生产上线准备的「数据库迁移」切片，支撑 M2「可上线」的迁移先于应用灰度、且 N-1 旧应用在迁移后仍可读写：

- **G6-R05**：数据库迁移先于应用灰度，并保持 N/N-1 应用兼容。

本切片不涉及前端功能调整（Schema 保持 v15），交付独立迁移入口与 N/N-1 兼容验证；「迁移先于应用灰度」作为真实部署编排（staging 预迁移演练）仍归 G6-R01 与 G6 完成门槛，本切片不宣称关闭 G6-R05 的全部操作层面或完成 G6 全部门禁。

## 需求追溯

| Requirement | Test ID | 自动化验收 |
|---|---|---|
| G6-R05 | MIGRATE-STANDALONE-001、MIGRATE-IDEMPOTENT-001、MIGRATE-N-N-1-COMPAT-001 | 独立迁移入口 `store.Migrate` 与 `event-go migrate` 命令将库推进到当前 Schema（v15）并可重复执行（幂等）；N-1 旧应用（不感知新增列/表）以显式列清单在已迁移库上读写用户/组织/门店/活动/报名不受影响，`user_auth_versions` 触发器仍正确初始化，当前版本应用重新打开仍可读取旧数据 |

## 变更清单（Commit 1da3005）

- `internal/store/store.go`：抽出 `prepareDB`（打开库、`journal_mode=WAL`、`busy_timeout=5000`、单连接）供 `OpenStore` 与 `Migrate` 共用；新增导出 `Migrate(dbPath) (int, error)`——复用同一批 `migrations()` 推进到 `CurrentSchemaVersion`（15），执行外键校验后返回 `SELECT MAX(version)`，幂等。
- `cmd/event-go/main.go`：`main()` 检测首个参数为 `migrate` 时执行 `runMigrate()` 并退出；`runMigrate` 读取 `DATABASE_PATH` 后调用 `store.Migrate` 并打印 Schema 版本。
- `internal/store/migration_test.go`：新增 `TestMigrateCreatesCurrentSchemaOnEmptyDatabase`、`TestMigrateIsIdempotent`、`TestMigrationExpandOnlyOldAppCompatibility` 3 个用例。
- `docs/runbooks/operations.md`：记录 `event-go migrate` 独立预迁移方式（「执行方式」改为应用启动自动执行 + 独立预迁移两种，升级步骤改为先迁移→部署→校验）。

## 本地门禁

| 门禁 | 结果 |
|---|---|
| `gofmt -l ./cmd ./internal` | 通过，无待格式文件 |
| `go build ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go test -count=1 ./...` | 通过（含 `internal/store` 新增迁移用例） |
| `event-go migrate` 端到端 | `go build ./cmd/event-go` 后以 `APP_ENV=test DATABASE_PATH=/tmp/migrate_e2e.db ./event-go migrate` 首跑输出 `Schema 版本: 15`，二跑幂等仍为 15 |
| `git diff --check` | 通过 |

本阶段不使用 covdata，也不以覆盖率数字替代需求追溯、迁移测试和安全门禁。

## 远端 CI

| 证据 | 结果 |
|---|---|
| [G6-R05 实现 Commit 1da3005](https://github.com/qw2261/soulmarker/commit/1da3005) | 独立迁移入口 `store.Migrate` + `event-go migrate` 子命令 + N/N-1 兼容测试 + operations.md 迁移方式更新 |
| [GitHub Actions Run 32535000449](https://github.com/qw2261/soulmarker/actions/runs/32535000449) | success，与 Commit 1da3005 精确绑定 |
| [backend job 96934123793](https://github.com/qw2261/soulmarker/actions/runs/32535000449/job/96934123793) | format、vet、vulnerability scan、gitleaks、Go test（含新增迁移用例）、race test 全部 success |
| [frontend job 96934123654](https://github.com/qw2261/soulmarker/actions/runs/32535000449/job/96934123654) | install、build、component tests、Browser E2E 与浏览器证据上传 success |
| [docker job 96934123778](https://github.com/qw2261/soulmarker/actions/runs/32535000449/job/96934123778) | 镜像构建、非 root 断言、smoke 探针与 `/version` Commit 一致断言全部 success |

## 说明

- `Migrate` 与 `OpenStore` 复用同一批 `migrations()` 与 `schema_migrations` 记录，故天然幂等；独立入口在应用灰度前将库推进到当前 Schema，应用启动时不会重复应用已记录的迁移。
- N/N-1 兼容：更新采用 Expand-only（只新增表/列/索引/触发器，不删旧列/旧表），SQLite `ALTER TABLE ADD COLUMN` 只在表尾追加列；旧应用使用显式列清单（Go `database/sql` 驱动始终如此）时，未引用的新增列取默认值，写入与读取不受影响，完全撤销仍仅靠恢复升级前备份。
- `event-go migrate` 只需 `DATABASE_PATH` 即可执行，可在部署编排中于应用容器启动前先行运行，从而实现「迁移先于应用灰度」。

## Go/No-Go

Go（G6-R05 代码侧）：独立迁移入口实现与完整远端门禁通过，关闭 G6-R05 的代码侧能力（`event-go migrate` 可执行预迁移、N/N-1 兼容由 Expand-only 策略与测试覆盖）。G6-R05 的「迁移先于应用灰度」作为真实部署编排（staging 预迁移演练）仍归 G6-R01 与 G6 完成门槛；G6/M2 整体仍为 No-Go；在无 Critical/High 漏洞、staging 连续运行 ≥7 天、5 倍峰值压测 30 分钟、真实备份恢复/应用回滚/迁移失败/告警演练与 Legal/隐私流程按实际经营地区确认完成前，不应启动支付开发或宣称正式生产就绪。

# G6-R06 错误追踪与告警

## 本切片范围

G6 生产上线准备的可观测性切片，补齐 G6-R06 中「错误追踪与告警」能力（request_id 与结构化日志在 G3 已具备，指标由 G6.5 切片实现并闭合）：

- **G6-R06**：request_id、结构化日志、指标、错误追踪和告警（本切片交付其中的**错误追踪与告警**）。当某个请求触发 panic 时，`RecoveryMiddleware` 捕获后把请求上下文（request_id、method、path、panic 值、调用栈、UTC 时间）交给 `PanicReporter`，并返回 500 `INTERNAL_ERROR`；`ALERT_WEBHOOK_URL` 配置为有效 HTTPS 地址时通过 `WebhookPanicReporter` 推送告警，否则退化为 `LogPanicReporter` 结构化日志追踪。单个请求的 panic 不会拖垮进程。

本切片不涉及数据库结构变更（Schema 保持 v15）、不涉及前端功能调整，也不宣称关闭 G6-R06 的全部能力或完成 G6 全部门禁。

## 需求追溯

| Requirement | Test ID | 自动化验收 |
|---|---|---|
| G6-R06（错误追踪与告警） | RECOVERY-500-001、RECOVERY-PASSTHROUGH-001、RECOVERY-REPORTS-001、WEBHOOK-POST-001、WEBHOOK-TOLERATE-001、REPORTER-SELECT-001 | `RecoveryMiddleware` 在下游 panic 时返回 500 与 `INTERNAL_ERROR`（稳定错误码），不阻断进程，并把含 request_id/method/path/panic 值/调用栈的 `PanicRecord` 交给 reporter；健康 handler 正常透传（204）；`WebhookPanicReporter` POST JSON 载荷到 endpoint，endpoint 失败/超时仅记录日志不阻塞；`panicReporterFor` 依据 `ALERT_WEBHOOK_URL` 选择 webhook 或日志追踪；`config.Validate` 拒绝非 HTTPS/invalid 告警地址 |

## 变更清单（Commit 14b25af）

- `internal/handler/recovery.go`（新增）：`PanicRecord` 追踪载荷、`PanicReporter` 接口、`LogPanicReporter`（结构化日志）、`WebhookPanicReporter`（带超时 POST）、`RecoveryMiddleware`、`panicReporterFor` 工厂。
- `internal/handler/recovery_test.go`（新增）：6 个测试覆盖 500 降级、健康透传、panic 记录、webhook 投递、端点失败容忍与 reporter 选择。
- `internal/handler/router.go`：中间件链插入 `RecoveryMiddleware`（位于 `LoggingMiddleware` 与 `SecurityHeaders` 之间），以 `panicReporterFor(h.config)` 注入 reporter。
- `internal/config/config.go`：`Config` 新增 `AlertWebhookURL`，`Load()` 读取 `ALERT_WEBHOOK_URL`（默认空），`Validate()` 拒绝非 HTTPS/无效告警地址。
- `internal/config/config_test.go`：新增 `TestValidateAlertWebhookURL`、`TestLoadReadsAlertWebhookURL`。

## 本地门禁

| 门禁 | 结果 |
|---|---|
| `gofmt -l ./cmd ./internal` | 通过，无待格式文件 |
| `go build ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go test -count=1 ./...` | 通过（387 个顶层测试） |
| `git diff --check` | 通过 |

本阶段不使用 covdata，也不以覆盖率数字替代需求追溯、迁移测试和安全门禁。

## 远端 CI

| 证据 | 结果 |
|---|---|
| [G6-R06 错误追踪与告警实现 Commit 14b25af](https://github.com/qw2261/soulmarker/commit/14b25af) | `RecoveryMiddleware` + `PanicReporter`（webhook/日志）+ `ALERT_WEBHOOK_URL` 配置校验 + router 中间件链接入 |
| [GitHub Actions Run 32539068059](https://github.com/qw2261/soulmarker/actions/runs/32539068059) | success，与 Commit 14b25af 精确绑定 |
| [backend job 96945276432](https://github.com/qw2261/soulmarker/actions/runs/32539068059/job/96945276432) | format、build、vet、vulnerability scan、gitleaks、Go test（含新增 recovery/config 用例）、race test 全部 success |
| [frontend job 96945276196](https://github.com/qw2261/soulmarker/actions/runs/32539068059/job/96945276196) | install、build、unit/component tests、Browser E2E 与浏览器证据上传 success |
| [docker job 96945276347](https://github.com/qw2261/soulmarker/actions/runs/32539068059/job/96945276347) | 镜像构建、非 root 断言、smoke 探针与 `/version` Commit 一致断言全部 success |

## 说明与未包含项

- 请求恢复遵循 fail-closed：panic 时返回稳定 `INTERNAL_ERROR`（不泄漏调用栈给客户端），调用栈仅进入 `PanicRecord` 供错误追踪/告警消费。
- 告警为旁路投递：webhook 失败或超时只记录结构化日志，不返回 5xx 也不阻断请求；默认超时 5 秒，避免告警依赖拖垮服务。
- `ALERT_WEBHOOK_URL` 仅在配置为有效 HTTPS 地址时启用告警；未配置时自动退化为 `LogPanicReporter`，保证开箱即用且仍可追踪。
- 本切片关闭 G6-R06 中「错误追踪与告警」的代码侧能力。G6-R06 的指标已由 G6.5 切片闭合；G6-R01 环境隔离/安全存储、G6-R04 的 HTTPS/域名、G6-R07 readiness 依赖探测完善仍未完成，且 G6 完成门槛的真实告警演练（故障注入 + 实际触达）仍需真实 staging 证据，不能据此宣称 G6-R06 与 G6/M2 整体完成。

## Go/No-Go

Go（G6-R06 错误追踪与告警）：代码侧实现与完整远端门禁通过，关闭 G6-R06 中「错误追踪与告警」能力（错误追踪由 `RecoveryMiddleware` + `PanicReporter` 提供，告警由 `ALERT_WEBHOOK_URL` webhook 或结构化日志降级承载）。G6-R06 的指标已由 G6.5 闭合；G6-R01 环境隔离/安全存储、G6-R04 的 HTTPS/域名、G6-R07 readiness 依赖探测完善仍未完成，G6 完成门槛的真实告警演练仍缺 staging 证据；G6/M2 整体仍为 No-Go；在无 Critical/High 漏洞、staging 连续运行 ≥7 天、5 倍峰值压测 30 分钟、真实备份恢复/应用回滚/迁移失败/告警演练与 Legal/隐私流程按实际经营地区确认完成前，不应启动支付开发或宣称正式生产就绪。

# G6-R07 readiness 依赖探测完善

## 本切片范围

G6 生产上线准备的可观测性切片，完善 `readyz` 探针，使依赖异常时返回可操作状态：

- **G6-R07**：liveness 与 readiness 分离，依赖异常时返回可操作状态（liveness/readiness 分离基础已由 G6.1 切片实现，本切片把 readiness 扩展为同时探测数据库连接与 Schema 迁移版本）。

本切片不涉及前端功能调整（Schema 保持 v15），交付 readiness 的 schema 版本探测；「staging 连续稳定运行」与真实的依赖故障响应演练仍归 G6 完成门槛，本切片不宣称关闭 G6-R07 的全部操作层面或完成 G6 全部门禁。

## 需求追溯

| Requirement | Test ID | 自动化验收 |
|---|---|---|
| G6-R07 | READINESS-SCHEMA-001、READINESS-SCHEMA-002、SCHVERSION-CURRENT-001、SCHVERSION-CLOSED-001 | `Store.SchemaVersion()` 读取 `schema_migrations` 的 `MAX(version)`（空库回退 0，关闭后返回错误）；`ReadinessHandler` 在数据库连通且 Schema 为当前版本（`current`）时返回 200 与 `schema_version`，在数据库断开（`disconnected`）或 Schema 落后（`pending_migration`）/不可知（`unknown`）时返回 503 `SERVICE_UNAVAILABLE`，并在响应中保留可操作 `Data`（db、schema、schema_version） |

## 变更清单（Commit 14b25af）

- `internal/store/store.go`：新增 `SchemaVersion() (int, error)`——`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`，库未初始化/查询失败返回错误。
- `internal/store/migration_test.go`：新增 `TestSchemaVersionReturnsCurrentVersion`、`TestSchemaVersionAfterCloseReturnsError`。
- `internal/handler/handler.go`：`ReadinessHandler` 扩展——探测 `SchemaVersion`，`schemaStatus` 可为 `current`/`pending_migration`/`unknown`；`dbStatus == "disconnected" || schemaStatus != "current"` 时返回 503 + `CodeServiceUnavailable`。
- `internal/handler/dto/types.go`：`ReadinessResponse` 增加 `schema`、`schema_version` 字段。
- `internal/handler/handler_test.go`：`TestReadinessHandlerHealthy` 新增 `schema=schema_version` 断言；`TestReadinessHandlerUnhealthy` 新增 `schema=unknown`、`schema_version=0` 断言。

## 本地门禁

| 门禁 | 结果 |
|---|---|
| `gofmt -l ./cmd ./internal` | 通过，无待格式文件 |
| `go build ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go test -count=1 ./...` | 通过（387 个顶层测试） |
| `git diff --check` | 通过 |

本阶段不使用 covdata，也不以覆盖率数字替代需求追溯、迁移测试和安全门禁。

## 远端 CI

| 证据 | 结果 |
|---|---|
| [G6-R07 readiness 依赖探测实现 Commit 14b25af](https://github.com/qw2261/soulmarker/commit/14b25af) | `Store.SchemaVersion` + `ReadinessHandler` schema 状态探测 + `ReadinessResponse` schema/schema_version 字段 |
| [GitHub Actions Run 32539068059](https://github.com/qw2261/soulmarker/actions/runs/32539068059) | success，与 Commit 14b25af 精确绑定 |
| [backend job 96945276432](https://github.com/qw2261/soulmarker/actions/runs/32539068059/job/96945276432) | format、build、vet、vulnerability scan、gitleaks、Go test（含新增 store/handler 用例）、race test 全部 success |
| [frontend job 96945276196](https://github.com/qw2261/soulmarker/actions/runs/32539068059/job/96945276196) | install、build、unit/component tests、Browser E2E 与浏览器证据上传 success |
| [docker job 96945276347](https://github.com/qw2261/soulmarker/actions/runs/32539068059/job/96945276347) | 镜像构建、非 root 断言、smoke 探针与 `/version` Commit 一致断言全部 success |

## 说明

- `SchemaVersion()` 对空库回退 0，使 readiness 在未迁移库上呈现 `pending_migration`/`unknown` 而非 panic；关闭后的库返回错误，readiness 据此报 `schema=unknown` 并不可就绪。
- readiness 以「数据库连通 + Schema 为当前版本」双条件判定，避免应用在迁移未完成的实例上提前接收流量；响应保留 db/schema/schema_version 供运维定位具体依赖。
- `GET /healthz` 仅反映进程存活，`GET /readyz` 反映依赖就绪——两者保持分离，readiness 探测在上游依赖异常时返回 503 + 可操作 `Data`，符合 `ErrorCode=SERVICE_UNAVAILABLE` 约定。

## Go/No-Go

Go（G6-R07 readiness 依赖探测完善）：代码侧实现与完整远端门禁通过，关闭 G6-R07 中「依赖异常时返回可操作状态」的能力（readiness 现已同时探测数据库连接与 Schema 迁移版本）。G6-R01 环境隔离/安全存储、G6-R04 的 HTTPS/域名仍未完成，G6 完成门槛的「staging 连续稳定运行 ≥7 天」与真实依赖故障响应演练仍缺 staging 证据；G6/M2 整体仍为 No-Go；在无 Critical/High 漏洞、staging 连续运行 ≥7 天、5 倍峰值压测 30 分钟、真实备份恢复/应用回滚/迁移失败/告警演练与 Legal/隐私流程按实际经营地区确认完成前，不应启动支付开发或宣称正式生产就绪。

# G6-R01 环境隔离与安全存储管理（密钥文件挂载）

## 本切片范围

G6 生产上线准备的「环境隔离与安全存储」切片，支撑 M2「可上线」的 staging/production 环境隔离与配置/密钥由安全存储管理：

- **G6-R01**：staging、production 环境隔离，配置和密钥由安全存储管理。

本切片交付其中的**密钥安全存储代码侧能力**：除环境变量外，支持从安全存储以文件方式挂载读取密钥（Docker/K8s Secret 常以文件挂载，如 `/run/secrets/<name>`），并保持 fail-closed（文件读取失败则拒绝启动）。本切片不涉及数据库结构变更（Schema 保持 v15）、不涉及前端功能调整，也不宣称关闭 G6-R01 的全部操作层面或完成 G6 全部门禁——staging/production 真实隔离环境、部署编排与安全存储（Secret 注入）仍归真实基础设施交付与完成门槛。

## 需求追溯

| Requirement | Test ID | 自动化验收 |
|---|---|---|
| G6-R01（密钥安全存储） | SECRET-FILE-001…004 | `ADMIN_TOKEN`/`JWT_SECRET`/`SMTP_PASSWORD` 支持以 `<KEY>_FILE` 指向的文件读取密钥内容（优先于默认值），并去除尾部 CRLF；环境变量 `<KEY>` 优先于 `<KEY>_FILE`；文件读取失败记录到 `fileReadErrors`，`Validate()` 检测到任何读取错误即返回含对应 `<KEY>_FILE` 的错误并拒绝启动（fail-closed） |

## 变更清单（Commit 1ad0984）

- `internal/config/config.go`：`Config` 新增 `fileReadErrors []string`；`Load()` 中 `AdminToken`/`JWTSecret`/`SMTPPassword` 三密钥改用 `getSecret()` 读取；`Validate()` 起始新增文件读取错误即拒绝启动；新增 `getSecret()` —— 优先级「环境变量 `<KEY>` > 文件 `<KEY>_FILE` > 默认值」，文件读取失败记入 `fileReadErrors`，读取内容用 `strings.TrimRight(data, "\r\n")` 去除尾部 CRLF。
- `internal/config/config_test.go`：新增 4 个测试——`TestLoadReadsSecretFromFile`（JWT 文件读取去掉末尾 `\n`）、`TestLoadSecretFileSupportsAdminAndSMTP`（ADMIN_TOKEN 与带 `\r\n` 的 SMTP_PASSWORD 文件读取，验证 `\r\n` 均去除）、`TestLoadSecretFileEnvVarPrecedence`（环境变量优先于文件）、`TestLoadRejectsUnreadableSecretFile`（不存在的文件使 `Validate()` 失败并含 `ADMIN_TOKEN_FILE`）。

## 本地门禁

| 门禁 | 结果 |
|---|---|
| `gofmt -l .` | 通过，无待格式文件 |
| `go build ./...` | 通过 |
| `go vet ./...` | 通过 |
| `go test -count=1 ./...` | 通过（391 个顶层测试，含 `internal/config` 新增 4 例） |
| `git diff --check` | 通过 |

本阶段不使用 covdata，也不以覆盖率数字替代需求追溯、迁移测试和安全门禁。

## 远端 CI

| 证据 | 结果 |
|---|---|
| [G6-R01 密钥文件挂载实现 Commit 1ad0984](https://github.com/qw2261/soulmarker/commit/1ad0984) | `getSecret` 优先环境变量、其次 `<KEY>_FILE` 文件挂载、最后默认值；文件读取失败 fail-closed；`\r\n` 去除修正 |
| [GitHub Actions Run 32540823076](https://github.com/qw2261/soulmarker/actions/runs/32540823076) | success，与 Commit 1ad0984 精确绑定 |
| [backend job 96950363363](https://github.com/qw2261/soulmarker/actions/runs/32540823076/job/96950363363) | format、build、vet、vulnerability scan、gitleaks、Go test（含新增 config 用例）、race test 全部 success |
| [frontend job 96950363272](https://github.com/qw2261/soulmarker/actions/runs/32540823076/job/96950363272) | install、build、unit/component tests、Browser E2E 与浏览器证据上传 success |
| [docker job 96950363208](https://github.com/qw2261/soulmarker/actions/runs/32540823076/job/96950363208) | 镜像构建、非 root 断言、smoke 探针与 `/version` Commit 一致断言全部 success |

## 说明与未包含项

- `getSecret` 的三类密钥（`ADMIN_TOKEN`/`JWT_SECRET`/`SMTP_PASSWORD`）统一读取路径，使 Docker/K8s Secret 可以文件方式挂载并由应用读取，密钥正文不进入环境变量/镜像层；`Validate()` 在任何密钥文件读取失败时拒绝启动，实现 fail-closed。
- 读取顺序与安全语义：环境变量 `<KEY>` 仍最优先，满足显式注入与向后兼容；`<KEY>_FILE` 提供文件挂载通道；两者均未提供时才回退默认值（development 的 `DefaultJWTSecret`）。
- 本切片只交付「密钥可由安全存储文件挂载读取 + fail-closed」的代码侧能力。G6-R01 的 staging/production 真实隔离环境、部署编排、安全存储 Secret 实际注入、HTTPS/域名与部署归档仍归真实基础设施交付、G6-R04 的 HTTPS/域名与 G6 完成门槛，不能据此宣称 G6-R01 或 G6/M2 整体完成。
- **部署编排与 Secret 注入方案已就绪**：详见 [运维手册](../runbooks/operations.md) 第 1 节——推荐以 `<KEY>_FILE` 只读挂载 Secret 文件（Docker `--read-only` 绑定挂载 / K8s `secret` 卷），密钥正文不进入环境变量或镜像层；部署时登记 Commit、镜像 Tag（`$(git rev-parse --short HEAD)`）、Schema 版本、Secret 注入方式与时间，作为「发布制品、Commit、测试报告与部署记录互相追溯」的凭证。该方案就绪，待真实 staging 环境执行。

## Go/No-Go

Go（G6-R01 密钥安全存储）：代码侧实现与完整远端门禁通过，交付 G6-R01 中「配置和密钥由安全存储管理」的密钥文件挂载与 fail-closed 能力。G6-R01 的 staging/production 环境真实隔离、部署编排与 Secret 注入、G6-R04 的 HTTPS/域名仍未完成，G6 完成门槛的真实备份恢复/应用回滚/迁移失败/告警演练、staging 连续运行 ≥7 天、5 倍峰值压测 30 分钟与 Legal/隐私流程按实际经营地区确认仍缺真实 staging 证据；G6/M2 整体仍为 No-Go；在各项完成门槛满足前，不应启动支付开发或宣称正式生产就绪。
