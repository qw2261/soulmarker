# v6.1.0 测试报告

> 状态：G5.4 Local Candidate；G5.1–G5.3 远端通过，G5.4 Commit/Run 待回填

## 版本身份

| 字段 | 值 |
|---|---|
| 上一远端安全基线 Commit | a5722da |
| G5.1 已验证 Commit | f548a298ca7bc7c8695b15e3d986c33e767b295d |
| G5.2 已验证 Commit | 780c4966cb20ce6384a4756db4443c4181d9161d |
| G5.3 已验证 Commit | 00ebef05634467f5a5befb8c1de0303c82cb2a4d |
| G5.4 候选 Commit | 待功能提交后回填 |
| 候选分支 | origin/codex/update_project |
| Schema | v13；v12 租户基础 + Event 稳定 tenant、索引/FK 与 `/events` N/N-1 写兼容 |

## 本切片范围

G5.1 交付租户数据基础，G5.2 交付集中授权内核，G5.3 交付稳定 Event tenant 和 scoped 业务资源。G5.4 候选新增组织自助创建、SMTP 邀请接受、成员/邀请管理和用户 JWT 驱动的租户工作台；owner/editor/checker/finance 可以在没有 Admin Token 的情况下完成免费活动运营。完整审计、PII 脱敏、所有权转移、真实三组织试点和 platform 兼容路由下线仍不在本切片。

## 需求追溯

| Requirement | Test ID | 自动化验收 |
|---|---|---|
| G5-R01 / G5-F01–F02 | MIG-TENANT-V12-001、DOM-ORGANIZATION-001 | 空库 Schema v12；v11→v12 保留活动和资料；历史资料回填 unclaimed；system 租户存在；不猜测 owner |
| G5-R02 foundation / G5-F03 | DOM-ORG-MEMBER-001、SEC-ORG-INVITE-001 | 组织/Profile/owner 原子创建；唯一 active owner；active owner/admin 邀请；禁止 owner 邀请；登录/已验证恢复邮箱匹配；过期清理与单次消费 |
| G5-R03 | DOM-ORG-ROLE-001 | 数据库只接受 owner/admin/editor/checker/finance；邀请只允许非 owner 角色 |
| G5-F04 / G6-R05 前置 | COMPAT-ORGANIZER-V12-001 | pre-v12 INSERT 自动创建 unclaimed 租户并回填；pre-v12 DELETE 自动暂停租户；部分唯一索引不阻断旧写入 |
| G5-R02 authorization | SEC-TENANT-POLICY-001、SEC-TENANT-CONTEXT-001 | 五角色精确 capability 矩阵；JWT 不缓存租户角色；每请求解析 Membership/Organization；未知角色、跨租户、revoked、suspended、能力不足拒绝 |
| G5-R04 foundation | SEC-PRINCIPAL-SEPARATION-001、SEC-TENANT-FLAG-001 | platform Token 与 tenant JWT 互不替代；管理响应/前端校验 principal_type；开关关闭返回 404，非法布尔配置拒绝启动 |
| G5.2 API contract | CT-API-TENANT-AUTH-001、FE-PLATFORM-PRINCIPAL-001 | `/me/organizations`、`/organizations/{id}/session` 的 v1/兼容路由、DTO、OpenAPI、39 个错误码与前端管理 session 契约一致 |
| G5-R05 migration | MIG-EVENT-TENANT-V13-001、COMPAT-EVENT-V13-001 | 空库/v12→v13；Event tenant 回填；删除 Organizer 后 tenant 保留；pre-v13 INSERT/organizer update；错配拒绝；重复迁移/FK/index/trigger |
| G5-R05 resources | SEC-TENANT-RESOURCE-SCOPE-001、SEC-TENANT-BUSINESS-MATRIX-001 | Store 与 HTTP 使用 B 的正确资源 ID 访问 A scope 仍拒绝；Event 不可跨 tenant 改绑；五角色 Event/Ticket/Registration/Checkin/Export 允许/拒绝矩阵 |
| G5.3 API/export | CT-API-TENANT-RESOURCES-001、SEC-TENANT-EXPORT-001 | tenant 与 platform 双轨复用 scoped Service；OpenAPI/Router 双向一致；CSV 由 tenant scope 查询并阻断公式注入；核销 actor 可追溯到 tenant member |
| G5-R02/G5-R06 invite | SEC-ORG-INVITE-TOKEN-001、DOM-ORG-SELF-SERVICE-001 | 256 位 Token 仅存 SHA-256 摘要；HTTP 不泄露 Token/hash；SMTP 链接匹配登录邮箱并单次消费；投递失败撤销；revoked member 可重新邀请恢复 |
| G5-R04 operational | SEC-PLATFORM-TOKEN-REPLACEMENT-001、SEC-TENANT-FLAG-001 | 自助创建、日常运营和角色旅程不设置 Admin Token；platform 兼容面不参与组织者流程；feature flag 关闭自助入口返回 404 |
| G5-R06 members/UI | SEC-ORG-MEMBER-MANAGE-001、CT-API-ORG-SELF-SERVICE-001、FE-ORG-WORKSPACE-001 | owner/admin 事务内管理规则、owner/自撤销保护、撤销后下一请求 403；Router/OpenAPI/DTO/45 错误码一致；工作台按 capability 请求和展示 |
| G5-R06 E2E | E2E-ORG-SELF-SERVICE-001 | desktop Chromium/Pixel 7 完成创建组织、活动、票种、邮件邀请接受、editor 编辑、checker 只核销、finance 只查看/导出；localStorage 无 admin_token |

## 当前本地结果

| 门禁 | 结果 |
|---|---|
| Go Test pass 事件数量 | 350（含表驱动 subtest；不使用 covdata） |
| `go test -count=1 ./...` | 通过 |
| `go test -race -count=1 ./...` | 通过，零数据竞争 |
| `go vet ./...` | 通过，无警告 |
| `govulncheck@v1.6.0` | 0 个可达漏洞；15 个依赖模块已知项均不在当前调用路径 |
| Vue unit/component tests | 19 passed（12 files）；包含站内邀请 redirect 与外站拒绝 |
| `npm run build` | 通过；主 JS 547.48 KB / 191.58 KB gzip，保留大于 500 KB 的既有 chunk 提示 |
| Playwright desktop/Pixel 7 E2E | 6 passed（3 个场景 × 2 个浏览器项目）；完整运营/用户回归与 G5.4 自助组织旅程通过 |
| `npm audit --omit=dev` | 0 vulnerabilities |
| gofmt / `git diff --check` | 通过 |
| G5.3 定向 Store 测试 | 通过；Schema v13、旧写兼容和跨租户资源负向矩阵 |
| G5.3 Handler/契约测试 | 远端完整 Go Test 与 race 通过；跨租户 HTTP、五角色业务路由、CSV 安全与 OpenAPI/Router 双向契约均纳入 |
| G5.4 Store/Service/Handler | 本地通过；随机 Token 摘要、投递失败撤销、重复接受、成员保护、重新邀请、撤销即时生效与 feature flag 关闭均纳入 |
| G5.4 OpenAPI 契约 | 本地通过；9 个新增 operation、7 个 DTO schema、4 个 request body、2 个 path parameter 与 45 个错误码双向一致 |
| G5.4 Playwright | 全量 6 passed；其中 desktop Chromium/Pixel 7 各 1 个自助组织场景通过，本地 SMTP 邀请与四角色正反向权限旅程通过 |

本阶段不使用 covdata，也不以覆盖率数字替代需求追溯、迁移测试和安全门禁。

## 远端 CI

| 证据 | 结果 |
|---|---|
| [首次候选 Commit f548a29](https://github.com/qw2261/soulmarker/commit/f548a298ca7bc7c8695b15e3d986c33e767b295d) | Schema v12、租户 Store、迁移/兼容测试和候选文档 |
| [GitHub Actions Run 29437366319](https://github.com/qw2261/soulmarker/actions/runs/29437366319) | success，与 f548a298ca7bc7c8695b15e3d986c33e767b295d 绑定 |
| [backend job 87427278795](https://github.com/qw2261/soulmarker/actions/runs/29437366319/job/87427278795) | format、vet、govulncheck、293 tests、race success |
| [frontend job 87427278794](https://github.com/qw2261/soulmarker/actions/runs/29437366319/job/87427278794) | build、17 unit/component tests、4 desktop/mobile E2E、浏览器证据上传 success |
| [G5.2 Commit 780c496](https://github.com/qw2261/soulmarker/commit/780c4966cb20ce6384a4756db4443c4181d9161d) | 集中 Policy、实时 tenant context、platform principal、只读 API、OpenAPI/前端契约与回退开关 |
| [GitHub Actions Run 29440070376](https://github.com/qw2261/soulmarker/actions/runs/29440070376) | success，与 780c4966cb20ce6384a4756db4443c4181d9161d 绑定 |
| [backend job 87436433655](https://github.com/qw2261/soulmarker/actions/runs/29440070376/job/87436433655) | format、vet、govulncheck 0、302 tests、race success |
| [frontend job 87436433616](https://github.com/qw2261/soulmarker/actions/runs/29440070376/job/87436433616) | build、18 unit/component tests、4 desktop/mobile E2E、浏览器证据上传 success |
| [G5.3 Commit 00ebef0](https://github.com/qw2261/soulmarker/commit/00ebef05634467f5a5befb8c1de0303c82cb2a4d) | Schema v13、scoped Store/Service、tenant/platform 双轨 API、隔离矩阵、ADR-006 与候选文档 |
| [GitHub Actions Run 29443464930](https://github.com/qw2261/soulmarker/actions/runs/29443464930) | success，与 00ebef05634467f5a5befb8c1de0303c82cb2a4d 绑定 |
| [backend job 87447966246](https://github.com/qw2261/soulmarker/actions/runs/29443464930/job/87447966246) | format、vet、govulncheck 0、309 tests、race success |
| [frontend job 87447966182](https://github.com/qw2261/soulmarker/actions/runs/29443464930/job/87447966182) | build、18 unit/component tests、4 desktop/mobile E2E 与浏览器证据上传 success |
| G5.4 功能候选 Commit | 待提交后回填 |
| G5.4 GitHub Actions Run / backend / frontend jobs | 待远端 CI success 后回填；回填前 G5-R04/G5-R06 保持未关闭 |

## Go/No-Go

Local Go（仅 G5.4 候选）：自助组织 API/UI、安全邀请和 desktop/mobile 四角色旅程已通过本地门禁，可以提交远端候选。G5-R04/G5-R06 在远端 CI success 并回填 Commit/Run 前仍保持未关闭。完整 G5、M2 与商业化仍为 No-Go；后续还必须完成 G5.5 审计/PII/真实试点，以及 G6–G8 生产、支付和商业化基线。
