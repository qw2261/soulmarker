# v6.1.0 三组织试点验收记录

> 状态：通过（真实受控部署 + 公开 API 驱动；全程无 platform admin token 介入）<br>
> 规程：[controlled-activity-acceptance.md](../../../testing/controlled-activity-acceptance.md)<br>
> 场景：3 个独立试点组织者完成免费活动运营闭环 + 跨租户隔离 + 审计可追溯
> 关联目标：G5.5 审计与试点（三组织试点项）

| 字段 | 值 |
|---|---|
| 候选完整 SHA | `53dfd0027f91d97fa2ce2664ceb999c9da982e91` |
| 部署/环境 | 本地受控部署：`APP_ENV=test`、`ADMIN_TOKEN=controlled-admin-token`、独立数据库 `data/controlled_acceptance.db`、`127.0.0.1:8080`，托管 `web/dist` SPA；对应 HEAD `53dfd00` 干净工作树。真实 staging 部署（HTTPS 域名、独立数据库）属 G6 环境隔离项，待配置。 |
| 组织数量 / 角色 | 3 个独立试点组织者（甲/乙/丙）。各自通过公开 `POST /auth/register` 注册用户后自助建组织，owner 全程无开发者介入、无 `X-Admin-Token`。 |
| 执行人 / 复核人 | 执行人：自动化受控驱动脚本 `/tmp/drive_org_pilot.py`；复核人：待人工确认 |
| 参与人数 | 每组织 2 名独立注册参与者，共 6 名（甲甲/乙甲、甲乙/乙乙、甲丙/乙丙）。 |
| 原始证据引用 | 驱动响应 `/tmp/pilot/`（`report.json`、`pilot-a-*/pilot-b-*/pilot-c-*` 各步骤 JSON、三个 `*_regs_export.csv`、三组隔离检查 JSON）。原始 PII 已脱敏（`pilot-*@example.com`），不进入 Git。 |
| 跨租户隔离检查 | 24 条跨租户请求全部返回 `403 ORGANIZATION_ACCESS_DENIED`（无权访问该组织）。 |

## 验收结果

| 场景 | 结果 | 证据/备注 |
|---|---|---|
| 试点组织者自助入驻建组织 | 通过 | 甲/乙/丙分别 `POST /api/v1/organizations`→`201`（org_id=`6`/`7`/`8`）；owner 自动成为 owner 角色，工作台返回 workspace `profile.id` 作为 organizer 身份。 |
| 自助发布免费活动 | 通过 | 分别 `POST /api/v1/organizations/{orgId}/events`→`201`（event_id=`6`/`7`/`8`，`status=published`、`price=0`）；`organizer_id` 取自 workspace `profile.id`。 |
| 配置免费票种 | 通过 | 分别 `POST /api/v1/organizations/{orgId}/events/{eventId}/tickets`→`201`（ticket_id=`6`/`7`/`8`，`price=0`、`stock` 正整数）。 |
| 参与者注册与凭证 | 通过 | 每组织 2 名参与者 `POST /api/v1/events/{eventId}/register`→`201`；Admission 凭证格式 `soulmark:admission:` + 32 位 hex（甲 `7c99…07`/`3cfd…6cd`、乙 `1477…b83`/`adb1…599`、丙 `c8bb…2d1`/`1037…566`，脱敏）。 |
| 报名 CSV 安全导出 | 通过 | `GET /api/v1/organizations/{orgId}/events/{eventId}/registrations/export`→CSV 带 UTF-8 BOM、列 `ID,姓名,联系方式,票种,报名时间`、每组织 2 行；`spreadsheetSafeCell` 转义 `= + - @` 防公式注入；各 CSV SHA-256 见下表。 |
| 两次核销 | 通过 | 分别 `POST /api/v1/organizations/{orgId}/events/{eventId}/checkins`（body `{"credential": "soulmark:admission:..."}`）→`201`（首次）；每组织 2 条 `checkInOrganizationAdmission` 审计。 |
| 关键动作审计可追溯 | 通过 | `GET /api/v1/organizations/{orgId}/audits?page=1&page_size=100`→每组织 `total=6`，action 集为 `createOrganization`、`createOrganizationEvent`、`createOrganizationTicket`、`exportOrganizationEventRegistrations`、`checkInOrganizationAdmission`；审计条目携带 actor/tenant/request_id，跨租户读审计被拒。 |
| 跨租户隔离 | 通过 | 甲→乙、乙→丙、甲→丙 三组 × 8 个端点（list events / get event / list tickets / list registrations / list members / list audits / write checkin / get session）全部返回 `403 ORGANIZATION_ACCESS_DENIED`「无权访问该组织」。 |

## 各组织 CSV SHA-256

| 组织 | org_id | event_id | CSV SHA-256 |
|---|---|---|---|
| 甲 | 6 | 6 | `69239d188b8c165d3426c6e930d9acdce57a917487dc325a709efaf10833dce9` |
| 乙 | 7 | 7 | `c4b72d2e1bf2d0e2e157126151688643c68f8dac44d34f9d0b53d89da0986f9f` |
| 丙 | 8 | 8 | `ae483bba715b0f5f19cf891ccea6e712e3bdfb4dd17751b24fc104d89b9012a8` |

## 缺陷与结论

- P0：无
- P1：无
- P2：无
- Go / No-Go：Go（3 个独立试点组织者无需开发者介入完成免费活动运营闭环；跨租户 24 条访问全部 403；关键管理动作均可通过审计日志追溯；G5.5 三组织试点项完成）
- 执行人确认：待填写
- 复核人确认：待填写
