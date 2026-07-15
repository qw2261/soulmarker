# v6.1.0 测试报告

> 状态：G5.2 Candidate Local Pass；完整本地门禁通过，等待首次候选 Commit 与远端 CI

## 版本身份

| 字段 | 值 |
|---|---|
| 上一远端安全基线 Commit | a5722da |
| G5.1 已验证 Commit | f548a298ca7bc7c8695b15e3d986c33e767b295d |
| G5.2 当前候选 Commit | Pending（首次候选提交后回填） |
| 候选分支 | origin/codex/update_project |
| Schema | v12，新增租户、成员、邀请、角色约束和 `/organizers` N/N-1 写兼容 |

## 本切片范围

G5.1 交付租户数据基础。G5.2 在不修改 Schema 的前提下交付集中 capability Policy、实时 Membership 授权、platform/tenant principal 分离、当前用户组织列表、租户 session、稳定拒绝错误码和回退开关。它仍不包含组织自助写 API/UI、业务资源 tenant scope、审计或 platform admin 业务路由移除。

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

## 当前本地结果

| 门禁 | 结果 |
|---|---|
| 顶层 Go Test 数量 | 302 |
| `go test -count=1 ./...` | 通过 |
| `go test -race -count=1 ./...` | 通过，零数据竞争 |
| `go vet ./...` | 通过，无警告 |
| `govulncheck@v1.6.0` | 0 个可达漏洞；15 个依赖模块已知项均不在当前调用路径 |
| Vue unit/component tests | 18 passed（11 files） |
| `npm run build` | 通过；主 JS 544.34 KB / 190.74 KB gzip，保留大于 500 KB 的既有 chunk 提示 |
| Playwright desktop/Pixel 7 E2E | 4 passed（2 个场景 × 2 个浏览器项目）；完整运营/用户回归通过 |
| `npm audit --omit=dev` | 0 vulnerabilities |
| gofmt / `git diff --check` | 通过 |

本阶段不使用 covdata，也不以覆盖率数字替代需求追溯、迁移测试和安全门禁。

## 远端 CI

| 证据 | 结果 |
|---|---|
| [首次候选 Commit f548a29](https://github.com/qw2261/soulmarker/commit/f548a298ca7bc7c8695b15e3d986c33e767b295d) | Schema v12、租户 Store、迁移/兼容测试和候选文档 |
| [GitHub Actions Run 29437366319](https://github.com/qw2261/soulmarker/actions/runs/29437366319) | success，与 f548a298ca7bc7c8695b15e3d986c33e767b295d 绑定 |
| [backend job 87427278795](https://github.com/qw2261/soulmarker/actions/runs/29437366319/job/87427278795) | format、vet、govulncheck、293 tests、race success |
| [frontend job 87427278794](https://github.com/qw2261/soulmarker/actions/runs/29437366319/job/87427278794) | build、17 unit/component tests、4 desktop/mobile E2E、浏览器证据上传 success |
| G5.2 首次候选 Commit / Run | Pending |

## Go/No-Go

No-Go（G5.2）：完整本地门禁已通过，仍需首次候选 Commit 与远端 CI 绑定。远端通过后也只允许进入 G5.3 资源租户化；完整 G5、M2 与商业化继续为 No-Go。
