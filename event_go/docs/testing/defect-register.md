# 缺陷登记与 P0/P1 清零审计

> 状态：Active<br>
> 审计日期：2026-07-16<br>
> 适用候选：v6.0.0 / `codex/update_project`<br>
> Owner：qw2261

## 1. 使用规则

- 本文件是 v5.2 之后 P0/P1/P2 缺陷与风险豁免的唯一登记入口；历史任务文档中的优先级列表不等于当前缺陷状态。
- P0/P1 必须有复现、修复、回归测试和 Commit/CI 证据，关闭前发布结论保持 No-Go。
- P2 必须记录 Owner、最晚阶段和绕行方式；范围门槛、外部验收与缺陷分开登记。
- 新发现缺陷使用 `BUG-<Goal>-NNN`，对应回归测试使用 `REG-BUG-<Goal>-NNN`。

## 2. 当前开放缺陷

| ID | 等级 | 描述 | Owner | 状态 | 解除条件 |
|---|---|---|---|---|---|
| — | — | 当前没有已知开放 P0/P1/P2 代码缺陷 | — | Empty | 新发现问题立即登记 |

## 3. 已关闭 P0/P1 审计项

下列 ID 是本次正式审计为既有高风险修复补建的追溯编号，不替代原 Requirement ID。

| ID | 等级 | 风险与关闭条件 | 回归/证据 | 状态 |
|---|---|---|---|---|
| BUG-G1-001 | P0 | staging/production 管理认证或 JWT 配置缺失时不得 fail-open | `SEC-CONFIG-001`、`SEC-JWT-001`、配置/管理认证测试、v5.4 远端候选 | Closed |
| BUG-G1-002 | P1 | 用户不得通过请求体联系方式冒用报名、取消、发帖或回复身份；报名 PII 不得公开 | `SEC-IDENTITY-001`、`SEC-REGISTRATION-PII-001`、可信 user_id HTTP 旅程 | Closed |
| BUG-G1-003 | P1 | 错误 event 路径不得读取或修改其他活动的 ticket/post/reply | `SEC-RESOURCE-001`、嵌套资源归属回归 | Closed |
| BUG-G2-001 | P0 | 并发报名不得超容量、库存不得为负、重复取消不得重复退库存 | `CONC-REG-001`、Store race 门禁 | Closed |
| BUG-G4-001 | P1 | 退出、密码重置和恢复邮箱确认必须撤销旧会话；一次性 Token 不得复用或泄露账户存在性 | `SEC-AUTH-SESSION-001`、`SEC-PASSWORD-RESET-001`、`SEC-RECOVERY-EMAIL-001`、Runs `29389509516` / `29428887570` | Closed |
| BUG-G4-002 | P1 | 同一 Admission 重复或并发核销不得产生重复履约记录，已核销报名不得取消 | `CONC-CHECKIN-001`、`SEC-CHECKIN-001`、Run `29386146688` | Closed |
| BUG-G4-003 | P1 | 已移除帖子/回复不得重新出现在公开查询，治理原始证据必须保留 | `SEC-CONTENT-REPORT-001`、`DOM-CONTENT-MODERATION-001`、Run `29424593002` | Closed |
| BUG-G4-004 | P1 | `jwt/v5.2.1` 的 `GO-2025-3553` 可达路径允许分隔符洪泛造成过量内存分配 | `jwt/v5.2.2`、`REG-BUG-G4-001`、pinned `govulncheck@v1.6.0`；Commits `df4e66c` / `0790db9`；Run `29434256135` | Closed |
| BUG-G4-005 | P1 | Go `1.25.0` 标准库存在多个可达漏洞，首次安全候选被门禁阻断 | Run `29433701071` 正确失败；固定 Go `1.25.12`；Commit `0790db9`、Run `29434256135` | Closed |

## 4. 外部门槛与延期风险

这些项目不计入“开放代码缺陷”，但对应阶段在证据完成前仍为 No-Go。

| ID | 类型/等级 | 说明与当前绕行 | Owner | 最晚关闭 |
|---|---|---|---|---|
| GATE-G4-SMTP | 外部验收 | 真实 staging SMTP 的收件、链接跳转、重置/绑定和旧会话撤销尚未执行；本地适配器与 HTTP 旅程不能替代真实投递 | qw2261 | G4 Done 前 |
| GATE-G4-UAT | 外部验收 | 仍需两场独立内部或受控活动，无人工改库并保留签字证据 | qw2261 | G4 Done 前 |
| RISK-G5-ADMIN | P2 | G5.4 已用用户 JWT、成员角色和 tenant scope 替换组织者日常 Admin Token 入口；当前为 Local Candidate，远端门禁前保持开放，platform Token 兼容面只限应急/治理 | qw2261 | G5-R04/R06 远端证据回填 |
| RISK-G6-RATE | P2 | 密码重置只有用户级一分钟限流，尚无边缘/IP 限流与异步重试；受控试点限制公开流量 | qw2261 | G6-R04 |
| RISK-G6-NOTIFY | P2 | 临近提醒仅支持单实例；多实例前升级为租约单执行者或外部任务系统 | qw2261 | G6 部署扩容前 |
| RISK-G6-BROWSER | P2 | 自动矩阵只有 desktop Chromium 与 Pixel 7 Chromium；生产前补 Safari/iPhone WebKit 和目标 Android 真机 | qw2261 | G6 Go-Live 前 |
| RISK-G6-MEDIA | P2 | 活动封面依赖外部 HTTP/HTTPS URL，尚无自有存储、裁剪和失效巡检 | qw2261 | G6/G8 复审 |
| RISK-G6-BUNDLE | P3 | 前端主包约 544 KB，构建有 500 KB 提示；通知页等业务页已懒加载，不阻塞受控 G4 | qw2261 | G6 性能门禁前 |

## 5. 2026-07-16 审计证据

| 检查 | 结论 |
|---|---|
| GitHub 公开 open Issues | 0 条（审计时点） |
| TODO/FIXME/XXX/HACK 扫描 | 无业务遗留标记；仅启动边界使用预期的 `log.Fatalf` |
| `go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...` | 修复前命中可达 `GO-2025-3553`；升级后 0 个可达漏洞 |
| 首次安全候选远端 CI | Run `29433701071` 发现 Go `1.25.0` 标准库可达漏洞并正确阻断；工具链修复候选提升到 `1.25.12` |
| `npm audit --omit=dev` | 0 vulnerabilities（最近 v6.0 候选门禁） |
| 身份、权限、并发、迁移、核销、治理、通知 | 已由追溯矩阵中的 Critical/High Requirement 和远端 CI 覆盖 |

## 6. 清零结论

- P0：0 个开放。
- P1：0 个开放；`BUG-G4-004`、`BUG-G4-005` 已由 Run `29434256135` 关闭。
- G4 仍为 No-Go：`GATE-G4-SMTP` 与 `GATE-G4-UAT` 尚未完成。
