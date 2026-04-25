# 亦闻 event-go MVP 任务跟踪

> 目标：跑通"创建活动 → 浏览活动 → 报名参与"核心闭环
> 技术栈：Go 标准库 + SQLite + Docker

---

## 第一阶段：项目初始化

- [x] 确定 MVP 范围和方案
- [x] 创建 Go 模块（`go.mod`）
- [x] 确认 Go 版本兼容性（本地 1.24.2 → 迁移后需 go 1.25）
- [x] 规划代码文件结构

## 第二阶段：数据模型（v1 — 内存版）

- [x] 定义 `Event` 结构体（ID、标题、描述、时间、地点、容量、价格、创建时间）
- [x] 定义 `Registration` 结构体（ID、活动ID、姓名、联系方式、报名时间）
- [x] 定义 API 请求/响应结构体
- [x] 实现线程安全的内存存储（`sync.RWMutex`）

## 第三阶段：API 接口（v1 — 内存版）

- [x] `POST /api/events` — 创建活动
- [x] `GET /api/events` — 获取活动列表
- [x] `GET /api/events/{id}` — 获取活动详情
- [x] `POST /api/events/{id}/register` — 报名参加活动
- [x] `GET /api/events/{id}/registrations` — 查看活动报名列表

## 第四阶段：健壮性与错误处理（v1 — 内存版）

- [x] 参数校验（必填字段、时间格式、容量限制）
- [x] 错误响应规范化（统一 JSON 格式）
- [x] 边界情况处理（活动已满、重复报名、活动不存在）
- [x] CORS 支持（方便前端调试）

## 第五阶段：Docker 部署（v1 — 内存版）

- [x] 编写多阶段构建 Dockerfile
- [x] Docker 镜像构建验证
- [x] 容器运行测试
- [x] 镜像体积优化（目标 < 20MB → 17.6MB ✅）

## 第六阶段：SQLite 持久化（v2）

- [x] 添加 `modernc.org/sqlite` 纯 Go 依赖
- [x] 设计数据库表结构（events + registrations）
- [x] 实现自动建表迁移（`CREATE TABLE IF NOT EXISTS`）
- [x] 重构 Store：内存实现 → SQLite 实现
- [x] 添加活动状态字段（draft / published / cancelled / ended）
- [x] 验证：重启后数据不丢失

## 第七阶段：活动编辑与管理（v2）

- [x] `PUT /api/events/{id}` — 编辑活动
- [x] `DELETE /api/events/{id}` — 删除活动（级联删除报名）
- [x] 状态校验：未发布的活动不可报名
- [x] 局部更新：只传需要改的字段

## 第八阶段：Docker 部署（v2）

- [x] 更新 Dockerfile 支持 Go 1.25
- [x] 配置 goproxy.cn 国内镜像加速
- [x] 验证构建成功（镜像 23.6MB）
- [x] 添加 VOLUME 指令 + DB_PATH 环境变量支持数据持久化

## 第九阶段：优化与复盘 ✅

- [x] 代码结构审查：移除了自定义 `contains`/`searchString`，改用标准库 `strings.Contains`
- [x] API 一致性检查：局部更新改用指针（`*string`, `*float64`），修复不传字段误覆盖 Bug
- [x] 错误处理完备检查：引入哨兵错误（`ErrNotFound`, `ErrDuplicate`, `ErrFull`），`errors.Is` 替代脆弱字符串比较
- [x] 数据库路径支持 `DB_PATH` 环境变量，Docker 可灵活映射 volume
- [x] Docker volume 映射：`docker run -v $(pwd)/data:/app/data ...`

## 第十阶段：代码结构重构 ✅

- [x] 明确"依赖注入"架构方向：无全局变量，显式传递依赖
- [x] 拆分文件：
  - `types.go` — 数据模型（Event、Registration、请求/响应结构体、哨兵错误）
  - `store.go` — 数据层（NewStore、建表迁移、所有 CRUD 方法）
  - `handlers.go` — HTTP 层（Handler struct、所有 handle* 处理器、中间件）
  - `main.go` — 入口（组装 Store → Handler → 注册路由 → 启动）
- [x] 消除全局变量 `store`，改为 Handler 结构体持有 Store
- [x] 更新架构文档到 README.md

## 第十一阶段：讨论区（第二步 — 互动） ✅

- [x] 设计数据模型：Post（帖子）+ Reply（回复）
- [x] 新增数据库表迁移（posts + replies）
- [x] 发帖/回复权限校验：仅已报名者可参与
- [x] 帖子列表（含回复数）、帖子详情（含回复列表）
- [x] 测试验证：全部 9 个场景通过

---

## 任务进度

| 阶段 | 状态 | 备注 |
|------|------|------|
| 一、项目初始化 | ✅ 已完成 | 方案确定，结构清晰 |
| 二、数据模型（v1） | ✅ 已完成 | Event + Registration，内存版 |
| 三、API 接口（v1） | ✅ 已完成 | 5 个接口全部实现 |
| 四、健壮性（v1） | ✅ 已完成 | 参数校验、重复报名、边界检查 |
| 五、Docker 部署（v1） | ✅ 已完成 | 多阶段构建，镜像 17.6MB |
| 六、SQLite 持久化（v2） | ✅ 已完成 | 纯 Go SQLite，零外部依赖 |
| 七、活动编辑管理（v2） | ✅ 已完成 | PUT + DELETE，7 个接口 |
| 八、Docker 部署（v2） | ✅ 已完成 | Go 1.25，镜像 23.6MB |
| 九、优化与复盘 | ✅ 已完成 | 代码审查、Bug 修复、全局改进 |
| 十、代码结构重构 | ✅ 已完成 | 依赖注入架构，4 文件拆分 |
| 十一、讨论区（互动） | ✅ 已完成 | 发帖/回复，报名者权限校验 |
| 十二、代码审计与修复 | ✅ 已完成 | 12 项 P0-P3 问题全部修复 |
| **十三、门票管理** | ✅ 已完成 | 门票 CRUD + 报名关联 + 库存扣减 |
| **十四、管理员认证** | ✅ 已完成 | API Token 保护管理端接口 |
| **十五、活动分类与搜索** | ✅ 已完成 | 状态筛选 + 价格类型 + 关键词搜索 |

---

## 快速启动

```bash
# 本地运行
cd event_go && go run .

# Docker 运行
cd event_go && docker build -t event-go . && docker run -p 8080:8080 event-go

# Docker 运行（数据持久化，重启不丢失）
docker run -p 8080:8080 -v $(pwd)/data:/app/data event-go
```

---

## 测试计划

### 测试范围

| 测试类别 | 测试项 | 预期结果 |
|---------|--------|---------|
| **基础功能** | 创建活动（免费） | 201 Created，返回活动信息 |
| **基础功能** | 创建活动（付费） | 201 Created，价格字段正确 |
| **基础功能** | 活动列表 | 200 OK，返回所有活动 |
| **基础功能** | 活动详情 | 200 OK，返回指定活动 |
| **基础功能** | 报名活动 | 201 Created，返回报名信息 |
| **基础功能** | 报名列表 | 200 OK，返回报名记录 |
| **编辑管理** | 编辑活动标题 | 200 OK，标题更新 |
| **编辑管理** | 编辑活动状态 | 200 OK，状态更新 |
| **编辑管理** | 编辑活动价格 | 200 OK，价格更新 |
| **编辑管理** | 删除活动 | 200 OK，活动移除 |
| **错误处理** | 重复报名 | 409 Conflict，提示已报名 |
| **错误处理** | 活动已满 | 409 Conflict，提示名额已满 |
| **错误处理** | 活动不存在 | 404 Not Found |
| **错误处理** | 参数校验失败 | 400 Bad Request |
| **错误处理** | 未发布活动报名 | 400 Bad Request |
| **错误处理** | 无效状态值 | 400 Bad Request |
| **错误处理** | 删除不存在的活动 | 404 Not Found |
| **持久化** | 容器重启后数据不丢失 | 数据仍在 |

### 测试命令速查

```bash
# 创建活动
curl -s -X POST http://localhost:8080/api/events \
  -H "Content-Type: application/json" \
  -d '{"title":"Go 入门讲座","description":"从零学 Go","event_time":"2026-05-01T14:00:00+08:00","location":"线上","capacity":50,"price":0}'

# 编辑活动
curl -s -X PUT http://localhost:8080/api/events/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Go 进阶讲座","price":19.9}'

# 获取活动详情
curl -s http://localhost:8080/api/events/1

# 创建门票
curl -s -X POST http://localhost:8080/api/events/1/tickets \
  -H "Content-Type: application/json" \
  -d '{"name":"普通票","price":0,"stock":100}'

# 查看门票列表
curl -s http://localhost:8080/api/events/1/tickets

# 报名活动（不选门票）
curl -s -X POST http://localhost:8080/api/events/1/register \
  -H "Content-Type: application/json" \
  -d '{"name":"张三","contact":"zs@email.com"}'

# 报名活动（关联门票）
curl -s -X POST http://localhost:8080/api/events/1/register \
  -H "Content-Type: application/json" \
  -d '{"name":"李四","contact":"ls@email.com","ticket_id":1}'

# 活动列表（筛选与搜索）
curl -s "http://localhost:8080/api/events?status=published"              # 按状态筛选
curl -s "http://localhost:8080/api/events?price_type=free"               # 免费活动
curl -s "http://localhost:8080/api/events?price_type=paid"               # 付费活动
curl -s "http://localhost:8080/api/events?q=Go"                          # 关键词搜索
curl -s "http://localhost:8080/api/events?status=published&price_type=paid"  # 组合筛选

# 删除活动
curl -s -X DELETE http://localhost:8080/api/events/1
```

---

## 第十二阶段：代码审计与修复 ✅

基于 `test_reports/observation_report_2026-04-25.md` 的审计发现，完成以下修复：

### P0 — 严重（数据一致性）
- [x] **修复 Register 并发超卖风险** (`store.go:237-244`)
  - 使用 `BEGIN` 事务包裹 `COUNT(*)` 查询和 `INSERT` 操作
  - 并发报名时不再可能超过容量限制
- [x] **修复 DeleteEvent 非原子操作** (`store.go:210-226`)
  - 使用事务包裹全部删除操作
  - 级联删除 `replies` → `posts` → `registrations` → `events`
  - 删除活动后不再残留孤儿数据

### P1 — 重要（资源泄漏 / 错误处理）
- [x] **添加 Store.Close 方法 + 优雅关闭** (`store.go`, `main.go`)
  - `Store` 新增 `Close() error` 方法
  - `main()` 使用 `defer store.Close()`
  - 监听 `SIGINT`/`SIGTERM` 信号，5 秒超时优雅关闭 HTTP 服务
- [x] **修复 QueryRow 错误忽略** (`store.go:238`)
  - `Register` 中 `Scan(&count)` 错误被显式处理
- [x] **修复 PRAGMA 错误忽略** (`store.go:34-35`)
  - `journal_mode=WAL` 和 `busy_timeout` 设置失败时记录警告日志

### P2 — 建议（代码质量）
- [x] **修复所有 time.Parse 错误忽略** (`store.go` 共 12 处)
  - `ListEvents`, `GetEvent`, `UpdateEvent`, `ListRegistrations`, `CreatePost`, `ListPosts`, `GetPost`, `CreateReply`, `ListReplies` 中的时间解析全部显式处理
- [x] **修复 LastInsertId 错误忽略** (`store.go:108`)
  - `CreateEvent`, `CreatePost`, `CreateReply` 中的 `LastInsertId()` 错误被显式处理
- [x] **修复 RowsAffected 错误忽略** (`store.go:221`)
  - `DeleteEvent` 中的 `RowsAffected()` 错误被显式处理
- [x] **提取 handlers.go 重复代码**
  - `parseEventID(r)` — 提取参数解析（原 8 处重复）
  - `parsePostID(r)` — 提取帖子 ID 解析
  - `getEventOr404(w, eventID)` — 提取事件存在性检查（原 6 处重复）
  - `checkRegistration(w, eventID, contact)` — 提取权限检查（原 2 处重复）

### P3 — 优化（安全 / 配置）
- [x] **修复 CORS 过于宽松** (`handlers.go:413`)
  - 支持通过 `CORS_ORIGIN` 环境变量配置允许的域名
  - 未设置时默认保持 `*`（开发环境兼容）
- [x] **端口支持环境变量配置** (`types.go`, `main.go`)
  - 新增 `getPort()` 函数，从 `PORT` 环境变量读取端口
  - 默认端口仍为 `8080`

### 验证结果
- [x] `go build` 编译通过
- [x] `go vet` 无警告
- [x] Docker 构建成功（镜像 `event-go:audit`）
- [x] 基础 API 测试通过（创建/编辑/删除/报名）
- [x] 讨论区 API 测试通过（发帖/回复/权限）
- [x] 容量限制测试通过（2 人容量，第 3 人正确返回 409）
- [x] 级联删除测试通过（删除活动后帖子/回复均返回 404）

---

## 第十三阶段：门票管理（第三步 — 完善） ✅

- [x] 设计 `Ticket` 数据模型（ID、活动ID、名称、价格、库存）
- [x] 新增数据库表迁移（tickets）+ 注册表扩展（ticket_id, ticket_name 列）
- [x] `POST /api/events/{id}/tickets` — 创建门票
- [x] `GET /api/events/{id}/tickets` — 门票列表
- [x] `GET /api/events/{id}/tickets/{ticketId}` — 门票详情
- [x] `PUT /api/events/{id}/tickets/{ticketId}` — 编辑门票
- [x] `DELETE /api/events/{id}/tickets/{ticketId}` — 删除门票
- [x] 报名时关联门票选择（`RegisterReq` 增加可选 `ticket_id` 字段）
- [x] 门票库存扣减（事务内原子操作，`UPDATE ... WHERE stock > 0` + 检查影响行数）
- [x] 售罄/不存在门票的错误响应（`ErrTicketNotFound` + `ErrTicketSoldOut`）
- [x] 删除活动时级联删除门票（`DeleteEvent` 事务加入 `DELETE FROM tickets`）

### 验证结果
- [x] 创建活动 → 创建 2 种门票（免费票库存2 + VIP票库存1）
- [x] 报名时关联免费票 → stock 2→1 ✅
- [x] 报名时关联VIP票 → stock 1→0 ✅
- [x] 尝试购买已售罄VIP票 → 409 "门票已售罄" ✅
- [x] 尝试购买不存在门票 → 404 "门票不存在" ✅
- [x] 普通报名（不选门票）→ 成功 ✅
- [x] 编辑门票（名称/价格/库存）→ 更新成功 ✅
- [x] 删除门票 → 确认已删除 ✅
- [x] 删除活动 → 级联清理门票数据 ✅

---

## 第十四阶段：管理员认证（安全管理） ✅

- [x] 新增 `ADMIN_TOKEN` 环境变量配置
- [x] 添加 `adminAuth` 中间件，不设置时自动跳过（开发模式兼容）
- [x] 保护活动创建/编辑/删除接口（`POST/PUT/DELETE /api/events`）
- [x] 保护门票创建/编辑/删除接口（`POST/PUT/DELETE /api/events/{id}/tickets`）
- [x] 报名、讨论区、列表查看等保持公开可访问
- [x] 测试验证：无 Token / 错误 Token → 401；正确 Token → 成功

---

## 第十五阶段：活动分类与搜索

- [x] 扩展 `Store.ListEvents` 方法，支持动态条件查询（status / price_type / keyword）
- [x] 更新 `ListEvents` Handler 解析 URL 查询参数
- [x] 状态筛选：`?status=published` 按活动状态过滤
- [x] 价格类型筛选：`?price_type=free`（免费） / `?price_type=paid`（付费）
- [x] 关键词搜索：`?q=Go` 按标题和描述模糊匹配
- [x] 多条件组合：`?status=published&price_type=paid` 同时满足

### 验证结果
- [x] 状态筛选测试：`GET /api/events?status=published` → 返回已发布活动 ✅
- [x] 价格类型筛选测试：`GET /api/events?price_type=free` → 仅返回免费活动 ✅
- [x] 价格类型筛选测试：`GET /api/events?price_type=paid` → 仅返回付费活动 ✅
- [x] 关键词搜索测试：`GET /api/events?q=Go` → 返回匹配活动 ✅
- [x] 组合筛选测试：`GET /api/events?status=published&price_type=paid` → 同时满足 ✅

---

## 后续思考 / 待讨论

### 待实现功能（优先级排序）

| 优先级 | 方向 | 说明 |
|--------|------|------|
| **D** | 🖥️ **前端界面** | Web / 小程序，等后端功能稳定后再投入 |
| **E** | 🧪 **自动化测试** | 单元测试 + 集成测试，使用 go test 框架 |
| **F** | 🚀 **CI/CD** | 配置 GitHub Actions，实现自动化构建和部署 |
| **G** | 📊 **监控** | 添加健康检查端点 + 日志中间件 |
| **H** | 📝 **文档** | 使用 Swagger 生成 API 文档 |
| **I** | 🔒 **安全审计** | 定期依赖扫描 + 代码审计 |

### 规划详情

#### 🖥️ D. 前端界面
- 待后端功能稳定后再投入
- 可考虑 Web / 小程序

#### 🧪 E. 自动化测试
- 为核心函数编写单元测试（Store CRUD、Handler 参数校验）
- 测试 API 端点和数据库交互（集成测试）
- 使用 `go test -cover` 监控测试覆盖率

#### 🚀 F. CI/CD
- 配置 GitHub Actions 工作流
- 代码提交时自动运行测试
- 实现自动化 Docker 镜像构建

#### 📊 G. 监控
- 添加 `/health` 健康检查端点
- 添加请求日志中间件（方法、路径、耗时、状态码）

#### 📝 H. 文档
- 使用 Swagger 注解生成 OpenAPI 文档
- 编写部署文档和开发规范

#### 🔒 I. 安全审计
- 使用 `gosec` 工具扫描代码
- 定期检查依赖包安全漏洞（`go list -m -u all`）
