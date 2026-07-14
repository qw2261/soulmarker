# v5.4.0 Release Notes

> 状态：Verification

## 目标

完成 G2 用户身份与数据模型 v2，使 user_id 成为报名、取消和讨论授权的唯一身份源。

## 主要变化

- 报名只接收可选 ticket_id，姓名和联系方式从持久化用户读取。
- 取消、发帖和回复必须携带有效 JWT，并按 user_id 校验归属和报名状态。
- 新增 GET /api/me/registrations 和“我的报名”页面。
- 新增受管理员保护的 GET /api/admin/identity-migration。
- Schema v4 精确回填历史身份并标记 verified、backfilled、legacy。
- Schema v5 启用 SQLite 外键、启动一致性检查并固化删除语义。
- 增加容量、库存、重复取消和备份恢复测试。

## 兼容性

- 旧的 name、contact、author_name、author_contact 请求字段会被严格 JSON 边界拒绝。
- 未登录的报名、取消、发帖和回复请求返回 401。
- 无法匹配用户的历史记录保持可读但不能用于授权，需通过迁移报告人工处理。
