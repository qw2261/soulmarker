# ADR-006：Event 稳定租户边界与资源作用域

> 状态：Accepted<br>
> 日期：2026-07-16<br>
> 适用：G5-R04–R05 / G5.3 资源租户化 / Schema v13

## 背景

G5.1 通过 `OrganizerProfile.organization_id` 建立租户基础，但 Event 仍只保存 `organizer_id`。这条间接链路不能作为长期授权边界：兼容删除 OrganizerProfile 时会把历史 Event 的 `organizer_id` 置为 0，Event 将无法再确定原租户；同时仅在 Handler 先查一次归属，后续按资源 ID 更新，也会留下跨租户竞态和绕过入口。

## 决策

1. Schema v13 为 `events` 增加非空 `organization_id`。它是 Event 的稳定授权边界；`organizer_id` 继续表示可删除、可替换的公开资料归属。删除 OrganizerProfile 不改变 Event 的 `organization_id`。
2. Ticket、Registration、Admission、Checkin 和报名导出不重复保存 tenant_id，而是通过不可变的 `event_id -> events.organization_id` 继承作用域。所有租户管理 SQL 必须在查询、更新或删除语句中携带 organization scope，不能只依赖 Handler 前置查询。
3. `OrganizationOperationsService` 是 Event/Ticket/Registration/Checkin/Export 管理用例的统一入口。租户路由和兼容期 platform 路由都调用同一 scoped Store API；无作用域 Store 写方法只保留源码兼容，并先解析当前 Organization 后转入 scoped 实现。
4. Event 创建和更新只接受同一 Organization 下的 OrganizerProfile。通用应用写方法不得把 Event 改绑到另一租户；数据库触发器继续拒绝显式 `organization_id`/`organizer_id` 错配。
5. 新租户业务路径使用 `/organizations/{organizationId}/events/...`，每次请求先按 ADR-005 实时解析 Membership 与 capability，再在资源 SQL 中二次绑定 organization_id。已通过组织授权但资源属于其他组织时返回 404；没有路径组织权限时返回 403。
6. capability 与入口对应：Event 使用 `events.manage`，Ticket 使用 `tickets.manage`，报名查看使用 `registrations.read`，CSV 导出使用 `registrations.export`，核销与核销记录使用 `checkins.manage`。租户核销 actor 为 `organization_member:<userID>`，平台兼容入口为 `platform_admin`。
7. platform 管理路由在 G5.4 自助后台可替代前继续保留，但只负责解析资源当前 Organization，实际业务调用不得回到无作用域写路径。此双轨是兼容策略，不代表 platform Token 已退出公开业务面。

## 迁移与兼容

- v12→v13 按 OrganizerProfile 回填现有 Event 的 `organization_id`，保留 Event、Membership、Registration、Admission 和 Checkin。
- pre-v13 INSERT 省略 `organization_id` 时，兼容触发器从 OrganizerProfile 自动回填。
- pre-v13 只更新 `organizer_id` 时，兼容触发器同步 Event tenant；v13 scoped Store 则禁止跨组织改绑。该旧写兼容只服务 N-1 回滚窗口，不是新业务 API。
- 新列、索引、外键和触发器均为 Expand。应用回滚时保留 Schema v13，不删除或清空 `events.organization_id`。

## 后果

- OrganizerProfile 删除后，历史 Event 仍有稳定授权和未来计费归属。
- 子资源作用域由 Event 单点继承，避免多份 tenant_id 漂移；代价是租户查询必须 JOIN 或使用 scoped Event 子查询。
- platform 与 tenant 双轨复用同一 Service/Store，降低兼容期出现两套业务规则的风险。
- G5.3 只完成后端资源隔离和契约。自助入驻/成员管理/UI、完整审计、request_id、PII 按角色脱敏仍属于 G5.4–G5.5。

## 回滚与退出条件

授权异常时可设置 `ORGANIZATION_AUTH_ENABLED=false` 关闭全部租户路由，platform 兼容路由继续通过 scoped Service 运营。应用可回滚到 pre-v13，但必须保留 v13 数据库和兼容触发器；回滚期间禁止新租户自助流量和跨 Organization Organizer 改绑。

G5.3 放行必须同时证明：空库和 v12→v13 迁移、重复迁移/FK/触发器、旧写兼容、删除 Organizer 后 tenant 保留、Store/Service 直接资源 ID 隔离、五角色业务路由矩阵、platform/tenant 双轨、OpenAPI 双向契约、完整本地与远端门禁。
