# G6 生产运维手册（Runbook）

> **适用范围**：G6-R08 要求的部署、迁移、备份恢复、回滚、支付关闭与故障响应 Runbook。
> **状态**：文档已建立；备份/恢复、应用回滚、迁移失败与告警的实际演练仍需在真实 staging 上执行并回填，见本文「未完成项」与 G6 完成门槛。
> **关联**：[goal.md](../goal.md) / [ADR-001](../adr/001-sqlite-foreign-key-and-deletion-semantics.md) / [ADR-004](../adr/004-organization-tenant-boundary-and-migration.md) / [ADR-006](../adr/006-stable-event-tenant-scope.md) / 各版本 [migration-rollback](../releases/)。

---

## 0. 前置事实（本文依据）

- 数据库：SQLite，启动时自动执行版本化迁移，`schema_migrations` 记录已应用版本；当前 Schema v14。
- 迁移模式：**Expand-only**，创建新表/新列/新索引/兼容触发器，**不删除旧列或旧表**；完全撤销仅靠恢复升级前备份。
- 备份（G6-R09）：`Store.Backup` 基于 `VACUUM INTO` 生成一致快照；备份管理器以只读校验 `integrity_check`/schema/表数量，按 UTC 时间戳保留最近 N 份，定期恢复演练复制最新备份到临时位置校验后清理。
- 运行环境变量（`internal/config/config.go`）：
  - `APP_ENV`（development/test/staging/production）
  - `DATABASE_PATH`（默认 `data/event_go.db`）
  - `PORT`（默认 `8080`）
  - `BACKUP_DIR`（默认 `data/backups`）
  - `BACKUP_INTERVAL_SECONDS`（默认 `0`=禁用）
  - `BACKUP_RETAIN`（默认 `7`；`<=0` 表示保留全部）
  - `BACKUP_DRILL_INTERVAL_SECONDS`（默认 `0`=禁用）
  - `ORGANIZATION_AUTH_ENABLED`（默认 `true`；`false` 关闭所有 `/organizations/...` 租户入口）
- 探针：`GET /healthz`（进程存活，不探测依赖）、`GET /readyz`（依赖就绪，否则 503，`ErrorCode=SERVICE_UNAVAILABLE`）。
- 支付：**尚未实现**（属 G7），本文第 5 节给出计划原则与未来的开关位置，当前不适用。

---

## 1. 部署（Deploy）

### 1.1 构建镜像

在 `event_go/` 目录执行：

```sh
docker build -t event-go:$(git rev-parse --short HEAD) .
```

- 基础镜像固定版本、非 root 用户 `app` 运行、`/app/data` 归属 `app:app`（见 [Dockerfile](../../Dockerfile)）。
- 每次部署记录 Git Commit；不可变制品把 Commit SHA、构建环境与依赖写入制品元数据（G6-R02 相关，当前仓库 CI 已产生后端/前端/Docker 三个 job）。

### 1.2 启动容器（staging / production）

```sh
docker run -d --name event-go \
  -p 8080:8080 \
  -v "$(pwd)/data:/app/data" \
  -e APP_ENV=staging \
  -e DATABASE_PATH=/app/data/event_go.db \
  -e ADMIN_TOKEN="$ADMIN_TOKEN" \
  -e JWT_SECRET="$JWT_SECRET" \
  -e CORS_ORIGIN="$CORS_ORIGIN" \
  -e PUBLIC_BASE_URL="$PUBLIC_BASE_URL" \
  -e SMTP_HOST=... -e SMTP_PORT=... -e SMTP_USERNAME=... -e SMTP_PASSWORD=... -e SMTP_FROM=... \
  -e BACKUP_DIR=/app/data/backups \
  -e BACKUP_INTERVAL_SECONDS=3600 \
  -e BACKUP_RETAIN=7 \
  -e BACKUP_DRILL_INTERVAL_SECONDS=86400 \
  event-go:$(git rev-parse --short HEAD)
```

- staging/production 必须满足 `config.Validate()` 的 fail-closed 校验：非默认 ≥32 字节 `JWT_SECRET`、明确 `CORS_ORIGIN`、HTTPS `PUBLIC_BASE_URL`、完整 SMTP 配置、ADMIN_TOKEN 非空；否则拒绝启动。
- 备份目录需可写且与数据库同卷，确保快照与主库在同一文件系统，便于原子复制。

### 1.3 启动后验证

```sh
curl -fsS http://localhost:8080/healthz     # 进程存活
curl -fsS http://localhost:8080/readyz      # 依赖就绪（否则 503）
```

容器健康检查与 smoke test 由 [smoke.sh](../../scripts/smoke.sh) 与 CI docker job 覆盖：断言运行用户 `uid != 0` 并验证 `/healthz` 与 `/readyz`。

---

## 2. 数据库迁移（Migration）

### 2.1 执行方式

迁移在应用启动时由 `store.OpenStore` → `migrate` 自动执行，按 `migrations()` 顺序，每个迁移在独立事务中应用并写入 `schema_migrations`，失败即回滚该事务并拒绝启动。**无独立迁移 CLI**。

### 2.2 升级步骤

1. 停止写入；记录当前应用 Commit、Schema 版本与 `DATABASE_PATH`。
2. 做一致性备份（推荐先正常停止应用再复制单文件，或用 `VACUUM INTO`，见第 3 节）。
3. 部署新镜像，让其自动应用迁移。
4. 应用后校验：

```sql
SELECT MAX(version) FROM schema_migrations;   -- 期望为最新版本，当前 14
PRAGMA foreign_key_check;                      -- 期望无行
```

具体版本的期望断言与兼容触发器校验见对应 [release migration-rollback](../releases/) 文档。

### 2.3 迁移失败处理

- 应用因迁移失败而启动失败时，进程不会监听端口；直接检查日志定位失败的迁移号与名称。
- 使用**前向修复**：提交修复该迁移的新迁移，而不是手工改库或 DROP。
- 需要立即恢复服务时，回滚到上一兼容制品并恢复升级前备份（见第 4 节）。

---

## 3. 备份与恢复（Backup & Restore，G6-R09）

### 3.1 自动备份

设置 `BACKUP_INTERVAL_SECONDS` 大于 0 即启用。启动时立即执行一次备份，随后按间隔创建一致性快照。脚本：

- `BACKUP_DIR` 默认 `data/backups`。
- 文件命名 `event-go-20060102T150405.db`（UTC 时间戳，同秒冲突追加序号）。
- 每份备份创建后用 `?mode=ro` 打开，执行 `PRAGMA integrity_check` 必须为 `ok`，并读取 schema 版本与表数量。
- 保留策略：按文件名（创建时间）升序，保留最近 `BACKUP_RETAIN` 份，删除更旧文件。

### 3.2 恢复演练（定期）

设置 `BACKUP_DRILL_INTERVAL_SECONDS` 大于 0 即启用。每次把最新备份复制到 `drill-<ts>.db` 临时位置，以可读写方式打开并校验，成功后清理。目的是持续证明「最新备份可恢复且可查询」。

### 3.3 手动备份

```sh
# 直接用 SQLite 生成一致快照（应用运行期安全）
# SQLite 无官方备份 CLI 依赖 Go 驱动，最稳妥为正常停机后复制主文件：
cp /app/data/event_go.db /app/data/backups/manual-$(date -u +%Y%m%dT%H%M%S).db
```

依赖较新 SQLite 能力时，可用 `VACUUM INTO '/app/data/backups/manual-<ts>.db'` 在运行期生成一致快照。

### 3.4 实际恢复（灾难）

1. 停止应用，避免写入。
2. 选择一份已校验的备份（可用恢复演练或 `PRAGMA integrity_check` 验证）。
3. 复制备份为 `DATABASE_PATH` 目标文件；确认目标文件不存在或已移走（避免覆盖冲突）。
4. 重新启动应用，让迁移补齐到当前 Schema 版本。
5. 校验 `readyz`、`SELECT MAX(version) FROM schema_migrations` 与关键业务数据。

> RPO/RTO 目标见 [goal.md](../goal.md) 初始 SLO（Public Beta：RPO ≤ 24h、RTO ≤ 4h）。

---

## 4. 回滚（Rollback）

### 4.1 应用回滚

1. 停止当前候选应用；记录正在使用的数据库与升级前备份路径。
2. 涉及租户授权时，设置 `ORGANIZATION_AUTH_ENABLED=false` 或在入口层关闭租户业务路由，停止 tenant 写流量。
3. 部署上一兼容制品（后端 + 前端必须整体回滚，因为新前端配旧后端会因缺少 `principal_type` 等字段拒绝管理登录，这是预期的 fail-closed）。
4. 复验核心路径：`readyz`、组织/门店/活动/报名/核销的创建读取更新删除。
5. 修复后以前滚方式重新部署；期间新增的租户数据仍保留。

### 4.2 Schema 回滚策略

- G2/G3/G5 迁移均为 **Expand-only**：不删除旧列、旧表、索引或兼容触发器。应用回滚时**不要**在 `schema_migrations` 中回退版本，也不要删除 Organizations/Memberships/Invitations/`events.organization_id`，否则破坏外键与后续前滚证据。
- **完全撤销**仅在确认无任何新写入且业务放弃候选时才允许：停机恢复升级前完整数据库备份。禁止在生产库手工 DROP 表/列/触发器。
- 前置条件：升级前必须存在可恢复的备份，且已执行恢复演练验证（见第 3 节）。

### 4.3 迁移失败回滚

同 4.2：优先前向修复新迁移；无法快速修复时，用 3.4 的实际恢复把库回退到升级前备份，再部署上一兼容制品。

---

## 5. 支付关闭（Payment Shutdown）

> **当前不适用**：订单/支付/退款属 G7，本阶段尚未实现。此处记录设计原则，G7 实现时按此落实。

- 生产真实支付必须可在不重启、不改库的前提下**一键停止新下单**，同时保留订单查询与退款处理能力。
- 计划开关位置：写入 `internal/config` 的环境变量（如 `PAYMENT_ACCEPTING_NEW_ORDERS` 或等价），由 `Order/Create` 入口 fail-closed 拒绝；底层使用集中 Capability 开关并保留只读降级，与 `ORGANIZATION_AUTH_ENABLED` 同模式。
- 支付上线必须满足 [goal.md](../goal.md) G7 资金与合规原则，并在 README、OpenAPI、ADR 中同步。

---

## 6. 故障响应（Fault Response）

### 6.1 探针与健康判断

| 探针 | 路径 | 语义 | 失败时动作 |
|---|---|---|---|
| Liveness | `GET /healthz` | 仅进程存活，不探测依赖 | 进程未响应 → 重启容器 |
| Readiness | `GET /readyz` | 依赖就绪，否则 503 + `SERVICE_UNAVAILABLE` | 503 → 停止引流，排查数据库/依赖 |

容器已配置 `HEALTHCHECK`（`/healthz`）。

### 6.2 日志

- 默认结构化 `json` 日志（`LOG_FORMAT`），级别 `LOG_LEVEL`。
- 备份/恢复演练失败通过 `log.Printf("备份/恢复演练失败: %v", err)` 输出。
- 未知错误统一走稳定 `error_code`，不向客户端泄漏内部细节。

### 6.3 常见故障与处置

| 症状 | 可能原因 | 处置 |
|---|---|---|
| 进程启动即退出 | `config.Validate()` 失败（staging/prod 缺密钥/错误配置） | 按日志补齐 `ADMIN_TOKEN`/`JWT_SECRET`/`CORS_ORIGIN`/`PUBLIC_BASE_URL`/SMTP |
| `readyz` 返回 503 | 数据库未就绪或迁移失败 | 检查迁移日志、磁盘、文件权限；就绪前不引流 |
| 备份失败 | `BACKUP_DIR` 不可写 / 磁盘满 | 检查目录权限与磁盘空间 |
| 恢复演练失败 | 备份缺失或校验异常 | 检查 `List()` 元数据，重建一份可用备份 |
| 迁移失败 | 迁移不适用当前库 | 前向修复新迁移；必要时恢复升级前备份 |
| 租户隔离异常 | 授权策略缺陷 | 设 `ORGANIZATION_AUTH_ENABLED=false` 关闭租户入口，仅保留 platform 应急运营，再修复 |

### 6.4 告警与指标

当前**尚未实现** metrics/错误追踪/告警（G6-R06 未完成）。本手册的故障响应以手动探针 + 日志为主；G6-R06 完成后补充告警规则与阈值，并回填到「未完成项」。

---

## 7. 未完成项与验证要求

下列项属于 G6 完成门槛，需在**真实 staging** 上执行并有证据后方可宣称达到 M2：

- [ ] 备份恢复、应用回滚、迁移失败、告警的**实际演练**通过并记录执行人、时间、备份路径与判定。
- [ ] staging 连续稳定运行至少 7 天。
- [ ] 5 倍预测峰值压测 30 分钟，错误率满足阶段 SLO。
- [ ] 无 Critical/High 安全漏洞。
- [ ] 发布制品、Tag、Commit、测试报告与部署记录可互相追溯。
- [ ] 法律文本、隐私同意、投诉与数据主体请求流程确认（G6-R10，含审计/隐私/账号注销/数据导出删除）。

> 本文档是 G6-R08 的交付物之一；其「回滚」部分与 [/goal] 中「备份和回滚完成前不建议启动支付开发或宣称正式生产就绪」的约束直接对齐。
