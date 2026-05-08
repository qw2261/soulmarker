# 亦闻 event-go MVP 任务跟踪（核心版）

> 目标：跑通"创建活动 → 浏览活动 → 报名参与 → 讨论互动"完整闭环
> 技术栈：Go 标准库 + SQLite + Docker
> 全量版见 `mvp_task.md`，测试数据见 `test_reports/`

---

## 进度总览

| 阶段 | 状态 | 要点 |
|------|------|------|
| 一、项目初始化 | ✅ | 方案确定，结构清晰 |
| 二、数据模型（v1） | ✅ | Event + Registration，内存版 |
| 三、API 接口（v1） | ✅ | 5 个接口全部实现 |
| 四、健壮性（v1） | ✅ | 参数校验、边界检查、CORS |
| 五、Docker 部署（v1） | ✅ | 多阶段构建，镜像 17.6MB |
| 六、SQLite 持久化（v2） | ✅ | 纯 Go SQLite，零外部依赖 |
| 七、活动编辑管理（v2） | ✅ | PUT/DELETE，局部更新（指针） |
| 八、Docker 部署（v2） | ✅ | Go 1.25，镜像 ~24MB，VOLUME 持久化 |
| 九、优化与复盘 | ✅ | 哨兵错误 + `errors.Is` |
| 十、代码结构重构 | ✅ | 依赖注入，消除全局变量 |
| 十一、讨论区 | ✅ | 帖子/回复 CRUD + 报名者权限 |
| 十二、代码审计与修复 | ✅ | P0-P3 全面修复（事务、错误处理等） |
| 十三、门票管理 | ✅ | CRUD + 报名关联 + 库存扣减 |
| 十四、管理员认证 | ✅ | API Token 中间件 |
| 十五、活动分类与搜索 | ✅ | 状态/价格/关键词组合筛选 |
| 十六、自动化测试 | ✅ | 81 用例，覆盖率 75.2% |
| 十七、目录结构重构 | ✅ | internal/ 分层（handler/model/store/config） |
| 十八、监控与日志 | ✅ | /health + JSON 结构化日志（slog） |
| 十九、全面综合评测 | ✅ | 质量/性能/安全/架构 6 维度评测 |
| 二十、持续优化 | ✅ | 索引、注释、配置集中、日志分级 |
| 二十一、分页 | ✅ | 4 列表接口 page/page_size/total |
| 二十二、报名取消 | ✅ | DELETE 取消 + 24h 截止 + 退还库存 |
| **二十三、门店 + 用户 + 前端** | ✅ | Organizer 层级 + JWT 注册登录 + Vue 3 前端 |
| **二十四、门店详情 + 按店筛选** | ✅ | 门店详情页 + `?organizer_id=` 筛选 |
| **二十五、覆盖率提升 + 依赖升级** | ✅ | 176 用例 / Store 80% / Handler 75% / race clean |

---

## v5.0 摘要

```
25 个 API 接口 | 176 测试用例 | Store 80% / Handler 75% 覆盖率
go test -race 零竞争 | go vet 零警告 | 所有依赖最新
门店列表 → 门店详情 → 旗下活动：完整闭环
Organizer → Event → Ticket → Registration → Post → Reply 两级实体架构
X-Admin-Token（管理）+ Authorization Bearer JWT（用户）双通道认证
JWT 自动识别 | SQLite WAL + 事务 | JSON 结构化日志
```

## v4.5 摘要（历史）

```
25 个 API 接口 | 门店详情页 + 按门店筛选活动 | Vue 3 前端
门店列表 → 门店详情 → 旗下活动：完整闭环
```

---

## 快速启动

```bash
cd event_go && go run .                # 本地
cd event_go && docker build -t event-go . && docker run -p 8080:8080 event-go   # Docker
docker run -p 8080:8080 -v $(pwd)/data:/app/data event-go   # 数据持久化
```

---

## 🔧 小改动

- [x] **main.go 走 config 统一配置** — `DB_PATH` / `PORT` 改用 `config.Load()`

---

## 待实现功能

### P1 — 近期重点

| 方向 | 说明 |
|------|------|
| 🎫 **报名免填身份 + 二维码核销** | 登录后报名只需选门票点确认；报名成功生成二维码，管理员扫码核销。Registration 表加 user_id + check_code |
| 📱 **H5 移动端适配** | 响应式 CSS + viewport + 底部 Tab 导航，现有 Vue 代码改样式即可，后续可平滑迁移小程序 |
| 🖼️ **活动封面图** | `Event` 加 `cover_image_url`，数据模型 + 卡片展示 + 详情页 |
| 🏷️ **按门店聚合视图** | 首页默认活动列表，增加"按门店浏览"视图切换 |
| 🛡️ **限流** | 基于 IP 的请求频率限制，防滥用 |

### P2 — 中期规划

| 方向 | 说明 |
|------|------|
| 📨 **报名通知** | 报名后邮箱/站内通知，增加用户留存 |
| 📤 **数据导出** | 报名 CSV 导出 |
| ✏️ **内容删除** | 帖子/回复管理端/作者删除 |
| 📞 **联系方式校验** | 邮箱/手机号格式校验 |
| 🏠 **树莓派部署** | ARM64 Docker + 内网穿透 |

### P3 — 后续关注

| 方向 | 说明 |
|------|------|
| 📖 **API 文档** | Swagger / OpenAPI |
| ⏰ **活动状态自动更新** | 过期活动自动归档 ended |

### P4 — 持续交付

| 方向 | 说明 |
|------|------|
| 🚀 **CI/CD** | GitHub Actions 自动测试 + 构建 |

---

## 测试

```bash
cd event_go && go test -v -count=1 ./...   # 全部测试（176 用例）
cd event_go && go test -cover ./...         # 覆盖率（Store 80%, Handler 75%）
cd event_go && go test -race ./...          # 数据竞争检测
cd event_go && go vet ./...                 # 静态检查
```
