# v5.5.0 测试报告

> 状态：In Progress，G3-R01/R02 本地与远端门禁已通过

## 版本身份

| 字段 | 值 |
|---|---|
| 基线 Commit | b4e97a8f6b20dfaa85bc3f1cdfea2c2242076a7d |
| 候选 Commit | fb6e0075925ca126086f6872ea3f68740f1ef550 |
| 候选分支 | origin/codex/update_project |
| Schema | v5，无数据库迁移 |

## 追溯范围

| Requirement | Test ID | 自动化测试 |
|---|---|---|
| G3-R01 | ARCH-CONFIG-001 | TestHandlerUsesStartupConfigSnapshot、既有 Config.Validate 与认证测试 |
| G3-R02 | ARCH-STORE-001 | TestNewStoreReturnsInitializationError、既有迁移失败返回测试 |

## 本地结果

| 门禁 | 结果 |
|---|---|
| 顶层 Go Test 数量 | 204 |
| go test -count=1 ./... | 通过 |
| go test -race -count=1 ./... | 通过 |
| go vet ./... | 通过 |
| gofmt / diff check | 通过 |
| npm run build | 通过；主包约 1.02 MB，保留 Vite chunk size 告警 |

本阶段继续不使用 covdata。

## 远端 CI

| 证据 | 结果 |
|---|---|
| [GitHub Actions Run 29378239877](https://github.com/qw2261/soulmarker/actions/runs/29378239877) | success |
| backend：format、vet、test、race | success |
| frontend：build | success |

## Go/No-Go

No-Go：G3 尚有 R03–R08；R01/R02 已完成本地与远端验证。
