# v5.5.0 测试报告

> 状态：In Progress，G3-R01–R06 当前已实现范围均已通过本地和远端门禁

## 版本身份

| 字段 | 值 |
|---|---|
| 已验证基线 Commit | b91b8dbeb84f600950ccd09cba855a581b4e5ae6 |
| 当前候选 Commit | e0605014a3b076db5920a843f66d145a7d88c998 |
| 候选分支 | origin/codex/update_project |
| Schema | v5，无数据库迁移 |

## 追溯范围

| Requirement | Test ID | 自动化测试 |
|---|---|---|
| G3-R01 | ARCH-CONFIG-001 | TestHandlerUsesStartupConfigSnapshot、既有 Config.Validate 与认证测试 |
| G3-R02 | ARCH-STORE-001 | TestNewStoreReturnsInitializationError、既有迁移失败返回测试 |
| G3-R03 | ARCH-SERVICE-001 | RegistrationService 与 DiscussionService 正常、拒绝、边界、故障测试；既有 Handler 契约回归 |
| G3-R05 | ARCH-REPO-001 | fakeRegistrationRepository 编译期契约与 service 正常/拒绝/故障测试 |
| G3-R06 | ARCH-DEPS-001 | TestJWTManagerSignAndVerifyUser、TestGenerateTokenUsesInjectedClockAndSigner、取消截止确定时间测试 |
| G3-R04 | CT-DTO-001 | TestResponseEnvelopeContract、TestUserAndLoginResponseDoNotExposePassword、TestEntityMappersFixPublicFieldSets、TestPostDetailUsesStableEmptyArray、既有 Handler 契约回归 |

## 本地结果

| 门禁 | 结果 |
|---|---|
| 顶层 Go Test 数量 | 224 |
| go test -count=1 ./... | 通过 |
| go test -race -count=1 ./... | 通过 |
| go vet ./... | 通过 |
| gofmt / diff check | 通过 |
| npm run build | 通过；主包约 1.02 MB，保留 Vite chunk size 告警 |
| Markdown 本地链接 | 通过，排除被忽略的 node_modules/test_reports 原始产物 |

本阶段继续不使用 covdata。

## 远端 CI

| 证据 | 结果 |
|---|---|
| [GitHub Actions Run 29378239877](https://github.com/qw2261/soulmarker/actions/runs/29378239877) | success |
| backend：format、vet、test、race | success |
| frontend：build | success |
| [GitHub Actions Run 29379040895](https://github.com/qw2261/soulmarker/actions/runs/29379040895) | service/依赖注入候选 success |
| [GitHub Actions Run 29379725406](https://github.com/qw2261/soulmarker/actions/runs/29379725406) | DiscussionService 候选 success |
| [GitHub Actions Run 29380643861](https://github.com/qw2261/soulmarker/actions/runs/29380643861) | HTTP DTO 候选 success |

## Go/No-Go

No-Go：R04 DTO 切片已完成验证；G3 仍有 OpenAPI 和业务错误码工作。
