# v6.0.0 Migration and Rollback

## 前向迁移

Schema v6 是 Expand 迁移，仅新增：

- `admissions`：Registration 之外的入场权益、随机凭证、active/revoked 状态和票种快照。
- `checkins`：每个 Admission 最多一条成功核销事件。
- Admission/Checkin 外键与查询索引。
- 禁止更新、删除 Checkin 的 SQLite 触发器。

迁移由 `schema_migrations` 独立事务执行。升级前仍按既有流程备份 SQLite 文件；失败时事务回滚，服务拒绝启动。

## 应用回滚

1. 停止 v6 候选应用。
2. 保留 v6 数据库和完整备份，不删除 Admission/Checkin 表。
3. 部署上一候选制品；旧应用会忽略新增表，现有活动、报名、库存与讨论数据仍兼容。
4. 复验登录、报名、取消、库存和讨论；核销入口在旧应用中不可用。

重新前滚到 v6 后，已签发凭证和核销审计继续可见。

## 禁止操作

生产回滚不得直接删除 `checkins`、修改核销时间或复用已吊销凭证。只有在确认没有任何真实 Admission/Checkin 数据、已完成备份且明确放弃本候选时，才可离线删除 v6 新表和迁移记录。
