# 亦闻 event-go

> 当前已发布能力与运行方式以本 README 为准。后续产品化目标、阶段门禁和迭代节奏统一维护在 [docs/goal.md](docs/goal.md)；v1–v5.1 的历史过程见 [docs/mvp_task.md](docs/mvp_task.md)。

## 项目想法

亦闻是一个活动管理平台，目标是让活动的组织、报名和交流变得简单。

**核心思路**：把每个活动想象成一个"线上门店"——

- 活动有组织者、地点、时间，就像门店有老板、地址、营业时间
- 活动可以卖门票，就像门店卖商品
- 买了票的人自动进入一个专属讨论区，可以在里面提问、分享、约伴

**往大了想**：其实一切营销都是活动。

- 饭店发优惠券 → 就是"限时领券"活动
- 超市试吃 → 就是"免费体验"活动
- 电商满减 → 就是"凑单"活动
- 线下讲座 → 就是"报名参会"活动
- 线上直播 → 就是"预约观看"活动

本质上都是同一件事：**有人发起、设定规则、吸引参与、完成互动**。亦闻想做的，就是提供一个通用的框架，不管什么类型的"活动"都能在上面跑起来。

## 核心概念

### 活动即门店

每个活动本质上就是一个独立的小社区：

- **组织者**：活动的创建者和管理者，相当于"店长"
- **参与者**：报名买票的人，相当于"顾客"
- **内容**：活动本身的信息（时间、地点、介绍）以及参与者产生的讨论
- **门槛**：通过门票区分参与者和围观者，保证讨论质量

### 为什么要做这个

市面上的活动平台通常只解决"报名"这一个环节，活动结束后关系就断了。亦闻想让活动的价值延续：

1. **活动前**：方便发布和传播，吸引报名
2. **活动中**：参与者之间可以交流，形成氛围
3. **活动后**：讨论和资料留存，变成一个有长期价值的内容沉淀

## 迭代路线

**第一步 — 核心闭环** ✅
先跑通最基本的流程：创建活动 → 展示活动 → 报名参与。集中精力把这一个流程做顺。

**第二步 — 互动** ✅
加入讨论区功能，让参与者能在活动前后交流。这是和传统活动平台拉开差距的关键。

**第三步 — 完善** ✅
门票管理、报名关联门票、库存扣减等周边功能逐步补充。

**第四步 — 生态**
支持系列化活动、主办方主页、活动推荐等，形成一个活动生态。

***

## 数据模型

### 核心实体关系

```
Organization (授权、审计、未来计费租户)
  ├── OrganizationMember (owner/admin/editor/checker/finance)
  └── OrganizerProfile (公开门店/品牌资料)
        └── Event (活动)
              ├── Registration (报名记录)
              │     └── Ticket (门票) — N:1，报名可选关联一张门票
              ├── Post (帖子)
              │     └── Reply (回复) — N:1，一个帖子有多个回复
              └── Ticket (门票) — N:1，一个活动可创建多种门票

User (用户) — 注册/登录获得 JWT
  报名/发帖/回复时自动携带身份
  └── Notification (站内通知) — 报名、取消、活动变更与临近提醒
```

### Organization & OrganizerProfile — 租户与公开资料

Schema v12 开始把授权边界与公开展示拆开：`Organization` 是成员权限、审计和未来计费的租户；`OrganizerProfile` 继续承载现有 `/organizers` API 的门店/品牌公开资料。当前基础切片保持一对一关系，历史门店迁移为 `unclaimed` Organization，不根据联系方式猜测 owner。

| 实体 | 说明 |
|---|---|
| `organizations` | 租户名称、slug 与 `unclaimed/active/suspended/system` 状态 |
| `organizers` / `OrganizerProfile` | 兼容现有 API 的公开门店资料，通过 `organization_id` 关联租户 |
| `organization_members` | 用户在租户内的角色与 active/revoked 状态；每个组织最多一个 active owner |
| `organization_invitations` | 只保存邀请 Token 摘要、规范化邮箱、非 owner 角色、过期和消费状态 |

邀请只能由 active Organization 的 active owner/admin 创建；接受者的登录邮箱或已验证恢复邮箱必须与邀请一致。完整租户 API、资源 scope、后台 UI 和审计仍属于后续 G5 切片，当前全局 Admin Token 仍只代表 platform admin。

### OrganizerProfile 字段

| 字段          | 类型      | 说明                          |
| ----------- | ------- | --------------------------- |
| id          | int64   | 主键                          |
| name        | string  | 门店名称                        |
| description | string  | 简介                          |
| contact     | string  | 联系方式                        |
| logo_url    | string  | Logo 图片 URL                 |
| address     | string  | 地址                          |
| website     | string  | 官网                          |
| tags        | string  | 标签（逗号分隔，如"教育,讲座"）|

### Event — 活动

| 字段             | 类型      | 说明                                       |
| -------------- | ------- | ---------------------------------------- |
| id             | int64   | 主键                                       |
| organizer\_id  | int64   | 所属门店                                     |
| organizer\_name | string  | 门店名称（查询时自动填充）                          |
| title          | string  | 活动标题                                     |
| description    | string  | 活动描述                                     |
| cover\_url     | string  | 活动封面 HTTP/HTTPS 图片地址                    |
| event\_time    | string  | 活动时间（RFC3339）                            |
| location       | string  | 活动地点                                     |
| capacity       | int     | 报名容量上限                                   |
| price          | float64 | 活动基础价格                                   |
| status         | string  | 状态：draft / published / cancelled / ended |

### User Recovery Identity — 用户恢复身份

| 字段/实体 | 说明 |
|---|---|
| `users.contact` | 登录标识；历史手机号账户绑定邮箱后仍保留原手机号登录 |
| `users.recovery_email` | 独立密码恢复邮箱；phone-only 账户只有验证成功后才可用于重置 |
| `users.recovery_email_verified_at` | 邮箱所有权确认时间；新注册邮箱在确认前显示“待验证” |
| `recovery_email_tokens` | 只保存一次性验证 Token 的 SHA-256 摘要、目标邮箱、过期和消费状态 |

### Ticket — 门票

| 字段        | 类型      | 说明                 |
| --------- | ------- | ------------------ |
| id        | int64   | 主键                 |
| event\_id | int64   | 所属活动               |
| name      | string  | 门票名称（如"普通票""VIP票"） |
| price     | float64 | 门票价格               |
| stock     | int     | 当前库存               |

### Registration — 报名记录

| 字段           | 类型      | 说明          |
| ------------ | ------- | ----------- |
| id           | int64   | 主键          |
| event\_id    | int64   | 关联活动        |
| name         | string  | 报名者姓名       |
| contact      | string  | 联系方式（手机/邮箱） |
| ticket\_id   | \*int64 | 可选，关联的门票    |
| ticket\_name | string  | 报名时的门票名称快照  |
| user\_id     | \*int64 | 新报名的可信用户 ID；历史无法匹配的数据为空并标记 legacy |
| identity\_status | string | verified / backfilled / legacy |

### Admission & Checkin — 入场权益与核销

| 实体 | 说明 |
|---|---|
| Admission | 免费报名在 Registration 事务内签发的独立入场权益；保存不可预测凭证、票种快照和 active/revoked 状态 |
| Checkin | 每个 Admission 最多一条成功核销事件；数据库触发器禁止更新和删除 |

Registration、Admission、Checkin 保持独立，取消报名会吊销未核销 Admission；已核销报名不能取消或退回库存。

### Post & Reply — 讨论区

| 实体    | 说明                       |
| ----- | ------------------------ |
| Post  | 帖子，关联 event\_id；user\_id 是新增讨论的唯一作者身份；`moderation_status` 支持保留原文的软删除 |
| Reply | 回复，关联 post\_id；user\_id 是新增回复的唯一作者身份；公开查询隐藏 removed 回复 |
| ContentReport | 参与者对帖子/回复的分类举报；同一举报人与目标仅允许一条待处理记录 |
| ContentModerationAction | 管理员移除、恢复或驳回的独立审计记录，不随公开内容隐藏而丢失 |

### Notification — 站内通知

| 能力 | 说明 |
|---|---|
| 事务通知 | 报名成功、取消和活动变更与业务写入同一 SQLite 事务 |
| 临近提醒 | 单实例调度器扫描提醒窗口，唯一幂等键避免重复创建 |
| 用户状态 | 支持分页、未读筛选、未读数、单条已读和全部已读 |
| 历史保留 | 活动删除后通知保留，`event_id` 自动置空，不保存联系方式或入场凭证 |

***

## 当前进度

**v6.0 验收与 v6.1 租户基础并行推进** — 免费活动自动化、JWT/Go 供应链安全与 P0/P1 清零已通过远端门禁；G4 仍待真实 staging SMTP 和两场受控活动。v6.1 当前只落地 Schema v12 租户基础、历史数据安全回填、成员/邀请存储不变量与 N/N-1 `/organizers` 写兼容，尚未开放自助组织 API，也未宣称业务资源已经 tenant scoped。

机器可读规范：[`GET /api/v1/openapi.json`](http://localhost:8080/api/v1/openapi.json)，源文件位于 [`internal/openapi/v1.json`](internal/openapi/v1.json)。

```
POST   /api/v1/auth/register                            用户注册
POST   /api/v1/auth/login                               用户登录（返回 JWT）
POST   /api/v1/auth/logout                              退出并撤销该用户全部现有 JWT
POST   /api/v1/auth/password-reset/request              请求一次性密码重置链接
POST   /api/v1/auth/password-reset/confirm              使用一次性 Token 重置密码
POST   /api/v1/auth/recovery-email/confirm              确认恢复邮箱并撤销旧会话
GET    /api/v1/me/registrations[?page=&page_size=]      当前用户报名列表
GET    /api/v1/me/admissions[?page=&page_size=]         当前用户入场凭证与历史状态
GET    /api/v1/me/activities[?page=&page_size=]          当前用户统一活动时间线（前端主入口）
POST   /api/v1/me/recovery-email/request                校验当前密码并发送恢复邮箱验证链接
GET    /api/v1/me/notifications[?unread_only=&page=&page_size=] 当前用户通知列表
GET    /api/v1/me/notifications/unread-count            当前用户未读通知数
PUT    /api/v1/me/notifications/{notificationId}/read   标记自己的单条通知已读
PUT    /api/v1/me/notifications/read-all                标记自己的全部通知已读
GET    /api/v1/admin/session                            校验平台管理员 Token 🔐
GET    /api/v1/admin/content-reports                    举报队列（状态/内容类型筛选）🔐
PUT    /api/v1/admin/content-reports/{reportId}         移除内容并处理或驳回举报 🔐
GET    /api/v1/admin/content-actions                    内容治理动作审计 🔐
GET    /api/v1/admin/identity-migration                 身份迁移统计与 legacy 清单 🔐
POST   /api/v1/organizers                               创建门店 🔐
GET    /api/v1/organizers[?page=&page_size=]            门店列表（分页，含活动数）
GET    /api/v1/organizers/{id}                          门店详情
PUT    /api/v1/organizers/{id}                          编辑门店 🔐
DELETE /api/v1/organizers/{id}                          删除门店（活动解绑）🔐
POST   /api/v1/events                                   创建活动（必须归属门店）🔐
GET    /api/v1/events[?status=&price_type=&q=&organizer_id=&page=&page_size=] 活动列表（筛选 + 分页，含门店名）
GET    /api/v1/events/{id}                              活动详情（含门店名）
PUT    /api/v1/events/{id}                              编辑活动 🔐
DELETE /api/v1/events/{id}                              删除活动 🔐
POST   /api/v1/events/{id}/register                     报名活动（必须登录，身份来自 JWT）
DELETE /api/v1/events/{id}/register                     取消自己的报名（活动开始前24h）
GET    /api/v1/events/{id}/registration                 当前登录用户的报名状态
GET    /api/v1/events/{id}/registrations[?page=&page_size=] 报名列表（分页）🔐
GET    /api/v1/events/{id}/admission                    当前用户的活动入场凭证
POST   /api/v1/events/{id}/checkins                     幂等核销入场凭证 🔐
GET    /api/v1/events/{id}/checkins                     核销审计列表 🔐
POST   /api/v1/events/{id}/posts                        发帖（需已报名，支持 JWT 自动识别）
GET    /api/v1/events/{id}/posts[?page=&page_size=]     帖子列表（分页）
GET    /api/v1/events/{id}/posts/{postId}               帖子详情（含回复）
DELETE /api/v1/events/{id}/posts/{postId}               软删除帖子并保留治理证据 🔐
PUT    /api/v1/events/{id}/posts/{postId}/restore       恢复已移除帖子 🔐
POST   /api/v1/events/{id}/posts/{postId}/reports       举报帖子（需已报名且非作者）
POST   /api/v1/events/{id}/posts/{postId}/replies       回复帖子（需已报名，支持 JWT 自动识别）
DELETE /api/v1/events/{id}/posts/{postId}/replies/{replyId} 软删除回复并保留治理证据 🔐
PUT    /api/v1/events/{id}/posts/{postId}/replies/{replyId}/restore 恢复已移除回复 🔐
POST   /api/v1/events/{id}/posts/{postId}/replies/{replyId}/reports 举报回复（需已报名且非作者）
POST   /api/v1/events/{id}/tickets                      创建门票 🔐
GET    /api/v1/events/{id}/tickets[?page=&page_size=]   门票列表（分页）
GET    /api/v1/events/{id}/tickets/{ticketId}           门票详情
PUT    /api/v1/events/{id}/tickets/{ticketId}           编辑门票 🔐
DELETE /api/v1/events/{id}/tickets/{ticketId}           删除门票 🔐
GET    /health                                        健康检查
```

**认证架构**：

```
用户 Token  →  Authorization: Bearer <JWT>  →  UserAuth 中间件 → context
管理员 Token →  X-Admin-Token: <token>       →  AdminAuth 中间件
```

后台登录会先调用受保护的 `/admin/session` 校验 Token；每次进入后台路由再次服务端复验。任意受保护请求返回 `ADMIN_AUTH_INVALID` 时，前端只清理发出该请求的当前管理会话并安全回到管理登录页。

**活动列表筛选参数**：

| 参数           | 类型     | 说明           | 示例                                            |
| ------------ | ------ | ------------ | --------------------------------------------- |
| `status`     | string | 按状态筛选        | `draft` / `published` / `cancelled` / `ended` |
| `price_type` | string | 按价格类型筛选      | `free`（免费） / `paid`（付费）                       |
| `q`          | string | 关键词搜索（标题+描述） | `Go`、`Docker`                                 |
| `organizer_id` | int    | 按门店筛选活动 | `1` |

**列表接口分页参数**（活动/报名/帖子/门票/通知）：

| 参数          | 类型 | 默认值 | 说明          |
| ----------- | ---- | --- | ----------- |
| `page`      | int  | 1   | 页码，从 1 开始    |
| `page_size` | int  | 20  | 每页条数，最大 100 |

分页响应额外返回 `total`、`page`、`page_size` 字段。

所有 JSON 写请求最多 1 MiB，未知字段、多个连续 JSON 对象和尾随内容会返回 400/413。错误响应保留数字 `code`，并增加与 HTTP 状态独立的稳定字符串 `error_code`；HTTP 500 只返回通用信息，内部错误写入服务日志。

业务错误码按用途分组如下，机器可读完整枚举以 OpenAPI `ErrorResponse` 为准：

| 类别 | `error_code` |
|---|---|
| 请求边界 | `VALIDATION_ERROR`、`INVALID_JSON`、`REQUEST_TOO_LARGE`、`API_ROUTE_NOT_FOUND`、`METHOD_NOT_ALLOWED` |
| 认证 | `USER_AUTH_REQUIRED`、`USER_TOKEN_INVALID`、`ADMIN_AUTH_INVALID`、`INVALID_CREDENTIALS`、`USER_ALREADY_EXISTS`、`PASSWORD_RESET_TOKEN_INVALID`、`RECOVERY_EMAIL_IN_USE`、`RECOVERY_EMAIL_ALREADY_BOUND`、`RECOVERY_EMAIL_TOKEN_INVALID`、`RECOVERY_EMAIL_RATE_LIMITED` |
| 资源 | `EVENT_NOT_FOUND`、`ORGANIZER_NOT_FOUND`、`TICKET_NOT_FOUND`、`POST_NOT_FOUND`、`NOTIFICATION_NOT_FOUND` |
| 报名与讨论 | `EVENT_NOT_PUBLISHED`、`REGISTRATION_DUPLICATE`、`EVENT_CAPACITY_FULL`、`TICKET_SOLD_OUT`、`REGISTRATION_NOT_FOUND`、`CANCELLATION_DEADLINE_EXCEEDED`、`PARTICIPATION_REQUIRED` |
| 入场与核销 | `ADMISSION_NOT_FOUND`、`ADMISSION_REVOKED`、`ADMISSION_ALREADY_CHECKED_IN`、`EVENT_HAS_ADMISSIONS` |
| 服务端 | `INTERNAL_ERROR` |

详细历史任务见 [docs/mvp_task.md](docs/mvp_task.md)，后续路线见 [docs/goal.md](docs/goal.md)。

***

## 核心流程

### 0. 用户注册与登录

```
注册 (POST /api/v1/auth/register) → name + email + password（8–72 字节，bcrypt 加密）
登录 (POST /api/v1/auth/login) → contact + password → 返回带 auth_version 的 JWT Token
退出 (POST /api/v1/auth/logout) → 服务端递增 auth_version，撤销该用户全部旧 JWT
忘记密码 → 申请 30 分钟一次性链接 → 设置新密码 → 撤销全部旧 JWT
历史手机号账户 → 保留手机号登录 → 当前密码确认 → 邮箱链接确认 → 启用恢复邮箱并撤销旧会话

JWT 有效期 7 天（可配置），前端 localStorage 持久化
报名、发帖、回复、取消和“我的活动”均从 JWT user_id 加载持久化用户，不接受联系方式授权
```

新注册仅接受邮箱，但格式合法不等于邮箱所有权已验证，账户安全页会明确显示“待验证”。历史 contact 登录保持兼容；恢复邮箱绑定不会覆盖手机号。密码重置和恢复邮箱验证 Token 均使用 256 位随机数，数据库只保存 SHA-256 摘要；新申请会替代旧 Token，同一用户一分钟内只接受一次申请。密码重置对已知和未知邮箱均返回相同 202 响应。

免费报名成功时会同时生成 Admission。用户二维码内容为 `soulmark:admission:<32位随机码>`；运营端首次核销返回 201，重复核销返回原 Checkin 且 `already_checked_in=true`，不会新增记录。

### 1. 活动发布

```
创建门店 (POST /api/v1/organizers) 🔐 → 创建活动 (POST /api/v1/events) 必选门店
→ 设置门票 (POST /api/v1/events/{id}/tickets)
→ 活动状态为 published → 对外开放报名
```

### 2. 用户报名

```
用户报名 (POST /api/v1/events/{id}/register)
  ├── 必须登录，姓名和联系方式从账户资料读取
  ├── 可选传入 ticket_id 关联门票
  ├── 关联门票时自动扣减库存（原子操作，事务保障）
  ├── 不传 ticket_id → 纯报名，不涉及门票
  ├── 超出容量 / 重复报名 / 门票售罄 → 明确错误提示
  └── 活动开始前24h可自由取消 (DELETE /api/v1/events/{id}/register)
       ├── 取消时自动退还门票库存（事务内原子操作）
       └── 超过截止时间返回 400，无法取消
```

### 3. 活动讨论

```
报名成功 → 获得发帖/回复权限
发帖 (POST /api/v1/events/{id}/posts) → 通过 JWT user_id 验证报名
回复 (POST /api/v1/events/{id}/posts/{postId}/replies) → 同上，不接受 author_contact 回退
```

### 4. 活动管理

```
编辑活动 (PUT /api/v1/events/{id}) → 局部更新，支持改标题/时间/状态等
删除活动 (DELETE /api/v1/events/{id})
  ├── 已签发 Admission → 拒绝硬删除，返回 EVENT_HAS_ADMISSIONS
  └── 无 Admission → 事务级联清理：回复 → 帖子 → 报名 → 门票 → 活动
```

***

## 技术架构

### 项目结构

```
event_go/
├── cmd/
│   └── event-go/
│       └── main.go              # 入口：组装依赖、注册路由、启动服务、优雅关闭
├── internal/
│   ├── auth/
│   │   └── token.go             # JWT TokenManager：签发与验证
│   ├── clock/
│   │   └── clock.go             # 可注入业务时钟
│   ├── config/
│   │   └── config.go            # 配置管理：环境变量统一加载
│   ├── emailaddr/
│   │   └── email.go             # 裸邮箱地址的严格规范化与大小写收口
│   ├── handler/
│   │   ├── dto/                 # HTTP 请求/响应 DTO、实体映射与 JSON 契约测试
│   │   ├── handler.go           # 基础设施：Handler 结构体, parseID, paginatedOK, getEventOr404 等
│   │   ├── handler_event.go     # 活动 API（Create/List/Get/Update/Delete）
│   │   ├── handler_ticket.go    # 门票 API（Create/List/Get/Update/Delete）
│   │   ├── handler_registration.go # 报名 API（Register, CancelRegistration, ListRegistrations）
│   │   ├── handler_admission.go # 用户凭证、运营核销与审计 API
│   │   ├── handler_post.go      # 帖子/回复 API（CreatePost/Reply, ListPosts, GetPost）
│   │   ├── handler_content_moderation.go # 举报、软删除、恢复与治理审计 API
│   │   ├── handler_auth.go      # 用户认证、JWT 解析与持久化用户校验
│   │   ├── handler_organizer.go # 门店 API（Create/Get/List/Update/Delete）
│   │   ├── handler_test.go      # Handler 集成测试
│   │   ├── handler_identity.go  # 身份迁移报告 API
│   │   └── middleware.go        # 中间件：日志、CORS、安全响应头、管理员认证
│   ├── openapi/
│   │   ├── openapi.go           # 内嵌并提供 OpenAPI v1 文档
│   │   └── v1.json              # OpenAPI 3.1 机器可读契约
│   ├── service/
│   │   ├── registration.go      # 报名/取消用例、业务规则与窄 Repository 接口
│   │   ├── admission.go         # 凭证查询、规范化、核销与审计用例
│   │   ├── discussion.go        # 讨论资格、可信作者与帖子/回复写入用例
│   │   └── content_moderation.go # 举报权限、幂等处理与治理用例编排
│   ├── identifier/
│   │   └── credential.go        # 加密随机 Admission 凭证生成器
│   ├── store/
│   │   ├── store.go             # Store、版本化事务迁移、schema_migrations
│   │   ├── store_event.go       # 活动 CRUD
│   │   ├── store_ticket.go      # 门票 CRUD
│   │   ├── store_registration.go # 报名 CRUD
│   │   ├── store_admission.go  # Admission 查询、幂等 Checkin 与审计
│   │   ├── store_post.go        # 帖子/回复 CRUD
│   │   ├── store_content_moderation.go # 举报队列、软删除、恢复与动作审计事务
│   │   ├── store_user.go        # 用户 CRUD
│   │   ├── store_organizer.go   # 门店 CRUD
│   │   ├── store_organization.go # Organization、成员和邀请基础存储
│   │   ├── store_identity.go    # 身份迁移统计与 legacy 清单
│   │   ├── store_test.go        # Store 单元测试
│   │   ├── store_concurrency_test.go # 容量、库存、取消并发测试
│   │   └── migration_test.go    # 空库、旧库、重复、失败与恢复测试
│   └── model/
│       └── types.go             # 数据模型：结构体定义、哨兵错误、常量、ListEventsParams
├── data/                        # 数据库文件（运行时生成）
├── docs/
│   ├── goal.md                  # 后续产品化目标、阶段门禁与迭代节奏
│   ├── mvp_task.md              # v1–v5.1 历史任务记录
│   └── mvp_task_core.md         # v1–v5.1 历史摘要
├── test_reports/                # 本地/CI 原始测试产物（被忽略，不作为发布证据）
├── Dockerfile                   # 多阶段构建（Node.js → Go → Alpine）
├── go.mod / go.sum
├── web/                         # Vue 3 前端（Vite + Element Plus + Pinia）
└── README.md
```

### 架构分层

```
cmd/event-go/main.go         入口层：组装依赖、启动服务、SPA fallback
         │
         v
internal/handler/*.go        HTTP 层：路由、参数校验、权限检查、JWT 认证与公开 DTO
         │
         v
internal/service/*.go        应用层：跨实体用例、业务边界、调用方定义的 Repository 接口
         │
         v
internal/store/*.go          数据层：SQLite CRUD、事务管理、版本化迁移
         │
         v
internal/model/types.go      模型层：类型定义、哨兵错误、常量、JWT Claims

internal/config/config.go    配置层：环境变量统一管理（横向）
internal/auth + clock        可注入安全与时间依赖（横向）
web/                         Vue 3 前端：Vite + Element Plus + Pinia（横向）
```

### 依赖注入设计

```
main.go
  │  加载并校验 Config（一次）
  │  创建 Store（数据层，显式处理 error）
  │  创建 RegistrationService、Clock、TokenManager
  │  创建 Handler（HTTP 层），注入显式依赖
  │  注册路由，启动服务（监听 SIGINT/SIGTERM 优雅关闭）
  │
  ├──→ Store          ← 封装数据库操作和 SQLite 事务
  │      (CreateEvent, ListEvents, GetEvent, ...)
  │
  ├──→ Application Services ← 报名/取消、讨论写入的跨实体规则与用例编排
  │
  └──→ Handler        ← HTTP 参数、认证上下文与响应映射
         (h.CreateEvent, h.ListEvents, ...)
```

核心思路：**不依赖全局变量，显式传递依赖**。

- `main.go` 一次性加载并校验 `Config`，再创建 `Store` 与 `Handler`；请求处理不重复读取环境变量
- `NewStore` / `OpenStore` 返回初始化错误，调用方显式决定启动失败策略
- 跨实体规则通过最小 service 协调；简单查询和单实体 CRUD 仍可直接调用 Store
- Repository 接口由 service 按实际用例定义，不为所有 CRUD 预建抽象
- HTTP DTO 显式列出公开字段，数据库实体新增字段不会自动进入 API 响应
- 加新功能时先判断是否存在跨实体不变量，再决定是否需要 service，避免机械分层

### 技术选型

| 选择 | 原因 |
|------|------|
| Go 标准库路由 | Go 1.25 路由目录同时注册 `/api/v1` 与兼容 `/api`，零外部依赖 |
| SQLite（modernc.org/sqlite） | 纯 Go，零 CGO，嵌入式，单文件数据库 |
| Vue 3 + Element Plus + Vite | 渐进式前端，极速 HMR，TypeScript 支持 |
| bcrypt + JWT (HS256) | 密码安全哈希 + 用户身份令牌 |
| 多阶段 Docker 构建 | Node.js → Go → Alpine |

### 数据一致性保障

| 场景 | 机制 |
|------|------|
| 并发报名超卖 | `BEGIN` 事务内 `COUNT` + `INSERT`，原子操作 |
| 门票库存超卖 | `UPDATE ... WHERE stock > 0` + 检查 `RowsAffected` |
| 删除活动数据保护 | 存在 Admission 时拒绝硬删除；否则事务级联 replies → posts → registrations → tickets → events |
| 删除门店保护 | 事务内解绑旗下活动（organizer_id = 0），不级联删除 |
| 密码安全 | bcrypt 哈希（`DefaultCost`），不存明文 |
| 数据库连接泄漏 | `Store.Close()` + `defer` + 信号监听优雅关闭 |
| Schema 漂移 | `schema_migrations` + 逐版本事务执行；迁移失败阻止启动 |
| 外键与孤儿数据 | 单连接 SQLite 强制 `foreign_keys=ON`，启动执行 `foreign_key_check` |
| 并发报名 | Store 内串行化关键写事务；容量、库存与取消均有并发回归测试 |
| 重复核销 | Admission 唯一约束 + 串行化事务；重复/并发扫描返回原 Checkin |
| 核销审计不可变 | SQLite 触发器拒绝 Checkin 的 UPDATE 与 DELETE |
| 用户活动分页 | 单一 SQL 投影合并 Admission 与无凭证 Registration，统一排序、去重和精确计数 |
| 会话撤销 | JWT 携带 auth_version；退出和密码重置递增数据库版本，旧 JWT 随即失效 |
| 密码重置 | 256 位一次性 Token、SHA-256 摘要存储、30 分钟默认过期、替代和消费均不可复用 |
| 恢复邮箱绑定 | 当前密码 + 邮箱链接双重确认；确认事务撤销旧 JWT、旧重置 Token 和其他验证 Token，原登录 contact 不变 |
| 内容治理 | Post/Reply 使用 `visible/removed` 软删除；举报处理与动作审计在事务内写入，公开查询只返回 visible 内容 |
| 租户基础 | Organization 与 OrganizerProfile 分离；历史资料回填为 unclaimed；单 active owner、邀请邮箱/过期/单次消费由约束和事务保护 |
| N/N-1 门店写兼容 | 旧应用省略 `organization_id` 创建资料时由触发器生成 unclaimed 租户；旧应用删除资料时自动暂停对应租户 |

### 自动化测试

| 指标 | 结果 |
|------|------|
| 测试文件 | Config、Handler、Store、Migration、Vue Component、Playwright E2E 测试 |
| 测试用例 | **293** 个顶层 Go 测试、17 个 Vue unit/component 测试、4 个 E2E 用例（2 个浏览器项目） |
| 数据竞争 | `go test -race` 零竞争 |
| 静态检查 | `go vet ./...` 无警告 |
| 前端构建 | Element Plus 按实际组件注册；主 JS 约 490 KB / 170 KB gzip，无 chunk size 告警 |
| 覆盖率策略 | 当前不使用 covdata，不以覆盖率作为发布门禁 |

**测试命令**：

```bash
cd event_go && go test -v -count=1 ./...   # 运行所有测试
cd event_go && go test -race -count=1 ./... # 数据竞争检测
cd event_go && go vet ./...                 # 静态检查
cd event_go && go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./... # Go 可达漏洞扫描
cd event_go/web && npm test                 # Vue 组件测试
cd event_go/web && npm run e2e              # 桌面与移动端浏览器 E2E
```

### 数据库迁移

应用启动时会自动执行版本化迁移，当前 `CurrentSchemaVersion=12`。Schema v7–v11 分别覆盖认证、活动封面、内容治理、恢复邮箱和站内通知；Schema v12 新增 `organizations`、`organization_members`、`organization_invitations`，并为 `organizers` 增加 `organization_id`。历史资料一对一回填为 unclaimed 租户，系统占位资料对应 system 租户，不猜测历史 owner；兼容触发器保护 pre-v12 应用的创建和删除写入。迁移逐版本写入 `schema_migrations`，每个版本在独立事务中执行；随后强制启用 SQLite 外键并执行一致性检查，失败时服务拒绝启动。

升级生产数据前先停止旧进程并备份数据库：

```bash
cd event_go
cp data/event_go.db data/event_go.db.pre-upgrade.bak
```

身份迁移会精确匹配已注册用户联系方式；无法匹配的数据保留为只读 legacy 并进入管理员处理清单。Schema v12 的升级、兼容和回滚步骤见 [docs/releases/v6.1.0/migration-rollback.md](docs/releases/v6.1.0/migration-rollback.md)。

### API 速查

```bash
# 用户注册
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"张三","contact":"zhangsan@example.com","password":"change-me-123"}'

# 用户登录
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"contact":"zhangsan@example.com","password":"change-me-123"}'

# 创建门店（管理端）
curl -s -X POST http://localhost:8080/api/v1/organizers \
  -H "Content-Type: application/json" \
  -H "X-Admin-Token: <ADMIN_TOKEN>" \
  -d '{"name":"XX大学","description":"综合大学","address":"大学路1号","tags":"教育,讲座"}'

# 创建活动（归属门店）
curl -s -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -H "X-Admin-Token: <ADMIN_TOKEN>" \
  -d '{"organizer_id":1,"title":"Go 进阶讲座","event_time":"2026-06-15T14:00:00+08:00","location":"线上","capacity":50,"price":19.9}'

# 报名（已登录用户自动携带 JWT）
curl -s -X POST http://localhost:8080/api/v1/events/1/register \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <JWT_TOKEN>" \
  -d '{}'

# 发帖（已登录 + 已报名）
curl -s -X POST http://localhost:8080/api/v1/events/1/posts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <JWT_TOKEN>" \
  -d '{"title":"好活动","content":"推荐"}'
```

***

## 前端

基于 Vue 3 + Element Plus + TypeScript，Vite 构建。

### 项目结构

```
event_go/
├── web/                          # 🆕 Vue 前端项目
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts            # Vite 配置（开发代理到 Go 8080）
│   ├── tsconfig.json
│   └── src/
│       ├── main.ts               # Vue 入口
│       ├── App.vue
│       ├── api/                  # API 层（axios + 各模块封装）
│       ├── stores/               # Pinia 状态管理（认证）
│       ├── router/               # Vue Router 路由
│       ├── components/           # 通用组件（导航、卡片、分页、报名表单等）
│       ├── views/                # 页面（活动列表/详情/讨论/帖子/门店/认证/管理）
│       └── utils/                # 工具函数（日期/价格格式化）
├── cmd/event-go/main.go          # 入口（含 SPA 静态文件服务 + Organizer + 用户认证路由）
├── Dockerfile                    # 多阶段构建（Node.js → Go → Alpine）
└── ...
```

### 页面路由

| 路由 | 页面 | 说明 |
|------|------|------|
| `/` | 活动列表 | 首页，搜索/筛选/分页，卡片展示门店名 |
| `/events/:id` | 活动详情 | 活动信息 + 门店链接 + 报名/取消 + 最近帖子 |
| `/events/:id/discussion` | 讨论区 | 帖子列表 + 发帖（JWT 自动填充） |
| `/events/:id/posts/:postId` | 帖子详情 | 内容 + 回复列表 + 写回复 |
| `/organizers` | 门店列表 | 卡片网格，含活动数，可点击进入详情 |
| `/organizers/:id` | 门店详情 | 门店信息 + 旗下活动列表（分页） |
| `/login` | 用户登录 | contact + password |
| `/register` | 用户注册 | name + email + password（8–72 字节） |
| `/forgot-password` | 忘记密码 | 申请一次性密码重置链接，未知账户返回相同结果 |
| `/reset-password?token=...` | 重置密码 | 消费一次性 Token 并撤销旧会话 |
| `/verify-recovery-email?token=...` | 验证恢复邮箱 | 消费一次性 Token，绑定邮箱并撤销旧会话 |
| `/me/registrations` | 我的活动 | 基于统一活动时间线展示待参加、已结束、已取消、已入场和 Admission 二维码 |
| `/me/security` | 账户安全 | 查看登录标识、恢复邮箱状态并用当前密码申请验证 |
| `/me/notifications` | 通知中心 | 全部/未读筛选、分页、单条或全部已读，并跳转到关联活动 |
| `/admin` | 管理登录 | 通过服务端 `/admin/session` 验证 X-Admin-Token |
| `/admin/organizers` | 门店管理 | 门店列表、创建、编辑和保留历史活动的安全删除 |
| `/admin/organizers/new`、`/:id/edit` | 门店表单 | 维护名称、联系方式、地址、官网、Logo 和标签 |
| `/admin/events` | 活动管理 | 列表、创建、编辑、删除及门店归属 |
| `/admin/events/new` | 创建活动 | 表单（先选门店） |
| `/admin/events/:id/edit` | 编辑活动 | 表单、状态和门店重新归属 |
| `/admin/events/:id/tickets` | 票种管理 | 创建、编辑库存/价格、删除并保留报名快照 |
| `/admin/events/:id/registrations` | 报名与核销 | 查看/安全导出报名、扫码枪/粘贴凭证核销、查看不可变审计记录 |

***

## 快速启动

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `APP_ENV` | `development` | 运行环境：development / test / staging / production |
| `PORT` | `8080` | 服务端口 |
| `ADMIN_TOKEN` | 空（不校验） | 管理员令牌 |
| `DATABASE_PATH` | `data/event_go.db` | SQLite 数据库路径 |
| `JWT_SECRET` | 内置 dev key | JWT 签名密钥（**生产务必修改**） |
| `JWT_EXPIRE_HOURS` | `168`（7 天） | JWT 有效期 |
| `CORS_ORIGIN` | `*` | 允许的跨域来源 |
| `CANCEL_DEADLINE_HOURS` | `24` | 取消报名截止小时数 |
| `PUBLIC_BASE_URL` | `http://localhost:<PORT>` | 密码重置与恢复邮箱验证链接的公开站点地址；生产必须为 HTTPS |
| `PASSWORD_RESET_TTL_MINUTES` | `30` | 密码重置 Token 有效分钟数，允许 1–1440 |
| `RECOVERY_EMAIL_TTL_MINUTES` | `30` | 恢复邮箱验证 Token 有效分钟数，允许 1–1440 |
| `NOTIFICATION_REMINDER_HOURS` | `24` | published 活动临近提醒窗口，允许 1–168 小时 |
| `NOTIFICATION_SCAN_INTERVAL_SECONDS` | `60` | 单实例提醒调度扫描间隔，允许 1–3600 秒 |
| `SMTP_HOST` / `SMTP_PORT` | 空 / `587` | SMTP 服务地址与端口 |
| `SMTP_USERNAME` / `SMTP_PASSWORD` | 空 | SMTP 认证信息 |
| `SMTP_FROM` | 空 | 密码重置与恢复邮箱验证邮件发件人 |

staging 和 production 会执行 fail-closed 配置校验：必须设置 `ADMIN_TOKEN`、至少 32 字节且非默认的 `JWT_SECRET`、明确的 `CORS_ORIGIN`、HTTPS `PUBLIC_BASE_URL` 和完整合法的 SMTP 配置；非法 Token TTL、通知提醒窗口或扫描间隔会拒绝启动。development 未配置 SMTP 时只把密码重置/邮箱验证链接写入服务日志，不发送邮件；业务通知不依赖 SMTP。

### 环境准备

前端开发和 CI 统一使用 Node.js 22、npm 10；`web/.nvmrc` 可用于切换版本。

```bash
# 如果 nvm 装了但没加载（终端提示 npm: command not found）
export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"
```

### 开发模式（前后端分离）

```bash
# 终端1：启动 Go 后端（端口 8080）
cd event_go && go run ./cmd/event-go

# 终端2：安装前端依赖 + 启动 Vite 开发服务器（端口 5173）
cd event_go/web
npm install
npm run dev

# 浏览器打开 http://localhost:5173
```

### 生产构建（单二进制）

```bash
# 1. 构建前端静态文件
cd event_go/web && npm install && npm run build

# 2. 构建并运行（Go 服务 SPA 静态文件）
cd event_go && go build -o event-go ./cmd/event-go/ && ./event-go

# 浏览器打开 http://localhost:8080
```

### Docker 部署

```bash
cd event_go
docker build -t event-go .
docker run -p 8080:8080 -v $(pwd)/data:/app/data event-go
```
