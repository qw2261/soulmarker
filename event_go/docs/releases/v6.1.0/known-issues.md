# v6.1.0 Known Issues

- G5.4 自助入驻、邀请/成员管理和租户工作台已由 Commit `8bc0c50` / Run `29447383045` 通过远端门禁；剩余多租户风险转入 G5.5 审计、PII 和试点。
- platform admin 业务路由仍在兼容期保留全局 Token，只作为应急和治理入口；下线前需要使用量审计和明确弃用窗口。
- 后续风险集中在 G5.5 的 actor/tenant/request_id 审计、PII 脱敏、所有权转移与真实三组织试点。
- 新管理前端要求 `/admin/session` 返回 `principal_type=platform_admin`；前后端整体回滚安全，若未来拆分部署必须为混合 N/N-1 增加单独兼容策略。
- v6.1 基础阶段每个 Organization 只允许一个 OrganizerProfile；真实多品牌需求需单独 ADR 和 Expand 迁移。
- 历史 Organization 状态为 unclaimed，必须通过后续受控认领流程绑定 owner，禁止人工按 contact 猜测并直接改库。
- Membership 目前只有 active/revoked 存储模型；成员撤销和重新邀请已实现，所有权转移与完整审计尚未实现。
- 邀请已具备安全随机 Token、SMTP 投递和稳定 HTTP 错误目录；限流、异步重试、退信与真实 staging SMTP 验收仍未完成。
- SQLite 和单实例部署仍是当前试点边界；多实例、真实支付和托管数据库决策未提前纳入本切片。
