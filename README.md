# BirthDayMemo

[中文](#中文文档) | [English](#english-doc)

---

# 中文文档

自托管生日提醒应用，支持日历视图、标签管理、PDF 导出与邮件提醒。单二进制部署，零外部依赖。

## 功能特性

- **单文件部署** - Go 编译的可执行文件内嵌前端，无需额外依赖
- **日历视图** - 月视图/年视图切换，直观展示生日分布
- **标签管理** - 自定义标签与颜色，按标签筛选生日
- **PDF 导出** - 横版 A4 日历，支持自定义背景、表格特效、5 种预设字体
- **邮件提醒** - SMTP 邮件发送，支持自定义模板与变量替换
- **多用户系统** - 管理员/普通用户角色分离，数据完全隔离
- **安全防护** - bcrypt 密码加密、图形验证码、IP 登录限流、操作审计日志
- **多语言** - 内置中英双语，支持管理员扩展语言包
- **主题切换** - 深色/浅色模式，10 种预设主题色
- **配置文件** - 支持 `//` 注释的 JSON 配置，首次运行交互式配置向导

## 快速开始

### Windows

1. 下载 `birthdaymemo-windows-amd64.exe`
2. 命令行运行：
   ```cmd
   birthdaymemo-windows-amd64.exe
   ```
3. 首次启动进入交互式配置向导（语言、监听地址、端口、管理员账号）
4. 配置完成后自动生成 `config.json` 和 `birthdaymemo.db`
5. 浏览器打开 `http://IP:<端口>`

### Linux

1. 下载对应架构的二进制文件：
   - x86_64 服务器：`birthdaymemo-linux-amd64`
   - ARM64 服务器：`birthdaymemo-linux-arm64`

2. 添加执行权限并运行：
   ```bash
   chmod +x birthdaymemo-linux-amd64
   ./birthdaymemo-linux-amd64
   ```

3. 首次启动进入交互式配置向导
4. 配置完成后自动生成 `config.json` 和 `birthdaymemo.db`
5. 浏览器打开 `http://IP:<端口>`

### 后台运行（Linux）

```bash
# 使用 nohup
nohup ./birthdaymemo-linux-amd64 > /dev/null 2>&1 &

# 或使用 systemd（推荐）
sudo tee /etc/systemd/system/birthdaymemo.service << 'EOF'
[Unit]
Description=BirthDayMemo
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/birthdaymemo
ExecStart=/opt/birthdaymemo/birthdaymemo-linux-amd64
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable birthdaymemo
sudo systemctl start birthdaymemo
```

### 命令行重置管理员密码

```bash
# Windows
birthdaymemo.exe -reset-admin-password
```
```bash
# Linux
./birthdaymemo-linux-amd64 -reset-admin-password
```

运行后进入交互式引导，按提示输入新密码即可。

## 配置说明

首次运行通过交互式向导生成 `config.json`（支持 `//` 注释）：

```json
{
    "server": {
        "host": "0.0.0.0",
        "port": 12345
    },
    "language": "zh",
    "log_retention_days": 14
}
```

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `server.host` | `0.0.0.0` | 监听地址。`0.0.0.0` 监听所有网卡，`127.0.0.1` 仅本机访问 |
| `server.port` | 随机 10000+ | 监听端口 |
| `language` | `zh` | 控制台语言（zh/en） |
| `log_retention_days` | `14` | 日志文件保留天数，0 为永久保留 |

Notice: 后期也可通过管理员后台管理修改

## 使用指南

### 管理员

登录后台可使用：

- **用户管理** - 创建/编辑/删除用户，重置密码
- **系统配置** - 监听地址/端口、日志保留天数
- **SMTP 设置** - 配置邮件服务器，支持连接测试
- **邮件模板** - 自定义提醒邮件内容，支持变量替换（`{user}`、`{name}`、`{age}` 等）
- **操作日志** - 查看审计日志，支持搜索与日志文件下载

### 普通用户

管理员创建账号后，用户可：

- **日历视图** - 月视图/年视图查看生日分布，按标签筛选
- **生日管理** - 添加/编辑/删除生日，设置姓名、性别、日期、标签
- **标签管理** - 创建自定义标签与颜色
- **PDF 导出** - 自定义背景、表格特效、字体，导出横版 A4 日历
- **个人设置** - 修改密码、语言、主题、提醒方式

### 提醒功能

用户可配置两种提醒方式：

- **到期提醒** - 生日前 N 天发送邮件（默认 3 天），支持设置提醒时间
- **定期汇总** - 每周/每月固定时间发送未来一段时间的生日汇总

## API 接口

### 认证

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| `POST` | `/api/auth/login` | 否 | 登录。Body: `{username, password, captcha_id?, captcha_answer?}` |
| `POST` | `/api/auth/logout` | 是 | 登出 |
| `GET` | `/api/auth/me` | 是 | 获取当前用户信息 |
| `GET` | `/api/captcha` | 否 | 获取验证码图片与 ID |
| `PUT` | `/api/auth/password` | 是 | 修改密码 |

### 生日管理

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| `GET` | `/api/birthdays` | 是 | 获取生日列表。Params: `tag_id` |
| `POST` | `/api/birthdays` | 是 | 添加生日。Body: `{name, gender, birth_date, tag_ids}` |
| `PUT` | `/api/birthdays/:id` | 是 | 更新生日 |
| `DELETE` | `/api/birthdays/:id` | 是 | 删除生日 |

### 标签管理

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| `GET` | `/api/tags` | 是 | 获取标签列表 |
| `POST` | `/api/tags` | 是 | 创建标签。Body: `{name, color}` |
| `PUT` | `/api/tags/:id` | 是 | 更新标签 |
| `DELETE` | `/api/tags/:id` | 是 | 删除标签 |

### PDF 导出

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| `POST` | `/api/pdf/preview` | 是 | 预览 PDF。Body: `{range, setting, resources}` |
| `POST` | `/api/pdf/export` | 是 | 导出 PDF |
| `GET` | `/api/pdf/settings` | 是 | 获取 PDF 设置 |
| `PUT` | `/api/pdf/settings` | 是 | 更新 PDF 设置 |

### 管理员接口

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| `GET` | `/api/admin/users` | 管理员 | 用户列表 |
| `POST` | `/api/admin/users` | 管理员 | 创建用户 |
| `PUT` | `/api/admin/users/:id` | 管理员 | 更新用户 |
| `DELETE` | `/api/admin/users/:id` | 管理员 | 删除用户 |
| `PUT` | `/api/admin/users/:id/password` | 管理员 | 重置密码 |
| `GET` | `/api/admin/settings` | 管理员 | 获取系统配置 |
| `PUT` | `/api/admin/settings` | 管理员 | 更新系统配置 |
| `GET` | `/api/admin/smtp` | 管理员 | 获取 SMTP 配置 |
| `PUT` | `/api/admin/smtp` | 管理员 | 更新 SMTP 配置 |
| `POST` | `/api/admin/smtp/test` | 管理员 | 测试 SMTP 连接 |
| `GET` | `/api/admin/email-template` | 管理员 | 获取邮件模板 |
| `PUT` | `/api/admin/email-template` | 管理员 | 更新邮件模板 |
| `GET` | `/api/admin/logs` | 管理员 | 操作日志列表 |
| `GET` | `/api/admin/logs/files` | 管理员 | 日志文件列表 |
| `GET` | `/api/admin/logs/files/:filename` | 管理员 | 下载日志文件 |

## 从源码构建

### 前置要求

- Go 1.24+
- Node.js 18+

### 字体文件说明

源代码默认使用以下 5 款字体：

| 字体名称 | 授权类型 | 下载链接 |
|----------|----------|----------|
| **思源黑体** (Noto Sans SC) | 免费可商用（开源字体） | 推荐从官方 GitHub 或 [Google Fonts](https://fonts.google.com/noto) 下载，确保获取最新版。 |
| **站酷小薇** (ZCOOL XiaoWei) | 免费可商用（站酷公益字体） | 站酷网官方专题页、[字加网](https://www.zijia.com.cn/) 等平台。 |
| **站酷快乐** (ZCOOL KuaiLe) | 免费可商用（站酷公益字体） | 站酷网官方专题页、[字加网](https://www.zijia.com.cn/) 等平台。 |
| **马善政** (Ma Shan Zheng) | 免费可商用（SIL OFL 开源协议） | Google Fonts、GitHub 等开源字体平台。部分下载站标注"商用须授权"，建议认准开源渠道。 |
| **龙藏** (Long Cang) | 免费可商用（寒蝉字库免费授权） | [字加网](https://www.zijia.com.cn/)、寒蝉字库官方渠道。部分下载站信息混乱，建议优先使用官方来源。 |

下载后将文件名改为以下对应名称，放入 `internal/pdfexport/fonts/` 目录即可：

| 预设名称 | 文件名 | 风格说明 |
|----------|--------|----------|
| 思源黑体 | `NotoSansSC.ttf` | 现代无衬线 |
| 站酷小薇 | `ZCOOLXiaoWei.ttf` | 细衬线 |
| 站酷快乐 | `ZCOOLKuaiLe.ttf` | 圆润活泼 |
| 马善政 | `MaShanZheng.ttf` | 毛笔行书 |
| 龙藏 | `LongCang.ttf` | 硬笔行楷 |

如果您不想使用上述默认字体，可自行挑选喜欢的 TTF 字体文件，放入 `internal/pdfexport/fonts/` 目录，并修改 `internal/pdfexport/fonts.go` 中的文件名映射和预设列表。

**提示：** 您可以将 `internal/pdfexport/fonts.go` 文件发给 AI，并说明：

> "我只有 'XX' 字体文件，文件名为 'XX.ttf'（可准备多个），请帮我删去原来的，并替换为这几个字体文件。"

AI 会自动帮您修改 `fonts.go` 中的文件名映射和预设列表。

### 使用构建工具（推荐）

项目自带 `build-tool.bat` 一键编译脚本（Windows），双击运行后选择目标平台即可自动完成前端构建和后端编译：

```
请选择编译目标:

  1 - Windows x64
  2 - Windows ARM64
  3 - Linux x64
  4 - Linux ARM64
  5 - macOS x64 (Intel)
  6 - macOS ARM64 (Apple Silicon)
  7 - 全部平台
```

编译产物输出到 `BDM\` 目录。

### 手动构建步骤

```bash
# 1. 安装前端依赖
cd frontend
npm install

# 2. 编译（以 Linux amd64 为例）
# 构建前端 + 编译后端（CGO_ENABLED=0 纯静态编译）
cd ..
cd frontend && npm run build && cd ..
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o BDM/birthdaymemo-linux-amd64 .
```

各平台编译命令：

```bash
# Windows x64
set CGO_ENABLED=0 && set GOOS=windows && set GOARCH=amd64 && go build -o BDM\birthdaymemo-windows-amd64.exe .

# Windows ARM64
set CGO_ENABLED=0 && set GOOS=windows && set GOARCH=arm64 && go build -o BDM\birthdaymemo-windows-arm64.exe .

# Linux x64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o BDM/birthdaymemo-linux-amd64 .

# Linux ARM64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o BDM/birthdaymemo-linux-arm64 .

# macOS x64 (Intel)
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o BDM/birthdaymemo-macos-amd64 .

# macOS ARM64 (Apple Silicon)
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o BDM/birthdaymemo-macos-arm64 .
```

## 技术栈

**后端：** Go、Chi、GORM、SQLite、bcrypt、fpdf

**前端：** Vue 3、TypeScript、Vite、Pinia、Vue Router

## 项目结构

```
.
├── main.go                  # 入口
├── frontend_dist/           # 内嵌前端 (go:embed)
├── internal/
│   ├── auth/                # 认证与会话管理
│   ├── config/              # 配置加载
│   ├── console/             # 交互式配置向导
│   ├── database/            # 数据库初始化
│   ├── email/               # SMTP 邮件发送
│   ├── i18n/                # 多语言支持
│   ├── logger/              # 日志系统
│   ├── models/              # 数据模型
│   ├── pdfexport/           # PDF 导出引擎
│   │   └── fonts/           # 字体文件（需自行准备）
│   ├── reminder/            # 提醒调度器
│   ├── security/            # 登录限流
│   └── web/                 # HTTP 处理器与中间件
└── frontend/
    └── src/
        ├── views/           # Vue 页面组件
        ├── components/      # 通用组件
        ├── router/          # Vue Router
        ├── stores/          # Pinia 状态管理
        ├── api/             # API 客户端
        ├── locales/         # 语言包
        └── styles/          # 样式文件
```

## 开源协议

本项目基于 **Apache License 2.0** 开源。

- 你可以自由使用、修改和分发本软件
- 允许商业使用，但**本项目明确禁止用于任何商业用途**
- 必须保留原作者版权声明和许可证文本
- 修改的文件需标注变更说明

详见 [LICENSE](LICENSE)。

---

# English Doc

A self-hosted birthday reminder application with calendar views, tag management, PDF export, and email notifications. Single binary deployment, zero external dependencies.

## Features

- **Single Binary Deployment** - Go compiled executable with embedded frontend, no extra dependencies needed
- **Calendar Views** - Month/Year view switching, intuitive birthday distribution display
- **Tag Management** - Custom tags with colors, filter birthdays by tag
- **PDF Export** - Landscape A4 calendar with custom backgrounds, table effects, 5 preset fonts
- **Email Reminders** - SMTP email sending with customizable templates and variable substitution
- **Multi-User System** - Admin/User role separation with complete data isolation
- **Security** - bcrypt password hashing, captcha, IP-based login rate limiting, audit logs
- **Multi-Language** - Built-in English/Chinese, extensible language packs
- **Theme Switching** - Dark/Light mode with 10 preset theme colors
- **Config File** - JSON config with `//` comment support, interactive setup wizard on first run

## Quick Start

### Windows

1. Download `birthdaymemo-windows-amd64.exe`
2. Run from command line:
   ```cmd
   birthdaymemo-windows-amd64.exe
   ```
3. First launch enters interactive setup wizard (language, listen address, port, admin account)
4. After setup, auto-generates `config.json` and `birthdaymemo.db`
5. Open `http://IP:<port>` in browser

### Linux

1. Download the binary for your architecture:
   - x86_64 server: `birthdaymemo-linux-amd64`
   - ARM64 server: `birthdaymemo-linux-arm64`

2. Add execute permission and run:
   ```bash
   chmod +x birthdaymemo-linux-amd64
   ./birthdaymemo-linux-amd64
   ```

3. First launch enters interactive setup wizard
4. After setup, auto-generates `config.json` and `birthdaymemo.db`
5. Open `http://IP:<port>` in browser

### Run in Background (Linux)

```bash
# Using nohup
nohup ./birthdaymemo-linux-amd64 > /dev/null 2>&1 &

# Or using systemd (recommended)
sudo tee /etc/systemd/system/birthdaymemo.service << 'EOF'
[Unit]
Description=BirthDayMemo
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/birthdaymemo
ExecStart=/opt/birthdaymemo/birthdaymemo-linux-amd64
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable birthdaymemo
sudo systemctl start birthdaymemo
```

### Reset Admin Password via CLI

```bash
# Windows
birthdaymemo.exe -reset-admin-password
```
```bash
# Linux
./birthdaymemo-linux-amd64 -reset-admin-password
```

This will start an interactive prompt to enter the new password.

## Configuration

First run generates `config.json` via interactive wizard (supports `//` comments):

```json
{
    "server": {
        "host": "0.0.0.0",
        "port": 12345
    },
    "language": "en",
    "log_retention_days": 14
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `server.host` | `0.0.0.0` | Listen address. `0.0.0.0` for all interfaces, `127.0.0.1` for local only |
| `server.port` | Random 10000+ | Listen port |
| `language` | `zh` | Console language (zh/en) |
| `log_retention_days` | `14` | Log file retention days, 0 for permanent |

Notice: You can also manage and modify this later through the admin backend.

## Usage Guide

### Admin

After login, admin can:

- **User Management** - Create/edit/delete users, reset passwords
- **System Config** - Listen address/port, log retention
- **SMTP Settings** - Configure mail server with connection test
- **Email Template** - Customize reminder email content with variable substitution (`{user}`, `{name}`, `{age}`, etc.)
- **Audit Logs** - View operation logs with search and log file download

### User

After admin creates an account, user can:

- **Calendar View** - Month/Year view for birthday distribution, filter by tags
- **Birthday Management** - Add/edit/delete birthdays with name, gender, date, tags
- **Tag Management** - Create custom tags with colors
- **PDF Export** - Custom backgrounds, table effects, fonts, export landscape A4 calendar
- **Personal Settings** - Change password, language, theme, reminder preferences

### Reminder Options

Users can configure two reminder types:

- **Due Date Reminder** - Send email N days before birthday (default 3), with configurable reminder time
- **Periodic Summary** - Weekly/monthly fixed time summary of upcoming birthdays

## Build from Source

### Prerequisites

- Go 1.24+
- Node.js 18+

### Font Files

The source code defaults to the following 5 fonts:

| Font Name | License | Download |
|-----------|---------|----------|
| **Noto Sans SC** | Free for commercial use (Open Source) | Official GitHub or [Google Fonts](https://fonts.google.com/noto). Get the latest version. |
| **ZCOOL XiaoWei** | Free for commercial use (ZCOOL Public Font) | ZCOOL official site, [Zijia](https://www.zijia.com.cn/), etc. |
| **ZCOOL KuaiLe** | Free for commercial use (ZCOOL Public Font) | ZCOOL official site, [Zijia](https://www.zijia.com.cn/), etc. |
| **Ma Shan Zheng** | Free for commercial use (SIL OFL) | Google Fonts, GitHub, and other open-source font platforms. |
| **Long Cang** | Free for commercial use (HanChan Free License) | [Zijia](https://www.zijia.com.cn/), HanChan official channels. |

After downloading, rename the files to match the following and place them in the `internal/pdfexport/fonts/` directory:

| Preset Name | Filename | Style |
|-------------|----------|-------|
| Noto Sans SC | `NotoSansSC.ttf` | Modern sans-serif |
| ZCOOL XiaoWei | `ZCOOLXiaoWei.ttf` | Thin serif |
| ZCOOL KuaiLe | `ZCOOLKuaiLe.ttf` | Rounded & playful |
| Ma Shan Zheng | `MaShanZheng.ttf` | Brush cursive |
| Long Cang | `LongCang.ttf` | Pen cursive |

If you prefer different fonts, place your own TTF font files in the `internal/pdfexport/fonts/` directory and update the filename mappings and preset list in `internal/pdfexport/fonts.go`.

**Tip:** You can send the `internal/pdfexport/fonts.go` file to an AI assistant and say:

> "I only have 'XX' font file, named 'XX.ttf' (can be multiple). Please help me remove the original ones and replace them with these font files."

The AI will automatically update the filename mappings and preset list in `fonts.go`.

### Using Build Tool (Recommended)

The project includes `build-tool.bat`, a one-click build script (Windows). Double-click to run and select the target platform — it automatically builds the frontend and compiles the backend:

```
Select build target:

  1 - Windows x64
  2 - Windows ARM64
  3 - Linux x64
  4 - Linux ARM64
  5 - macOS x64 (Intel)
  6 - macOS ARM64 (Apple Silicon)
  7 - All platforms
```

Build output is placed in the `BDM\` directory.

### Manual Build Steps

```bash
# 1. Install frontend dependencies
cd frontend
npm install

# 2. Build (Linux amd64 example)
# Build frontend + compile backend (CGO_ENABLED=0 for static binary)
cd ..
cd frontend && npm run build && cd ..
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o BDM/birthdaymemo-linux-amd64 .
```

Platform-specific build commands:

```bash
# Windows x64
set CGO_ENABLED=0 && set GOOS=windows && set GOARCH=amd64 && go build -o BDM\birthdaymemo-windows-amd64.exe .

# Windows ARM64
set CGO_ENABLED=0 && set GOOS=windows && set GOARCH=arm64 && go build -o BDM\birthdaymemo-windows-arm64.exe .

# Linux x64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o BDM/birthdaymemo-linux-amd64 .

# Linux ARM64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o BDM/birthdaymemo-linux-arm64 .

# macOS x64 (Intel)
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o BDM/birthdaymemo-macos-amd64 .

# macOS ARM64 (Apple Silicon)
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o BDM/birthdaymemo-macos-arm64 .
```

## Tech Stack

**Backend:** Go, Chi, GORM, SQLite, bcrypt, fpdf

**Frontend:** Vue 3, TypeScript, Vite, Pinia, Vue Router

## Project Structure

```
.
├── main.go                  # Entry point
├── frontend_dist/           # Embedded frontend (go:embed)
├── internal/
│   ├── auth/                # Authentication & session management
│   ├── config/              # Configuration loading
│   ├── console/             # Interactive setup wizard
│   ├── database/            # Database initialization
│   ├── email/               # SMTP email sending
│   ├── i18n/                # Multi-language support
│   ├── logger/              # Log system
│   ├── models/              # Data models
│   ├── pdfexport/           # PDF export engine
│   │   └── fonts/           # Font files (must be provided by user)
│   ├── reminder/            # Reminder scheduler
│   ├── security/            # Login rate limiting
│   └── web/                 # HTTP handlers & middleware
└── frontend/
    └── src/
        ├── views/           # Vue page components
        ├── components/      # Shared components
        ├── router/          # Vue Router
        ├── stores/          # Pinia state management
        ├── api/             # API client
        ├── locales/         # Language packs
        └── styles/          # Style files
```

## License

This project is licensed under the **Apache License 2.0**.

- You are free to use, modify, and distribute this software
- Commercial use is allowed in general, but **this project explicitly prohibits any commercial use**
- You must preserve the original author's copyright notice and license text
- Modified files must include a notice of changes

See [LICENSE](LICENSE) for details.

---

Made By mcBill | [GitHub](https://github.com/mcbill1) | [Site](https://mcbill.top)
