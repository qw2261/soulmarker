# v6.1.0 Migration and Rollback

## 前向迁移

Schema v12–v13 是 Expand-only，新增：

- `organizations`：租户名称、slug、状态和时间；id=0 为 system。
- `organizers.organization_id`：非空默认 0，迁移后历史非系统资料一对一关联 unclaimed Organization。
- `organization_members`：用户、角色和 active/revoked 状态；部分唯一索引保证每组织最多一个 active owner。
- `organization_invitations`：规范化邮箱、非 owner 角色、Token 摘要、pending/accepted/revoked/expired 状态和过期/消费时间。
- 创建兼容触发器：旧应用 INSERT 不提供 `organization_id` 时自动创建 unclaimed Organization 并回填。
- 删除兼容触发器：旧应用 DELETE OrganizerProfile 时自动暂停对应 Organization。
- `events.organization_id`：Schema v13 的稳定授权边界，按 OrganizerProfile tenant 回填，删除公开资料后仍保留。
- Event 兼容触发器：pre-v13 INSERT 省略 tenant 时自动回填；旧应用只更新 `organizer_id` 时同步 tenant；显式错配写入被拒绝。
- `idx_events_organization` 与 `events.organization_id -> organizations.id` 外键。

迁移不会创建历史 Membership，也不会根据 `organizers.contact` 猜测 owner。Event 继续保留 `organizer_id` 作为公开资料关联，Ticket/Registration/Admission/Checkin 通过 Event 继承 tenant，不重复写入 tenant_id。

## 升级步骤

1. 停止写入并记录当前应用 Commit、Schema 版本和数据库文件路径。
2. 复制 SQLite 主文件及同目录 WAL/SHM 状态所需的一致性备份；推荐先正常停止应用再复制单文件。
3. 在备份副本运行新应用或迁移测试，确认 v11→v12→v13 或 v12→v13 成功。
4. 启动候选后执行并保存：

```sql
SELECT MAX(version) FROM schema_migrations;
PRAGMA foreign_key_check;
SELECT COUNT(*) FROM organizers WHERE id <> 0 AND organization_id = 0;
SELECT organization_id, COUNT(*) FROM organizers
WHERE organization_id <> 0 GROUP BY organization_id HAVING COUNT(*) > 1;
SELECT COUNT(*) FROM organization_members;
SELECT COUNT(*) FROM events WHERE organization_id = 0 AND organizer_id <> 0;
SELECT COUNT(*) FROM events e JOIN organizers o ON o.id = e.organizer_id
WHERE e.organizer_id <> 0 AND e.organization_id <> o.organization_id;
```

预期版本为 13、foreign_key_check 无行、历史非系统资料与 Event 不存在零 organization_id、Event/Profile tenant 无错配、每个非零 organization_id 只有一份 Profile；迁移后成员数保持原值。

## N/N-1 兼容

- pre-v12 读取和更新旧 Organizer 字段不受新增列影响。
- pre-v12 INSERT 省略 `organization_id` 时，触发器生成 unclaimed Organization；旧应用取得的 Organizer ID 保持可用。
- pre-v12 DELETE 时，触发器把对应 Organization 标记为 suspended，避免留下无公开资料的 active 租户。
- pre-v13 Event INSERT 省略 `organization_id` 时从 OrganizerProfile 自动回填；旧应用只更新 `organizer_id` 时同步 tenant。
- 新 v13 Store/Service 禁止跨 Organization 改绑 OrganizerProfile；只有 N-1 原始 SQL 兼容路径会按旧语义同步 tenant，回滚窗口必须限制为受控 platform 运营。
- pre-v12 应用不了解 Membership/Invitation，也不会执行租户授权。应用回滚期间必须关闭未来自助租户入口，仅允许受控 platform admin 运营。

## 应用回滚

1. 停止 v13 候选应用，保留 v13 数据库和升级前备份。
2. 设置 `ORGANIZATION_AUTH_ENABLED=false` 或在入口层关闭租户业务路由，停止 tenant 写流量。
3. 部署 pre-v13 应用。不要删除 organizations、memberships、invitations、`events.organization_id`、索引或触发器。
4. 复验旧 `/organizers` 与 `/events` 创建、读取、更新、删除；确认新 Event 自动获得 tenant，Organizer 删除后历史 Event tenant 保留。
5. 修复后以前滚方式重新部署 v13；新增租户数据仍保留。

## 完全撤销 Schema v12–v13

只有在确认升级后没有任何新 Organization、Membership、Invitation、Organizer 或 Event 写入，并且业务明确放弃候选时，才允许停机恢复升级前完整数据库备份。不要在生产库手工 DROP 表、索引、触发器或尝试删除任一 `organization_id` 列；这会破坏外键和后续前滚证据。

## Go/No-Go 检查

- 空库、v11→v12→v13、v12→v13、重复迁移、迁移失败和备份恢复测试通过。
- FK、部分唯一索引、两个兼容触发器存在且行为测试通过。
- 历史 Event/Organizer 数量和关联保持不变，Membership 不被猜测生成。
- Organizer 与 Event 的 N/N-1 创建、更新、删除兼容通过。
- 候选 Commit、远端 CI Run、备份路径、执行人和回滚判定已记录。

当前代码候选证据：Commit `f548a29`，GitHub Actions Run `29437366319` success；backend job `87427278795` 与 frontend job `87427278794` 均通过。

## G5.2 应用授权切片

G5.2 不增加 Schema 版本。它新增只读 `/me/organizations`、`/organizations/{organizationId}/session`、实时 capability Policy、`principal_type` 响应字段和 `ORGANIZATION_ACCESS_DENIED` 错误码。

授权异常时优先设置 `ORGANIZATION_AUTH_ENABLED=false` 并重启应用：`/me/organizations` 与全部 `/organizations/...` 入口统一返回 404，既有 platform admin 业务路由与租户数据继续工作。若需要整体回滚，部署 G5.1 前后端同一制品；不要删除 Membership 或回退数据库。新前端配旧后端会因缺少 `principal_type` 拒绝管理登录，这是预期的 fail-closed 行为，因此当前必须整体发布/回滚前后端。

G5.2 放行检查已验证：五角色精确矩阵、跨租户/暂停/撤销 403、platform/tenant 凭证互斥、开关关闭 404、非法开关启动失败、OpenAPI/DTO/错误码契约和完整 CI。

G5.2 代码候选证据：Commit `780c496`，GitHub Actions Run `29440070376` success；backend job `87436433655` 与 frontend job `87436433616` 均通过。

## G5.3 资源租户化切片

G5.3 使用 Schema v13 和 [ADR-006](../../adr/006-stable-event-tenant-scope.md)。新 `/organizations/{organizationId}/events/...` 路由与 platform 兼容路由复用 `OrganizationOperationsService` 和 scoped Store；关闭 `ORGANIZATION_AUTH_ENABLED` 只关闭租户入口，不改变数据库或 platform 应急运营能力。

G5.3 候选证据：Commit `00ebef0`，GitHub Actions Run `29443464930` success；backend job `87447966246` 与 frontend job `87447966182` 均通过。Schema v13 上线仍需按本页记录真实备份路径、执行人和回滚判定。
