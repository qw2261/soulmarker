# 亦闻 event-go

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
Event (活动)
  ├── Registration (报名记录) — N:1，一个活动有多个报名
  │     └── Ticket (门票) — N:1，报名可选关联一张门票
  ├── Post (帖子) — N:1，一个活动有多个讨论帖
  │     └── Reply (回复) — N:1，一个帖子有多个回复
  └── Ticket (门票) — N:1，一个活动可创建多种门票
```

### Event — 活动

| 字段          | 类型      | 说明                                       |
| ----------- | ------- | ---------------------------------------- |
| id          | int64   | 主键                                       |
| title       | string  | 活动标题                                     |
| description | string  | 活动描述                                     |
| event\_time | string  | 活动时间（RFC3339）                            |
| location    | string  | 活动地点                                     |
| capacity    | int     | 报名容量上限                                   |
| price       | float64 | 活动基础价格                                   |
| status      | string  | 状态：draft / published / cancelled / ended |

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

### Post & Reply — 讨论区

| 实体    | 说明                       |
| ----- | ------------------------ |
| Post  | 帖子，关联 event\_id，仅已报名者可创建 |
| Reply | 回复，关联 post\_id，仅已报名者可创建  |

***

## 当前进度

**v4.1** — 综合优化（配置集中管理 + 测试覆盖增强 + 代码注释完善），共 **17 个 API 接口**。

```
POST   /api/events                          创建活动
GET    /api/events[?status=&price_type=&q=] 活动列表（支持筛选搜索）
GET    /api/events/{id}                     活动详情
PUT    /api/events/{id}                     编辑活动
DELETE /api/events/{id}                     删除活动
POST   /api/events/{id}/register            报名活动
GET    /api/events/{id}/registrations       报名列表
POST   /api/events/{id}/posts               发帖（需已报名）
GET    /api/events/{id}/posts               帖子列表
GET    /api/events/{id}/posts/{postId}      帖子详情（含回复）
POST   /api/events/{id}/posts/{postId}/replies 回复帖子（需已报名）
POST   /api/events/{id}/tickets             创建门票
GET    /api/events/{id}/tickets             门票列表
GET    /api/events/{id}/tickets/{ticketId}  门票详情
PUT    /api/events/{id}/tickets/{ticketId}  编辑门票
DELETE /api/events/{id}/tickets/{ticketId}  删除门票
```

**活动列表筛选参数**：

| 参数           | 类型     | 说明           | 示例                                            |
| ------------ | ------ | ------------ | --------------------------------------------- |
| `status`     | string | 按状态筛选        | `draft` / `published` / `cancelled` / `ended` |
| `price_type` | string | 按价格类型筛选      | `free`（免费） / `paid`（付费）                       |
| `q`          | string | 关键词搜索（标题+描述） | `Go`、`Docker`                                 |

详细任务跟踪见 [mvp\_task.md](mvp_task.md)。

***

## 核心流程

### 1. 活动发布

```
创建活动 (POST /api/events) → 设置门票 (POST /api/events/{id}/tickets)
→ 活动状态为 published → 对外开放报名
```

### 2. 用户报名

```
用户报名 (POST /api/events/{id}/register)
  ├── 可选传入 ticket_id 关联门票
  ├── 关联门票时自动扣减库存（原子操作，事务保障）
  ├── 不传 ticket_id → 纯报名，不涉及门票
  └── 超出容量 / 重复报名 / 门票售罄 → 明确错误提示
```

### 3. 活动讨论

```
报名成功 → 获得发帖/回复权限
发帖 (POST /api/events/{id}/posts) → 需传入报名时的 contact 验证
回复 (POST /api/events/{id}/posts/{postId}/replies) → 同上
```

### 4. 活动管理

```
编辑活动 (PUT /api/events/{id}) → 局部更新，支持改标题/时间/状态等
删除活动 (DELETE /api/events/{id}) → 事务级联清理：
  回复 → 帖子 → 门票 → 报名 → 活动
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
│   ├── config/
│   │   └── config.go            # 配置管理：环境变量统一加载
│   ├── handler/
│   │   ├── handler.go           # HTTP 层：请求处理、参数校验、权限检查、中间件
│   │   ├── handler_test.go      # Handler 集成测试（34 个用例）
│   │   └── middleware.go        # 中间件：日志、CORS、管理员认证
│   ├── store/
│   │   ├── store.go             # 数据层：SQLite 建表迁移、所有 CRUD 方法、事务管理
│   │   └── store_test.go        # Store 单元测试（32 个用例）
│   └── model/
│       └── types.go             # 数据模型：结构体定义、哨兵错误、常量
├── data/                        # 数据库文件（运行时生成）
├── docs/
│   └── mvp_task.md              # 任务跟踪文档
├── test_reports/                # 阶段性测试报告
├── Dockerfile                   # 多阶段构建（Go 1.25 → Alpine 3.19）
├── go.mod                       # Go 模块定义
├── go.sum                       # 依赖锁文件
└── README.md                    # 项目文档
```

### 架构分层

```
cmd/event-go/main.go         入口层：组装依赖、启动服务
         │
         v
internal/handler/handler.go  HTTP 层：路由、参数校验、权限检查
         │
         v
internal/store/store.go      数据层：SQLite CRUD、事务管理
         │
         v
internal/model/types.go      模型层：类型定义、哨兵错误、常量
```

- 单向依赖：`main → handler → store → model`
- 无循环依赖，各层职责清晰
- `internal` 包防止外部导入，强制封装

### 依赖注入设计

```
main.go
  │  创建 Store（数据层）
  │  创建 Handler（HTTP 层），注入 Store
  │  注册路由，启动服务（监听 SIGINT/SIGTERM 优雅关闭）
  │
  ├──→ Store          ← 封装所有数据库操作
  │      (CreateEvent, ListEvents, GetEvent, ...)
  │
  └──→ Handler        ← 封装所有 HTTP 处理器
         (h.CreateEvent, h.ListEvents, ...)
```

核心思路：**不依赖全局变量，显式传递依赖**。

- `main.go` 创建 `Store`，再创建 `Handler` 把 `Store` 注入进去
- `Handler` 的方法直接调用 `h.store.Xxx()`，不走全局变量
- 加新功能时：`model/` 加结构体 → `store/` 加方法 → `handler/` 加处理器 → `cmd/` 加路由

### 技术选型

| 选择                         | 原因                                                     |
| -------------------------- | ------------------------------------------------------ |
| Go 标准库路由                   | Go 1.25 路由语法（`"POST /api/events/{id}/register"`），零外部依赖 |
| SQLite（modernc.org/sqlite） | 纯 Go 实现，零 CGO，嵌入式，单文件数据库                               |
| 多阶段 Docker 构建              | 最终镜像 23.7MB，Go 1.25 → Alpine 3.19                      |

### 数据一致性保障

| 场景       | 机制                                                        |
| -------- | --------------------------------------------------------- |
| 并发报名超卖   | `BEGIN` 事务内 `COUNT` + `INSERT`，原子操作                       |
| 门票库存超卖   | `UPDATE ... WHERE stock > 0` + 检查 `RowsAffected`          |
| 删除活动数据残留 | 事务级联删除：replies → posts → tickets → registrations → events |
| 数据库连接泄漏  | `Store.Close()` + `defer` + 信号监听优雅关闭                      |

### 自动化测试

| 指标 | 结果 |
|------|------|
| 测试文件 | `internal/store/store_test.go` + `internal/handler/handler_test.go` |
| 测试用例 | **66**（Store 32 + Handler 34） |
| 覆盖率 | **67.8%** |
| 静态检查 | `go vet ./...` 无警告 |

**测试命令**：

```bash
cd event_go && go test -v -count=1 ./...   # 运行所有测试
cd event_go && go test -cover ./...         # 查看覆盖率
cd event_go && go vet ./...                 # 静态检查
```

