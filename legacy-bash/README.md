# EmDash 移行診断スクリプト

WordPress サイトの **複雑度を定量化** して、EmDash（Astro + Cloudflare Workers + D1 + R2）への移行難易度・概算見積りレンジを自動で算出するための診断ツール。

「投稿数 × 画像数」という浅い量だけではなく、**EmDash でそのまま移せるもの / 再実装が必要なもの / 移行事故になりやすいもの** をスコアリングする観点に立って設計している。

```
emdash-migration-audit.sh
        │
        ▼
emdash-audit-output/
  ├ summary.json         # 機械可読のメトリクス
  ├ risk-report.md       # 営業/見積り提出用の人間向けレポート
  ├ plugins.json         # 有効/無効プラグイン一覧
  ├ themes.json          # インストール済みテーマ
  ├ post-types.json      # 登録済み post type
  ├ taxonomies.json      # 登録済み taxonomy
  ├ postmeta-top.csv     # postmeta key 上位 100
  └ uploads-extensions.txt  # uploads 配下の拡張子ヒストグラム
```

## 使い方

WordPress のインストール先 (`wp-config.php` がある場所) で実行する。リモートなら SSH で入って同じところで実行。

```bash
# このリポジトリから対象サーバへスクリプトだけコピーして実行する例
scp scripts/audit/emdash-migration-audit.sh user@target:/var/www/html/
ssh user@target 'cd /var/www/html && chmod +x emdash-migration-audit.sh && ./emdash-migration-audit.sh'

# あるいは直接対象サーバの WP ルートで
chmod +x emdash-migration-audit.sh
./emdash-migration-audit.sh                  # → ./emdash-audit-output/
./emdash-migration-audit.sh ./audit-2026-04  # 出力先を任意に指定
```

### 必要なもの

- `bash` 4 以上（macOS の標準 bash 3.2 では配列展開が動かないことがあるので、対象サーバ側で実行する）
- [WP-CLI](https://wp-cli.org)
- WordPress DB への read 権限（`wp db query` 経由）
- `wp-content/themes` / `plugins` / `mu-plugins` / `uploads` の read 権限

WP-CLI 未導入の環境では先に入れる:

```bash
curl -O https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar
chmod +x wp-cli.phar
sudo mv wp-cli.phar /usr/local/bin/wp
wp --info
```

## 診断観点

スクリプトが見ているのは以下 14 観点。

| # | 観点 | 主な指標 |
| -- | --- | --- |
| 1  | コンテンツ規模 | posts / pages / drafts / private / categories / tags / users / comments |
| 2  | uploads 複雑度 | 総容量・ファイル数・拡張子分布・本文中の `wp-content/uploads` 出現数・`http://` 直書き数 |
| 3  | テーマ複雑度 | active theme の PHP/CSS/JS 数・page templates・hook 出現数・jQuery 系出現数 |
| 4  | functions.php 複雑度 | テーマ内の `add_action` / `add_filter` / `register_post_type` / `register_taxonomy` / `add_shortcode` / `register_rest_route` / `add_meta_box` / `wp_schedule_event` / `wp_remote_*` 出現数 |
| 5  | プラグイン複雑度 | 有効プラグイン件数 + ACF / WooCommerce / SEO / フォーム / リダイレクト / 会員 / 多言語 / キャッシュ系の有無 |
| 6  | カスタム投稿タイプ・タクソノミー | core を除いた登録数 |
| 7  | postmeta 複雑度 | 上位 100 key・ACF 痕跡・serialized data 件数 |
| 8  | ショートコード複雑度 | 本文中で `[xxx ...]` 形式が見つかる投稿数 |
| 9  | SEO 複雑度 | Yoast / Rank Math / AIOSEO の postmeta 件数 |
| 10 | リダイレクト複雑度 | `.htaccess` の redirect/rewrite 行数・コード内 `wp_redirect` / `header('Location` 出現数 |
| 11 | ユーザー / 権限 | 全体件数（ロール別の深掘りは追加実装の余地あり） |
| 12 | mu-plugins | ファイル数・hook 出現数 |
| 13 | 外部連携 | `wp_remote_*` / `curl_init` / `admin-ajax.php` / `register_rest_route` / webhook / stripe / line / slack / mailchimp の出現数 |
| 14 | 基本メタ | WordPress version / PHP version / multisite フラグ / DB prefix |

## スコア → 見積りレンジ

加点式。100 点満点ではなく合計を見て段階判定する。

| 加点 | 条件 |
| --- | --- |
| +5  | 投稿 100 件超 |
| +10 | 投稿 500 件超 |
| +5  | 固定ページ 20 件超 |
| +5  | 有効プラグイン 10 個超 |
| +10 | 有効プラグイン 20 個超 |
| +15 | ACF / カスタムフィールド系プラグイン |
| +30 | WooCommerce |
| +25 | 会員系プラグイン |
| +20 | 多言語系プラグイン |
| +10 | カスタム投稿タイプあり |
| +15 | カスタム投稿タイプ 3 個以上 |
| +10 | カスタムタクソノミーあり |
| +10 | ショートコード利用投稿 20 件超 |
| +10 | テーマ周辺の hook が 50 件超 |
| +10 | mu-plugins あり |
| +10 | 外部連携・API・Ajax らしきコードあり |
| +10 | リダイレクト系プラグイン |
| +5  | SEO プラグインあり |
| +10 | SEO meta 100 件超 |
| +10 | serialized postmeta 100 件超 |
| +10 | `.htaccess` rewrite/redirect 10 行超 |
| +10 | コード内 `wp_redirect` あり |
| +10 | jQuery / admin-ajax 等の依存が 20 件超 |

| スコア | 判定 | 見積り目安 |
| --- | --- | --- |
| 0–20    | Simple          | 5万〜20万円 |
| 21–50   | Standard        | 20万〜60万円 |
| 51–90   | Complex         | 60万〜150万円 |
| 91–130  | High Risk       | 150万〜300万円 |
| 131+    | Rebuild Project | 300万円〜 / 個別見積り |

これは **営業会話の出発点** であり、固定見積りではない。`risk-report.md` の Risk Items を見て根拠を説明する材料にする。

## レポートの読み方

最初に見るべきフィールド:

```
risk.score
risk.level
plugins.has_acf
plugins.has_woocommerce
plugins.has_member
plugins.has_multilingual
customization.custom_post_type_count
customization.shortcode_post_count
customization.serialized_meta_count
uploads.file_count / uploads.size
theme.hook_like_occurrences
customization.external_integration_like_occurrences
```

これらが高い場合は **再構築寄り** で扱う。

判定別の営業トーク雛形:

- **Simple** — 基本的なブログ・小規模サイト構成。EmDash への PoC 移行または小規模移行で対応可能。
- **Standard** — 一般的な WordPress サイト構成。投稿・画像・SEO・フォーム等の確認は必要だが段階的な EmDash 移行は現実的。
- **Complex** — WordPress 固有機能への依存が多く一部再実装が必要。まず PoC 環境で移行検証を行うことを推奨。
- **High Risk** — プラグイン・カスタム投稿・SEO・外部連携などの依存が強い。単純移行ではなく再構築案件として扱う。
- **Rebuild Project** — CMS / フロント / インフラの再設計プロジェクト。要件定義と段階移行計画が必要。

## まだ実装していない（v2 以降の候補）

- ロール別ユーザー数の集計 (`wp_capabilities` 内訳)
- `wp_options` の肥大度（autoload あり総バイト数）
- WP cron event 件数
- multisite ブロック (`is_multisite` フラグだけ取得済み、件数や子サイト構成までは未取得)
- 画像 alt 欠損率・内部リンク旧 URL 率・404 候補
- DB prefix の wpil_ 等カスタム検出
- upload path の custom 設定検出
- Cloudflare 既存設定（DNS / WAF / page rules）の取得は別ツール

## ロードマップ

- **v1: Bash + WP-CLI**（このスクリプト）— 安く速く診断、初期営業向け
- **v2: Node.js / TypeScript CLI** — JSON ハンドリング・テスト・複数環境対応をきれいに。社内ツール化
- **v3: 管理画面付き SaaS** — URL / SSH / WP-CLI 出力から EmDash 移行難易度を自動判定する WebUI、移行リード獲得用

## 参考

スコアリング設計と運用方針の元ネタは社内方針（このリポジトリの議論ログ）。指標と重み付けは現場で使いながら継続的に調整する想定。
