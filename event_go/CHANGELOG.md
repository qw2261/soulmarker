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
- 当前用户报名列表 API 与前端“我的报名”页面。
- 管理员身份迁移统计和 legacy 人工处理清单 API。
- 容量、门票库存和重复取消的真实并发测试。
- Schema v6 Admission/Checkin 模型、用户凭证 API、幂等核销 API 和运营审计列表。
- 用户活动详情与“我的活动”二维码凭证，运营报名页核销工作台。
- Vitest 组件测试、Playwright 桌面/移动核心旅程和 CI 浏览器证据。

### Changed

- 报名名单路由进入管理员认证边界。
- 帖子详情使用明确的 PostDetail 响应 DTO。
- 活动、门票、门店更新接口增加字段校验。
- 门票和帖子嵌套路由校验活动归属。
- 所有 JSON 写请求限制为 1 MiB，并拒绝未知字段和尾随 JSON。
- API 错误响应建立 26 个稳定业务 `error_code` 的集中目录，HTTP 状态与业务原因独立选择，数字 `code` 保持兼容。
- 报名、取消、发帖和回复统一使用持久化 user_id 授权，不再接受联系方式回退。
- SQLite 固定单连接写入模型并在启动时强制启用外键和执行一致性检查。
- 删除门票时保留报名票种快照并解除 ticket_id 引用；删除门店使用隐藏的系统占位门店保持历史活动完整。
- Config 只在启动入口加载和校验一次，并显式注入 Handler、JWT、管理员认证、CORS 与日志配置。
- NewStore 显式返回初始化 error，库层不再通过 panic 处理启动失败。
- 报名与取消的跨实体规则进入 RegistrationService，Handler 只负责 HTTP 边界和错误映射。
- JWT signer/verifier 与业务 Clock 改为显式依赖，取消截止边界和 Token 时间可确定性测试。
- 讨论写入进入 DiscussionService，活动/帖子归属、报名资格和可信作者身份不再由 Handler 跨实体编排。
- HTTP 请求、响应 Envelope 与公开资源 DTO 从数据库实体分离，所有 Handler 通过显式映射固定 JSON 字段集合。
- 新增 `/api/v1` 稳定路由、内嵌 OpenAPI 3.1 文档和双向路由/Schema 契约测试；前端默认切换到 v1。
- OpenAPI 固定 `ErrorResponse.error_code` 枚举，并与 Go 错误目录自动双向校验；未知错误码安全降级为通用内部错误。
- 免费报名与 Admission 在同一事务内创建；取消会吊销凭证，已核销报名不可取消。
- 移动导航改为抽屉，“我的报名”升级为显示参与状态与凭证的“我的活动”。

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
- Admission 使用 128 位加密随机凭证；跨活动、已吊销凭证拒绝核销。

### Migration

- Schema 进入版本 3；为 registrations、posts、replies 增加 nullable `user_id`。
- registrations 增加仅对非空 `user_id` 生效的活动内唯一索引。
- Schema v3 迁移为纯 Expand；Schema v5 完成 user_id 授权切换并关闭联系方式兼容路径。
- Schema 升级到版本 5：精确回填可匹配身份，无法匹配的记录标记为 legacy。
- 新写入记录标记为 verified；管理员可以查询各实体迁移总量和 legacy 清单。
- Schema 升级到版本 6：新增 Admission/Checkin 外键、索引和 Checkin 不可变触发器。
