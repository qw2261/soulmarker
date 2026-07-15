# v6.0.0 Migration and Rollback

## 前向迁移

Schema v6 与 v7 都是 Expand 迁移，仅新增：

- `admissions`：Registration 之外的入场权益、随机凭证、active/revoked 状态和票种快照。
- `checkins`：每个 Admission 最多一条成功核销事件。
- Admission/Checkin 外键与查询索引。
- 禁止更新、删除 Checkin 的 SQLite 触发器。
- `user_auth_versions`：独立保存用户认证版本，不重建历史 `users` 表。
- `password_reset_tokens`：保存一次性 Token 的 SHA-256 摘要、过期和消费状态。
- `users_create_auth_version` 触发器与重置 Token 查询索引。

迁移由 `schema_migrations` 独立事务执行。升级前仍按既有流程备份 SQLite 文件；失败时事务回滚，服务拒绝启动。

## 应用回滚

1. 停止 v6 候选应用。
2. 保留 v6 数据库和完整备份，不删除 Admission/Checkin 表。
3. 若回滚到不理解 Schema v7 认证版本的应用，必须先旋转 `JWT_SECRET`，使所有 v7 前签发的 JWT 失效。
4. 部署上一候选制品；旧应用会忽略新增表，现有活动、报名、库存与讨论数据仍兼容。
5. 复验登录、报名、取消、库存和讨论；核销入口在旧应用中不可用，密码重置入口不可用。

重新前滚到 v6 后，已签发凭证和核销审计继续可见。

旋转 `JWT_SECRET` 是 pre-v7 回滚的安全前置条件。旧应用不检查 `auth_version`；若沿用原密钥，已通过退出或密码重置撤销的 JWT 会在回滚后重新有效。

## 禁止操作

生产回滚不得直接删除 `checkins`、修改核销时间、复用已吊销凭证，或通过删除认证版本记录恢复会话。只有在确认没有任何真实 Admission/Checkin/PasswordReset 数据、已完成备份且明确放弃本候选时，才可离线删除新增表和迁移记录。
