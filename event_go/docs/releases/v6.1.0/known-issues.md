# v6.1.0 Known Issues

- Schema v12 只建立租户数据基础；现有运营 API 仍由 platform admin Token 保护，不能向不受信任的组织者开放。
- 业务资源仍通过 OrganizerProfile/Event 旧链路访问，尚无 tenant context；跨租户安全承诺必须等待 G5.2–G5.3 权限与 scope 测试。
- v6.1 基础阶段每个 Organization 只允许一个 OrganizerProfile；真实多品牌需求需单独 ADR 和 Expand 迁移。
- 历史 Organization 状态为 unclaimed，必须通过后续受控认领流程绑定 owner，禁止人工按 contact 猜测并直接改库。
- Membership 目前只有 active/revoked 存储模型；成员撤销、重新邀请和所有权转移的完整 Service/API 与审计尚未实现。
- 邀请 Store 接受的是 Token 摘要；安全随机明文 Token 生成、邮件发送、限流和 HTTP 错误目录属于后续自助邀请切片。
- SQLite 和单实例部署仍是当前试点边界；多实例、真实支付和托管数据库决策未提前纳入本切片。
