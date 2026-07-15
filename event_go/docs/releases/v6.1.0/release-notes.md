# v6.1.0 Release Notes

> 状态：G5.4 Local Candidate；G5.1–G5.3 已通过远端门禁，G5.4 Commit/Run 待回填

## 当前切片

- Schema v12 新增 Organization、OrganizationMember 和 OrganizationInvitation。
- Organization 作为授权、审计和未来计费租户；OrganizerProfile 继续作为公开门店/品牌资料。
- 历史门店一对一回填为 unclaimed Organization，系统占位门店映射为 system，不根据联系方式猜测 owner。
- 新组织、公开资料和唯一 active owner 在同一事务创建。
- 角色集合固定为 owner、admin、editor、checker、finance；普通邀请不能授予 owner。
- 邀请仅允许 active owner/admin 创建，接受者必须匹配登录邮箱或已验证恢复邮箱，并实施过期、待处理唯一和单次消费。
- pre-v12 应用继续使用旧 `/organizers` 写法：创建触发器自动生成租户，删除触发器自动暂停租户。
- [ADR-004](../../adr/004-organization-tenant-boundary-and-migration.md) 与 [迁移回滚说明](migration-rollback.md) 固化后续渐进租户化边界。
- [ADR-005](../../adr/005-platform-and-tenant-authorization.md) 将 platform_admin 与 organization_member 定义为互不继承的 principal。
- 新增 13 项集中 capability Policy，精确约束 owner/admin/editor/checker/finance；租户角色不写入 JWT，每次请求读取实时 Membership 和 Organization 状态。
- 新增当前用户组织列表与租户 session API；跨租户、revoked、suspended 和能力不足统一返回 403 `ORGANIZATION_ACCESS_DENIED`。
- `PlatformAdminAuth` 为旧管理路由写入平台 principal，前端管理守卫同时校验 `authenticated` 与 `principal_type=platform_admin`。
- `ORGANIZATION_AUTH_ENABLED=false` 可关闭全部 `/organizations/...` 租户入口；非法配置拒绝启动，platform scoped 兼容路由继续可用。
- Schema v13 为 Event 增加稳定 `organization_id`，v12→v13 回填历史活动，并在 OrganizerProfile 删除后保留 tenant。
- [ADR-006](../../adr/006-stable-event-tenant-scope.md) 固化 Event tenant、子资源继承、N/N-1 兼容和双轨回滚策略。
- 新增 tenant-scoped Event、Ticket、Registration、Checkin 与 CSV Export API；五角色通过 capability 进入各自最小业务路由。
- `OrganizationOperationsService` 与 scoped Store SQL 同时服务 tenant 和 platform 兼容路由；正确资源 ID 不能绕过 URL organization scope。
- 租户核销 actor 记录为 `organization_member:<userID>`；服务端 CSV 导出继续使用 UTF-8 BOM 与公式注入防护。
- 新增组织自助创建、邀请接受、成员列表/改角色/撤销与邀请列表/撤销 API；OpenAPI、DTO 和 45 个稳定错误码同步。
- 邀请使用 256 位随机 Token，数据库只存 SHA-256 摘要，明文只进入一次 SMTP 链接；投递失败自动撤销 pending 邀请。
- revoked 成员可以通过新邀请恢复；owner、自我撤销和 admin 越权管理由事务内规则阻断。
- 新增用户 JWT 驱动的组织工作台，覆盖组织选择/创建、活动发布、票种、报名、导出、核销和成员邀请，不依赖 `X-Admin-Token`。
- 工作台按实时 capability 区分 editor、checker、finance；每次进入组织路由重新验证 tenant session。
- 登录/注册在只允许站内路径的前提下保留邀请 redirect，未注册收件人可从邮件链接直接完成注册和接受。
- Playwright 通过隔离本地 SMTP 捕获器在 desktop Chromium 与 Pixel 7 验证完整邀请和四角色运营旅程，且 localStorage 始终不存在 `admin_token`。
- [ADR-007](../../adr/007-organization-self-service-and-invitation-delivery.md) 固化自助运营、邀请投递失败语义、成员保护和回退边界。

## 明确未完成

- G5.4 当前只完成本地候选，必须等待功能提交远端 CI success 并回填 Commit/Run，才能关闭 G5-R04/G5-R06。
- platform admin 兼容路由仍使用全局 Token，但仅保留应急和治理；移除兼容面需要单独的使用量审计与弃用计划。
- actor/tenant/request_id 审计、成员操作审计和 PII 脱敏仍属于 G5.5。
- 所有权转移、邀请限流/异步重试/退信处理和三个真实组织试点尚未完成。
- G4 的真实 staging SMTP 与两场受控活动仍是独立上线门禁；本地 SMTP 捕获器不能替代外部投递验收。

因此本版本是 G5 的可回滚基础迁移，不是完整多租户发布，也不改变 G4 真实 SMTP 与受控活动的待验收状态。
