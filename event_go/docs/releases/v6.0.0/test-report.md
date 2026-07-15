# v6.0.0 测试报告

> 状态：In Progress，G4-R02 统一时间线加固已通过本地门禁，待新候选远端回填

## 版本身份

| 字段 | 值 |
|---|---|
| 已验证基线 Commit | a80c45c274edcb1c2d4e5dc890a20585e46bfdaa |
| 当前候选 Commit | 待提交与远端验证；上一候选 674fd623513f1504c6325ed985192630d3a57c16 |
| 候选分支 | origin/codex/update_project |
| Schema | v6，新增 admissions/checkins 与不可变触发器 |

## 追溯范围

| Requirement | Test ID | 自动化/验收 |
|---|---|---|
| G4-R02 | DOM-ACTIVITY-001、CT-WEB-ACTIVITY-001、CT-WEB-ADMISSION-001 | 23 条免费/付费/已取消混合历史跨 3 页精确总数、稳定排序、无重复/遗漏；单 API 前端分页；二维码状态；Playwright “我的活动” |
| G4-R07 | DOM-ADMISSION-001、MIG-ADMISSION-001、SEC-CREDENTIAL-001 | 免费/付费签发、v5→v6 数据保留、迁移外键/触发器、随机凭证、可信用户作用域与 HTTP 旅程 |
| G4-R08 | CONC-CHECKIN-001、SEC-CHECKIN-001、IT-CHECKIN-001 | 首次/重复/并发核销、不可变触发器、吊销、跨活动、已核销取消、审计列表 |
| G4 browser gate | E2E-ADMISSION-001 | desktop-chromium 与 Pixel 7：注册、浏览、报名、凭证、首次/重复核销、审计、已入场状态 |

## 当前本地结果

| 门禁 | 结果 |
|---|---|
| 顶层 Go Test 数量 | 250 |
| Vue component tests | 3 passed |
| Playwright E2E | 2 passed（desktop-chromium、mobile-chromium） |
| go test -count=1 ./... | 通过 |
| go test -race -count=1 ./... | 通过 |
| go vet ./... | 通过 |
| OpenAPI JSON / 路由 / DTO / error_code 契约 | 通过 |
| npm run build | 通过；主 JS 约 479 KB / 167 KB gzip，无 chunk size 告警 |
| npm audit --omit=dev | 0 vulnerabilities |
| Markdown 本地链接 / git diff --check / gofmt | 通过 |
| Browser 可视验收 | 桌面/412x915 用户凭证、移动菜单、运营核销布局通过；二维码 184x184 非空 |

本阶段继续不使用 covdata。

## 远端 CI

| 证据 | 结果 |
|---|---|
| [GitHub Actions Run 29386146688](https://github.com/qw2261/soulmarker/actions/runs/29386146688) | Commit 674fd623，success |
| [backend job 87259740454](https://github.com/qw2261/soulmarker/actions/runs/29386146688/job/87259740454) | format、vet、test、race success |
| [frontend job 87259740437](https://github.com/qw2261/soulmarker/actions/runs/29386146688/job/87259740437) | Node 22 build、component、desktop/mobile Playwright success；浏览器证据已上传 |

## Go/No-Go

No-Go：R02 统一时间线加固及 R07、R08 纵向切片已通过本地门禁；新候选仍待远端验证，且 G4 其余 Requirements、完整 E2E 和两场受控测试活动尚未完成。
