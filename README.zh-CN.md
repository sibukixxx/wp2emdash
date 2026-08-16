# wp2emdash

[![CI](https://github.com/sibukixxx/wp2emdash/actions/workflows/ci.yml/badge.svg)](https://github.com/sibukixxx/wp2emdash/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/sibukixxx/wp2emdash.svg)](https://pkg.go.dev/github.com/sibukixxx/wp2emdash)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/sibukixxx/wp2emdash)](go.mod)

[English](README.md) | [日本語](README.ja.md) | 简体中文

`wp2emdash` 是一个用 Go 编写的 WordPress → EmDash 迁移 CLI。它遵循 Unix 思想，将迁移拆分为按阶段执行的小命令，并对 `wp-cli`、`wrangler`、`rclone` 等现有工具进行轻量封装。命令输出 JSON 或 Markdown，便于与其他工具组合使用。

## 目录

- [为什么使用小命令](#为什么使用小命令)
- [安装](#安装)
- [快速开始](#快速开始)
- [命令](#命令)
- [预设](#预设)
- [设计原则](#设计原则)
- [路线图](#路线图)
- [参与贡献](#参与贡献)
- [许可证](#许可证)

```text
wp2emdash audit                  -> 评估迁移复杂度并计算风险分数
wp2emdash db plan                -> 根据 summary.json 生成数据库迁移计划
wp2emdash media scan             -> 为 wp-content/uploads 生成 JSON 清单
wp2emdash report                 -> 根据 summary.json 重新生成风险报告
wp2emdash run --preset           -> 执行迁移阶段预设
wp2emdash secrets check          -> 检查预设所需的密钥
wp2emdash seo extract-meta       -> 将文章 SEO 元数据导出为 JSON
wp2emdash seo extract-redirects  -> 将重定向规则导出为 JSON
wp2emdash seo url-map            -> 比较新旧 URL 列表
wp2emdash doctor                 -> 检查外部依赖工具
```

## 为什么使用小命令

实际迁移通常需要分阶段进行：

```text
最低限度验证 -> 小型生产环境 -> SEO 敏感型生产环境 -> 大量媒体 -> 自定义重建
```

`wp2emdash` 作为这些阶段的编排器，自动完成机械性工作，同时明确保留需要人工判断的部分。它不会提供一个包办所有操作的 `migrate-all` 命令。

## 安装

需要 Go 1.22 或更高版本。

```bash
git clone https://github.com/sibukixxx/wp2emdash.git
cd wp2emdash
make build
./bin/wp2emdash --help

# 或者
go install github.com/sibukixxx/wp2emdash/cmd/wp2emdash@latest
```

## 快速开始

请在 WordPress 主机上运行，或确保当前环境能够访问 WordPress 安装目录。

```bash
# 1. 检查外部依赖
wp2emdash doctor

# 2. 审计本地 WordPress 站点
wp2emdash audit --wp-root /var/www/html

# 也可以通过只读 HTTP agent 审计
wp2emdash audit \
  --agent-url https://example.com/wp-json/wp2emdash/v1/audit \
  --agent-token secret-token

# 3. 扫描媒体文件并生成清单
wp2emdash media scan --dir /var/www/html/wp-content/uploads --hash

# 4. 先 dry-run，再明确执行预设
wp2emdash run --preset minimal --wp-root /var/www/html --dry-run
wp2emdash run --preset minimal --wp-root /var/www/html --apply

# 5. 根据审计结果生成数据库迁移计划
wp2emdash db plan \
  --from wp2emdash-output/summary.json \
  --preset small-production

# 6. 检查环境中是否已有预设所需的密钥
wp2emdash secrets check --profile small-production

# 7. 导出 SEO 元数据和重定向
wp2emdash seo extract-meta --wp-root /var/www/html
wp2emdash seo extract-redirects --wp-root /var/www/html
```

默认输出目录为 `wp2emdash-output/`，主要产物包括：

- `summary.json`
- `risk-report.md`
- `media-manifest.json`
- `db-plan.json`
- `db-plan.md`
- `seo-meta.json`
- `seo-redirects.json`

## 命令

| 命令 | 用途 | 主要选项 |
| --- | --- | --- |
| `doctor` | 检查 `wp`、`wrangler`、`git` 等工具 | `--json` |
| `audit` | 通过本地 WP-CLI、SSH 或 HTTP agent 评估迁移风险 | `--wp-root` `--write` `--json` `--ssh` `--agent-url` |
| `db plan` | 根据 `summary.json` 生成 JSON/Markdown 数据库迁移计划 | `--from` `--preset` `--write` `--json` |
| `media scan` | 通过本地目录、SSH 或 HTTP agent 生成媒体清单 | `--dir` `--hash` `--max-files` `--ssh` `--agent-url` |
| `report` | 根据 `summary.json` 重新生成 `risk-report.md` | `--from` `--stdout` |
| `run --preset` | 执行迁移阶段预设 | `--preset` `--wp-root` `--dry-run` `--apply` |
| `secrets check` | 检查所需的 secret 环境变量 | `--profile` `--json` |
| `seo extract-meta` | 合并并导出 Yoast、Rank Math、AIOSEO 元数据 | `--wp-root` `--write` `--ssh` |
| `seo extract-redirects` | 导出 `.htaccess` 和重定向插件中的规则 | `--wp-root` `--write` `--ssh` |
| `seo url-map` | 比较新旧 URL 列表 | `--old` `--new` `--write` |

风险评分采用累加方式。可以使用 `--risk-bands path/to/custom.json` 替换面向用户的风险级别和估算区间。

## 预设

`wp2emdash run --preset <name>` 提供五种预设：

| 预设 | 范围 |
| --- | --- |
| `minimal` | PoC 审计与迁移可行性报告 |
| `small-production` | 小型博客或落地页的生产迁移 |
| `seo-production` | 对 SEO 敏感内容的生产迁移 |
| `media-heavy` | 包含大量图片、PDF 或视频的迁移 |
| `custom-rebuild` | 涉及主题、插件、mu-plugins 和外部集成的重建项目 |

目前 `minimal` 是唯一完整实现的预设。其他预设已经包含 `db plan` 和 `secrets check`，部署、媒体同步和验证等后续步骤仍在逐步实现。

## 设计原则

- 一个命令只承担一个职责
- 输出 JSON 或 Markdown
- 破坏性操作默认使用 dry-run
- 轻量封装外部工具，不重复实现它们
- 永不生成或覆盖 `.env`

代码依赖方向为：`cli -> usecase -> {domain, infra} -> shell`。`domain` 不依赖外部系统。详细规则请参阅 [CONTRIBUTING.md](CONTRIBUTING.md)。

## HTTP Agent

除了 SSH，`wp2emdash` 还可以从 WordPress 内的只读 HTTP agent 获取审计数据和媒体清单。Go 客户端优先选择 `agent-url`，其次是 `ssh`，最后是本地访问。完整请求和响应 schema 请参阅[英文文档](README.md#http-agent-schema)。

## 路线图

| 版本 | 计划范围 | 状态 |
| --- | --- | --- |
| v0.1 | doctor / audit / media scan / report / minimal preset | 已完成 |
| v0.2 | `env generate` / `secrets check` / `db plan` | 后两项已完成 |
| v0.3 | `media sync` / `media verify` / 旧媒体路径 Worker | 同步和验证已完成 |
| **v0.4** | SEO 元数据、重定向和 URL map | 已完成 |
| v0.5 | 主题和插件分析、重建计划报告 | 未开始 |
| v1.0 | 完整实现五种预设和 GitHub Actions 脚手架 | 未开始 |

## 参与贡献

核心命令集稳定之前，Issues 暂时关闭；欢迎提交 Pull Request。PR 流程和质量门槛请参阅 [CONTRIBUTING.md](CONTRIBUTING.md)。主要检查包括：

```bash
make build
make test
make vet
```

旧版 Bash 实现位于 [`legacy-bash/`](legacy-bash/)，作为受限远程环境中的备用方案和行为参考。

## 支持项目

如果 `wp2emdash` 为你的迁移工作节省了时间，可以通过 [GitHub Sponsors](https://github.com/sponsors/sibukixxx) 支持项目。

## 许可证

[MIT](LICENSE)
