# v6.0.0 测试报告

> 状态：In Progress，JWT 与 Go 标准库漏洞修复、P0/P1 清零审计为本地候选

## 版本身份

| 字段 | 值 |
|---|---|
| 已验证基线 Commit | a80c45c274edcb1c2d4e5dc890a20585e46bfdaa |
| 当前候选 Commit | 待 JWT 依赖安全修复提交后回填 |
| 候选分支 | origin/codex/update_project |
| Schema | v11，新增持久化站内通知、未读/活动索引和幂等键；保留 v10 恢复邮箱、v9 内容治理、v8 活动封面、v7 认证与 v6 admissions/checkins |

## 追溯范围

| Requirement | Test ID | 自动化/验收 |
|---|---|---|
| G4-R02 | DOM-ACTIVITY-001、CT-WEB-ACTIVITY-001、CT-WEB-ADMISSION-001 | 23 条免费/付费/已取消混合历史跨 3 页精确总数、稳定排序、无重复/遗漏；单 API 前端分页；二维码状态；Playwright “我的活动” |
| G4-R07 | DOM-ADMISSION-001、MIG-ADMISSION-001、SEC-CREDENTIAL-001 | 免费/付费签发、v5→v6 数据保留、迁移外键/触发器、随机凭证、可信用户作用域与 HTTP 旅程 |
| G4-R08 | CONC-CHECKIN-001、SEC-CHECKIN-001、IT-CHECKIN-001 | 首次/重复/并发核销、不可变触发器、吊销、跨活动、已核销取消、审计列表 |
| G4 browser gate | E2E-ADMISSION-001 | desktop-chromium 与 Pixel 7：注册、浏览、报名、凭证、首次/重复核销、审计、已入场状态 |
| G4-R03 | SEC-AUTH-SESSION-001、SEC-PASSWORD-RESET-001、MIG-AUTH-V7-001、MIG-RECOVERY-EMAIL-001、SEC-RECOVERY-EMAIL-001、CT-API-AUTH-001、CT-API-RECOVERY-EMAIL-001、FE-AUTH-SESSION-001、FE-RECOVERY-EMAIL-001 | 注册邮箱/密码策略；旧 JWT 版本兼容与撤销；一次性重置 Token；v6→v7 与 v9→v10 迁移；phone-only 登录标识保留；当前密码 + 邮箱链接绑定；确认后撤销旧会话/重置令牌；生产配置 fail-closed |
| G4-R03 browser | E2E-AUTH-001 | desktop-chromium 与 Pixel 7：无效 JWT 清理、安全回跳重新登录、服务端退出、受保护路由、未知邮箱统一重置成功页 |
| G4-R03 recovery browser | E2E-RECOVERY-EMAIL-001 | desktop-chromium 与 Pixel 7：账户安全入口、登录邮箱“待验证”状态和无横向溢出；phone-only 申请/确认/撤销/重置由 HTTP 集成旅程覆盖 |
| G4-R06 | SEC-ADMIN-SESSION-001、FE-ADMIN-SESSION-001、FE-CSV-001、CT-API-ADMIN-001 | 管理 Token 服务端校验和失效竞态；门店/活动/票种运营页面；CSV 转义、UTF-8 与公式注入防护；OpenAPI 双向契约 |
| G4-R06 browser | E2E-OPERATOR-001 | desktop-chromium 与 Pixel 7：后台守卫/登录、门店增删改、活动创建、票种增删改、用户报名、CSV 下载内容、首次/重复核销、管理退出 |
| G4-R01 | FE-RESPONSIVE-001、E2E-ATTENDEE-001 | 活动列表、封面详情、报名、二维码凭证、发帖、回复、取消、“我的活动”取消/已入场状态；关键用户页断言无页面级横向溢出 |
| G4-R04 | MIG-EVENT-COVER-001、CT-EVENT-COVER-001、FE-REQUEST-STATE-001、E2E-UX-STATE-001 | v7→v8 保留历史活动并默认空封面；封面 URL 安全边界；加载失败重试、无匹配空状态、404、离线和恢复提示 |
| G4-R09 | MIG-CONTENT-MODERATION-001、SEC-CONTENT-REPORT-001、DOM-CONTENT-MODERATION-001、CT-API-CONTENT-MODERATION-001 | v8→v9 保留历史讨论并默认 visible；仅活动参与者可举报他人可见内容；重复举报幂等；移除/恢复/驳回事务和独立动作审计；DTO、错误码、OpenAPI 与路由双向契约 |
| G4-R09 browser | E2E-CONTENT-MODERATION-001 | desktop-chromium 与 Pixel 7：举报帖子/回复、移除回复、驳回举报、直接移除/恢复帖子、公开隐藏 removed 内容、核对 4 条本轮动作记录 |
| G4-R05 | DOM-NOTIFICATION-001、MIG-NOTIFICATION-001、IT-NOTIFICATION-001、SEC-NOTIFICATION-001、SCHED-NOTIFICATION-001、CT-API-NOTIFICATION-001、FE-NOTIFICATION-001 | 报名/取消/活动变更事务通知；v10→v11 和空库/重复迁移；用户隔离、未读状态、活动删除保留；提醒幂等和时间变更；DTO/错误码/OpenAPI/路由契约；通知中心与未读徽标 |
| G4-R05 browser | E2E-NOTIFICATION-001 | desktop-chromium 与 Pixel 7：报名和取消后通知中心显示对应标题、活动正文、无横向溢出并可全部标为已读 |
| BUG-G4-004 | REG-BUG-G4-001、SEC-GOVULN-001 | delimiter-flood JWT 必须拒绝；`jwt/v5.2.2` 修复 `GO-2025-3553`；CI pinned `govulncheck@v1.6.0` 阻断可达 Go 漏洞 |
| BUG-G4-005 | SEC-GO-TOOLCHAIN-001、SEC-GOVULN-001 | 首次 Run `29433701071` 因 Go `1.25.0` 标准库可达漏洞正确失败；工具链提升到 Go `1.25.12` 后必须通过同一扫描与全量门禁 |

## 当前本地结果

| 门禁 | 结果 |
|---|---|
| 顶层 Go Test 数量 | 286 |
| Vue unit/component tests | 17 passed（10 files） |
| Playwright E2E | 4 passed（2 个场景 × desktop-chromium / Pixel 7） |
| go test -count=1 ./... | 通过 |
| go test -race -count=1 ./... | 通过 |
| go vet ./... | 通过 |
| govulncheck@v1.6.0 | 修复前发现 JWT `GO-2025-3553`；首次远端候选进一步发现 Go `1.25.0` 标准库漏洞；`jwt/v5.2.2` + Go `1.25.12` 本地扫描为 0 个可达漏洞 |
| OpenAPI JSON / 路由 / DTO / error_code 契约 | 通过 |
| npm run build | 通过；主 JS 544.28 KB / 190.74 KB gzip；存在大于 500 KB 的 chunk 提示，通知页本身保持路由懒加载 |
| npm audit --omit=dev | 0 vulnerabilities |
| Markdown 本地链接 / git diff --check / gofmt | 通过 |
| Browser 可视验收 | 4 passed（2 个场景 × desktop-chromium / Pixel 7）；通知中心报名、取消、全部已读、无横向溢出与截图证据通过 |

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
| [GitHub Actions Run 29394150565](https://github.com/qw2261/soulmarker/actions/runs/29394150565) | Commit 72dca57，响应式用户旅程与反馈闭环候选 success |
| [backend job 87283755655](https://github.com/qw2261/soulmarker/actions/runs/29394150565/job/87283755655) | format、vet、265 tests、race、Schema v8 与 OpenAPI 契约 success |
| [frontend job 87283755644](https://github.com/qw2261/soulmarker/actions/runs/29394150565/job/87283755644) | Node 22 build、10 unit/component tests、4 desktop/mobile Playwright cases success；6 张成功截图证据已上传 |
| [GitHub Actions Run 29424593002](https://github.com/qw2261/soulmarker/actions/runs/29424593002) | Commit 270d64d，内容治理闭环候选 success |
| [backend job 87383598780](https://github.com/qw2261/soulmarker/actions/runs/29424593002/job/87383598780) | format、vet、271 tests、race、Schema v9、DTO/错误码/OpenAPI/路由契约 success |
| [frontend job 87383598785](https://github.com/qw2261/soulmarker/actions/runs/29424593002/job/87383598785) | Node 22 build、10 unit/component tests、4 desktop/mobile Playwright cases success；8 张成功截图证据已上传 |
| [GitHub Actions Run 29428887570](https://github.com/qw2261/soulmarker/actions/runs/29428887570) | Commit 5880d13，历史账户恢复邮箱候选 success |
| [backend job 87398406467](https://github.com/qw2261/soulmarker/actions/runs/29428887570/job/87398406467) | format、vet、277 tests、race、Schema v10、恢复身份与 API 契约 success |
| [frontend job 87398406297](https://github.com/qw2261/soulmarker/actions/runs/29428887570/job/87398406297) | Node 22 build、13 unit/component tests、4 desktop/mobile Playwright cases success；浏览器证据已上传 |
| [GitHub Actions Run 29432032120](https://github.com/qw2261/soulmarker/actions/runs/29432032120) | Commit 04908db，基础通知闭环候选 success |
| [backend job 87409182693](https://github.com/qw2261/soulmarker/actions/runs/29432032120/job/87409182693) | format、vet、285 tests、race、Schema v11、通知事务/调度/隔离与 OpenAPI 契约 success |
| [frontend job 87409182702](https://github.com/qw2261/soulmarker/actions/runs/29432032120/job/87409182702) | Node 22 build、17 unit/component tests、4 desktop/mobile Playwright cases success；通知中心浏览器证据已上传 |

## Go/No-Go

No-Go：R01、R02、R04、R05、R06、R07、R08、R09 已通过本地和远端候选门禁；R03 仍缺真实 staging SMTP 验收，两场受控活动尚未执行。P0/P1 台账已建立，JWT 与 Go 标准库漏洞已本地修复但仍需候选远端 CI 后正式清零。
