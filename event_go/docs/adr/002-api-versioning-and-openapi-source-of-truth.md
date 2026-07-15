# ADR-002：API 版本化与 OpenAPI 单一事实源

> 状态：Accepted
> 日期：2026-07-15
> 适用：G3-R07 / API v1

## 背景

早期前端和外部调用直接使用 `/api`，路由、README 和 TypeScript 类型分别维护，存在契约漂移风险。商业化前需要稳定版本边界、机器可读规范以及可审计的兼容策略。

## 决策

1. `/api/v1` 是新客户端和前端的稳定入口。
2. `/api` 与 `/api/v1` 在一个共享路由目录中注册并调用同一 Handler，不维护两套实现。
3. 旧 `/api` 至少保留一个发布周期。移除前必须确认访问日志无有效旧客户端，发布说明需给出迁移窗口。
4. OpenAPI 3.1 JSON 与二进制一起构建，通过 `/api/v1/openapi.json` 提供。
5. 路由目录记录 method、path、operationId 和 request schema；测试双向验证实际路由与 OpenAPI 操作数量、operationId 和 requestBody。
6. OpenAPI Schema 字段与 Go HTTP DTO 的 JSON 字段通过反射契约测试校验。
7. 数据库实体不得作为 OpenAPI 或公开 API 字段的事实源。

## 后果

- 新客户端可以依赖明确的 v1 契约，旧客户端拥有受控迁移窗口。
- 路由、DTO 或规范的单边修改会使 CI 失败。
- 破坏性变更必须进入后续版本前缀，不能静默修改 v1。
- 兼容路由会短期增加路由数量，但不会产生重复业务代码。
