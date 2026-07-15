# 需求与测试追溯矩阵

> 状态：In Progress
> 更新规则：每个行为变更必须在同一 PR 更新对应行

| Requirement | 版本 | 风险 | 验收条件 | 实现 | 自动化测试 | E2E/手工 | 证据 | 状态 |
|---|---|---|---|---|---|---|---|---|
| G0-R04 | v5.2 | High | 生产和集成测试使用同一 Router | internal/handler/router.go | IT-ROUTER-001 | — | v5.2 test-report | Local Pass |
| G1-R01 | v5.3 | High | PostDetail 正确展示 post 与 replies | PostDetailResp、PostDetail.vue | CT-API-POST-001 | 待 G4 | v5.2 test-report | Local Pass |
| G1-R02 | v5.3 | High | 错误 event 路径不能访问 ticket/post | Handler scope checks | SEC-RESOURCE-001 | — | v5.2 test-report | Local Pass |
| G1-R03 | v5.3 | High | 更新接口拒绝空名称、非法时间、负数 | Event/Ticket/Organizer handlers | IT-API-VALIDATION-001 | — | v5.2 test-report | Local Pass |
| G1-R04 | v5.3 | High | 报名名单只允许管理员读取 | Router AdminAuth | SEC-REGISTRATION-PII-001 | — | v5.2 test-report | Local Pass |
| G1-R05 | v5.3 | High | 登录用户无需扫描名单即可查询报名状态 | Registration status API | IT-API-REGSTATUS-001 | 待 G4 | v5.2 test-report | Local Pass |
| G1-R06 | v5.3 | High | staging/production 缺管理令牌时拒绝启动 | Config.Validate | SEC-CONFIG-001 | — | v5.2 test-report | Local Pass |
| G1-R07 | v5.3 | High | 默认 JWT 密钥被拒绝，无效 Token 返回 401 | Config.Validate、UserAuth | SEC-JWT-001 | — | v5.2 test-report | Local Pass |
| G1-R08 | v5.3 | High | HTTP 超时、明确 CORS、1 MiB 请求限制、未知字段与尾随 JSON 拒绝、安全响应头 | main、Config、Handler | SEC-HTTP-001 | — | v5.2 test-report | Local Pass |
| G1-R09 | v5.3 | High | 所有错误响应有稳定 error_code，HTTP 500 不泄露 SQL/数据库细节 | APIResp、Handler | SEC-ERROR-001 | — | v5.2 test-report | Local Pass |
| G2-R01 | v5.4 | High | 迁移按版本、事务执行并记录 schema_migrations，错误返回启动入口 | internal/store/store.go | MIG-SCHEMA-001 | — | v5.2 test-report | Local Pass |
| G2-R02 | v5.4 | High | registrations.user_id 可空，非空时活动内唯一 | migration v3 | MIG-IDENTITY-EXPAND-001 | — | v5.2 test-report | Local Pass |
| G2-R03 | v5.4 | High | 新增讨论写入 user_id，报名校验仅使用 user_id | Handler、Store、migration v3-v5 | SEC-IDENTITY-001 | — | v5.4 test-report | Local Pass |
| G2-R04 | v5.4 | High | 当前用户可分页查询自己的报名，单活动状态按 user_id 返回 | /api/me/registrations、registration status | IT-ME-REG-001 | MyRegistrations.vue build | v5.4 test-report | Local Pass |
| G2-R05 | v5.4 | Critical | 报名、取消、发帖、回复必须登录且请求体不能指定身份 | Handler DTO、requireUser | SEC-IDENTITY-001 | 前端表单 build | v5.4 test-report | Local Pass |
| G2-R06 | v5.4 | High | 精确回填匹配用户，未匹配记录标记 legacy 并输出人工清单 | migration v4、identity report | MIG-IDENTITY-BACKFILL-001 | 管理接口待 UI | v5.4 test-report | Local Pass |
| G2-R07 | v5.4 | Critical | 每个 SQLite 连接启用外键，启动检查孤儿；删除语义有 ADR 和测试 | Store、ADR-001 | DB-FK-001 | — | v5.4 test-report | Local Pass |
| G2-R08 | v5.4 | High | 真实文件空库、旧库、重复、失败和备份恢复通过 | migration_test.go | MIG-LEGACY-001 | — | v5.4 test-report | Local Pass |
| G2-R09 | v5.4 | Critical | 并发报名不超容量、库存不为负、重复取消只退一次 | Store mutex + transaction | CONC-REG-001 | — | v5.4 test-report | Local Pass |
| G3-R01 | v5.5 | High | Config 仅在启动入口加载校验，Handler 与中间件使用同一注入实例 | main、Handler、Router、Middleware | ARCH-CONFIG-001 | — | v5.5 test-report | Remote Pass |
| G3-R02 | v5.5 | High | NewStore 初始化失败返回 error，不 panic 或退出进程 | internal/store/store.go | ARCH-STORE-001 | — | v5.5 test-report | Remote Pass |
| G3-R03 | v5.5 | High | 报名/取消和讨论写入的跨实体规则由 application service 承载，Handler 只处理 HTTP 边界 | RegistrationService、DiscussionService、Handler | ARCH-SERVICE-001 | — | v5.5 test-report | Remote Pass |
| G3-R04 | v5.5 | High | Handler 不直接解码 model 请求或序列化数据库实体，公开 JSON 字段由 DTO 契约固定 | handler/dto、全部 Handler | CT-DTO-001 | 前端 TypeScript build | v5.5 test-report | Remote Pass |
| G3-R05 | v5.5 | Medium | Repository 接口仅包含报名用例需要的 GetEvent、Register、Cancel 方法 | service.RegistrationRepository | ARCH-REPO-001 | — | v5.5 test-report | Remote Pass |
| G3-R06 | v5.5 | High | 业务时间和 JWT 签发/验证可注入并可使用确定时间测试 | Clock、TokenManager、Handler Dependencies | ARCH-DEPS-001 | — | v5.5 test-report | Remote Pass |
| G3-R07 | v5.5 | Critical | 所有业务操作同时注册 v1/兼容路由，OpenAPI operationId、request schema 与 DTO 字段双向一致 | Router catalog、OpenAPI 3.1、ADR-002 | CT-OPENAPI-001 | 前端使用 /api/v1 | v5.5 test-report | Remote Pass |

后续新增 Requirement 时，不得只写实现文件；必须同时填写可验证验收条件和测试 ID。
