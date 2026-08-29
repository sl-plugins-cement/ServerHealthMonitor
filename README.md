# Server Health Monitor

面向生产环境的通用服务器健康监控与自动修复平台。项目包含 Go 监控 Agent、Web 管理控制台、兼容版 Bash monitor，以及 Prometheus/Grafana 集成。SCP:SL 仅作为一个可选应用适配器保留。

## ⚠️ 首次初始化（必读）

**管理控制台首次启动时必须创建管理员账户。**

### 初始管理员设置

#### 命令行初始化（推荐）

```bash
# 服务器端
server-health-monitor-console --admin-user <username> --admin-pass '<password>'
```

密码必须满足以下要求：
- **最少 12 个字符**
- 包含**大写字母** (A-Z)
- 包含**小写字母** (a-z)
- 包含**数字** (0-9)
- 包含**特殊字符** (!@#$%^&*._-~)

#### 示例（生成强随机密码）

```bash
# Linux / macOS
python3 -c "import secrets; print('P@ssw0rd' + secrets.token_hex(8))"

# PowerShell
$p = -join ((65..90) + (97..122) + (48..57) + (33,64,35,36,37,94,38,42,46,95,45,126) | Get-Random -Count 12 | % {[char]$_}); Write-Host $p
```

### 首次登录提示

⚠️ **重要**：首次使用初始管理员账户登录后，**必须立即修改密码**。系统会强制拦截：修改密码前无法访问控制台其他页面（仅可退出登录或完成改密）。API 密钥只显示一次，请妥善保存。

### CLI 兜底（锁定 / 遗失密钥）

当唯一管理员被锁或遗失 API 密钥、无法通过 Web 登录时，可在服务器上用命令行兜底：

```bash
# 解锁被永久锁定的账户
server-health-monitor-console --unlock-user <用户名>

# 重置密码（下次登录强制修改密码），需配合 --new-pass
server-health-monitor-console --reset-user-password <用户名> --new-pass '<新密码>'

# 轮换 API 密钥（只打印一次，请妥善保存）
server-health-monitor-console --rotate-user-key <用户名>
```

### 登录方式

Web 控制台使用**双重验证**：
- **第一因素**：用户名 + 密码
- **第二因素**：API 密钥（初始化时生成，仅显示一次）

如果遗失 API 密钥，管理员可通过 Web 界面重新生成。

## 功能概览

- **智能环境适配**：自动识别 Linux 环境、systemd、容器和可用诊断工具
- **多实例支持**：可同时监控多个服务、容器或应用实例
- **多维度健康检测**：
  - systemd 服务状态（active / inactive / failed）
  - UDP 游戏端口监听
  - 进程数量监控
  - 磁盘使用率
  - 内存使用率
  - CPU 使用率（基于 /proc/stat）
  - 系统负载（1 分钟，自动按 CPU 核心数归一化）
  - 最近 5 分钟服务错误日志扫描
  - 网络连通性检测（可选）
- **受控自动恢复**：
  - 连续 N 次确认异常后才执行 `systemctl restart`
  - 重启冷却时间防止重启循环
  - 重启后验证 UDP 端口恢复
- **Webhook 通知**：支持 Discord / Slack 兼容的 JSON Webhook
- **广泛兼容性**：自动检测 Linux 发行版和包管理器，兼容常见 systemd Linux 发行版
- **配置文件**：通过 `/etc/server-health-monitor.conf` 灵活配置所有参数
- **安全加固**：systemd unit 含 NoNewPrivileges、ProtectSystem、ProtectKernelTunables 等

## 文件结构

```
.
├── cmd/agent/                       # Go Agent 入口
├── cmd/console/                     # Web 管理控制台入口
├── internal/                        # 配置、采集、监控、通知和 Web 实现
├── deploy/                          # 部署脚本、systemd、Prometheus、Grafana
├── server-health-monitor.sh         # 通用 Bash 监控
├── server-health-monitor.service    # Bash monitor systemd service
├── server-health-monitor.timer      # Bash monitor 15 秒 timer
├── server-health-monitor-agent.conf.example # Go Agent 配置示例
├── dashboard-client.py              # 本地客户端/SSH 隧道配套客户端
├── install.sh                       # Linux 服务器安装脚本
├── start-*.bat                      # Windows 启动脚本
├── go.mod / go.sum                  # Go 模块依赖
├── LICENSE
└── README.md
```

## 快速安装

```bash
git clone https://github.com/Oxen112774/ServerHealthMonitor.git
cd ServerHealthMonitor
sudo bash install.sh
```

安装脚本会自动：
- 检测 Linux 发行版和包管理器
- 检查依赖命令（iproute2、procps、util-linux 等）
- 安装 Bash monitor、systemd timer、Go Agent 和配置文件
- 启用并启动定时器

安装脚本需要 root 权限和 systemd。没有检测到游戏服务适配器时仍可安装通用 Agent；它不会安装任何业务应用本身。

Go Agent 默认监听 `0.0.0.0:8080`。公网环境请先配置认证、防火墙和 HTTPS，不要直接暴露未认证的 HTTP 面板。

## 手动安装

```bash
sudo install -o root -g root -m 0750 server-health-monitor.sh /usr/local/sbin/server-health-monitor
sudo install -o root -g root -m 0644 server-health-monitor.service /etc/systemd/system/
sudo install -o root -g root -m 0644 server-health-monitor.timer /etc/systemd/system/
sudo install -o root -g root -m 0640 server-health-monitor.conf.example /etc/server-health-monitor.conf
sudo mkdir -p /var/lib/server-health-monitor && sudo chmod 0750 /var/lib/server-health-monitor
sudo systemctl daemon-reload
sudo systemctl enable --now server-health-monitor.timer
```

## 配置

编辑配置文件：

```bash
sudo vi /etc/server-health-monitor.conf
```

### 关键参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `SERVICE` | 空（自动检测） | 空值时自动发现所有 `scpsl-*.service`；指定则单实例模式 |
| `PORT` | 自动从服务名提取 | 显式指定 UDP 端口 |
| `FAILURE_THRESHOLD` | 3 | 连续异常次数阈值，达到后触发重启 |
| `RESTART_COOLDOWN` | 300 秒 | 两次重启间最小间隔，防止循环 |
| `SOCKET_WAIT_TIMEOUT` | 30 秒 | 重启后等待 UDP 端口恢复的超时 |
| `WARN_DISK_PERCENT` | 85 | 磁盘使用率告警阈值 |
| `WARN_MEMORY_PERCENT` | 90 | 内存使用率告警阈值 |
| `WARN_CPU_PERCENT` | 90 | CPU 使用率告警阈值 |
| `WARN_LOAD_MULTIPLE` | 4 | 负载 / CPU 核心数比值阈值 |
| `WARN_PROCESS_COUNT` | 500 | 进程数告警阈值 |
| `CONNECTIVITY_CHECK_HOST` | 空（禁用） | 设置后检测网络连通性 |
| `WEBHOOK_URL` | 空（禁用） | Discord / Slack Webhook 地址 |
| `NOTIFY_COOLDOWN` | 600 秒 | 每实例最小通知间隔 |

### 单实例模式

如果你只有一个 SCP:SL 服务器实例，可以显式指定：

```bash
SERVICE="scpsl-7777.service"
PORT="7777"
```

### 多实例自动模式（默认）

保持 `SERVICE` 为空，脚本会自动发现并逐个监控所有 `scpsl-*.service` 实例。每个实例独立维护状态文件，互不干扰。

### Webhook 通知

```bash
# Discord
WEBHOOK_URL="https://discord.com/api/webhooks/xxxx/yyyy"

# Slack (compatible endpoint)
WEBHOOK_URL="https://hooks.slack.com/services/xxx/yyy/zzz"
```

通知会在以下场景触发（受 `NOTIFY_COOLDOWN` 冷却限制）：
- 服务异常确认并执行重启（critical 红色）
- 重启后 UDP 端口未恢复（critical 红色）
- 磁盘使用率超阈值（warning 黄色）
- 网络连通性异常（warning 黄色）
- 服务恢复正常（info 绿色）

## 查看状态

```bash
# 定时器状态
systemctl status server-health-monitor.timer

# 最近 100 条日志
journalctl -t scpsl-health-monitor -n 100 --no-pager

# 实时跟踪日志
journalctl -t scpsl-health-monitor -f

# 手动执行一次
sudo /usr/local/sbin/scpsl-health-monitor

# 查看状态文件
ls -la /var/lib/scpsl-health-monitor/
```

## 运维操作

### Go Agent

```bash
sudo systemctl status server-health-monitor-agent.service
sudo journalctl -u server-health-monitor-agent.service -n 100 --no-pager
curl -fsS http://127.0.0.1:8080/api/health
curl -fsS http://127.0.0.1:8080/metrics
```

### 升级与卸载

升级前备份 `/etc/server-health-monitor-agent.conf`、`/etc/server-health-monitor.conf` 和控制台数据目录。替换二进制或重新运行安装脚本后执行：

```bash
sudo systemctl daemon-reload
sudo systemctl restart server-health-monitor-agent.service
sudo systemctl restart server-health-monitor.timer
```

项目目前没有自动卸载脚本。确认不再需要后，先停止服务，再手动删除对应 unit、二进制、配置和状态目录；删除前请备份配置与审计数据。

## Prometheus + Grafana

```bash
cd deploy
set -a; . ./.env; set +a
docker compose up -d
```

创建 `deploy/.env`（不要提交）并设置强随机密码：

```dotenv
GRAFANA_ADMIN_PASSWORD=replace-with-a-long-random-password
```

- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000

启动前请编辑 [prometheus.yml](deploy/prometheus/prometheus.yml) 中的 Agent 地址。模板中的 target 只是本机示例，不适用于所有环境；如果 Agent 开启认证，还需要配置 Prometheus 的 `basic_auth`。Grafana 初始账号由 [docker-compose.yml](deploy/docker-compose.yml) 设置，首次登录后必须立即修改密码。

Go Agent 默认监听 `0.0.0.0:8080`。公网环境请先配置认证、防火墙和 HTTPS，不要直接暴露未认证的 HTTP 面板。

### Go Agent 配置

配置模板见 [monitor-agent.conf.example](monitor-agent.conf.example)。常用配置如下：

```ini
# 仅本机访问，推荐配合 SSH 隧道
host = "127.0.0.1"
port = 8080

# 远程访问时至少启用认证，并通过 HTTPS 反向代理保护传输
auth_user = "admin"
auth_pass = "change-this-password"

collect_interval = 3
check_interval = 5
failure_threshold = 3
restart_cooldown = 300
socket_wait_timeout = 30
service = ""
service_port = ""
```

修改后执行：

```bash
sudo systemctl restart server-health-monitor-agent.service
sudo journalctl -u server-health-monitor-agent.service -n 100 --no-pager
```

### 端口与访问方式

| 组件 | 默认地址 | 用途 |
|------|----------|------|
| Go Agent | `0.0.0.0:8080` | 面板、`/api/status`、`/metrics` |
| 管理控制台 | `0.0.0.0:8081` | 用户、服务器和部署管理 |
| Prometheus | `0.0.0.0:9090` | 指标查询 |
| Grafana | `0.0.0.0:3000` | 可视化面板 |

推荐让 Agent 监听 `127.0.0.1`，再使用 SSH 隧道或 HTTPS 反向代理。若必须监听公网地址，请配置认证、防火墙白名单和 TLS；不要把管理控制台、Prometheus 或 Grafana 直接暴露给互联网。

### 客户端连接路径

本地运行时有三种路径：

1. **SSH 隧道**：双击 `启动服务器监控.cmd`，输入服务器地址；这是公网服务器的默认推荐方式。
2. **HTTPS 直连**：通过 Nginx/Caddy 提供 TLS 和认证，使用 `python dashboard-client.py --mode direct --server https://monitor.example.com --port 443`。
3. **内网反向代理**：通过企业 VPN 或统一网关访问，使用 `--mode proxy --server https://gateway.example.com --port 443`。

三种方式最终都只把 Agent API 代理到本机 `127.0.0.1:8090`。客户端不会自动保存密码；SSH 私钥、网关凭据和生产地址应放在本机私有配置或密钥管理系统中。

### 通用运维诊断（实验性）

管理控制台提供受保护的 `GET /console/api/diagnostics` 接口，用于盘点控制台所在主机的操作系统、架构、systemd、容器工具、防火墙工具和基础诊断能力，并返回 0-100 风险评分。

接口同时返回修复计划预览。所有高风险计划都必须明确确认，并且在对应适配器提供备份、执行和回滚实现前保持不可执行；平台不会把用户输入直接拼接成远程 shell 命令。后续适配器应遵循：盘点 → 预览 → 确认 → 最小权限执行 → 验证 → 审计 → 回滚。

### 控制台权限与工单

控制台内置权限组：服主（全部权限）、技术总监（运维与权限组管理）、管理员（兼容旧部署）、运维人员（服务器管理和部署）、查看者（查看监控并提交工单）、报告提交者（查看仪表盘并提交工单），以及自定义权限组。自定义组只能由服主或技术总监创建和修改，并且不能授予用户管理、敏感数据查看、修复执行等核心权限。

服务器列表默认过滤 SSH 密钥路径等敏感字段；工单提交者只能看到自己的工单，运维人员、技术总监和服主可以处理全部工单。角色分配始终在后端校验，网页隐藏选项不是安全边界。

首次部署控制台时必须通过命令行安全初始化管理员：

```bash
server-health-monitor-console --admin-user <username> --admin-pass '<strong-password>'
```

登录需要密码和一次性 API 密钥。连续三次失败会永久锁定账户，只能由管理员（或在服务器上通过 `--unlock-user` CLI）解锁。生产环境请设置 `CONSOLE_TOKEN_SECRET`，并通过 HTTPS、VPN 或 SSH 隧道访问 8081。

## 工作原理

```
每 15 秒触发一次（开机约 90 秒后开始）
     │
     ▼
┌─────────────────────┐
│  自动发现 SCP:SL     │  ← systemctl list-units + 磁盘扫描
│  systemd 服务实例     │
└─────────┬───────────┘
          │
          ▼ (对每个实例)
┌─────────────────────┐
│  收集指标            │  服务状态 / UDP 端口 / 进程数
│                     │  磁盘 / 内存 / CPU / 负载 / 错误日志
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│  健康评估            │  healthy / socket-missing / service-inactive
└─────────┬───────────┘
          │
     ┌────┴────┐
     │         │
  healthy    unhealthy
     │         │
     │         ▼
     │    ┌──────────────────┐
     │    │ 连续 N 次异常?   │── No → 记录状态，等待下次检查
     │    └────────┬─────────┘
     │             │ Yes
     │             ▼
     │    ┌──────────────────┐
     │    │ 冷却时间已过?     │── No → 记录 suppressed，等待
     │    └────────┬─────────┘
     │             │ Yes
     │             ▼
     │    ┌──────────────────┐
     │    │ systemctl restart │── webhook 通知
     │    └────────┬─────────┘
     │             │
     │             ▼
     │    ┌──────────────────┐
     │    │ 等待 UDP 恢复     │  最长 SOCKET_WAIT_TIMEOUT 秒
     │    └────────┬─────────┘
     │             │
     ▼             ▼
┌──────────────────────────┐
│  持久化状态文件           │  /var/lib/scpsl-health-monitor/<port>.env
│  webhook 恢复通知         │
└──────────────────────────┘
```

## 安全说明

- 脚本需要 root 权限运行（读取 systemd/journal 状态，可能执行 `systemctl restart`）
- systemd unit 已做安全加固：NoNewPrivileges、ProtectSystem=full、ProtectHome、ProtectKernelTunables/Modules/Logs 等
- 状态文件权限 0640，目录 0750
- Webhook 通知使用 curl 发送，超时 10 秒，失败不影响主流程
- Agent 和控制台当前使用 HTTP，不提供内置 TLS；生产环境应通过 SSH 隧道或 HTTPS 反向代理访问
- 双凭据登录是密码加 API key，不等同于 TOTP 或硬件密钥；API key 只在创建或轮换时显示一次
- 控制台默认不信任 `X-Forwarded-For`（防伪造 IP 绕过限流）；仅在反向代理之后设置 `CONSOLE_TRUST_PROXY=1`
- HTTPS 之后请设置 `CONSOLE_COOKIE_SECURE=1`，让登录 Cookie 携带 Secure 标志

## 兼容性

| 发行版 | 支持状态 | 备注 |
|--------|---------|------|
| Debian / Ubuntu | ✅ 完全支持 | 需要 iproute2、procps、util-linux |
| RHEL / CentOS / Rocky / Alma | ✅ 完全支持 | 需要 iproute、procps-ng |
| openSUSE | ✅ 完全支持 | 需要 iproute2 |
| Arch Linux | ✅ 完全支持 | 需要 iproute2、procps-ng |
| Alpine Linux | ⚠️ 需额外配置 | 默认不是 systemd，需先提供 systemd 环境 |
| 其他 systemd 发行版 | ⚠️ 通常可运行 | 需要 bash 4+、systemd 和系统工具 |

脚本会自动检测发行版和包管理器，在缺少依赖时给出安装提示。Bash monitor 和 Go Agent 是两套独立实现，同时启用时可能对同一实例重复检查或重启，生产环境应选择一套作为自动修复主入口。

## 常见问题

- **没有发现实例**：确认服务名称符合 `scpsl-<数字>.service`，并运行 `systemctl list-units --type=service --all` 检查。
- **面板打不开**：检查 `systemctl status server-health-monitor-agent.service`、监听地址、防火墙和 `ss -lntp | grep 8080`。
- **自动重启未触发**：确认服务状态或 UDP 端口确实异常，并检查 `failure_threshold` 与 `restart_cooldown`。
- **Prometheus 无数据**：检查 target 地址、容器到 Agent 的网络连通性、防火墙以及 `/metrics` 是否可访问。
- **控制台无法登录**：首次运行必须用 `--admin-user` 和 `--admin-pass` 创建管理员；API key 使用创建时输出的值。账户被锁或遗失密钥时，在服务器上用 `--unlock-user`、`--reset-user-password`、`--rotate-user-key` 兜底（见上文）。

## CI/CD

配置 GitHub `production` Environment 以及 `DEPLOY_HOST`、`DEPLOY_PORT`、`DEPLOY_USER`、`DEPLOY_KEY` Secrets 后，推送到 `main` 分支会编译 Linux amd64 Agent 并尝试通过 SSH 部署。该流程依赖目标服务器权限和 systemd，不代表 clone 后即可自动部署。

项目要求的 Go 版本应与 [go.mod](go.mod) 保持一致。当前仓库的 `go.mod` 要求 Go 1.25，而 CI 工作流仍配置为 Go 1.21；启用自动构建前必须统一这两个版本。

## 许可证

[MIT](LICENSE)
