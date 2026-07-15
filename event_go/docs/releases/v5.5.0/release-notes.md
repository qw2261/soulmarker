# v5.5.0 Release Notes

> 状态：In Progress

## 当前切片

- 启动入口只加载并校验一次 Config。
- Handler、JWT、管理员认证、CORS、取消时限、版本与日志使用显式注入配置。
- NewStore 返回 `(*Store, error)`，初始化失败不再触发库层 panic。
- 现有 API、Schema v5 和前端行为保持不变。

## 后续范围

G3-R03 至 G3-R08 将按 service、DTO、可测试依赖、OpenAPI 和业务错误码拆分为独立切片，不在本次结构调整中混合实现。
