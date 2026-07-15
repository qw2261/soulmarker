# v6.0.0 测试报告

> 状态：In Progress，G4-R06 免费活动运营闭环已通过本地与远端候选门禁

## 版本身份

| 字段 | 值 |
|---|---|
| 已验证基线 Commit | a80c45c274edcb1c2d4e5dc890a20585e46bfdaa |
| 当前候选 Commit | 4b753eec78053855e0b41e7708926eb4a76273bf |
| 候选分支 | origin/codex/update_project |
| Schema | v7，新增认证版本、密码重置 Token 表与触发器；保留 v6 admissions/checkins |

## 追溯范围

| Requirement | Test ID | 自动化/验收 |
|---|---|---|
| G4-R02 | DOM-ACTIVITY-001、CT-WEB-ACTIVITY-001、CT-WEB-ADMISSION-001 | 23 条免费/付费/已取消混合历史跨 3 页精确总数、稳定排序、无重复/遗漏；单 API 前端分页；二维码状态；Playwright “我的活动” |
| G4-R07 | DOM-ADMISSION-001、MIG-ADMISSION-001、SEC-CREDENTIAL-001 | 免费/付费签发、v5→v6 数据保留、迁移外键/触发器、随机凭证、可信用户作用域与 HTTP 旅程 |
| G4-R08 | CONC-CHECKIN-001、SEC-CHECKIN-001、IT-CHECKIN-001 | 首次/重复/并发核销、不可变触发器、吊销、跨活动、已核销取消、审计列表 |
| G4 browser gate | E2E-ADMISSION-001 | desktop-chromium 与 Pixel 7：注册、浏览、报名、凭证、首次/重复核销、审计、已入场状态 |
| G4-R03 | SEC-AUTH-SESSION-001、SEC-PASSWORD-RESET-001、MIG-AUTH-V7-001、CT-API-AUTH-001、FE-AUTH-SESSION-001 | 注册邮箱/密码策略；旧 JWT 版本兼容与撤销；一次性重置 Token；v6→v7 迁移；生产配置 fail-closed；迟到 401 不清除新会话 |
| G4-R03 browser | E2E-AUTH-001 | desktop-chromium 与 Pixel 7：无效 JWT 清理、安全回跳重新登录、服务端退出、受保护路由、未知邮箱统一重置成功页 |
| G4-R06 | SEC-ADMIN-SESSION-001、FE-ADMIN-SESSION-001、FE-CSV-001、CT-API-ADMIN-001 | 管理 Token 服务端校验和失效竞态；门店/活动/票种运营页面；CSV 转义、UTF-8 与公式注入防护；OpenAPI 双向契约 |
| G4-R06 browser | E2E-OPERATOR-001 | desktop-chromium 与 Pixel 7：后台守卫/登录、门店增删改、活动创建、票种增删改、用户报名、CSV 下载内容、首次/重复核销、管理退出 |

## 当前本地结果

| 门禁 | 结果 |
|---|---|
| 顶层 Go Test 数量 | 263 |
| Vue unit/component tests | 8 passed（5 files） |
| Playwright E2E | 2 passed（desktop-chromium、mobile-chromium） |
| go test -count=1 ./... | 通过 |
| go test -race -count=1 ./... | 通过 |
| go vet ./... | 通过 |
| OpenAPI JSON / 路由 / DTO / error_code 契约 | 通过 |
| npm run build | 通过；主 JS 约 490 KB / 170 KB gzip，无 chunk size 告警 |
| npm audit --omit=dev | 0 vulnerabilities |
| Markdown 本地链接 / git diff --check / gofmt | 通过 |
| Browser 可视验收 | 桌面/Pixel 7 门店、活动、票种、报名导出与核销全流程通过；移动表格局部滚动；二维码 184x184 非空 |

本阶段继续不使用 covdata。

## 远端 CI

| 证据 | 结果 |
|---|---|
| [GitHub Actions Run 29386146688](https://github.com/qw2261/soulmarker/actions/runs/29386146688) | Commit 674fd623，success |
| [backend job 87259740454](https://github.com/qw2261/soulmarker/actions/runs/29386146688/job/87259740454) | format、vet、test、race success |
| [frontend job 87259740437](https://github.com/qw2261/soulmarker/actions/runs/29386146688/job/87259740437) | Node 22 build、component、desktop/mobile Playwright success；浏览器证据已上传 |
| [GitHub Actions Run 29387047591](https://github.com/qw2261/soulmarker/actions/runs/29387047591) | Commit df969f2，统一时间线/首包加固候选 success |
| [backend job 87262384999](https://github.com/qw2261/soulmarker/actions/runs/29387047591/job/87262384999) | format、vet、250 tests、race success |
| [frontend job 87262384985](https://github.com/qw2261/soulmarker/actions/runs/29387047591/job/87262384985) | Node 22 build、3 component tests、desktop/mobile Playwright success；浏览器证据已上传 |
| [GitHub Actions Run 29389509516](https://github.com/qw2261/soulmarker/actions/runs/29389509516) | Commit c6fc2cf，认证闭环候选 success |
| [backend job 87269619413](https://github.com/qw2261/soulmarker/actions/runs/29389509516/job/87269619413) | format、vet、262 tests、race success |
| [frontend job 87269619409](https://github.com/qw2261/soulmarker/actions/runs/29389509516/job/87269619409) | Node 22 build、5 unit/component tests、desktop/mobile Playwright success；浏览器证据已上传 |
| [GitHub Actions Run 29391643931](https://github.com/qw2261/soulmarker/actions/runs/29391643931) | Commit 4b753ee，免费活动运营闭环候选 success |
| [backend job 87276252748](https://github.com/qw2261/soulmarker/actions/runs/29391643931/job/87276252748) | format、vet、263 tests、race success |
| [frontend job 87276252755](https://github.com/qw2261/soulmarker/actions/runs/29391643931/job/87276252755) | Node 22 build、8 unit/component tests、desktop/mobile 完整运营 Playwright success；浏览器证据已上传 |

## Go/No-Go

No-Go：R02、R06、R07、R08 已通过本地和远端候选门禁；R03 仍缺 legacy phone-only 恢复和真实 staging SMTP。G4-R01、R04、R05、R09、完整讨论/取消用户旅程和两场受控测试活动尚未完成。
