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

# 报名活动
curl -s -X POST http://localhost:8080/api/events/1/register \
  -H "Content-Type: application/json" \
  -d '{"name":"张三","contact":"zs@email.com"}'

# 删除活动
curl -s -X DELETE http://localhost:8080/api/events/1
```

---

## 后续思考 / 待讨论

- API 是否要加认证（token）？
- 是否要提供前端 UI（Web / 小程序）？
- 要不要支持活动分类和搜索？
- Docker volume 映射让数据持久化
- 讨论区功能（迭代路线第二步）
