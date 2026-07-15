# v5.5.0 测试报告

> 状态：Verification，G3-R01–R08 已通过本地和远端门禁

## 版本身份

| 字段 | 值 |
|---|---|
| 已验证基线 Commit | f1b7227846cb6aa48e4e234523d77460b588999a |
| 当前候选 Commit | 48f91a3135b662f46a91acfbed21ffb75ae5e4a5 |
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
| G3-R07 | CT-OPENAPI-001 | TestOpenAPIRoutesMatchRouterCatalog、TestOpenAPISchemasMatchDTOJSONFields、TestVersionedAndLegacyRoutesAreEquivalent、TestOpenAPISpecIsServedFromVersionedAPI |
| G3-R08 | CT-ERROR-CODE-001、SEC-ERROR-002 | TestErrorCodeCatalog、TestNewErrorResponsePreservesLegacyNumericCode、TestNewErrorResponseRejectsUnknownCode、TestSameHTTPStatusUsesDistinctBusinessErrorCodes、TestAPIFallbackReturnsStructuredErrors、TestErrorResponseFormat、TestInternalErrorsDoNotLeak、TestOpenAPIErrorCodesMatchCatalog |

## 本地结果

| 门禁 | 结果 |
|---|---|
| 顶层 Go Test 数量 | 234 |
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
| [GitHub Actions Run 29381736629](https://github.com/qw2261/soulmarker/actions/runs/29381736629) | API v1/OpenAPI 候选 success |
| [GitHub Actions Run 29382749232](https://github.com/qw2261/soulmarker/actions/runs/29382749232) | 稳定业务错误码候选 success |
| Run 29382749232 backend | format、vet、test、race success |
| Run 29382749232 frontend | build success |

## Go/No-Go

No-Go：G3-R01–R08 与候选 SHA 已通过本地和远端门禁；仍需 v5.5.0 Tag 与正式发布证据后才能标记 Done。
