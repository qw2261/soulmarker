# v6.0.0 Release Notes

> 状态：In Progress，当前为 G4 Admission/Checkin 纵向切片

## 当前切片

- Schema v6 新增独立 `admissions` 与 `checkins`，不把核销状态塞入 Registration。
- 免费报名和免费票在同一事务内签发 Admission；付费票不生成免费入场权益。
- Admission 凭证使用注入的加密随机生成器，用户可在活动详情和“我的活动”查看二维码。
- 取消报名会吊销未核销 Admission；已核销报名禁止取消，避免库存和履约状态冲突。
- Checkin 由数据库触发器禁止更新和删除；同一凭证的重复/并发核销只保留一条记录并返回原结果。
- 运营报名页支持扫码枪/粘贴凭证核销和审计列表。
- 建立 Vitest 组件门禁与 Playwright 桌面/移动核心旅程，CI 保存浏览器证据。
- 移动导航改为抽屉，用户凭证页与运营核销工具无页面级横向溢出。
- 新增 `/me/activities` 统一只读投影，在服务端精确合并、排序和分页 Admission 历史与其他 Registration。
- Element Plus 改为显式注册实际组件，认证页面加入后主 JS 从约 1.03 MB 降至约 481 KB，消除 Vite chunk size 告警。
- Schema v7 新增独立用户认证版本与密码重置 Token；退出和密码重置会立即撤销全部旧 JWT。
- 新增密码重置申请/确认 API 和前端页面；Token 为 256 位随机值，数据库只保存 SHA-256 摘要，并实施过期、替代、单次消费和一分钟限流。
- 新注册要求邮箱与 8–72 字节密码，旧 contact 登录保持兼容；生产环境缺 HTTPS 公开地址或 SMTP 配置时拒绝启动。
- 前端增加受保护路由、Token 过期安全回跳和迟到 401 竞态保护；桌面/移动 E2E 覆盖过期、重新登录、退出与未知账户重置申请。

## 未完成范围

G4-R01、R04-R06、R09、完整用户/运营 E2E 和两场受控测试活动仍未完成。G4-R03 已完成本地代码闭环，但历史 phone-only 账户恢复与真实 staging SMTP 验收仍未完成，因此 R03 和 M1 均不标记完成。
