# sbc — sing-box commander 维护指南

## 架构

```
.env (SBC_CONFIG_URLS)
  → ReadEnvURLs()
    → DownloadConfigs() → temp dir → SHA256 proof (meta.json)
      → VerifyDownloads() → active variant check
        → copy to ~/.config/sing-box/
          → ActiveVariantTemplatePath() → config-{variant}.json
            → RenderProfile(templatePath, output, vars) [本地 envsubst]
              → sing-box check → InstallConfig → restart
```

## 配置分发

模板托管在 hk-edge，由 `sing-box-private-prod` 的 CI 自动发布：
```
https://hk-edge.cagedbird.cn/sbc-config/<secret>/{linux,macos,android}/config-fakeip-prefer-ipv4.json
https://hk-edge.cagedbird.cn/sbc-config/<secret>/{linux,macos,android}/config-realip-v4-only.json
```

每个配置附带 `.meta.json`（sha256 + bytes + updated_at）。sbc 下载后校验 sha256。

**CI 不做 envsubst**（linux/macos 保留 `$VAR` 占位符，各机器本地渲染）。
安卓例外：CI 用 GitHub Secrets 做 envsubst 全渲染后发布成品。

## 变体

- `fakeip-prefer-ipv4` — FakeIP + prefer_ipv4（主流）
- `realip-v4-only` — Real IP + IPv4-only fallback

无硬编码默认值。变体必须通过 `sbc config variant set <name>` 显式设置。
变体列表由文件系统扫描 `~/.config/sing-box/config-*.json` 动态发现。

## 数据流

1. `readEnvURLs()` — 从 `.env` 读 `SBC_CONFIG_URLS`（逗号+引号分隔）
2. `DownloadConfigs()` — 下载所有 URL，校验 meta.json sha256
3. `VerifyDownloads()` — 比对声明/成功数量，激活变体必须存在
4. 复制到 `~/.config/sing-box/`
5. `RenderProfile()` — 本地 envsubst（不依赖外部 `envsubst` 命令）
6. `sing-box check` 校验语法
7. `InstallConfig()` → `serviceRestart()`

## 硬规则

- **禁止远程 `sed`**。系统文件本地改 + `scp`。
- **不要硬编码变体名、模板路径、平台名**。变体由文件系统发现，平台由 `runtime.GOOS` 决定。
- **`NormalizeConfigVariant` 只做 lowercase + trim**，不做别名映射。别名映射由文件系统决定。
- **CI 的 `strip-jsonc.py` 必须正确处理字符串上下文**，不能像 `sed 's|//.*||'` 那样撞坏 URL。
- **补全去 emoji**：`completeSelectorNames/Nodes` 返回 `StripEmoji()` 处理后的候选名，原名字串匹配兜底。
- **`config edit` 已废弃**（模板真源是服务器，编辑本地下载副本无意义）。

## 模板格式

所有模板用 `.template.jsonc` 后缀（LSP 不报注释错误）。命名规则：
```
config.template.jsonc             → 旧版（已弃用）
config-fakeip-prefer-ipv4.template.jsonc
config-realip-v4-only.template.jsonc
```

CI 发布时去 `.template.jsonc` → `.json`，去 JSONC 注释，保留 `$VAR` 占位符。

## 测试

```bash
go test -race -cover ./...
```

核心测试覆盖：
- `internal/template_test.go` — `expandEnvsubst`（13 用例）、`RenderProfile`
- `internal/api_test.go` — `GetSelectors`、`ResolveSelector`、`ResolveNode`（mock JSON）
- `internal/env_test.go` — `ReadEnvFile`（注释、引号、空白）、`RequireEnvVars`
- `internal/variant_test.go` — `NormalizeConfigVariant`、`ActiveConfigVariant`、`SetConfigVariant`
- `internal/ui_test.go` — `ExtractUIZip`、`InstallUIDir`、`copyDir`
- `internal/emoji_test.go` — `StripEmoji`（常见 Selector/Node 名）
- `cmd/common_test.go` — 命令注册、help 不 panic、补全生成

## 发版

```bash
git tag v0.x.y && git push origin v0.x.y
```

CI 自动：build（3 平台）→ checksums → GitHub Release → Homebrew 公式更新。

## 本地构建

```bash
make build        # 编译到 ./sbc（ldflags 注入版本号）
make install      # 编译 + 安装到 /usr/local/bin/sbc
```

## 历史教训

- **JSONC URL bug**：`sed 's|//.*||'` 会把 `https://` 的 `//` 当注释删掉。必须用 `strip-jsonc.py`（理解字符串上下文）。
- **`NormalizeConfigVariant` 不该有别名映射**：早期版本把 `REALIP` → `realip-v4-only`，导致文件名不匹配。改为只做 lowercase，由文件系统做真相源。
- **`DefaultConfigVariant` 是毒瘤**：旧版 fallback 到 `"default"` 但文件名是 `config-fakeip-prefer-ipv4.json`，永远找不到文件。改为必须显式设置变体。
- **`runner.os` vs `runtime.GOOS`**：GitHub Actions 的 `runner.os` = `macOS`（大写），Go 的 `runtime.GOOS` = `darwin`（小写）。用 os matrix 显式定义 os_name。
