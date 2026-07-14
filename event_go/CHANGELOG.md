# Changelog

本项目从 v5.2 起采用语义化版本，并按 Added、Changed、Fixed、Security、Migration、Breaking 分类记录变化。

## Unreleased

### Added

- 生产与集成测试共用的统一 Router。
- 当前登录用户的单活动报名状态接口。
- GitHub Actions 后端和前端基础质量门禁。
- 测试策略、追溯矩阵和版本发布证据模板。
- Docker 构建上下文忽略规则。
- 版本化 SQLite 迁移框架与 `schema_migrations` 记录表。
- 空库、旧库、重复迁移和失败返回的真实文件数据库测试。

### Changed

- 报名名单路由进入管理员认证边界。
- 帖子详情使用明确的 PostDetail 响应 DTO。
- 活动、门票、门店更新接口增加字段校验。
- 门票和帖子嵌套路由校验活动归属。
- 所有 JSON 写请求限制为 1 MiB，并拒绝未知字段和尾随 JSON。
- API 错误响应增加稳定的 `error_code` 字段。

### Fixed

- 修复帖子详情页错误解析后端响应的问题。
- 修复活动报名人数超过 100 时前端无法判断当前用户报名状态的问题。

### Security

- 防止通过错误活动路径读取或修改其他活动的门票和帖子。
- 公开页面不再通过报名名单接口扫描参与者联系方式。
- staging/production 缺少安全配置时拒绝启动。
- HTTP Server 增加读取、写入、Header 和空闲连接超时。
- 增加 CSP、frame、MIME、Referrer 和 Permissions Policy 安全响应头。
- HTTP 500 与健康检查不再向客户端返回 SQL 或数据库内部错误。

### Migration

- Schema 进入版本 3；为 registrations、posts、replies 增加 nullable `user_id`。
- registrations 增加仅对非空 `user_id` 生效的活动内唯一索引。
- 迁移为纯 Expand，现有联系信息授权路径暂时保持兼容。
