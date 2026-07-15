# v6.0.0 测试报告

> 状态：In Progress，本地候选；远端 CI 待提交后回填

## 版本身份

| 字段 | 值 |
|---|---|
| 已验证基线 Commit | a80c45c274edcb1c2d4e5dc890a20585e46bfdaa |
| 当前候选 Commit | 待提交与远端验证 |
| 候选分支 | origin/codex/update_project |
| Schema | v6，新增 admissions/checkins 与不可变触发器 |

## 追溯范围

| Requirement | Test ID | 自动化/验收 |
|---|---|---|
| G4-R02 | CT-WEB-ADMISSION-001 | AdmissionCredential 组件状态/二维码/复制测试；Playwright “我的活动”状态与二维码 |
| G4-R07 | DOM-ADMISSION-001、MIG-ADMISSION-001、SEC-CREDENTIAL-001 | 免费/付费签发、v5→v6 数据保留、迁移外键/触发器、随机凭证、可信用户作用域与 HTTP 旅程 |
| G4-R08 | CONC-CHECKIN-001、SEC-CHECKIN-001、IT-CHECKIN-001 | 首次/重复/并发核销、不可变触发器、吊销、跨活动、已核销取消、审计列表 |
| G4 browser gate | E2E-ADMISSION-001 | desktop-chromium 与 Pixel 7：注册、浏览、报名、凭证、首次/重复核销、审计、已入场状态 |

## 当前本地结果

| 门禁 | 结果 |
|---|---|
| 顶层 Go Test 数量 | 248 |
| Vue component tests | 2 passed |
| Playwright E2E | 2 passed（desktop-chromium、mobile-chromium） |
| go test -count=1 ./... | 通过 |
| go test -race -count=1 ./... | 通过 |
| go vet ./... | 通过 |
| OpenAPI JSON / 路由 / DTO / error_code 契约 | 通过 |
| npm run build | 通过；主包约 1.03 MB，保留既有 chunk 告警 |
| npm audit --omit=dev | 0 vulnerabilities |
| Markdown 本地链接 / git diff --check / gofmt | 通过 |
| Browser 可视验收 | 桌面/412x915 用户凭证、移动菜单、运营核销布局通过；二维码 184x184 非空 |

本阶段继续不使用 covdata。

## Go/No-Go

No-Go：R02、R07、R08 本地切片成立，但 G4 其余 Requirements、完整 E2E 和两场受控测试活动尚未完成。
