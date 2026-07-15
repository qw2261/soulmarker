# ADR-004：Organization 租户边界与渐进迁移

> 状态：Accepted<br>
> 日期：2026-07-16<br>
> 适用：G5-R01–R05 / Schema v12 及后续租户化切片

## 背景

现有 `Organizer` 同时被理解为公开门店资料、活动归属和潜在管理主体，但它没有用户成员、角色或授权边界。直接把 owner、计费和权限字段继续堆进 `organizers` 会把公开资料与安全主体绑定，并迫使一次性改写全部 API、Store、前端和历史数据。历史 `contact` 也不能证明某个已注册用户就是门店所有者。

G5 需要在不破坏现有 `/organizers` API、活动外键和 N/N-1 发布兼容的前提下，先建立稳定租户身份，再逐步切换授权与业务资源 scope。

## 决策

1. `Organization` 是授权、审计和未来计费的租户；`OrganizerProfile` 是公开品牌/门店资料。v6.1 基础阶段保持一对一，未来只有在真实多品牌需求成立后才通过新 ADR 放开一对多。
2. 保留 `organizers` 表、`/organizers` API 和 `type Organizer = OrganizerProfile` 兼容层；租户字段不自动暴露到公开 JSON。
3. Schema v12 对每个历史 OrganizerProfile 创建同 ID 的 `unclaimed` Organization；id=0 系统占位资料对应 `system` Organization。迁移不根据 contact、名称或用户数据猜测 owner。
4. 新的自助组织创建必须在同一事务写入 active Organization、OrganizerProfile 和唯一 active owner Membership；任一步失败则全部回滚。
5. 角色集合固定为 owner、admin、editor、checker、finance。v12 只固化存储约束和邀请边界，具体资源权限矩阵在 G5.2 集中策略层实现：owner 负责所有权与全量管理，admin 负责非所有权运营与成员邀请，editor 负责内容/活动配置，checker 只负责核销，finance 负责财务与受控导出。
6. 每个组织最多一个 active owner。普通邀请不能授予 owner；所有权转移必须由后续显式事务流程完成，并留下审计。
7. 只有 active Organization 的 active owner/admin 可以创建邀请。邀请邮箱规范化后存储，Token 只保存摘要；接受者的登录邮箱或已验证 recovery email 必须精确匹配。邀请单次消费，过期 pending 记录在新邀请创建或接受时转为 expired。
8. Schema v12 是 Expand-only。`organizers.organization_id` 默认 0；pre-v12 应用省略该列插入时，数据库触发器创建 unclaimed Organization 并回填；pre-v12 应用删除资料时，删除触发器暂停对应 Organization。部分唯一索引只约束非零 `organization_id`，避免破坏旧 INSERT。
9. G5 按“基础模型 → 授权内核 → 资源 tenant scope → 自助运营 → 审计与试点”切片推进。全局 Admin Token 在 G5.1 仍是 platform admin，不能被解释为租户角色；在权限矩阵与跨租户测试完成前，不切断旧管理入口。

## 被否决方案

- **直接给 organizers 增加 owner_user_id 和 role 字段**：无法表达多成员、邀请和未来多品牌，同时混淆公开资料与安全主体。
- **按历史 contact 自动认领 owner**：公开联系方式可能是客服邮箱、电话或过期数据，自动认领会造成租户接管漏洞。
- **一次性重写全部管理 API 和资源外键**：迁移面过大，无法保持 N/N-1，也难以对跨租户越权进行逐层验证。
- **在 v12 直接删除旧 `/organizers` 写路径**：会让应用回滚和滚动发布失败，不符合 G6 的迁移先行要求。

## 后果

- 授权主体、公开展示和未来计费边界清晰，历史数据无需破坏性重建。
- 基础阶段会暂时同时存在 platform admin 与尚未接入 API 的租户角色；文档和状态必须明确“已建模”不等于“已租户化”。
- 一对一 Profile 限制降低首阶段复杂度，但未来若出现多品牌组织，需要新的 Expand 迁移和 ADR。
- 兼容触发器承担 N/N-1 写保护；每次修改 organizers 迁移或删除语义时必须有触发器存在性和旧应用写入回归。
- SQLite 仍适合当前单实例试点；进入真实支付前的数据库选择继续由 G7 决策，不由本 ADR 提前扩大范围。

## 回滚与退出条件

应用可回滚到 pre-v12 版本并忽略新增表；不得为回滚删除 Organization、Membership 或 Invitation 数据。旧应用创建/删除门店由触发器维持租户一致性。若必须完全撤销 Schema v12，只能在确认没有 v12 新写数据后停机并恢复升级前完整数据库备份。

进入 G5.3 资源租户化前必须满足：G5.2 权限矩阵全绿、platform_admin 与租户角色语义分离、所有 Store/Service 资源入口都有 tenant 参数或受控上下文、跨租户负向测试可证明 403/404。最晚在 G5.2 结束或 2026-08-15 复审本 ADR。
