# v5.5.0 Release Notes

> 状态：In Progress

## 当前切片

- 启动入口只加载并校验一次 Config。
- Handler、JWT、管理员认证、CORS、取消时限、版本与日志使用显式注入配置。
- NewStore 返回 `(*Store, error)`，初始化失败不再触发库层 panic。
- 新增 RegistrationService，承载活动发布状态、可信报名身份、容量错误和取消截止规则。
- service 按用例定义最小 RegistrationRepository，不抽象全部 Store CRUD。
- 新增可注入 Clock 与 TokenManager，JWT 签发/验证不再固定在 Handler 实现中。
- 新增 DiscussionService，统一活动/帖子归属、报名资格、可信作者和帖子/回复写入规则。
- 报名状态查询通过 RegistrationService，报名 Handler 不再直接组合 Event 与 Registration。
- 新增 handler/dto，请求、响应 Envelope 和公开资源字段不再由 model 实体隐式决定。
- DTO 契约测试固定敏感字段排除、分页 Envelope 和空数组行为。
- 27 个业务操作同时提供 `/api/v1` 和兼容 `/api` 路径，前端默认调用 v1。
- OpenAPI 3.1 文档内嵌在制品中，通过 `/api/v1/openapi.json` 提供。
- 路由目录、operationId、requestBody 和 DTO Schema 字段由自动化测试双向校验。
- 现有 API、Schema v5 和前端行为保持不变。

## 后续范围

稳定业务错误码继续作为独立切片，不在本次 API 版本化变更中混合实现。
