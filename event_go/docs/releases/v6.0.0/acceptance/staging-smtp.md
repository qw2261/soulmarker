# v6.0.0 Staging SMTP 验收记录

> 状态：本地受控 e2e 验证（主体通过）<br>
> 规程：[staging-smtp-acceptance.md](../../../testing/staging-smtp-acceptance.md)

| 字段 | 值 |
|---|---|
| 候选完整 SHA | `53dfd0027f91d97fa2ce2664ceb999c9da982e91` |
| staging 部署/制品 | 本地受控 e2e：`APP_ENV=test` 应用（`127.0.0.1:18080`）+ 本地 SMTP 捕获箱（SMTP `12525` / API `12526`），对应 HEAD 干净工作树。真实 staging 部署（HTTPS 域名、独立数据库、真实 SMTP 凭证）属 G6 环境隔离项，待配置。 |
| 执行时间 | 2026-08-22（Asia/Shanghai） |
| 执行人 / 复核人 | 执行人：自动化驱动脚本；复核人：待人工确认 |
| 脱敏测试账户 | 原账号 `b-***@example.com`，恢复邮箱 `br-***@example.com`（密码与原始 Token 不记录） |
| 原始证据引用 | 本地脚本输出 `/tmp/smtp_evidence.txt`（密码重置 A2–A6）、`/tmp/recovery_evidence.txt`（恢复邮箱 B2–B6）、`/tmp/ratelimit_evidence.txt`（一分钟限流） |

## 结果

| 场景 | 结果 | 证据/备注 |
|---|---|---|
| 已知邮箱收件与链接 | 通过（本地） | `POST /api/v1/auth/password-reset/request` → `202`；邮件 From=`"Soulmark E2E" <no-reply@example.com>`、Subject=`Soulmark password reset`、正文含 `reset-password?token=…` 与过期时间。链接为 `http://127.0.0.1:18080/…`，HTTPS 域名待真实 staging 验证。 |
| 密码重置与旧会话撤销 | 通过 | 重置确认 → `200`；旧密码登录 `401`、新密码登录 `200`；已登录会话重置后访问受保护接口 `401`（`AuthVersion` 撤销生效）。 |
| Token 单次使用、过期与替代 | 通过 | 单次复用 → `400 PASSWORD_RESET_TOKEN_INVALID`（实测）；过期与替代由单测锁定：过期 token 拒绝（`store_authentication_test.go:76`）、重新申请作废旧 token（`store_authentication_test.go:34`）。 |
| 未知账户枚举安全 | 通过 | 未注册邮箱申请重置 → `202` 统一文案，且捕获箱 **0 封邮件**。 |
| phone-only 恢复邮箱绑定 | 通过（本地） | 错误密码绑定 → `401 INVALID_CREDENTIALS`；正确密码 → `202`；验证邮件 Subject=`Soulmark recovery email verification`、含 `verify-recovery-email?token=…`；确认 → `200`，提示旧会话失效。 |
| 保留原账号登录 | 通过 | 确认恢复邮箱后，原账号仍以原密码登录 → `200`；但此前旧 token 访问受保护接口 → `401`（旧会话撤销）。 |
| 恢复邮箱再次重置密码 | 通过 | 以恢复邮箱申请 → `202` 且邮件到达；确认重置 → `200`；重置后旧密码登录 → `401`。 |
| 一分钟限流 | 通过 | 本地实测：首次申请 `202` 发邮件，**立即二次申请 `202` 但邮件计数不变（不发）**（`/tmp/ratelimit_evidence.txt`）；store 层 1 分钟窗口返回 `ErrPasswordResetRateLimit`（`store_user.go:227-229`）+ 单测覆盖；handler 仅 `slog.Warn("password reset request rate limited")`。 |
| 日志与证据脱敏 | 通过 | staging/prod 使用 `SMTPPasswordResetSender`，不打印 token/email（`notification/password_reset.go`）；handler 日志为固定文案（`handler_auth.go:140-142`）。开发模式 `LogPasswordResetSender` 打印链接属本地场景，非生产配置。 |

## 结论

- 本地受控 e2e 验证已覆盖 SMTP 核心闭环：真实邮件投递、密码重置、恢复邮箱绑定、旧会话撤销、Token 单次/过期/替代、未知账户枚举安全、一分钟限流、日志脱敏，**全部通过**。
- 唯一待补：**真实 staging 部署**（HTTPS `PUBLIC_BASE_URL`、独立数据库、真实 SMTP 凭证），属 G6 环境隔离/密钥管理项，非代码缺陷。
- Go / No-Go：**条件通过**（本地受控链路全项通过；转真实 Go 前需完成真实 staging 部署并复测 HTTPS 域名与真实 SMTP 投递）
- 新增缺陷：无
- 执行人确认：待填写
- 复核人确认：待填写
