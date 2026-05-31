# sbc — sing-box commander

`sbc` 是 sing-box 的 Go 原生命令行控制器，替代旧的 shell 脚本 `sbc-lib.sh`。
提供服务管理、配置渲染、代理切换、面板更新、一键部署等全链路运维能力。

## 安装

### Homebrew（推荐）

```bash
brew install cagedbird043/tap/sbc
```

### Go

```bash
go install github.com/cagedbird043/sbc@latest
```

### 源码编译

```bash
git clone https://github.com/cagedbird043/sbc.git
cd sbc
make install
```

## 快速开始

```bash
# 查看状态概览（不指定任何命令时的默认行为）
sbc

# 查看配置变体与模板路径
sbc config status

# 检查渲染语法（不部署）
sbc validate
```

## 命令参考

### service — 服务管理

| 命令 | 说明 |
|------|------|
| `sbc service start` | 启动 sing-box 服务 |
| `sbc service stop` | 停止 sing-box 服务 |
| `sbc service restart` | 重启 sing-box 服务 |
| `sbc service status` | 查看服务运行状态 |
| `sbc service log` | 查看服务日志 |

Linux 使用 systemctl，macOS 使用 launchctl。

### config — 配置管理

| 命令 | 说明 |
|------|------|
| `sbc config status` | 查看当前配置状态（变体、模板路径、目标文件） |
| `sbc config show` | 渲染并显示最终配置内容 |
| `sbc config edit` | 用编辑器打开模板文件 |
| `sbc config diff` | 比较模板渲染结果 vs 已部署配置 |
| `sbc config variant` | 查看当前配置变体 |
| `sbc config variant set <name>` | 切换变体（default / realip-v4-only） |
| `sbc config variant list` | 列出可用变体 |
| `sbc config template` | 查看模板路径信息 |
| `sbc config env` | 查看 .env 变量状态 |

### proxy — 代理控制

通过 sing-box Clash API (localhost:9090) 动态管理代理节点。

| 命令 | 说明 |
|------|------|
| `sbc proxy list` | 列出所有 Selector 及当前节点 |
| `sbc proxy groups` | 列出组代号 |
| `sbc proxy nodes [group]` | 列出节点（可指定组过滤） |
| `sbc proxy use <group> <node>` | 切换节点（支持子串匹配） |

### ui — 面板管理

| 命令 | 说明 |
|------|------|
| `sbc ui status` | 面板状态与已安装的资源文件 |
| `sbc ui update` | 下载并更新面板（zashboard） |

### 部署与验证

| 命令 | 说明 |
|------|------|
| `sbc update` | 完整部署流程：git pull → 渲染 → sing-box check → 部署 → 重启 |
| `sbc validate` | 渲染配置模板并做语法检查（不部署） |
| `sbc check` | 检查已部署的 config.json 语法 |

### completion — Shell 补全

| 命令 | 说明 |
|------|------|
| `sbc completion zsh` | 生成 Zsh 补全脚本 |
| `sbc completion bash` | 生成 Bash 补全脚本 |
| `sbc completion fish` | 生成 Fish 补全脚本 |
| `sbc completion powershell` | 生成 PowerShell 补全脚本 |

Zsh 用户可将输出重定向到 `~/.local/share/zsh/site-functions/` 或通过 `source <(sbc completion zsh)` 动态加载。

## 环境变量

`sbc` 从 `~/.config/sing-box/.env` 读取配置。必需的环境变量：

| 变量 | 说明 |
|------|------|
| `CLASH_API_SECRET` | Clash API 认证密钥（用于 proxy 命令和管理面板） |
| `MIXED_PROXY_USERNAME` | SOCKS5 / HTTP 代理用户名 |
| `MIXED_PROXY_PASSWORD` | SOCKS5 / HTTP 代理密码 |
| `PROVIDER_NAME_1` | 订阅 provider 名称 |
| `SUB_URL_1` | 订阅 URL |

运行时重载变量（非必需）：

| 变量 | 说明 |
|------|------|
| `SBC_PROFILE` | 强制指定配置轨道（linux / macos），默认自动检测 |
| `SBC_TEMPLATE_ROOT` | 覆盖模板仓库路径，默认从可执行文件位置推导 |
| `SBC_CONFIG_VARIANT` | 覆盖配置变体（default / realip-v4-only） |

## 配置变体

`sbc` 支持两个配置变体：

| 变体 | 说明 |
|------|------|
| `default` | FakeIP + prefer_ipv4（主流方向） |
| `realip-v4-only` | Real IP + IPv4-only（保守备用） |

切换变体需先执行 `sbc update` 才会生效。变体状态保存在 `~/.config/sing-box/config-variant`。

## 从旧版 Shell sbc 迁移

`sbc`（Go）是旧版 shell 实现 `bin/sbc`、`scripts/sbc-lib.sh` 的直接替代品：

| 旧命令 | 新命令 | 说明 |
|--------|--------|------|
| `sbc`（无参数） | `sbc` | 状态概览，行为一致 |
| `sbc start` | `sbc service start` | 归类到 service 子命令 |
| `sbc stop` | `sbc service stop` | 同上 |
| `sbc restart` | `sbc service restart` | 同上 |
| `sbc status` | `sbc service status` | 同上 |
| `sbc log` | `sbc service log` | 同上 |
| `sbc config show` | `sbc config show` | 一致 |
| `sbc config edit` | `sbc config edit` | 一致 |
| `sbc config diff` | `sbc config diff` | 一致 |
| `sbc variant` | `sbc config variant` | 归类到 config 子命令 |
| `sbc variant set` | `sbc config variant set` | 同上 |
| `sbc proxy ...` | `sbc proxy ...` | 一致，纯 Go 实现 |
| `sbc ui ...` | `sbc ui ...` | 一致，纯 Go 实现 |
| `sbc update` | `sbc update` | 一致 |
| `sbc check` | `sbc check` | 一致 |
| — | `sbc validate` | 新增：仅渲染+检查，不部署不重启 |

### 主要改进

- **零外部依赖**：不再需要 `envsubst`、`python3`、`diff` 等系统工具。模板渲染、zip 解压、差异比较全部由 Go 原生实现。
- **统一命令结构**：两级子命令体系，`service` / `config` / `proxy` / `ui` 清晰分组。
- **自动补全**：原生支持 zsh / bash / fish / powershell 补全，无需手动维护补全脚本。
- **类型安全**：编译期错误检查，避免 shell 脚本的运行时拼写错误。
- **跨平台**：Linux 和 macOS 共用同一代码库，通过编译目标区分。
