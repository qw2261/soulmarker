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
| G2-R03 | v5.4 | High | posts/replies 增加可空 user_id，后续授权切换到用户 ID | migration v3 | MIG-IDENTITY-EXPAND-001 | — | v5.2 test-report | Partial: Expand only |
| G2-R08 | v5.4 | High | 真实文件空库、旧库、重复执行和迁移失败可诊断 | migration_test.go | MIG-LEGACY-001 | 备份恢复演练待完成 | v5.2 test-report | Partial |

后续新增 Requirement 时，不得只写实现文件；必须同时填写可验证验收条件和测试 ID。
