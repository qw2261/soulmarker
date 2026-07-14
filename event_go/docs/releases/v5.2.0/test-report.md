# v5.2.0 测试报告

> 状态：Verification，本地门禁通过且候选分支已推送，等待远端 CI

## 版本身份

| 字段 | 值 |
|---|---|
| Tag | 待创建 |
| 基线 Commit | b196b3c754139fe20346252ae1cbc38a47dcf389 |
| 候选实现 Commit | bc9db6b（feat: 推进 v5.2 安全基线与版本化迁移） |
| 候选分支 | origin/codex/update_project |
| CI Run | 待运行 |
| Schema | v3，事务化版本迁移；user_id Expand |

## 环境

| Go | Node | npm | OS |
|---|---|---|---|
| 1.25.0 | 20.20.2（目标与 CI 为 22） | 10.8.2 | Darwin 24.6.0 x86_64 |

## 追溯范围

- G0-R04
- G1-R01
- G1-R02
- G1-R03
- G1-R04
- G1-R05
- G1-R06
- G1-R07
- G1-R08
- G1-R09
- G2-R01
- G2-R02（Expand）
- G2-R08（空库、旧库、重复执行、错误返回；备份恢复待演练）

## 本地结果

| 门禁 | 结果 |
|---|---|
| gofmt | 通过，无未格式化文件 |
| go test -count=1 ./... | 通过 |
| go test -race -count=1 ./... | 通过 |
| go vet ./... | 通过 |
| 顶层 Go Test 数量 | 192 |
| npm run build | 通过，保留大包告警 |
| Markdown 本地链接 | 通过 |
| CI YAML 解析 | 通过 |

本阶段按用户决策不使用 covdata，不生成覆盖率产物。

## 关键回归测试

| Test ID | 自动化测试 |
|---|---|
| CT-API-POST-001 | TestGetPostHandlerWithReplies + 前端 TypeScript 构建 |
| SEC-RESOURCE-001 | TestPostRoutesRequireEventOwnership、TestTicketRoutesRequireEventOwnership |
| IT-API-VALIDATION-001 | TestUpdateValidationRejectsInvalidValues |
| SEC-REGISTRATION-PII-001 | TestRegistrationListRequiresAdminToken |
| IT-API-REGSTATUS-001 | TestRegistrationStatusUsesAuthenticatedUser |
| SEC-CONFIG-001 | TestValidateSecureEnvironmentRequiresSecrets、TestValidateProductionConfig |
| SEC-JWT-001 | TestUserAuthWithInvalidToken |
| SEC-HTTP-001 | TestStrictJSONRequestBoundary、TestSecurityHeaders |
| SEC-ERROR-001 | TestInternalErrorsDoNotLeak、TestHealthHandlerDBDisconnected |
| MIG-SCHEMA-001 | TestMigrationEmptyDatabase、TestMigrationRepeatedExecution、TestMigrationFailureIsReturned |
| MIG-IDENTITY-EXPAND-001 | TestMigrationEmptyDatabase |
| MIG-LEGACY-001 | TestMigrationLegacyDatabasePreservesData |

## 未完成证据

- 远端 GitHub Actions 尚未执行。
- 最终发布 Commit 与 Tag 尚未生成；当前候选实现 Commit 为 bc9db6b。
- Docker daemon 当前不可用，Docker smoke 未执行。
- 真实备份恢复演练尚未执行。

## Go/No-Go

No-Go：候选提交已推送且本地验证通过，但必须取得远端 CI 与 Tag 证据后才能结束 G0。
