# v5.5.0 Known Issues

- G3-R03 至 G3-R08 尚未实现，当前 Handler 仍直接协调部分跨实体事务。
- clock 与 token signer 尚未抽象为可注入依赖。
- OpenAPI、`/api/v1` 和更细粒度的稳定业务错误码尚未建立。
- 前端 unit/component 与浏览器 E2E 尚未建立。
- 前端主包约 1.02 MB，仍有 Vite chunk size 告警。
