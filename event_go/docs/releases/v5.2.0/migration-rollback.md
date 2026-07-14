# v5.2.0 Migration and Rollback

## 数据迁移

本版本将 SQLite schema 升级到版本 3：

- 建立 `schema_migrations` 迁移记录表。
- 为 `registrations`、`posts`、`replies` 增加 nullable `user_id`。
- 为非空报名用户增加活动内唯一索引，并增加讨论作者索引。

迁移在应用启动时逐版本、逐事务执行；任何迁移失败都会阻止服务启动并返回明确的初始化错误。变更为纯 Expand，不删除或重命名旧字段。

生产执行前必须停止旧进程并备份数据库：

```bash
cd event_go
cp data/event_go.db data/event_go.db.pre-v5.2.bak
```

当前已自动验证空库、旧库、重复执行和错误返回；真实生产备份恢复演练仍是发布前置项。

## 应用回滚

1. 保留上一版本制品。
2. 停止当前应用。
3. 若仅回滚应用，直接启动上一版本制品；新增表和 nullable 列可由旧版本忽略。
4. 若迁移后发现数据异常，停止应用并用 `data/event_go.db.pre-v5.2.bak` 恢复数据库文件。
5. 执行健康检查和活动列表、登录、报名 smoke test。

## 兼容性

- 现有 API 保持兼容。
- schema v3 为向后兼容的 Expand；本版本不执行破坏性 down migration。
- 新增 GET /api/events/{id}/registration。
- GET /api/events/{id}/registrations 进入管理员认证边界；生产环境配置 ADMIN_TOKEN 后，旧匿名调用将收到 401。
