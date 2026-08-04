# ImageShare

[中文](#中文文档) | [English](#english-doc)

---

# 中文文档

轻量级自建图床系统，支持管理员后台、用户图库、游客分享上传。单文件部署，零依赖。

## 功能特性

- **单文件部署** - Go 编译的可执行文件内嵌前端，无需额外依赖
- **管理后台** - 图片管理、用户管理、游客链接管理、系统日志
- **用户系统** - 独立图库，支持存储空间和图片数量配额
- **游客上传** - 可分享的上传链接，支持过期时间和上传次数限制
- **图片直链** - 通过 `/i/{code}` 访问图片，不暴露真实路径
- **安全防护** - JWT 认证、bcrypt 密码加密、上传校验（扩展名+MIME）、限流、XSS/SQL 注入防护
- **自动刷新** - 仅当前页面 5 秒轮询，不分页模式滚动加载
- **格式支持** - JPG、JPEG、PNG、GIF、WebP
- **配置文件** - 支持 `//` 注释的 JSON 配置
- **日志系统** - 自动轮转、保留数量控制、ANSI 彩色控制台输出

## 快速开始

### Windows

1. 下载 `imageshare-windows-amd64.exe`
2. 命令行运行：
   ```cmd
   imageshare-windows-amd64.exe
   ```
3. 首次启动自动创建 `config.json`、`database/`、`uploads/`、`logs/`
4. 浏览器打开 `http://localhost:8080`

### Linux

1. 下载对应架构的二进制文件：
   - x86_64 服务器：`imageshare-linux-amd64`
   - ARM64 服务器：`imageshare-linux-arm64`

2. 添加执行权限并运行：
   ```bash
   chmod +x imageshare-linux-amd64
   ./imageshare-linux-amd64
   ```

3. 首次启动自动创建 `config.json`、`database/`、`uploads/`、`logs/`
4. 浏览器打开 `http://localhost:8080`

### 后台运行（Linux）

```bash
# 使用 nohup
nohup ./imageshare-linux-amd64 > /dev/null 2>&1 &

# 或使用 systemd（推荐）
sudo tee /etc/systemd/system/imageshare.service << 'EOF'
[Unit]
Description=ImageShare
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/imageshare
ExecStart=/opt/imageshare/imageshare-linux-amd64
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable imageshare
sudo systemctl start imageshare
```

### 默认管理员

```
用户名：admin
密码：image123456
```

首次登录后强制修改密码。

### 命令行修改管理员密码

```bash
# Windows
imageshare.exe -changepasswd <新密码>
```
```bash
# Linux
./imageshare-linux-amd64 -changepasswd <新密码>
```

## 配置说明

编辑 `config.json`（支持 `//` 注释）：

```json
{
    "server": {
        "host": "0.0.0.0",
        "port": 8080
    },
    "jwt_secret": "your-random-secret",
    "upload_path": "./uploads",
    "default_user_storage_limit_mb": 100,
    "default_user_image_limit": 50,
    "default_user_single_image_limit_mb": 10,
    "log_retention": 7
}
```

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `server.host` | `0.0.0.0` | 监听地址。`0.0.0.0` 监听所有网卡，`127.0.0.1` 仅本机访问 |
| `server.port` | `8080` | 监听端口 |
| `jwt_secret` | `image_share_secret_key_2024` | JWT 签名密钥，请修改为随机字符串 |
| `upload_path` | `./uploads` | 图片存储路径，相对可执行文件目录或绝对路径 |
| `default_user_storage_limit_mb` | `100` | 新用户默认存储配额（MB） |
| `default_user_image_limit` | `50` | 新用户默认图片数量上限 |
| `default_user_single_image_limit_mb` | `10` | 新用户默认单张图片大小上限（MB） |
| `log_retention` | `7` | 日志文件保留数量 |

## 使用指南

### 管理员

登录后可使用：

- **仪表盘** - 查看图片总数、用户数、游客任务数、存储占用
- **图片管理** - 浏览、预览、复制链接、删除图片，支持分页或滚动加载
- **用户管理** - 创建/编辑/删除用户，设置配额，重置密码
- **游客链接** - 创建可分享的上传链接，设置过期时间和上传次数
- **系统日志** - 查看操作日志，自动刷新

### 普通用户

管理员创建用户账号后，用户可：

- 上传图片（在配额内）
- 查看自己的图库
- 复制图片直链
- 删除自己的图片

### 游客

通过分享链接访问上传页面：

```
http://your-domain.com/upload/{code}
```

无需登录，受链接过期时间和上传次数限制。

## API 接口

### 认证

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| `POST` | `/api/auth/login` | 否 | 登录。Body: `{username, password}` |
| `POST` | `/api/auth/logout` | 是 | 登出 |
| `GET` | `/api/auth/verify` | 是 | 验证 Token |
| `PUT` | `/api/profile/password` | 是 | 修改密码。Body: `{old_password, new_password}` |

### 管理员接口

所有管理员接口需要 JWT Token 且角色为 admin。

**用户管理**

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/admin/users` | 创建用户。Body: `{username, password, storage_limit_mb, image_limit, single_image_limit_mb}` |
| `GET` | `/api/admin/users` | 用户列表。Params: `page`, `page_size`, `offset`, `limit` |
| `GET` | `/api/admin/users/:id` | 获取用户 |
| `PUT` | `/api/admin/users/:id` | 更新用户。Body: `{storage_limit_mb, image_limit, single_image_limit_mb}` |
| `PUT` | `/api/admin/users/:id/password` | 重置密码。Body: `{new_password}` |
| `DELETE` | `/api/admin/users/:id` | 删除用户 |

**游客任务**

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/admin/tasks` | 创建任务。Body: `{max_count, expire_days}` |
| `GET` | `/api/admin/tasks` | 任务列表。Params: `page`, `page_size`, `offset`, `limit` |
| `GET` | `/api/admin/tasks/:id` | 获取任务 |
| `PUT` | `/api/admin/tasks/:id` | 更新任务。Body: `{max_count, expire_days}` |
| `DELETE` | `/api/admin/tasks/:id` | 删除任务。Params: `delete_files` |

**图片**

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/admin/upload` | 上传图片。`multipart/form-data`，字段: `file` |
| `GET` | `/api/admin/images` | 图片列表。Params: `page`, `page_size`, `offset`, `limit` |
| `DELETE` | `/api/admin/images/:id` | 删除图片 |

**其他**

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/admin/stats` | 仪表盘统计 |
| `GET` | `/api/admin/logs` | 系统日志。Params: `file` |

### 用户接口

所有用户接口需要 JWT Token。

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/user/upload` | 上传图片。`multipart/form-data`，字段: `file` |
| `GET` | `/api/user/images` | 我的图片。Params: `page`, `page_size`, `offset`, `limit` |
| `DELETE` | `/api/user/images/:id` | 删除图片 |
| `GET` | `/api/user/stats` | 存储统计 |

### 游客接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/upload/:code` | 游客上传。`multipart/form-data`，字段: `file` |
| `GET` | `/api/upload/:code` | 检查链接状态 |
| `GET` | `/api/guest/:code` | 获取链接信息（无需认证） |

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/i/:code` | 图片直链访问 |

### 分页参数

列表接口支持两种模式：

**分页模式**（仅加载当前页）：
```
?page=1&page_size=20
```

**滚动模式**（批量加载）：
```
?offset=0&limit=30
```

响应格式：
```json
{
    "data": [...],
    "total": 100
}
```

## 从源码构建

### 前置要求

- Go 1.24+
- Node.js 18+

### 构建

可以优先使用**build.bat**构建
**clean.bat**是用来清理残留的（编译中下载的一些东西）

```bash
# 1. 构建前端
cd frontend
npm install
npm run build
cd ..

# 2. 复制前端到后端嵌入目录
# Windows PowerShell:
Remove-Item -Recurse -Force backend\cmd\frontend
Copy-Item -Recurse frontend\dist backend\cmd\frontend
Remove-Item -Recurse -Force backend\frontend
Copy-Item -Recurse frontend\dist backend\frontend

# Linux/macOS:
rm -rf backend/cmd/frontend && cp -r frontend/dist backend/cmd/frontend
rm -rf backend/frontend && cp -r frontend/dist backend/frontend

# 3. 编译后端
cd backend

# Windows amd64:
GOOS=windows GOARCH=amd64 go build -o ../../build/imageshare-windows-amd64.exe ./cmd/

# Linux amd64:
GOOS=linux GOARCH=amd64 go build -o ../../build/imageshare-linux-amd64 ./cmd/

# Linux arm64:
GOOS=linux GOARCH=arm64 go build -o ../../build/imageshare-linux-arm64 ./cmd/
```

## 技术栈

**后端：** Go、Gin、GORM、SQLite、JWT、bcrypt

**前端：** Vue 3、TypeScript、Vite、Element Plus、Pinia、Axios

## 项目结构

```
.
├── backend/
│   ├── cmd/
│   │   ├── main.go          # 入口 & 路由
│   │   └── frontend/        # 内嵌前端 (go:embed)
│   ├── config/              # 配置
│   └── internal/
│       ├── controller/      # HTTP 处理器
│       ├── service/         # 业务逻辑
│       ├── repository/      # 数据访问
│       ├── models/          # 数据模型
│       ├── middleware/       # JWT、限流
│       └── logger/          # 日志系统
└── frontend/
    └── src/
        ├── views/           # Vue 组件
        ├── router/          # Vue Router
        ├── stores/          # Pinia 状态
        └── utils/           # Axios 实例
```

## 开源协议

本项目基于 **GNU Affero General Public License v3.0** 开源。

- 你可以自由使用、修改和分发本软件
- 修改后的版本必须以相同协议开源
- 必须保留原作者版权声明
- 网络服务使用也必须公开源码（AGPL 特有条款）

详见 [LICENSE](LICENSE)。

---

# English Doc

A lightweight, self-hosted image hosting system with admin management, user galleries, and guest upload sharing. Single binary deployment, zero dependencies.

## Features

- **Single Binary Deployment** - Go compiled executable with embedded frontend, no extra dependencies needed
- **Admin Dashboard** - Image management, user management, guest link management, system logs
- **User System** - Independent galleries with storage/image count quotas
- **Guest Upload** - Shareable links with expiration time and upload count limits
- **Image Direct Link** - Access images via `/i/{code}`, no real path exposure
- **Security** - JWT auth, bcrypt password hashing, upload validation (extension + MIME), rate limiting, XSS/SQL injection protection
- **Auto Refresh** - 5s polling on active page only, scroll-to-load for non-paged mode
- **Format Support** - JPG, JPEG, PNG, GIF, WebP
- **Config File** - JSON config with comments support
- **Log System** - Auto rotation, retention control, ANSI colored console output

## Quick Start

### Windows

1. Download `imageshare-windows-amd64.exe`
2. run from command line:
   ```cmd
   imageshare-windows-amd64.exe
   ```
3. First launch auto-creates `config.json`, `database/`, `uploads/`, `logs/`
4. Open `http://localhost:8080` in browser

### Linux

1. Download the binary for your architecture:
   - x86_64 server: `imageshare-linux-amd64`
   - ARM64 server: `imageshare-linux-arm64`

2. Add execute permission and run:
   ```bash
   chmod +x imageshare-linux-amd64
   ./imageshare-linux-amd64
   ```

3. First launch auto-creates `config.json`, `database/`, `uploads/`, `logs/`
4. Open `http://localhost:8080` in browser

### Run in Background (Linux)

```bash
# Using nohup
nohup ./imageshare-linux-amd64 > /dev/null 2>&1 &

# Or using systemd (recommended)
sudo tee /etc/systemd/system/imageshare.service << 'EOF'
[Unit]
Description=ImageShare
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/imageshare
ExecStart=/opt/imageshare/imageshare-linux-amd64
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable imageshare
sudo systemctl start imageshare
```

### Default Admin

```
Username: admin
Password: image123456
```

You will be forced to change password on first login.

### Change Admin Password via CLI

```bash
# Windows
imageshare.exe -changepasswd <New-passwd>
```
```bash
# Linux
./imageshare-linux-amd64 -changepasswd <New-passwd>
```

## Configuration

Edit `config.json` (supports `//` comments):

```json
{
    "server": {
        "host": "0.0.0.0",
        "port": 8080
    },
    "jwt_secret": "your-random-secret",
    "upload_path": "./uploads",
    "default_user_storage_limit_mb": 100,
    "default_user_image_limit": 50,
    "default_user_single_image_limit_mb": 10,
    "log_retention": 7
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `server.host` | `0.0.0.0` | Listen address. `0.0.0.0` for all interfaces, `127.0.0.1` for local only |
| `server.port` | `8080` | Listen port |
| `jwt_secret` | `image_share_secret_key_2024` | JWT signing key. Change this for security |
| `upload_path` | `./uploads` | Image storage path, relative to executable or absolute |
| `default_user_storage_limit_mb` | `100` | Default storage quota for new users (MB) |
| `default_user_image_limit` | `50` | Default image count limit for new users |
| `default_user_single_image_limit_mb` | `10` | Default single image size limit for new users (MB) |
| `log_retention` | `7` | Number of log files to keep |

## Usage Guide

### Admin

After login, admin can:

- **Dashboard** - View total images, users, guest tasks, storage usage
- **Image Management** - Browse, preview, copy link, delete images. Support pagination or scroll-to-load
- **User Management** - Create/edit/delete users, set quotas, reset passwords
- **Guest Links** - Create shareable upload links with expiration and count limits
- **System Logs** - View operation logs with auto-refresh

### User

Admin creates user accounts. After login, user can:

- Upload images (within quota)
- View own gallery
- Copy image direct links
- Delete own images

### Guest

Guest accesses upload page via shared link:

```
http://your-domain.com/upload/{code}
```

No login required. Upload limited by link's expiration time and count limit.

## API Reference

### Authentication

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/auth/login` | No | Login. Body: `{username, password}` |
| `POST` | `/api/auth/logout` | Yes | Logout |
| `GET` | `/api/auth/verify` | Yes | Verify token validity |
| `PUT` | `/api/profile/password` | Yes | Change password. Body: `{old_password, new_password}` |

### Admin APIs

All admin APIs require JWT token with admin role.

**Users**

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/admin/users` | Create user. Body: `{username, password, storage_limit_mb, image_limit, single_image_limit_mb}` |
| `GET` | `/api/admin/users` | Get user list. Params: `page`, `page_size`, `offset`, `limit` |
| `GET` | `/api/admin/users/:id` | Get user by ID |
| `PUT` | `/api/admin/users/:id` | Update user. Body: `{storage_limit_mb, image_limit, single_image_limit_mb}` |
| `PUT` | `/api/admin/users/:id/password` | Reset user password. Body: `{new_password}` |
| `DELETE` | `/api/admin/users/:id` | Delete user |

**Guest Tasks**

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/admin/tasks` | Create task. Body: `{max_count, expire_days}` |
| `GET` | `/api/admin/tasks` | Get task list. Params: `page`, `page_size`, `offset`, `limit` |
| `GET` | `/api/admin/tasks/:id` | Get task by ID |
| `PUT` | `/api/admin/tasks/:id` | Update task. Body: `{max_count, expire_days}` |
| `DELETE` | `/api/admin/tasks/:id` | Delete task. Params: `delete_files` |

**Images**

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/admin/upload` | Upload image. `multipart/form-data`, field: `file` |
| `GET` | `/api/admin/images` | Get image list. Params: `page`, `page_size`, `offset`, `limit` |
| `DELETE` | `/api/admin/images/:id` | Delete image |

**Other**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/admin/stats` | Dashboard statistics |
| `GET` | `/api/admin/logs` | Get system logs. Params: `file` |

### User APIs

All user APIs require JWT token.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/user/upload` | Upload image. `multipart/form-data`, field: `file` |
| `GET` | `/api/user/images` | Get own images. Params: `page`, `page_size`, `offset`, `limit` |
| `DELETE` | `/api/user/images/:id` | Delete own image |
| `GET` | `/api/user/stats` | Get own storage stats |

### Guest APIs

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/upload/:code` | Upload image via guest link. `multipart/form-data`, field: `file` |
| `GET` | `/api/upload/:code` | Check guest link status |
| `GET` | `/api/guest/:code` | Get guest link info (no auth) |

### Public

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/i/:code` | Serve image by file code (direct link) |

### Pagination Parameters

List endpoints support two modes:

**Paged mode** (load current page only):
```
?page=1&page_size=20
```

**Scroll mode** (load in batches):
```
?offset=0&limit=30
```

Response format:
```json
{
    "data": [...],
    "total": 100
}
```

## Build from Source

### Prerequisites

- Go 1.24+
- Node.js 18+

### Build

It is recommended to use **build.bat** for building first.
**clean.bat** is used to clean up residual files (such as temporary files downloaded during the build process).

```bash
# 1. Build frontend
cd frontend
npm install
npm run build
cd ..

# 2. Copy frontend to backend embed directory
# Windows PowerShell:
Remove-Item -Recurse -Force backend\cmd\frontend
Copy-Item -Recurse frontend\dist backend\cmd\frontend
Remove-Item -Recurse -Force backend\frontend
Copy-Item -Recurse frontend\dist backend\frontend

# Linux/macOS:
rm -rf backend/cmd/frontend && cp -r frontend/dist backend/cmd/frontend
rm -rf backend/frontend && cp -r frontend/dist backend/frontend

# 3. Build backend
cd backend

# Windows amd64:
GOOS=windows GOARCH=amd64 go build -o ../../build/imageshare-windows-amd64.exe ./cmd/

# Linux amd64:
GOOS=linux GOARCH=amd64 go build -o ../../build/imageshare-linux-amd64 ./cmd/

# Linux arm64:
GOOS=linux GOARCH=arm64 go build -o ../../build/imageshare-linux-arm64 ./cmd/
```

## Tech Stack

**Backend:** Go, Gin, GORM, SQLite, JWT, bcrypt

**Frontend:** Vue 3, TypeScript, Vite, Element Plus, Pinia, Axios

## Project Structure

```
.
├── backend/
│   ├── cmd/
│   │   ├── main.go          # Entry point & router
│   │   └── frontend/        # Embedded frontend (go:embed)
│   ├── config/              # Configuration
│   └── internal/
│       ├── controller/      # HTTP handlers
│       ├── service/         # Business logic
│       ├── repository/      # Data access
│       ├── models/          # Data models
│       ├── middleware/       # JWT, rate limiting
│       └── logger/          # Log system
└── frontend/
    └── src/
        ├── views/           # Vue components
        ├── router/          # Vue Router
        ├── stores/          # Pinia stores
        └── utils/           # Axios instance
```

## License

This project is licensed under the **GNU Affero General Public License v3.0**.

- You are free to use, modify, and distribute this software
- Modified versions must be open-sourced under the same license
- You must preserve the original author's copyright notice
- Network use (SaaS) also requires source code disclosure (AGPL-specific clause)

See [LICENSE](LICENSE) for details.

---

Made By mcBill | [GitHub](https://github.com/mcbill1) | [Site](https://mcbill.top)
