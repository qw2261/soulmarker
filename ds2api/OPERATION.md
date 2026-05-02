# DS2API 操作文档

本目录为 DS2API v4.1.3-beta2 (darwin/amd64) Release 构建包，以下是如何启动服务的完整操作指南。

---

## 目录结构

```
ds2api_v4.1.3_beta2_darwin_amd64/
├── ds2api            # 可执行文件（已编译好的 Go 二进制）
├── .env              # 环境变量配置（DS2API 自动加载）
├── config.json       # 主配置文件
├── static/           # WebUI 管理面板静态文件
│   └── admin/
│       ├── index.html
│       └── assets/
├── OPERATION.md      # 本操作文档
├── README.MD         # 项目说明
├── README.en.md      # 英文说明
└── LICENSE           # 许可证
```

---

## 第一步：配置文件 ✅ 已完成

配置文件已就绪，使用 `.env` + `config.json` 组合。无需额外操作。

如需修改，直接编辑 `config.json`：

**关键字段速查：**

| 字段 | 说明 |
|------|------|
| `keys` | API Key 列表，客户端请求时需要携带 |
| `api_keys` | API Key 的详细配置（key/名称/备注） |
| `accounts` | DeepSeek 账号，邮箱或手机号 + 密码 |

---

## 第二步：赋予可执行权限

```bash
chmod +x /Users/qi/Documents/trae_projects/ds2api/ds2api_v4.1.3_beta2_darwin_amd64/ds2api
```

（通常 Release 包已有执行权限，若报 `Permission denied` 再执行此步。）

---

## 第三步：启动服务

### 方式一：在 Release 目录下直接运行（推荐）

**v4.1.3-beta2:**

```bash
cd /Users/qi/Documents/trae_projects/ds2api/ds2api_v4.1.3_beta2_darwin_amd64
./ds2api
```

**v4.2.1:**

```bash
cd /Users/qi/Documents/trae_projects/ds2api/ds2api_v4.2.1_darwin_amd64
./ds2api
```

此时 `ds2api` 会自动查找当前目录下的 `config.json` 并监听默认端口 `:5001`。

### 方式二：从任意目录运行（指定配置文件路径）

```bash
/Users/qi/Documents/trae_projects/ds2api/ds2api_v4.1.3_beta2_darwin_amd64/ds2api \
  -config /Users/qi/Documents/trae_projects/ds2api/ds2api_v4.1.3_beta2_darwin_amd64/config.json \
  -addr :5001
```

### 方式三：使用环境变量

```bash
# 自定义端口
DS2API_ADDR=":8080" /Users/qi/Documents/trae_projects/ds2api/ds2api_v4.1.3_beta2_darwin_amd64/ds2api

# 指定配置文件路径
DS2API_CONFIG="/path/to/your/config.json" /Users/qi/Documents/trae_projects/ds2api/ds2api_v4.1.3_beta2_darwin_amd64/ds2api
```

---

## 第四步：验证服务

服务启动后，可以通过以下接口验证：

```bash
# 健康检查
curl http://localhost:5001/healthz
# 预期返回: ok

# 就绪检查
curl http://localhost:5001/readyz
# 预期返回: ready

# 模型列表（需要 API Key）
curl http://localhost:5001/v1/models \
  -H "Authorization: Bearer sk-your-custom-api-key-here"
```

- **WebUI 管理台**: 浏览器打开 `http://localhost:5001/admin`

---

## 命令行参数速查

| 参数 | 环境变量 | 默认值 | 说明 |
|------|---------|--------|------|
| `-config` | `DS2API_CONFIG` | `config.json`（二进制所在目录） | 配置文件路径 |
| `-addr` | `DS2API_ADDR` | `:5001` | HTTP 监听地址与端口 |
| `-loglevel` | `DS2API_LOGLEVEL` | `info` | 日志级别（debug/info/warn/error） |

---

## 设置全局别名（可选）

如果希望在任何目录都能一键启动，可将以下内容添加到 `~/.zshrc`：

```bash
alias ds2api-start='/Users/qi/Documents/trae_projects/ds2api/ds2api_v4.1.3_beta2_darwin_amd64/ds2api -config /Users/qi/Documents/trae_projects/ds2api/ds2api_v4.1.3_beta2_darwin_amd64/config.json'
```

之后在任何终端输入 `ds2api-start` 即可启动服务。

---

## 客户端接入示例

```bash
# OpenAI SDK / 兼容客户端
export OPENAI_API_KEY="sk-your-custom-api-key-here"
export OPENAI_BASE_URL="http://localhost:5001/v1"

# 测试请求
curl http://localhost:5001/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-custom-api-key-here" \
  -d '{
    "model": "deepseek-v4-flash",
    "messages": [
      {"role": "user", "content": "你好，请用一句话介绍你自己"}
    ]
  }'
```

---

## 停止服务

在运行服务的终端按 `Ctrl + C` 即可停止。

---

## 常见问题

**Q: 启动报 `config.json: no such file or directory`？**

A: 确认 `config.json` 和 `.env` 文件是否在 `ds2api` 二进制同目录下，且文件名是否正确。

**Q: 请求返回 401 Unauthorized？**

A: 检查 `Authorization` header 中的 API Key 是否与 `config.json` 中 `keys` 列表中的某个值一致。

**Q: 请求返回账号/登录相关错误？**

A: 检查 `config.json` 中 `accounts` 的邮箱/手机号和密码是否正确，确保账号在 DeepSeek 网页端可以正常登录。

**Q: 如何升级到新版本？**

A: 从 GitHub Release 页面下载新版本的构建包，替换 `ds2api` 二进制文件即可（`config.json` 和 `static/` 目录通常不需要更新）。

---

## 参考链接

- 项目仓库: [github.com/CJackHwang/ds2api](https://github.com/CJackHwang/ds2api)
- 接口文档: [API.md](https://github.com/CJackHwang/ds2api/blob/main/API.md)
- 部署指南: [docs/DEPLOY.md](https://github.com/CJackHwang/ds2api/blob/main/docs/DEPLOY.md)
- 管理后台: [http://localhost:5001/admin/](http://localhost:5001/admin/)

---

> 最后更新: 2026-05-01
