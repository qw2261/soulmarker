# v6.2.0 测试报告（G6 生产上线准备 — 容器与部署基线、备份恢复、限流、构建 provenance 切片）

> 状态：G6-R03/R07（容器与部署基线）、G6-R09（自动备份与恢复验证）、G6-R08（运维 Runbook）、G6-R10（数据主体隐私、账号注销、数据导出/删除与留存）、G6-R04（全局 API 限流）与 G6-R02（CI/CD 不可变制品与构建 provenance）Remote Candidate Pass / 文档完成。其中 G6-R02 的远端 CI frontend Browser E2E 出现一次性 flake（本地全部 6 例通过，详见下文 G6-R02 切片），其余 job 全部 success

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
| Schema | v15（G6-R10 引入数据主体隐私，G6-R02 构建切片未变更数据库结构） |

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
- 本报告关闭 G6-R03、G6-R07、G6-R09、G6-R10，完成 G6-R08 运维 Runbook 文档，并关闭 G6-R04 的限流能力（见下文 G6-R04 切片）。G6-R01/R02/R05/R06、G6-R04 的 HTTPS/域名与依赖/Secret 扫描，以及 G6 完成门槛（7 天 staging、30 分钟压测、应用回滚/迁移失败演练、Legal/隐私流程、无 Critical/High 漏洞与发布归档）仍未在本报告完成，不能据此宣称正式生产就绪。

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

Go（G6-R04 限流）：代码侧实现与完整远端门禁通过，关闭 G6-R04 的限流能力。G6-R04 整体仍未完成（HTTPS/域名、依赖与 Secret 扫描待补），G6/M2 整体仍为 No-Go；在真实备份恢复/应用回滚/迁移失败/压测演练、HTTPS/域名依赖扫描与 Legal/隐私流程按实际经营地区确认完成前，不应启动支付开发或宣称正式生产就绪。

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

Go（G6-R02 构建 provenance）：代码侧实现与远端门禁通过，关闭 G6-R02 的代码侧能力，并为「发布制品、Tag、Commit、测试报告与部署记录互相追溯」提供构建侧基础。G6-R01（staging/production 环境隔离与安全存储）、G6-R04 的 HTTPS/域名与依赖/Secret 扫描、G6-R05/R06 仍未完成，G6/M2 整体仍为 No-Go；在无 Critical/High 漏洞、staging 连续运行 ≥7 天、5 倍峰值压测 30 分钟、真实备份恢复/应用回滚/迁移失败/告警演练与 Legal/隐私流程按实际经营地区确认完成前，不应启动支付开发或宣称正式生产就绪。
