# ADR-001：SQLite 外键与删除语义

> 状态：Accepted
> 日期：2026-07-15
> 适用：Schema v5 / G2-R07

## 背景

早期版本声明了部分外键但没有可靠启用 `PRAGMA foreign_keys`，并依赖手工删除顺序。门店删除会把活动 `organizer_id` 设置为 0，门票删除则可能与报名快照发生冲突。身份模型 v2 又新增了 registrations、posts、replies 到 users 的可空引用。

## 决策

1. SQLite 运行期固定一个数据库连接；连接初始化后执行 `PRAGMA foreign_keys=ON`。
2. 每次启动执行 `PRAGMA foreign_key_check`，存在必要关系孤儿时拒绝启动。
3. Schema v5 创建隐藏的 id=0 系统门店。删除真实门店时，历史活动转移到该占位记录；公开门店列表和详情不展示系统门店。
4. 删除活动继续由应用事务显式清理，顺序为 replies → posts → registrations → tickets → events。
5. 删除门票时将历史报名的 `ticket_id` 置空，但保留 `ticket_name` 快照用于审计。
6. 用户目前没有删除接口。未来删除用户默认采用 RESTRICT 或匿名化流程，不级联删除报名、帖子和回复。
7. 身份迁移只自动修复可空引用：无效 user_id 回退为 legacy、无效 ticket_id 置空、无效 organizer_id 转移到系统门店。必要 event_id/post_id 孤儿不自动删除，必须人工处理。

## 后果

- 单实例 SQLite 阶段能够可靠执行外键与写入串行化。
- 多实例或显著写并发不适用该连接模型；进入真实支付前仍需按 G6 数据库闸门评估 PostgreSQL。
- 删除门店和门票不会破坏历史活动、报名与票种快照。
- 应用回滚必须兼容 schema v5；数据库恢复遵循 v5.4 发布回滚文档。
