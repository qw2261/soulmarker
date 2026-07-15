# v5.5.0 Release Notes

> 状态：In Progress

## 当前切片

- 启动入口只加载并校验一次 Config。
- Handler、JWT、管理员认证、CORS、取消时限、版本与日志使用显式注入配置。
- NewStore 返回 `(*Store, error)`，初始化失败不再触发库层 panic。
- 新增 RegistrationService，承载活动发布状态、可信报名身份、容量错误和取消截止规则。
- service 按用例定义最小 RegistrationRepository，不抽象全部 Store CRUD。
- 新增可注入 Clock 与 TokenManager，JWT 签发/验证不再固定在 Handler 实现中。
- 现有 API、Schema v5 和前端行为保持不变。

## 后续范围

讨论权限 service、DTO、OpenAPI 和业务错误码继续拆分为独立切片，不在本次结构调整中混合实现。
