# v6.1.0 Release Notes

> 状态：G5.3 Remote Candidate Pass（Commit 00ebef0 / Run 29443464930）；G5.1/G5.2 同样已通过远端门禁

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

## 明确未完成

- 没有新增组织自助创建、邀请接受或成员管理 HTTP API/UI。
- G5-R05 已由 Commit `00ebef0` / Run `29443464930` 关闭；G5-R04 仍待 G5.4 替换公开业务面的 platform Token。
- platform admin 兼容路由仍使用全局 Admin Token；虽已复用 scoped Service，但租户自助后台尚未替换公开业务面的管理认证。
- tenant API 已可按角色操作资源，但尚无 G5.4 自助入驻、邀请/成员管理与租户后台 UI。
- actor/tenant/request_id 审计与 PII 按角色脱敏尚未实现。

因此本版本是 G5 的可回滚基础迁移，不是完整多租户发布，也不改变 G4 真实 SMTP 与受控活动的待验收状态。
