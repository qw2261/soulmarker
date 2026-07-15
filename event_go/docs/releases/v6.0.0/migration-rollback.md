# v6.0.0 Migration and Rollback

## 前向迁移

Schema v6、v7、v8 与 v9 都是 Expand 迁移，仅新增：

- `admissions`：Registration 之外的入场权益、随机凭证、active/revoked 状态和票种快照。
- `checkins`：每个 Admission 最多一条成功核销事件。
- Admission/Checkin 外键与查询索引。
- 禁止更新、删除 Checkin 的 SQLite 触发器。
- `user_auth_versions`：独立保存用户认证版本，不重建历史 `users` 表。
- `password_reset_tokens`：保存一次性 Token 的 SHA-256 摘要、过期和消费状态。
- `users_create_auth_version` 触发器与重置 Token 查询索引。
- `events.cover_url`：非空默认空字符串的封面地址列；历史活动无需回填即可继续读取。
- `posts` / `replies` 治理列：默认 `visible` 的状态、处理时间、处理人和原因；历史讨论无需回填即可继续公开读取。
- `content_reports`：分类、状态、处理结论和处理人的举报队列；部分唯一索引保证同一举报人与目标只有一条 open 举报。
- `content_moderation_actions`：移除、恢复和驳回的独立动作审计，不依赖目标当前是否公开可见。

迁移由 `schema_migrations` 独立事务执行。升级前仍按既有流程备份 SQLite 文件；失败时事务回滚，服务拒绝启动。

## 应用回滚

1. 停止 v6 候选应用。
2. 保留 v6 数据库和完整备份，不删除 Admission/Checkin 表。
3. 若回滚到不理解 Schema v7 认证版本的应用，必须先旋转 `JWT_SECRET`，使所有 v7 前签发的 JWT 失效。
4. 若数据库已有 `moderation_status='removed'` 的帖子或回复，不得直接部署 pre-v9 公开应用；旧查询不会过滤治理状态，会重新暴露已移除内容。必须使用保留 v9 读取过滤的兼容回滚制品，或在维护模式下完成前滚修复。
5. 在没有任何 removed 内容时，上一候选应用会忽略新增表、治理列和 `cover_url`，现有活动、报名、库存与讨论数据仍兼容；举报队列和治理入口暂不可用。
6. 复验登录、报名、取消、库存、讨论公开可见性和管理员治理记录；核销入口在 pre-v6 应用中不可用，密码重置入口在 pre-v7 应用中不可用。

重新前滚到 v6 后，已签发凭证和核销审计继续可见。

旋转 `JWT_SECRET` 是 pre-v7 回滚的安全前置条件。旧应用不检查 `auth_version`；若沿用原密钥，已通过退出或密码重置撤销的 JWT 会在回滚后重新有效。

## 禁止操作

生产回滚不得直接删除 `checkins`、修改核销时间、复用已吊销凭证、通过删除认证版本记录恢复会话，或硬删除举报/治理动作来迁就旧版本。只有在确认没有任何真实 Admission/Checkin/PasswordReset/ContentModeration 数据、已完成备份且明确放弃本候选时，才可离线删除新增表和迁移记录。
