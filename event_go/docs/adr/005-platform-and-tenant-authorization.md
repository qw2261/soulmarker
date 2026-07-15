# ADR-005：Platform 与 Tenant 授权模型

> 状态：Accepted<br>
> 日期：2026-07-16<br>
> 适用：G5-R02–R05 / G5.2 授权内核及后续资源租户化

## 背景

Schema v12 已建立 Organization、Membership 和角色，但“数据库里有 role”不等于请求已经获得租户权限。现有 `X-Admin-Token` 是平台级运营凭证，若直接把它解释为任意租户 owner，或把 Membership 固化进长期 JWT，会分别造成跨租户超级权限和角色撤销延迟。G5.3 还需要逐项迁移 Event、Ticket、Registration、Admission、Checkin 和 Export，授权内核必须先形成单一、可测试且可回退的判断入口。

## 决策

1. `platform_admin` 与 `organization_member` 是两种独立 principal。平台 Token 只进入 platform context，不自动获得任何 tenant capability；用户 JWT 也不能替代平台 Token。
2. JWT 只证明用户身份和会话版本，不保存 organization_id、role 或 capability。每次租户请求都以 JWT user_id 和 URL 中的 organizationId 查询当前 Membership，角色撤销、组织暂停随下一请求生效。
3. 租户上下文以路径 `/organizations/{organizationId}/...` 为唯一组织选择来源，不使用可被代理或客户端误传的全局“当前租户”Header。
4. capability 是 Handler/Service 使用的稳定授权语言，角色只在集中 Policy 中映射：

| 角色 | capability |
|---|---|
| owner | 全部 13 项，包括 owner.transfer、finance.read、registrations.export |
| admin | 组织/成员/活动/票种/报名查看/核销/审计；不含 owner.transfer、finance.read、registrations.export |
| editor | organization.read/manage、events.manage、tickets.manage |
| checker | organization.read、checkins.manage |
| finance | organization.read、registrations.read/export、finance.read |

5. 未登录或无效 JWT 返回 401；非法 organizationId 返回 400；不存在的 Membership、跨租户访问、revoked Membership、suspended Organization 和角色能力不足统一返回 403 `ORGANIZATION_ACCESS_DENIED`，避免通过差异响应枚举组织。
6. `GET /me/organizations` 返回当前用户 active Membership 对应的组织状态、角色和实时有效 capability；suspended Organization 保留可见但 capability 为空。`GET /organizations/{organizationId}/session` 通过统一中间件解析租户 principal。
7. G5.2 不切换现有管理业务路由。它们在兼容周期内明确使用 `PlatformAdminAuth`；G5.3 按资源逐项接入 tenant capability，验证跨租户负向矩阵后再移除公开业务面的全局平台 Token。
8. `ORGANIZATION_AUTH_ENABLED=false` 可关闭两个只读租户入口并返回 404；非法布尔值导致启动配置校验失败。数据库结构和旧管理路由不受该开关影响。
9. OpenAPI 和前端管理会话都校验 `principal_type`。新前端遇到不含 `platform_admin` 的管理会话响应时 fail closed；旧客户端可安全忽略新增响应字段。

## 后果

- 角色或组织状态变化无需等待 JWT 过期，降低离职成员和暂停组织继续操作的窗口。
- platform 与 tenant 凭证不能互相替代，后续资源迁移可以明确记录每条路由属于哪一类 principal。
- 每次租户请求多一次 Membership 查询；当前单实例 SQLite 可承受，达到容量证据后再评估短 TTL 授权缓存，缓存必须支持撤销失效。
- G5.2 只有只读上下文 API。capability 尚未接入业务资源时，不能宣称 event/checkin/export 已完成租户隔离。
- 新前端配旧后端时管理登录会因缺少 `principal_type` 安全失败；当前前后端由同一制品发布，回滚应整体回滚。若未来拆分部署，需新增显式兼容窗口。

## 回滚与退出条件

出现授权上下文异常时，先设置 `ORGANIZATION_AUTH_ENABLED=false` 关闭新入口，旧 platform admin 运营路径继续工作；无需回滚 Schema v12 或删除 Membership。应用整体回滚到 G5.1 时新增只读路由和 `principal_type` 消失，租户数据保持不变。

进入 G5.3 前必须证明：五角色精确 capability 矩阵、platform/tenant 凭证互斥、跨租户/暂停/撤销拒绝、OpenAPI 双向契约、feature flag 回退和完整 CI 均通过。G5.3 每迁移一种资源都必须在 Service/Store 查询入口携带 organization scope，不能只在 Handler 做一次表面检查。
