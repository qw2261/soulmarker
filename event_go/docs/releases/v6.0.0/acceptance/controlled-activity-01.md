# v6.0.0 受控活动验收记录 01

> 状态：通过（真实受控部署 + 公开 API 驱动；UI 可用性以候选代码的桌面/移动视口 e2e 佐证）<br>
> 规程：[controlled-activity-acceptance.md](../../../testing/controlled-activity-acceptance.md)<br>
> 场景：报名、讨论、导出与取消闭环

| 字段 | 值 |
|---|---|
| 候选完整 SHA | `53dfd0027f91d97fa2ce2664ceb999c9da982e91` |
| 部署/环境 | 本地受控部署：`APP_ENV=test`、`ADMIN_TOKEN=controlled-admin-token`、独立数据库 `data/controlled_acceptance.db`、`127.0.0.1:8080`，托管 `web/dist` SPA；对应 HEAD 干净工作树（除 `staging-smtp.md` 待提交）。真实 staging 部署（HTTPS 域名、独立数据库）属 G6 环境隔离项，待配置。 |
| 活动 ID / 脱敏名称 | 活动 ID=`1` / 名称「受控活动一·报名与取消闭环」（运营门店 ID=`1`「受控活动一运营方」） |
| 开始/结束时间 | 活动 `event_time=2026-08-26T19:40:54Z`（UTC）；验收执行：`2026-08-22 03:41（Asia/Shanghai）` |
| 执行人 / 复核人 | 执行人：自动化受控驱动脚本；复核人：待人工确认 |
| 参与人数 | 3 名注册参与者：参与者甲（报名+发帖+回复）、参与者乙（报名+回复）、未报名者丙（发帖被拒） |
| 原始证据引用 | 驱动响应 `/tmp/ca01/`（`reg_a.json`、`reg_b.json`、`post_a.json`、`reply_b.json`、`reply_a.json`、`post_c_denied.json`、`regs_export.csv`、`cancel_a.json`、`cancel_a_again.json`、`admission_a.json`、`tickets_after_reg.json`、`tickets_after_cancel.json`、`notifications_a_after_cancel.json` 等）。原始 PII 已脱敏，不进入 Git。 |
| CSV SHA-256 | `c84167de4002b02322a951413cf9fca6cc04371f201be28b7b65f42ffb01d294`（原始含脱敏后占位，见备注） |

## 验收结果

| 场景 | 结果 | 证据/备注 |
|---|---|---|
| 建店、建活动、建免费票 | 通过 | `POST /api/v1/organizers`→`201`（门店 id=1）；`POST /api/v1/events`→`201`（活动 id=1，`status=published`、`capacity=20`、`price=0`）；`POST /api/v1/events/1/tickets`→`201`（免费票 id=1，`price=0`、`stock=10`）。 |
| 两名以上参与者报名与凭证 | 通过 | 参与者甲/乙分别 `POST /api/v1/events/1/register`→`201`；Admission 凭证格式 `soulmark:admission:` + 32 位 hex（甲 `549b…47ef`、乙 `8b33…0995`，脱敏），`status=active`。 |
| 报名通知 | 通过 | `GET /api/v1/me/notifications`（甲、乙）各返回 1 条 `registration_confirmed`（标题「报名成功」，含活动时间/地点/票种）。 |
| 讨论权限、发帖与回复 | 通过 | 甲 `POST /api/v1/events/1/posts`→`201`；乙 `POST …/posts/1/replies`→`201`；甲再回复→`201`。未报名者丙发帖→`403 PARTICIPATION_REQUIRED`（「未报名该活动，无法参与讨论」）。 |
| 报名列表与安全 CSV 导出 | 通过 | `GET /api/v1/events/1/registrations`（admin）→`total=2`；`GET …/registrations/export`→CSV 带 UTF-8 BOM、列 `ID,姓名,联系方式,票种,报名时间`、含 2 行；`spreadsheetSafeCell` 将 `= + - @` 前缀转义以防公式注入；其 SHA-256 见上。 |
| 取消、库存单次恢复与凭证吊销 | 通过 | 甲 `DELETE /api/v1/events/1/register`→`200`「已取消报名」；库存 `stock=10`→报名 2 人后 `8`→取消后 `9`（仅恢复一次）；甲 `GET …/admission`→`status=revoked`（`revoked_at` 已置位）；通知列表新增 `registration_cancelled`（「报名已取消」）。 |
| 重复取消幂等 | 通过 | 甲再次 `DELETE /api/v1/events/1/register`→`404 REGISTRATION_NOT_FOUND`；库存仍 `9`（未再变化）、`GET …/registrations` 仍仅乙 1 条、无重复业务记录（无副作用，符合幂等语义）。 |
| 桌面/移动可用性 | 通过（佐证） | 候选代码的 Playwright e2e（desktop-chromium + mobile-chromium/Pixel 7）覆盖报名→讨论→取消→通知中心→我的报名，均断言 `document.documentElement.scrollWidth <= window.innerWidth`（无阻断性横向溢出），并向测试报告附截图；该 e2e 用独立 `:memory:` 环境复现，不污染受控部署。 |

## 缺陷与结论

- P0：无
- P1：无
- P2：重复取消返回 `404 REGISTRATION_NOT_FOUND`（语义上为幂等无副作用；如需严格 200 幂等，可作为低优先级优化项跟踪）
- Go / No-Go：Go（业务闭环全部通过；唯一待补为真实 staging 部署，属 G6 环境项）
- 执行人确认：待填写
- 复核人确认：待填写
