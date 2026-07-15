# v6.1.0 Known Issues

- G5.2 已提供只读租户 context 与 capability，但现有运营业务 API 仍由 platform admin Token 保护，不能向不受信任的组织者开放。
- 业务资源仍通过 OrganizerProfile/Event 旧链路访问，尚未强制 tenant scope；当前跨租户拒绝证据只覆盖授权 session，完整数据隔离必须等待 G5.3 资源级 Store/Service 测试。
- capability 是后续授权入口，不代表对应业务功能已开放；finance/export、checker/checkin 等能力只有在 G5.3–G5.4 接入具体路由后才可使用。
- 新管理前端要求 `/admin/session` 返回 `principal_type=platform_admin`；前后端整体回滚安全，若未来拆分部署必须为混合 N/N-1 增加单独兼容策略。
- v6.1 基础阶段每个 Organization 只允许一个 OrganizerProfile；真实多品牌需求需单独 ADR 和 Expand 迁移。
- 历史 Organization 状态为 unclaimed，必须通过后续受控认领流程绑定 owner，禁止人工按 contact 猜测并直接改库。
- Membership 目前只有 active/revoked 存储模型；成员撤销、重新邀请和所有权转移的完整 Service/API 与审计尚未实现。
- 邀请 Store 接受的是 Token 摘要；安全随机明文 Token 生成、邮件发送、限流和 HTTP 错误目录属于后续自助邀请切片。
- SQLite 和单实例部署仍是当前试点边界；多实例、真实支付和托管数据库决策未提前纳入本切片。
