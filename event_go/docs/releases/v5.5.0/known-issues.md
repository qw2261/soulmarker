# v5.5.0 Known Issues

- G3-R03 只完成报名/取消；讨论发帖仍由 Handler 协调报名检查与写入。
- 当前实体 ID 由 SQLite 生成；Admission/Order 引入领域 ID 时仍需注入 ID generator。
- OpenAPI、`/api/v1` 和更细粒度的稳定业务错误码尚未建立。
- 前端 unit/component 与浏览器 E2E 尚未建立。
- 前端主包约 1.02 MB，仍有 Vite chunk size 告警。
