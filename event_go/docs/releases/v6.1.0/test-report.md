# v6.1.0 测试报告

> 状态：Candidate Local Pass，完整本地门禁通过，等待首次候选 Commit 与远端 CI

## 版本身份

| 字段 | 值 |
|---|---|
| 上一远端安全基线 Commit | a5722da |
| 当前候选 Commit | Pending（首次候选提交后回填） |
| 候选分支 | origin/codex/update_project |
| Schema | v12，新增租户、成员、邀请、角色约束和 `/organizers` N/N-1 写兼容 |

## 本切片范围

G5.1 只交付租户数据基础：Organization 与 OrganizerProfile 分离、历史 unclaimed 回填、唯一 active owner、邀请存储语义和兼容迁移。不包含组织自助 HTTP API/UI、tenant context、业务资源 scope、完整权限矩阵、审计或 platform admin 替换。

## 需求追溯

| Requirement | Test ID | 自动化验收 |
|---|---|---|
| G5-R01 / G5-F01–F02 | MIG-TENANT-V12-001、DOM-ORGANIZATION-001 | 空库 Schema v12；v11→v12 保留活动和资料；历史资料回填 unclaimed；system 租户存在；不猜测 owner |
| G5-R02 foundation / G5-F03 | DOM-ORG-MEMBER-001、SEC-ORG-INVITE-001 | 组织/Profile/owner 原子创建；唯一 active owner；active owner/admin 邀请；禁止 owner 邀请；登录/已验证恢复邮箱匹配；过期清理与单次消费 |
| G5-R03 | DOM-ORG-ROLE-001 | 数据库只接受 owner/admin/editor/checker/finance；邀请只允许非 owner 角色 |
| G5-F04 / G6-R05 前置 | COMPAT-ORGANIZER-V12-001 | pre-v12 INSERT 自动创建 unclaimed 租户并回填；pre-v12 DELETE 自动暂停租户；部分唯一索引不阻断旧写入 |

## 当前本地结果

| 门禁 | 结果 |
|---|---|
| 顶层 Go Test 数量 | 293 |
| `go test -count=1 ./...` | 通过 |
| `go test -race -count=1 ./...` | 通过，零数据竞争 |
| `go vet ./...` | 通过，无警告 |
| `govulncheck@v1.6.0` | 0 个可达漏洞；依赖模块存在但当前代码不可达的已知项不作为通过项隐藏 |
| Vue unit/component tests | 17 passed（10 files） |
| `npm run build` | 通过；主 JS 544.28 KB / 190.74 KB gzip，保留大于 500 KB 的既有 chunk 提示 |
| Playwright desktop/Pixel 7 E2E | 4 passed（2 个场景 × 2 个浏览器项目）；本切片无新增公开页面，执行全量回归 |
| `npm audit --omit=dev` | 0 vulnerabilities |
| gofmt / `git diff --check` | 通过 |

本阶段不使用 covdata，也不以覆盖率数字替代需求追溯、迁移测试和安全门禁。

## 远端 CI

| 证据 | 结果 |
|---|---|
| 首次候选 Commit | Pending |
| GitHub Actions Run | Pending |

## Go/No-Go

No-Go：G5.1 完整本地门禁已通过，仍等待候选 Commit 与远端 CI 绑定。远端通过后也只允许进入 G5.2 授权内核，不代表完整 G5、M2 或商业化完成。
