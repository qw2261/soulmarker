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
- `/me/activities` 统一参与时间线 API，合并 Admission 历史与没有 Admission 的 Registration。
- 用户服务端退出、一次性密码重置申请/确认 API，以及忘记密码和设置新密码页面。
- SMTP、开发日志和测试丢弃三种密码重置通知适配器。
- 平台管理员会话校验 API，以及门店、活动、票种、报名导出和核销的一体化运营工作台。
- 防表格公式注入、UTF-8 BOM 和跨页聚合的报名 CSV 导出。
- 持久化站内通知、通知中心、导航未读徽标，以及报名成功、取消、活动变更和临近提醒闭环。
- 当前用户通知分页、未读计数、单条已读和全部已读 API；OpenAPI、DTO 与错误码契约同步扩展。
- Schema v12 Organization、OrganizationMember 与 OrganizationInvitation 租户基础模型。
- 组织与公开 OrganizerProfile 原子创建、用户组织查询，以及邀请创建/接受的 Store 能力。
- [ADR-004](docs/adr/004-organization-tenant-boundary-and-migration.md) 固化租户边界、历史 unclaimed 回填、角色和分阶段切换策略。

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
- 前端从全量安装 Element Plus 改为显式注册实际使用组件，运营页面加入后主 JS 约 490 KB。
- 新注册收敛为邮箱，密码策略统一为 8–72 字节；历史 contact 登录继续兼容。
- 前端受保护路由、Token 过期安全回跳与服务端退出形成统一会话状态闭环。
- 后台登录与每次后台路由进入均服务端复验 Token；活动编辑允许重新选择门店。
- 删除确认统一使用中文按钮，后台表格在移动端使用局部横向滚动。
- 业务通知按 ADR-003 以站内持久化记录为事实源；外部业务邮件/短信和多实例投递延后到生产基础设施阶段。
- `Organizer` 保留为 `OrganizerProfile` 的源码兼容别名，现有 `/organizers` API 暂不暴露租户授权字段。

### Fixed

- 修复帖子详情页错误解析后端响应的问题。
- 修复活动报名人数超过 100 时前端无法判断当前用户报名状态的问题。
- 修复“我的活动”双路分页导致混合历史总数偏小、跨页重复和排序不一致的问题。

### Security

- 防止通过错误活动路径读取或修改其他活动的门票和帖子。
- 公开页面不再通过报名名单接口扫描参与者联系方式。
- staging/production 缺少安全配置时拒绝启动。
- HTTP Server 增加读取、写入、Header 和空闲连接超时。
- 增加 CSP、frame、MIME、Referrer 和 Permissions Policy 安全响应头。
- HTTP 500 与健康检查不再向客户端返回 SQL 或数据库内部错误。
- Admission 使用 128 位加密随机凭证；跨活动、已吊销凭证拒绝核销。
- JWT 增加认证版本；退出或密码重置会撤销该用户全部旧 Token，旧无版本 JWT 按版本 1 兼容。
- 密码重置使用 256 位随机 Token，数据库只存 SHA-256 摘要，并实施一次性消费、过期、替代与用户级一分钟限流。
- staging/production 缺 HTTPS 公开地址、SMTP 配置或合法重置有效期时拒绝启动。
- 报名 CSV 对 `= + - @` 开头字段增加公式注入防护；无效管理 Token 不得仅凭本地存储进入后台。
- 通知查询和已读操作只使用 JWT 用户 ID 作用域，跨用户通知统一隐藏为 `NOTIFICATION_NOT_FOUND`；通知正文不保存联系方式、恢复令牌或入场凭证。
- 升级 `github.com/golang-jwt/jwt/v5` 到 v5.2.2，修复可达的 `GO-2025-3553` 分隔符洪泛过量内存分配问题，并在 CI 增加 pinned `govulncheck` 门禁。
- 将 Go 工具链基线从 1.25.0 提升到 1.25.12，消除首次远端漏洞扫描发现的可达标准库漏洞。
- 组织邀请只允许 active owner/admin 创建，禁止普通邀请授予 owner；接受时必须匹配登录邮箱或已验证恢复邮箱，且 Token 单次使用并受过期约束。
- 每个组织通过数据库部分唯一索引限制为最多一个 active owner。

### Migration

- Schema 进入版本 3；为 registrations、posts、replies 增加 nullable `user_id`。
- registrations 增加仅对非空 `user_id` 生效的活动内唯一索引。
- Schema v3 迁移为纯 Expand；Schema v5 完成 user_id 授权切换并关闭联系方式兼容路径。
- Schema 升级到版本 5：精确回填可匹配身份，无法匹配的记录标记为 legacy。
- 新写入记录标记为 verified；管理员可以查询各实体迁移总量和 legacy 清单。
- Schema 升级到版本 6：新增 Admission/Checkin 外键、索引和 Checkin 不可变触发器。
- Schema 升级到版本 7：新增用户认证版本、密码重置 Token 表、索引和新用户认证版本触发器。
- Schema 升级到版本 11：新增通知表、用户时间线/未读/活动索引和全局唯一幂等键；活动删除后历史通知保留并将引用置空。
- Schema 升级到版本 12：历史门店一对一回填为 unclaimed Organization，不猜测 owner；新增成员、邀请、角色/FK/索引，并以创建/删除触发器保持 pre-v12 `/organizers` 写兼容。
