# v5.4.0 Known Issues

- legacy 身份记录仍需运营人工确认，当前只提供受保护的 API 清单，尚无管理 UI。
- 前端 unit/component 与浏览器 E2E 尚未建立。
- 前端主包约 1.02 MB，仍有 Vite chunk size 告警。
- SQLite 采用单连接、单实例写入模型；多实例和真实支付前需完成 G6 数据库决策。
- OpenAPI、稳定业务错误码细分和 application service 重构属于 G3。
