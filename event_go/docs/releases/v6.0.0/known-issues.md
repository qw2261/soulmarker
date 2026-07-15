# v6.0.0 Known Issues

- 当前运营核销使用全局 Admin Token；租户角色、核销员身份和完整 actor 审计归 G5。
- 当前页面支持扫码枪或粘贴凭证，尚未集成移动浏览器摄像头扫码。
- “我的活动”完整保留免费 Admission 的已取消历史；没有 Admission 的付费报名仍沿用 Registration 列表语义。
- “我的活动”当前在前端合并 Admission 与 Registration 两路分页；大量混合历史可能出现总数偏小或跨页重复，统一活动查询模型归 G4-R01 收敛。
- 前端主包约 1.03 MB，仍有 Vite chunk size 告警，需在 G4-R01/R04 体验收敛时拆分 Element Plus 相关加载。
- 密码重置、通知、内容治理和运营导出尚未实现。
