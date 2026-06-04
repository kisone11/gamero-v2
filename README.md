# Gamero v2

Gamero 是一个游戏共创社区平台，包含 Web 前端、Go 后端和 HarmonyOS ArkTS 原生 App。

## 目录

- `frontend/` Web 前端，Vite + React
- `backend/` Go 后端 API
- `harmony/` HarmonyOS ArkTS App
- `scripts/` 本地部署和启动脚本

## 一键本地部署

Windows 环境直接运行：

```bat
deploy-local-reset.bat
```

脚本会执行：

- 拉起 Docker 服务：PostgreSQL、Redis、MinIO、Mailpit
- 删除同名数据库 `gamero` 和同名用户 `gamero`
- 新建数据库和用户
- 创建 MinIO bucket `gamero`
- 安装前后端依赖
- 运行数据库迁移
- 插入管理员账号
- 插入演示数据
- 构建前端和后端
- 启动后端和前端

默认地址：

```text
Frontend: http://127.0.0.1:5714
Backend:  http://127.0.0.1:8081/health
MinIO:    http://127.0.0.1:9001
Mailpit:  http://127.0.0.1:8025
```

默认管理员：

```text
email: admin@gamero.local
password: Gamero123
```

## 配置

首次部署会从模板生成：

```text
backend/config/config.yaml
```

模板文件：

```text
backend/config/config.example.yaml
```

`config.yaml` 属于本地配置，已被 `.gitignore` 排除。

## HarmonyOS App

用 DevEco Studio 6.1.1.280 打开：

```text
harmony
```

模拟器默认后端地址：

```text
http://10.0.2.2:8081/api/v1
```

真机运行时，需要把 `harmony/entry/src/main/ets/services/ApiConfig.ets` 中的 `10.0.2.2` 改为电脑局域网 IP。
