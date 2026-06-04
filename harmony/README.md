# Gamero HarmonyOS App

这是 Gamero 网站的鸿蒙版适配工程，使用 ArkTS + Stage 模型实现。界面风格参考 Web 端：深色背景、琥珀色主色、卡片式布局、游戏开发社区氛围。

## 已实现内容

- 登录/注册前置，未登录只能访问认证页
- 移动端首页动态流，登录后从真实 `/feed` 拉取
- 项目发现页
- 招募协作页
- 我的/管理员概览页
- 登录页、注册页、退出登录
- 邮箱验证码注册
- 每个页面均可滚动
- Web 端功能入口清单页
- 项目任务看板展示
- 管理后台数据卡片展示
- 管理后台用户治理、举报处理和帖子/日志隐藏审核
- 管理后台敏感词列表、批量添加和删除
- 管理后台按 ID 删帖、关闭招募、下架项目、软删除项目
- 我的开发日志和我的举报列表
- 我的项目/帖子/日志收藏列表
- 通知偏好开关设置
- 账号设置中的技能管理和作品集管理
- 项目封面/截图/视频上传
- 帖子详情图片/视频追加上传
- 开发日志详情图片/视频追加上传
- 与 Web 端一致的暗色视觉风格
- 真实后端 API 配置

当前版本是可导入 DevEco Studio 的 ArkTS 应用工程，已接入真实后端接口。未登录时只显示登录/注册页；登录或注册成功后才进入动态、项目、招募、功能、我的主界面。

动态页不会再用演示数据伪装成真实动态。如果 `/api/v1/feed` 没有返回数据，会显示“暂无真实动态数据”。

## 工程位置

```text
D:\Game development\gamero-v2\harmony
```

## DevEco Studio 6.1.1.280 环境配置

### 1. 安装 DevEco Studio

安装你提到的版本：

```text
devecostudio-windows-6.1.1.280
```

建议安装路径不要包含中文，例如：

```text
D:\Huawei\DevEco Studio
```

### 2. 首次启动配置 SDK

打开 DevEco Studio 后按以下步骤：

1. 进入 `Settings`。
2. 打开 `OpenHarmony SDK` 或 `HarmonyOS SDK`。
3. 选择 SDK 安装目录，例如：

```text
D:\Huawei\HarmonyOS\Sdk
```

4. 安装以下组件：
   - ArkTS
   - JS/ArkUI
   - Toolchains
   - Previewer
   - Emulator 或 Local Emulator

如果 DevEco Studio 提示自动下载 SDK，直接允许下载。

### 3. 配置 Node.js

DevEco Studio 6.x 通常会自带 Node.js。若提示 Node 不可用：

1. 打开 `Settings`。
2. 搜索 `Node.js`。
3. 选择 DevEco 自带 Node，或选择本机 Node。

建议 Node 版本使用 DevEco 推荐版本，不要强行使用太新的版本。

### 4. 打开项目

在 DevEco Studio 里：

```text
File -> Open
```

选择目录：

```text
D:\Game development\gamero-v2\harmony
```

注意：选择 `harmony` 目录，不是 `frontend`，也不是 `backend`。

### 5. 等待工程同步

打开后等待 DevEco Studio 自动执行：

- hvigor sync
- SDK index
- ArkTS analysis

如果底部提示 `Sync Now`，点击同步。

### 6. 配置签名

调试运行一般可以使用自动签名：

```text
File -> Project Structure -> Signing Configs
```

启用：

```text
Automatically generate signature
```

如果只是用预览器，可以先不配置真机签名。

### 7. 启动模拟器

打开：

```text
Device Manager
```

创建或启动一个 Phone 模拟器。

建议选择 HarmonyOS 5.x / API 12 附近的设备镜像。

### 8. 运行 App

顶部运行配置选择：

```text
entry
```

然后点击绿色运行按钮。

运行成功后会打开 Gamero 鸿蒙版首页。

## 后端地址配置

鸿蒙 App 的后端地址配置在：

```text
entry/src/main/ets/services/ApiConfig.ets
```

默认：

```text
http://10.0.2.2:8081/api/v1
```

已接入的真实接口：

```text
POST /api/v1/auth/email/code
POST /api/v1/auth/register/email
POST /api/v1/auth/login/email
GET  /api/v1/projects
POST /api/v1/projects
GET  /api/v1/projects/:id
PATCH /api/v1/projects/:id
DELETE /api/v1/projects/:id
POST /api/v1/projects/:id/follow
DELETE /api/v1/projects/:id/follow
POST /api/v1/projects/:id/collect
DELETE /api/v1/projects/:id/collect
POST /api/v1/projects/:id/members
PATCH /api/v1/projects/:id/members/:userId
DELETE /api/v1/projects/:id/members/:userId
DELETE /api/v1/projects/:id/members/me
GET  /api/v1/recruitments
POST /api/v1/recruitments
GET  /api/v1/recruitments/:id
PUT  /api/v1/recruitments/:id
PATCH /api/v1/recruitments/:id/close
POST /api/v1/recruitments/:id/reopen
DELETE /api/v1/recruitments/:id
POST /api/v1/recruitments/:id/apply
GET  /api/v1/applications/mine
GET  /api/v1/feed
GET  /api/v1/community/posts
POST /api/v1/community/posts
PATCH /api/v1/community/posts/:id
DELETE /api/v1/community/posts/:id
GET  /api/v1/discover/logs
POST /api/v1/projects/:id/logs
GET  /api/v1/projects/:id/releases
PATCH /api/v1/logs/:id
DELETE /api/v1/logs/:id
GET  /api/v1/notifications
GET  /api/v1/notifications/unread-count
PATCH /api/v1/notifications/:id/read
PATCH /api/v1/notifications/read-all
DELETE /api/v1/notifications/:id
GET  /api/v1/search
GET  /api/v1/talents
GET  /api/v1/users/:id
GET  /api/v1/users/me
GET  /api/v1/me/logs
GET  /api/v1/me/reports
GET  /api/v1/me/collections/projects
GET  /api/v1/me/collections/posts
GET  /api/v1/me/collections/logs
GET  /api/v1/notifications/preferences
PATCH /api/v1/notifications/preferences
PATCH /api/v1/users/me/profile
PATCH /api/v1/users/me/availability
PUT  /api/v1/users/me/skills
POST /api/v1/users/me/portfolio
DELETE /api/v1/users/me/portfolio/:id
POST /api/v1/users/:id/follow
DELETE /api/v1/users/:id/follow
GET  /api/v1/admin/stats
GET  /api/v1/admin/users
POST /api/v1/admin/users/:id/ban
DELETE /api/v1/admin/users/:id/ban
GET  /api/v1/admin/reports
PATCH /api/v1/admin/reports/:id
PATCH /api/v1/admin/posts/:id/hide
DELETE /api/v1/admin/posts/:id
PATCH /api/v1/admin/logs/:id/hide
DELETE /api/v1/admin/recruitments/:id
PUT  /api/v1/admin/projects/:id/ban
DELETE /api/v1/admin/projects/:id
GET  /api/v1/admin/sensitive-words
POST /api/v1/admin/sensitive-words
DELETE /api/v1/admin/sensitive-words/:word
POST /api/v1/upload/token
PATCH /api/v1/projects/:id/cover
POST /api/v1/projects/:id/screenshots
POST /api/v1/projects/:id/videos
PUT  /api/v1/community/posts/:id/images
PUT  /api/v1/community/posts/:id/videos
POST /api/v1/logs/:id/images
PUT  /api/v1/logs/:id/videos
GET  /api/v1/projects/:id/tasks
POST /api/v1/projects/:id/tasks
PATCH /api/v1/projects/:id/tasks/:taskID
DELETE /api/v1/projects/:id/tasks/:taskID
```

App 打开时只显示登录/注册页，不会进入主界面。

登录页默认填入管理员账号，点击“登录并加载真实数据”后会调用真实后端登录接口，并拉取动态流、项目列表、招募列表、项目任务、管理员统计等数据。

```text
邮箱：admin@gamero.local
密码：Gamero123
```

登录后会继续加载：动态流、项目列表、招募列表、管理员统计、项目任务。

登录成功后进入五个底部 Tab：

```text
动态
项目
招募
功能
我的
```

注册流程：

```text
1. 在启动认证页切换到“注册”并输入邮箱
2. 点击“发送验证码”调用 POST /api/v1/auth/email/code
3. 填入验证码、昵称、密码
4. 点击“注册并登录”调用 POST /api/v1/auth/register/email
```

退出登录：

```text
我的 -> 退出登录
```

### 模拟器访问本机后端

如果你用 DevEco 模拟器，通常使用：

```text
http://10.0.2.2:8081/api/v1
```

### 真机访问本机后端

如果你用真机，需要改成本机局域网 IP，例如：

```text
http://192.168.1.10:8081/api/v1
```

同时确保：

- 手机和电脑在同一个 Wi-Fi
- 后端监听 `0.0.0.0` 或电脑局域网 IP
- Windows 防火墙允许 8081 端口

## 启动 Web 后端服务

在项目根目录可以使用你现有的启动脚本：

```text
D:\Game development\gamero-v2\start-local.bat
```

首次安装依赖和检查配置可以运行：

```text
D:\Game development\gamero-v2\install-deps-and-config.bat
```

该脚本会执行：

```text
frontend npm install
backend go mod download
backend go mod tidy
harmony ohpm install（如果 ohpm 在 PATH 中）
harmony hvigor --sync（如果 hvigor 在 PATH 中）
frontend npm run build
backend go build ./...
```

或者分别启动后端和前端。鸿蒙 App 只需要后端 `8081` 可访问。

管理员账号：

```text
邮箱：admin@gamero.local
密码：Gamero123
```

## 当前页面说明

### 动态

展示开发日志、项目进展、技术分享等信息流。

数据来自真实 `/api/v1/feed`。没有真实数据时显示空状态，不再展示 demo 动态。

### 项目

展示游戏项目卡片，并包含任务看板入口样式。

### 招募

展示项目招募岗位、合作方式和有效期。

### 我的

展示当前账号、后端地址、管理员模块概览。

### 功能

按 Web 端路由整理移动端功能入口，包括：

```text
动态流
社区帖子
项目发现
项目创建/编辑
项目成员
项目版本
项目任务看板
开发日志
招募协作
招募申请/编辑
人才市场
搜索发现
通知中心
个人主页
账号设置
我的日志/举报
管理后台
```

当前 App 已完成认证、真实动态流、项目列表、创建项目、编辑项目、删除项目、项目详情、项目关注/收藏、项目成员添加/改角色/移除/退出、项目版本发布/版本记录、项目任务创建/状态更新/删除、招募列表、创建/编辑/关闭/重开/删除招募、招募详情、申请招募、我的申请、社区帖子列表、发布/编辑/删除帖子、开发日志列表、发布/编辑/删除开发日志、通知列表、未读通知数、通知已读/全部已读/删除、搜索、人才市场、用户主页、关注用户、账号资料和合作状态设置、管理员统计等核心接口接入。Web 端所有功能已先补齐移动端入口清单；上传、管理审核等复杂交互还需要继续逐项实现为 ArkTS 原生页面。

## 后续真实接口接入建议

当前已接入基础接口。后续建议继续接：

1. 登录接口：`POST /api/v1/auth/login/email`
2. 项目列表：`GET /api/v1/projects`
3. 招募列表：`GET /api/v1/recruitments`
4. 动态流：`GET /api/v1/feed`
5. 项目任务：`GET /api/v1/projects/:id/tasks`
6. 管理后台统计：`GET /api/v1/admin/stats`
7. 创建/编辑项目任务：`POST/PATCH /api/v1/projects/:id/tasks`
8. 社区帖子详情与评论：`GET /api/v1/community/posts/:id`
9. 开发日志详情与评论：`GET /api/v1/logs/:id`
10. 通知偏好设置：`GET/PATCH /api/v1/notifications/preferences`
11. 创建/编辑社区帖子：`POST/PATCH /api/v1/community/posts`
12. 创建/编辑开发日志：`POST /api/v1/projects/:id/logs`、`PATCH /api/v1/logs/:id`

## 与 Web 端风格对应关系

| Web 端 | 鸿蒙端 |
| --- | --- |
| 深色背景 `surface-void` | `#0B0D12` |
| 卡片背景 `surface-card` | `#151922` |
| 主色 amber | `#F5A623` |
| 圆角卡片 | ArkUI `borderRadius(18)` |
| 顶部导航 | 移动端 Logo Header |
| Web Tabs | 底部五 Tab 导航 |
