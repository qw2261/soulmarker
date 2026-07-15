# ADR-007：组织自助运营与邀请投递边界

> 状态：Accepted<br>
> 日期：2026-07-16<br>
> 适用：G5-R04 / G5-R06 / G5.4 自助运营

## 背景

G5.1–G5.3 已建立 Organization、实时 capability 和业务资源 tenant scope，但组织创建、成员维护和日常运营仍缺少普通用户入口。若继续让组织者使用全局 `X-Admin-Token`，平台凭证会越过租户边界；若邀请 API 返回明文 Token，则浏览器历史、日志或响应采集都可能泄露可消费凭证。

## 决策

1. 登录用户通过 `POST /organizations` 自助创建 active Organization、OrganizerProfile 和唯一 active owner，三者沿用 Store 原子事务。日常活动、票种、报名、导出和核销只调用 `/organizations/{organizationId}/...`，不要求或读取 platform Admin Token。
2. 工作台每次进入组织路由都重新请求 tenant session；服务端每次业务请求继续实时读取 Membership、Organization 状态和 capability。前端隐藏无权限操作只改善交互，不能替代服务端授权。
3. 邀请 Token 使用 256 位加密随机数，数据库只保存 SHA-256 摘要。HTTP 创建/列表响应不返回明文 Token 或摘要；唯一明文只进入一次 SMTP 邀请链接。`ORGANIZATION_INVITATION_TTL_HOURS` 默认 72，允许 1–168。
4. SMTP 投递失败时，同一用例立即撤销刚创建的邀请，避免留下用户无法取得链接的 pending 状态。接受邀请必须由登录用户完成，并匹配登录邮箱或已验证恢复邮箱；未注册收件人的登录/注册跳转必须保留站内邀请路径，外站 redirect 被拒绝。Token 单次消费、过期和撤销规则继续由 Store 事务保护。
5. 被撤销成员可通过新邀请恢复 active，并采用新角色。普通成员管理 API 禁止修改或撤销 owner，也禁止自我撤销。
6. owner 可管理所有非 owner 成员；admin 只能管理或授予 editor、checker、finance，不能管理或授予 admin/owner。成员变更在 Store 事务内重新校验 actor，避免只依赖进入 Handler 时的旧 capability。
7. platform 管理路由暂时保留为应急和治理兼容面，但不出现在组织者日常旅程。`ORGANIZATION_AUTH_ENABLED=false` 可整体关闭自助与 tenant 业务入口，回退后 platform scoped Service 仍可应急运营。
8. Playwright 使用独立本地 SMTP 捕获器验证邀请链接投递和消费；它只存在于测试进程，不增加生产 API、数据库后门或明文 Token 响应。

## 权限与界面

| 角色 | 工作台能力 |
|---|---|
| owner | 组织、成员/邀请、活动、票种、报名、导出、核销 |
| admin | 非 owner/admin 成员管理、邀请、活动、票种、报名、核销 |
| editor | 活动与票种，不读取成员或报名 PII |
| checker | 只进入核销工具和核销记录，不请求成员或报名名单 |
| finance | 只读取/导出报名，不编辑活动、成员或执行核销 |

## 后果

- 新组织可以在没有 platform admin 和人工改库的情况下完成免费活动运营，G5-R04/R06 的产品路径具备自动化验收基础。
- 邀请投递成为同步用例依赖；失败会向操作者返回错误并撤销邀请。异步队列、重试和退信处理属于 G6 生产基础设施，不在本切片伪造。
- platform 兼容路由继续存在，不代表组织者仍需全局 Token；后续移除兼容路由必须另做使用量审计和弃用计划。
- 完整 actor/tenant/request_id 审计、PII 脱敏策略、所有权转移和真实三组织试点仍属于 G5.5。

## 回滚与退出条件

异常时设置 `ORGANIZATION_AUTH_ENABLED=false`，关闭 `/me/organizations`、组织创建/邀请和全部 `/organizations/...` 入口；不删除 Organization、Membership、Invitation 或 Event 数据。应用回滚继续保留 Schema v13 和 scoped platform Service。

G5.4 放行必须同时证明：自助创建原子性、Token 不落库/不出 API、投递失败撤销、邀请接受与重复消费、owner/admin 成员保护、撤销即时生效、OpenAPI/Router/DTO/错误码一致，以及 desktop/Pixel 7 上 owner/editor/checker/finance 不使用 Admin Token 的完整旅程。
