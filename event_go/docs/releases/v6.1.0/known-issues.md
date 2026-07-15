# v6.1.0 Known Issues

- G5.3 已新增 tenant-scoped 业务 API，但尚无自助入驻、邀请/成员管理和租户后台 UI，不能视为组织者无需开发者介入的完整运营产品。
- platform admin 业务路由仍在兼容期保留全局 Token；它们已进入 scoped Service，但 G5-R04 要到 G5.4 替换公开运营入口后才能关闭。
- G5.3 候选 Commit、完整本地门禁与远端 CI 尚待回填；证据完成前 G5-R05 保持未完成。
- 新管理前端要求 `/admin/session` 返回 `principal_type=platform_admin`；前后端整体回滚安全，若未来拆分部署必须为混合 N/N-1 增加单独兼容策略。
- v6.1 基础阶段每个 Organization 只允许一个 OrganizerProfile；真实多品牌需求需单独 ADR 和 Expand 迁移。
- 历史 Organization 状态为 unclaimed，必须通过后续受控认领流程绑定 owner，禁止人工按 contact 猜测并直接改库。
- Membership 目前只有 active/revoked 存储模型；成员撤销、重新邀请和所有权转移的完整 Service/API 与审计尚未实现。
- 邀请 Store 接受的是 Token 摘要；安全随机明文 Token 生成、邮件发送、限流和 HTTP 错误目录属于后续自助邀请切片。
- SQLite 和单实例部署仍是当前试点边界；多实例、真实支付和托管数据库决策未提前纳入本切片。
