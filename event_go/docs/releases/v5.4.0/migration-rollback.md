# v5.4.0 Migration and Rollback

## 前置备份

停止应用，确认 WAL 已关闭后备份数据库：

```bash
cd event_go
cp data/event_go.db data/event_go.db.pre-v5.4.bak
```

## 前向迁移

应用启动会自动执行：

1. Schema v4 增加 identity_status。
2. 按 users.contact 精确回填 registrations、posts、replies.user_id。
3. 无法匹配的数据保持 user_id=NULL、identity_status=legacy。
4. Schema v5 清理无效可空引用、创建隐藏系统门店并启用外键。
5. 执行 foreign_key_check；必要关系存在孤儿时拒绝启动。

迁移后通过管理员接口检查总量和 legacy 清单：

```bash
curl -s http://localhost:8080/api/admin/identity-migration \
  -H "X-Admin-Token: <ADMIN_TOKEN>"
```

## 应用回滚

- v5.4 已关闭联系方式授权回退，不能只回滚到依赖 contact 授权的应用版本后继续写入。
- 若候选版本启动失败或迁移结果异常，停止应用并恢复 `data/event_go.db.pre-v5.4.bak`，再启动上一制品。
- 恢复后执行健康检查、登录、活动列表、报名和取消 smoke test。

## Forward-fix

- legacy 数据不得直接改 user_id；人工确认用户归属后，使用受审计的后续迁移脚本处理。
- foreign_key_check 发现必要孤儿时保留数据库和备份，先导出问题记录，再通过独立迁移修复。
