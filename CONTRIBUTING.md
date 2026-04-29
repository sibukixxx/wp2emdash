# Contributing to wp2emdash

wp2emdash は **EmDash 移行を Unix 思想で扱う orchestrator** です。設計の軸を外さない PR を歓迎します。

## 設計原則

1. **1 コマンド = 1 責務** — 全部入りの `migrate-all` は作らない
2. **JSON / Markdown 出力** — 他ツールに pipe できる形にする
3. **dry-run 既定** — 破壊的操作には `--apply` か `--confirm-domain <d>` を必須に
4. **外部ツールは薄くラップ** — `wp` / `wrangler` / `rclone` を再実装しない
5. **`.env` を生成・上書きしない** — 認証情報の置き場を強制しない

これに反する PR はリジェクトされる前提でレビューします。

## 開発フロー

```bash
git clone <this-repo>
cd wp2emdash
make build      # bin/wp2emdash
make test       # go test ./...
make vet        # go vet ./...
```

新しいサブコマンドを足す手順:

1. `internal/cli/<name>.go` に cobra コマンドを定義
2. ロジック本体は `internal/<topic>/` 以下のパッケージに（`cli` は薄くする）
3. 単体テストは対象パッケージ内に `_test.go` で。`internal/score/score_test.go` を雛形に
4. `internal/cli/root.go` の `NewRootCmd` で `AddCommand`
5. プリセットに組み込むなら `internal/preset/preset.go` の対応 phase に Step を追加し、`internal/cli/run.go` の `runStep` switch に case を足す

## コミットメッセージ

Conventional Commits を使用（英語 type、日本語 subject 可）:

```
feat(media): mediaカテゴリにsync wrapperを追加
fix(score): WooCommerce検出のplugin name 一致条件を修正
docs(readme): preset一覧をv0.3に更新
```

絵文字・Co-Authored-By は付けない。

## 品質ゲート

PR で minimum 通すべきもの:

- `go vet ./...` 0 警告
- `go test ./...` 全て pass
- `go build ./...` 成功
- スコアリング規則を変えるなら `legacy-bash/emdash-migration-audit.sh` の重みも合わせて更新（互換性維持）

## ロードマップ

[README.md の「ロードマップ」](README.md#ロードマップ) を参照。新機能は roadmap にある項目を優先します。
