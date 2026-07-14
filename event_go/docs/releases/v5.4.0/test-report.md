# v5.4.0 测试报告

> 状态：Verification，本地完整门禁已通过，候选提交已推送，等待远端 CI

## 版本身份

| 字段 | 值 |
|---|---|
| 基线 Commit | 212300047252ce2913020d3b776fd25abeb20a50 |
| 候选 Commit | 63ff5b8f69d96314616c3fa882f98c9742a2e373 |
| 候选分支 | origin/codex/update_project |
| Schema | v5，身份回填、legacy 标记、外键启用 |

## 追溯范围

- G2-R03 至 G2-R09
- ADR-001

## 核心证据

| Test ID | 自动化测试 |
|---|---|
| SEC-IDENTITY-001 | TestTrustedUserIDAuthorizationFlow、TestCancelRegistrationRejectsContactBody |
| IT-ME-REG-001 | TestTrustedUserIDAuthorizationFlow、前端 MyRegistrations 构建 |
| MIG-IDENTITY-BACKFILL-001 | TestMigrationBackfillsMatchingIdentityAndMarksLegacy |
| DB-FK-001 | TestSQLiteForeignKeysEnabled、TestMigrationRebuildsEventsWhenOrganizerForeignKeyIsMissing、TestDeleteTicketPreservesRegistrationSnapshot、既有级联删除测试 |
| MIG-LEGACY-001 | TestMigrationEmptyDatabase、TestMigrationLegacyDatabasePreservesData、TestMigrationRepeatedExecution、TestMigrationFailureIsReturned、TestDatabaseBackupRestoreRoundTrip |
| CONC-REG-001 | TestConcurrentRegistrationNeverExceedsCapacity、TestConcurrentTicketRegistrationNeverMakesStockNegative、TestConcurrentCancellationRestoresTicketExactlyOnce |

## 本地结果

| 门禁 | 结果 |
|---|---|
| 顶层 Go Test 数量 | 202 |
| go test -count=1 ./... | 通过 |
| go test -race -count=1 ./... | 通过 |
| go vet ./... | 通过 |
| gofmt / diff check | 通过 |
| npm run build | 通过；主包约 1.02 MB，保留 Vite chunk size 告警 |
| Markdown 本地链接 | 通过 |

本阶段继续不使用 covdata。

## Go/No-Go

No-Go：需要完成远端 CI 和版本发布证据。
