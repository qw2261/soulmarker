# v6.2.0 测试报告（G6 生产上线准备 — 容器与部署基线、备份恢复切片）

> 状态：G6-R03/R07（容器与部署基线）、G6-R09（自动备份与恢复验证）与 G6-R08（运维 Runbook）Remote Candidate Pass / 文档完成；backend、frontend、docker 三 job 全部通过完整远端 CI

## 版本身份

| 字段 | 值 |
|---|---|
| 候选分支 | origin/codex/update_project |
| 上一阶段基线 Commit | 63a72a9（G5.5 验收证据） |
| G6-R03/R07 实现 Commit | db62b83 |
| G6-R03/R07 门禁修复 Commit | ac75417（gofmt 对齐） |
| G6-R09 实现 Commit | e390d99 |
| Schema | v14（本切片未变更数据库结构） |

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
- 本报告关闭 G6-R03、G6-R07、G6-R09，并完成 G6-R08 运维 Runbook 文档。G6-R01/R02/R04/R05/R06/R10 与 G6 完成门槛（7 天 staging、30 分钟压测、应用回滚/迁移失败演练、Legal/隐私流程、无 Critical/High 漏洞与发布归档）未在本报告完成，不能据此宣称正式生产就绪。

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
- 本切片关闭 G6-R09。G6-R01/R02/R04/R05/R06/R10 与 G6 完成门槛中的「备份恢复演练」（真实 staging 持续运行、SLO 实测与压测）仍待完成，不能据此宣称正式生产就绪。

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

## Go/No-Go

Go（G6-R08 运维 Runbook）：文档交付完成。G6/M2 整体仍为 No-Go；在审计、隐私（R10）、应用回滚/迁移失败实际演练与压测门槛完成前，不应启动支付开发或宣称正式生产就绪。
