# wp2emdash

English documentation for `wp2emdash`. Japanese documentation lives in [README.md](README.md).

`wp2emdash` is a Go CLI that breaks a WordPress → EmDash migration into small, phase-oriented commands. It follows a Unix-style approach: wrap existing tools such as `wp-cli`, `wrangler`, and `rclone` thinly, then emit JSON or Markdown that can be piped into other tooling.

```
wp2emdash audit          -> measure migration complexity and score risk
wp2emdash media scan     -> build a JSON manifest of wp-content/uploads
wp2emdash report         -> regenerate risk-report.md from summary.json
wp2emdash run --preset   -> execute a migration phase preset
wp2emdash doctor         -> check required external tools
```

## Why Small Commands

Real migrations are usually phased. In practice, projects tend to split into stages such as:

```
minimum validation -> small production -> SEO-sensitive production -> media-heavy -> custom rebuild
```

`wp2emdash` acts as an orchestrator for those phases. It automates the mechanical parts while leaving human decisions explicit.

## Install

### From Source

Go 1.22+ is required.

```bash
git clone <this-repo>
cd wp2emdash
make build
./bin/wp2emdash --help

# or
go install ./cmd/wp2emdash
```

## Quick Start

Run on a WordPress host, or anywhere that has access to a WordPress install:

```bash
# 1. Check external dependencies
wp2emdash doctor

# 2. Audit a WordPress site from a local path
wp2emdash audit --wp-root /var/www/html

# 3. Audit via HTTP agent
wp2emdash audit \
  --agent-url https://example.com/wp-json/wp2emdash/v1/audit \
  --agent-token secret-token

# 4. Override public-facing level/estimate policy
wp2emdash audit \
  --wp-root /var/www/html \
  --risk-bands ./config/custom-risk-bands.json

# 5. Scan uploads locally
wp2emdash media scan --dir /var/www/html/wp-content/uploads --hash

# 6. Scan uploads via HTTP agent
wp2emdash media scan \
  --agent-url https://example.com/wp-json/wp2emdash/v1/media-scan \
  --agent-token secret-token \
  --dir wp-content/uploads

# 7. Dry-run a preset, then apply it
wp2emdash run --preset minimal --wp-root /var/www/html --dry-run
wp2emdash run --preset minimal --wp-root /var/www/html --apply

# 8. Run preset minimal via split agent endpoints
wp2emdash run --preset minimal \
  --agent-audit-url https://example.com/wp-json/wp2emdash/v1/audit \
  --agent-media-url https://example.com/wp-json/wp2emdash/v1/media-scan \
  --agent-token secret-token \
  --wp-root /var/www/html \
  --apply

# 9. Use the same public risk-band policy during preset execution
wp2emdash run --preset minimal \
  --wp-root /var/www/html \
  --risk-bands ./config/custom-risk-bands.json \
  --apply
```

Artifacts are written to `wp2emdash-output/` by default:

- `summary.json`
- `risk-report.md`
- `media-manifest.json`

## v0.1 Commands

| Command | Purpose | Main Flags |
| --- | --- | --- |
| `doctor` | Check required tools such as `wp`, `wrangler`, and `git` | `--json` |
| `audit` | Measure 14 migration signals via local WP-CLI, SSH, or HTTP agent | `--wp-root` `--write` `--json` `--ssh` `--agent-url` `--risk-bands` |
| `media scan` | Build a JSON manifest via local path, SSH, or HTTP agent | `--dir` `--hash` `--max-files` `--histogram-only` `--ssh` `--agent-url` |
| `report` | Regenerate `risk-report.md` from `summary.json` | `--from` `--stdout` |
| `run --preset` | Execute one of the predefined migration presets | `--preset` `--wp-root` `--dry-run` `--apply` `--ssh` `--agent-audit-url` `--agent-media-url` `--risk-bands` |

The core scoring rubric is additive. Public-facing level labels and estimate bands are replaceable through `--risk-bands path/to/custom.json`. The table below is only the example shipped in the default bundled policy:

| Level | Score Range | Example Estimate |
| --- | --- | --- |
| Simple | 0–20 | 50k–200k JPY |
| Standard | 21–50 | 200k–600k JPY |
| Complex | 51–90 | 600k–1.5M JPY |
| High Risk | 91–130 | 1.5M–3M JPY |
| Rebuild Project | 131+ | 3M+ JPY / custom estimate |

## Presets

`wp2emdash run --preset <name>` currently exposes five presets:

| Preset | Scope |
| --- | --- |
| `minimal` | PoC-level audit and migration feasibility report |
| `small-production` | Small production blog / landing page migration |
| `seo-production` | Production migration with SEO-sensitive content |
| `media-heavy` | Media-heavy migration with large uploads footprint |
| `custom-rebuild` | Rebuild-heavy migration involving theme, plugins, mu-plugins, and integrations |

As of v0.1, `minimal` is the fully implemented preset. The others are partially implemented and still contain placeholder `todo` steps.

## Architecture

The repository is structured in three layers:

```text
cmd/
  wp2emdash/main.go
internal/
  cli/          cobra command definitions
  usecase/      orchestration per command
  domain/       pure data structures and rules
  infra/        adapters for external systems
  shell/        thin os/exec wrapper
test/
  e2e/          end-to-end helpers and fixtures
legacy-bash/    reference bash implementation
```

Dependency direction: `cli -> usecase -> {domain, infra} -> shell`.

## Design Principles

- One command = one responsibility
- JSON / Markdown output
- Dry-run by default for destructive flows
- Thin wrappers around external tools
- Never generate or overwrite `.env`

## HTTP Agent Schema

`wp2emdash` can consume metrics from a read-only WordPress HTTP agent instead of SSH.

### `GET /wp-json/wp2emdash/v1/audit`

Headers:

```http
Authorization: Bearer <token>
Accept: application/json
```

Response:

```json
{
  "audit": {
    "site": {
      "home_url": "https://example.com",
      "site_url": "https://example.com",
      "wp_version": "6.5.0",
      "php_version": "8.2.12",
      "db_prefix": "wp_",
      "is_multisite": "no"
    },
    "content": {
      "posts": 120,
      "pages": 12,
      "drafts": 3,
      "private_posts": 1,
      "categories": 8,
      "tags": 15,
      "users": 4,
      "approved_comments": 22
    },
    "uploads": {
      "exists": true,
      "size": "12KB",
      "file_count": 3,
      "posts_with_uploads_paths": 7,
      "posts_with_http_urls": 2
    },
    "theme": {
      "active_theme": "example-theme",
      "php_files": 12,
      "css_files": 2,
      "js_files": 4,
      "page_templates": 3,
      "hook_like_occurrences": 12,
      "jquery_like_occurrences": 3
    },
    "plugins": {
      "active_count": 2,
      "has_acf": true,
      "has_woocommerce": false,
      "has_seo": true,
      "has_form": false,
      "has_redirect": true,
      "has_member": false,
      "has_multilingual": false,
      "has_cache": true
    },
    "customization": {
      "custom_post_type_count": 1,
      "custom_taxonomy_count": 1,
      "mu_plugin_count": 0,
      "mu_plugin_hook_like_occurrences": 0,
      "shortcode_post_count": 5,
      "seo_meta_count": 11,
      "serialized_meta_count": 9,
      "htaccess_redirect_like_lines": 0,
      "code_redirect_like_occurrences": 0,
      "external_integration_like_occurrences": 0
    }
  },
  "warnings": [
    {
      "code": "content.posts",
      "message": "probe failed"
    }
  ]
}
```

### `GET /wp-json/wp2emdash/v1/media-scan`

Query parameters:

- `dir`
- `hash=1`
- `max_files=200`
- `histogram_only=1`

Response:

```json
{
  "base_dir": "wp-content/uploads",
  "total_files": 3,
  "total_bytes": 12,
  "extensions": {
    "txt": 1
  },
  "files": [
    {
      "path": "2024/01/hello.txt",
      "size": 12,
      "sha256": "optional",
      "mime": "text/plain",
      "ext": "txt"
    }
  ]
}
```

Selection priority in Go is:

- `agent-url`
- `ssh`
- local

For `run --preset`, the preferred form is:

- `agent-audit-url`
- `agent-media-url`

`--agent-url` is still accepted as a backward-compatible fallback.

## Roadmap

| Version | Planned Scope |
| --- | --- |
| v0.1 | doctor / audit / media scan / report / run --preset minimal |
| v0.2 | `env generate`, `secrets check`, `db plan` |
| v0.3 | `media sync`, `media verify`, legacy uploads route worker scaffolding |
| v0.4 | `seo extract-meta`, `seo extract-redirects`, URL map comparison |
| v0.5 | `theme analyze`, `plugins analyze`, `mu-plugins analyze`, rebuild planning report |
| v1.0 | Full implementation of all five presets plus GitHub Actions scaffolding |

## Legacy Bash

`legacy-bash/emdash-migration-audit.sh` is the pre-Go reference implementation.

- Useful in very constrained remote environments
- Keeps the same scoring weights
- Serves as a fallback and behavioral reference

## License

MIT
