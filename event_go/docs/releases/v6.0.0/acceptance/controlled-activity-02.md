# v6.0.0 受控活动验收记录 02

> 状态：通过（真实受控部署 + 公开 API 驱动；UI 可用性以候选代码的桌面/移动视口 e2e 佐证）<br>
> 规程：[controlled-activity-acceptance.md](../../../testing/controlled-activity-acceptance.md)<br>
> 场景：活动变更、提醒、核销与治理闭环

| 字段 | 值 |
|---|---|
| 候选完整 SHA | `53dfd0027f91d97fa2ce2664ceb999c9da982e91` |
| 部署/环境 | 本地受控部署：`APP_ENV=test`、`ADMIN_TOKEN=controlled-admin-token`、独立数据库 `data/controlled_acceptance.db`、`127.0.0.1:8080`，托管 `web/dist` SPA；通知调度 `NOTIFICATION_REMINDER_HOURS=72`、`NOTIFICATION_SCAN_INTERVAL_SECONDS=5`。真实 staging 部署（HTTPS 域名、独立数据库）属 G6 环境隔离项，待配置。 |
| 活动 ID / 脱敏名称 | 活动 ID=`2` / 名称「受控活动二·变更与履约闭环」（运营主体 ID=`2`「受控活动二运营方」） |
| 开始/结束时间 | 活动 `event_time=2026-08-23T07:44:11Z`（UTC）；验收执行：`2026-08-22 03:44–03:46（Asia/Shanghai）` |
| 执行人 / 复核人 | 执行人：自动化受控驱动脚本；复核人：待人工确认 |
| 参与人数 | 3 名报名参与者（与活动一甲/乙/丙独立）：参与者丁（报名+发帖+核销）、参与者戊（报名+取消）、参与者己（报名+举报+核销闭环） |
| 原始证据引用 | 驱动响应 `/tmp/ca02/`（`org_create.json`、`ev_create.json`、`tk_create.json`、`user_d.json`、`user_e.json`、`reg_d.json`、`reg_e.json`、`post_d.json`、`report_spam.json`、`resolve_report.json`、`admin_reports_after.json`、`admin_reports_resolved.json`、`admin_actions_final.json`、`ev_update.json`、`notifications_d.json`、`notifications_e.json`、`checkin_d.json`、`checkin_d_again.json`、`checkins_list.json`、`cancel_d_rejected.json`、`cancel_e.json`、`cancel_e_again.json`、`regs_snapshot.json`、`user_c2.json`、`reg_c2.json` 等）。原始 PII 已脱敏，不进入 Git。 |
| 日志时间窗 | `2026-08-21T19:44:11Z – 2026-08-21T19:46:44Z`（UTC，即 `2026-08-22 03:44–03:46 +08:00`） |

## 验收结果

| 场景 | 结果 | 证据/备注 |
|---|---|---|
| 独立活动与参与者报名 | 通过 | `POST /api/v1/organizers`→`201`（组织 id=2）；`POST /api/v1/events`→`201`（活动 id=2，`status=published`、`capacity=20`、`price=0`）；`POST /api/v1/events/2/tickets`→`201`（免费票 id=2，`stock=10`、`price=0`）。丁/戊 `POST /api/v1/events/2/register`→`201`；Admission 凭证格式 `soulmark:admission:` + 32 位 hex（丁 `f7c1…d999`、戊 `7efb…7464`，脱敏），`status=active`。活动时间窗 `event_time=2026-08-23` 与活动一并行独立（活动一为 2026-08-26）。 |
| 活动变更通知 | 通过 | `PUT /api/v1/events/2`→`200`，地点由「受控活动二 场馆 B」调整为「受控活动二 场馆 B2（变更后）」；丁/戊 `GET /api/v1/me/notifications` 各含 1 条 `event_updated`（丁 id=6、戊 id=7，正文「地点调整为 受控活动二 场馆 B2（变更后）」）。 |
| 临近提醒幂等 | 通过 | 活动置于提醒窗口（`event_time=now+36h`、`NOTIFICATION_REMINDER_HOURS=72`），调度触发；丁/戊各仅 1 条 `event_reminder_24h`（丁 id=8、戊 id=9，正文「将在 2026-08-23 07:44 UTC 开始」）；重复扫描未产生重复通知（幂等键 `event-reminder-24h:{eventID}:{userID}:{eventTime}`）。 |
| 首次核销 | 通过 | 运营端 `POST /api/v1/events/2/checkins`（`credential=丁`）→`201`「核销成功」，`already_checked_in=false`，checkin id=1。 |
| 重复核销审计幂等 | 通过 | 再次 `POST /api/v1/events/2/checkins`（`credential=丁`）→`200`「该凭证已核销」，`already_checked_in=true`，返回同一 checkin id=1；`GET /api/v1/events/2/checkins`→`total=1`（审计列表仅一条成功记录）。 |
| 已核销取消拒绝 | 通过 | 已核销的丁 `DELETE /api/v1/events/2/register`→`409 ADMISSION_ALREADY_CHECKED_IN`（「入场凭证已核销」）；未核销的戊 `DELETE /api/v1/events/2/register`→`200`「已取消报名」，再次取消→`404 REGISTRATION_NOT_FOUND`（幂等无副作用）。 |
| 举报与治理动作审计 | 通过 | 己（非作者、已报名）`POST /api/v1/events/2/posts/2/reports`（`category=spam`）→`201`（举报 id=1，`status=open`）；`GET /api/v1/admin/content-reports`→`total=1`（open）。运营端 `PUT /api/v1/admin/content-reports/1`（`resolution=remove`）→`200`，`status=resolved`、`resolved_by=platform_admin`、`target_moderation_status=removed`；`GET /api/v1/admin/content-actions`→3 条（id=3 `remove` 关联 `report_id=1`，加早前 `remove`+`restore`，`actor=platform_admin`），全程动作审计可溯。举报分类使用服务端合法枚举 `spam`（UI 中文标签「垃圾广告」仅作展示），避免了首轮用中文值导致的 `400 VALIDATION_ERROR`。 |
| 桌面/移动可用性 | 通过（佐证） | 候选代码的 Playwright e2e（desktop-chromium + mobile-chromium/Pixel 7）覆盖报名→讨论→通知→核销→重复核销→取消→治理，均断言 `document.documentElement.scrollWidth <= window.innerWidth`（无阻断性横向溢出），并向测试报告附截图；该 e2e 用独立 `:memory:` 环境与独立 `PORT=18080` 复现，不污染受控部署。 |

## 缺陷与结论

- P0：无
- P1：无
- P2：无新发现；沿袭活动一 P2（重复取消返回 `404 REGISTRATION_NOT_FOUND`，幂等无副作用）；举报首轮用中文分类值 `垃圾广告` 返回 `400 VALIDATION_ERROR`（属调用方误用，服务端正确返回合法枚举校验错误，无需修复）
- Go / No-Go：Go（业务闭环全部通过；唯一待补为真实 staging 部署，属 G6 环境项）
- 执行人确认：待填写
- 复核人确认：待填写
