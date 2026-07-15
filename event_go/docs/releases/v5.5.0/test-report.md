# v5.5.0 测试报告

> 状态：In Progress，G3-R01–R03、R05–R06 的当前实现均已通过本地和远端门禁

## 版本身份

| 字段 | 值 |
|---|---|
| 已验证基线 Commit | d40d24f61cd91208b31dbff120b1c95baaeffefd |
| 当前候选 Commit | 08fa3011397169de37b0d351ac7475e85ebc48e0 |
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

## 本地结果

| 门禁 | 结果 |
|---|---|
| 顶层 Go Test 数量 | 220 |
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

## Go/No-Go

No-Go：DiscussionService 切片已完成验证；G3 仍有 DTO、OpenAPI 和业务错误码工作。
