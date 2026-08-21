# v6.2.0 测试报告（G6 生产上线准备 — 容器与部署基线切片）

> 状态：G6-R03/R07 容器与部署基线 Remote Candidate Pass；backend、frontend、docker 三 job 全部通过完整远端 CI

## 版本身份

| 字段 | 值 |
|---|---|
| 候选分支 | origin/codex/update_project |
| 上一阶段基线 Commit | 63a72a9（G5.5 验收证据） |
| G6-R03/R07 实现 Commit | db62b83 |
| G6-R03/R07 门禁修复 Commit | ac75417（gofmt 对齐） |
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
- 本切片仅关闭 G6-R03、G6-R07。G6-R01/R02/R04/R05/R06/R08/R09/R10 与 G6 完成门槛（7 天 staging、30 分钟压测、备份/回滚/迁移失败演练、Legal/隐私流程、无 Critical/High 漏洞与发布归档）未在本切片完成，不能据此宣称正式生产就绪。

## Go/No-Go

Go（G6-R03/R07 容器与部署基线切片）：Docker 非 root、固定基础镜像、Healthcheck、smoke test 与 liveness/readiness 分离均通过完整远端 CI 并回填证据。G6/M2 整体仍为 No-Go；在审计、隐私、备份恢复、回滚与压测门槛完成前，不应启动支付开发或宣称正式生产就绪。
