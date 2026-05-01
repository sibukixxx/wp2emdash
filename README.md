# wp2emdash

English documentation: [README.en.md](README.en.md)

WordPress → EmDash 移行を **フェーズ別の小さなコマンド群** として実行する Go 製 CLI。Unix 思想に倣い、`wp-cli` / `wrangler` / `rclone` などの既存ツールを薄くラップして JSON / Markdown を出力するので、他ツールに繋げやすい。

```
wp2emdash audit          → 複雑度を計測・スコア化
wp2emdash media scan     → wp-content/uploads を JSON manifest 化
wp2emdash report         → summary.json から risk-report.md を再生成
wp2emdash run --preset   → フェーズプリセットを実行
wp2emdash doctor         → 必要な外部ツールが揃っているか確認
```

## なぜ別ツールか

EmDash 移行は「全部入りの自動移行」よりも、案件ごとに

```
最低検証 → 小規模本番 → SEO 込み本番 → メディア本格 → 独自機能込み
```

のように **フェーズ別に組み合わせる** ほうが現実的。`wp2emdash` は orchestrator として個別フェーズの作業を機械化し、人間判断が必要な部分は明示的に残す。

## インストール

### ソースから

Go 1.22 以上。

```bash
git clone <this-repo>
cd wp2emdash
make build              # ./bin/wp2emdash
./bin/wp2emdash --help

# あるいは
go install ./cmd/wp2emdash
```

## クイックスタート

WordPress サーバ（または `wp-config.php` がある場所）で:

```bash
# 1. 依存ツールが揃っているか確認
wp2emdash doctor

# 2. WordPress 複雑度を計測してスコア化
wp2emdash audit --wp-root /var/www/html
#   → wp2emdash-output/summary.json
#   → wp2emdash-output/risk-report.md

# HTTP agent 経由でも監査可能
wp2emdash audit \
  --agent-url https://example.com/wp-json/wp2emdash/v1/audit \
  --agent-token secret-token

# 公開版ではリスク帯/見積り帯の policy を差し替え可能
wp2emdash audit \
  --wp-root /var/www/html \
  --risk-bands ./config/custom-risk-bands.json

# 3. uploads を manifest 化（R2 同期前の差分計算用）
wp2emdash media scan --dir /var/www/html/wp-content/uploads --hash
#   → wp2emdash-output/media-manifest.json

# HTTP agent 経由でも media scan 可能
wp2emdash media scan \
  --agent-url https://example.com/wp-json/wp2emdash/v1/media-scan \
  --agent-token secret-token \
  --dir wp-content/uploads

# 4. 「最低検証」プリセットを dry-run で確認 → apply
wp2emdash run --preset minimal --wp-root /var/www/html --dry-run
wp2emdash run --preset minimal --wp-root /var/www/html --apply

# HTTP agent 経由で preset minimal を実行
wp2emdash run --preset minimal \
  --agent-audit-url https://example.com/wp-json/wp2emdash/v1/audit \
  --agent-media-url https://example.com/wp-json/wp2emdash/v1/media-scan \
  --agent-token secret-token \
  --wp-root /var/www/html \
  --apply

# preset 実行でも同じ policy file を使える
wp2emdash run --preset minimal \
  --wp-root /var/www/html \
  --risk-bands ./config/custom-risk-bands.json \
  --apply
```

## v0.1 の機能

| サブコマンド | 役割 | 主なフラグ |
| --- | --- | --- |
| `doctor` | `wp` / `wrangler` / `git` 等の存在確認 | `--json` |
| `audit` | WP-CLI / SSH / HTTP agent で 14 観点を計測してスコア化 | `--wp-root` `--write` `--json` `--ssh` `--agent-url` |
| `media scan` | ローカル / SSH / HTTP agent で JSON manifest を生成 | `--dir` `--hash` `--max-files` `--histogram-only` `--ssh` `--agent-url` |
| `report` | `summary.json` から `risk-report.md` を再生成 | `--from` `--stdout` |
| `run --preset` | 5 種のフェーズプリセットを実行 | `--preset` `--wp-root` `--dry-run` `--apply` `--ssh` `--agent-audit-url` `--agent-media-url` |

スコアリング規則は加点式。公開版では level / 見積り帯を `--risk-bands path/to/custom.json` で差し替えられる。以下は **同梱デフォルト policy の例**:

| Level | スコア | 見積り目安 |
| --- | --- | --- |
| Simple | 0–20 | 5万〜20万円 |
| Standard | 21–50 | 20万〜60万円 |
| Complex | 51–90 | 60万〜150万円 |
| High Risk | 91–130 | 150万〜300万円 |
| Rebuild Project | 131+ | 300万円〜 / 個別見積り |

## プリセット

`wp2emdash run --preset <name>` で実行する 5 種の組み合わせ:

| Preset | スコープ |
| --- | --- |
| `minimal` | PoC: 複雑度を測り EmDash 移行可否レポートを出すだけ |
| `small-production` | 小規模ブログ/LP を本番化（投稿/固定ページ/uploads/standard SEO） |
| `seo-production` | SEO を落とさない本番移行（meta / canonical / redirect / OGP） |
| `media-heavy` | 大量画像・PDF・動画を R2 に安全移送 |
| `custom-rebuild` | functions.php / plugins / mu-plugins / 外部連携を含む再構築案件 |

v0.1 では `minimal` を完全実装、それ以外は audit + media scan + report 部分のみ実装、後段は `todo` ステップとしてプレースホルダで残し、後続バージョンで埋めていく方針。

## 設計

Clean Architecture 風に 3 層に整理：

```
cmd/
  wp2emdash/main.go          エントリポイント
internal/
  cli/                        cobra コマンド定義（flag 解析 + 出力フォーマットのみ）
  usecase/                    各サブコマンドの orchestration
    {audit,doctor,media_scan,report,run_preset}.go
    reporting/                JSON / Markdown レポート生成
  domain/                     純粋なデータ型・ビジネスルール
    audit/                    Audit / SiteInfo / ContentStats など
    media/                    Manifest / File
    preset/                   フェーズプリセット定義
    score/                    スコアリングルール（純粋関数）
  infra/                      外部システム adapter
    wpcli/                    wp-cli を叩く auditor
    filesystem/               uploads スキャナ
  shell/                      os/exec の薄いラッパ（DryRun 対応）
test/
  e2e/                        E2E テストヘルパー（fixtures / stubs / runner）
    tests/                    実テストケース
legacy-bash/                  v0 相当の bash スクリプト（同じ重み付け、参照用）
```

依存方向: `cli → usecase → {domain, infra} → shell`。`domain` は外部に依存しない。

設計原則は [`CONTRIBUTING.md`](CONTRIBUTING.md#設計原則) を参照。要約:

- **1 コマンド = 1 責務**
- **JSON / Markdown 出力**
- **dry-run 既定**
- **外部コマンドは薄くラップ**
- **`.env` を生成・上書きしない**

## HTTP Agent Schema

`wp2emdash` は SSH の代わりに、WordPress 内の read-only HTTP agent からも監査値を取得できる。現時点で Go 側が期待する response schema は以下で固定。

### `GET /wp-json/wp2emdash/v1/audit`

認証:

```http
Authorization: Bearer <token>
Accept: application/json
```

response:

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

query:

- `dir`
- `hash=1`
- `max_files=200`
- `histogram_only=1`

認証は `audit` と同じ bearer token。

response:

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

Go 側の優先順位は `agent-url > ssh > local`。`--agent-url` と `--ssh` の併用はエラー。

### Preset Execution With HTTP Agent

`run --preset minimal` では `--agent-audit-url` と `--agent-media-url` を受け取る。

推奨例:

```bash
wp2emdash run --preset minimal \
  --agent-audit-url https://example.com/wp-json/wp2emdash/v1/audit \
  --agent-media-url https://example.com/wp-json/wp2emdash/v1/media-scan \
  --agent-token secret-token \
  --wp-root /var/www/html \
  --apply
```

後方互換のため `--agent-url` も残してあり、`--agent-audit-url` / `--agent-media-url` が未指定のときの fallback として使われる。

## ロードマップ

| バージョン | 含めるもの |
| --- | --- |
| **v0.1（current）** | doctor / audit / media scan / report / run --preset minimal |
| v0.2 | `env generate` (`wrangler.jsonc` 雛形) / `secrets check` / `db plan` |
| v0.3 | `media sync` (rclone/wrangler ラッパ) / `media verify` / 旧 `/wp-content/uploads/*` 維持 Worker 雛形 |
| v0.4 | `seo extract-meta` / `seo extract-redirects` / URL map 比較 |
| v0.5 | `theme analyze` / `plugins analyze` / `mu-plugins analyze` / 再構築計画レポート |
| v1.0 | 5 プリセット全実装 + GitHub Actions ワークフロー雛形生成 |

## legacy-bash/

`legacy-bash/emdash-migration-audit.sh` は Go 化前の同等機能を持つ bash スクリプト。

- 軽量（Go バイナリを置けないリモート環境向け）
- 同じスコア重み付け（変更時は両方更新するルール）
- `wp2emdash audit` が動かない環境での fallback / リファレンス実装

詳細は [`legacy-bash/README.md`](legacy-bash/README.md)。

## ライセンス

MIT — [`LICENSE`](LICENSE)。

## 関連

- [EmDash CMS](https://github.com/emdash-cms/emdash) — 移行先 CMS
- [Cloudflare D1](https://developers.cloudflare.com/d1/) / [R2](https://developers.cloudflare.com/r2/) — EmDash の標準デプロイ先
- [WP-CLI](https://wp-cli.org/) — `wp2emdash audit` が裏で叩く
## Testing

```bash
make test
make test TEST_RUN=TestComputeAccumulatesSignals
make test-e2e
make test-e2e E2E_RUN=TestAuditCommand
make test-all
make lint
make fix
```

`make test` は通常の Go テストを実行し、`test/e2e/tests` は `make test-e2e` でのみ有効化される。
