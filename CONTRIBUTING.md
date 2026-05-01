# Contributing to wp2emdash

English project overview: [README.en.md](README.en.md)

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
make test       # go test -race -count=1 ./...
make test-e2e   # CLI の end-to-end tests
make test-all   # unit + E2E
make vet        # go vet ./...
make lint       # golangci-lint + vet
make fix        # gofmt + golangci-lint --fix
```

新しいサブコマンドを足す手順:

1. `internal/cli/<name>.go` に cobra コマンドを定義（flag 解析と出力整形のみ・薄く保つ）
2. orchestration は `internal/usecase/<name>.go` に書く
3. 外部システムが必要なら `internal/infra/<adapter>/` に adapter を切る
4. 純粋なデータ型・ルールは `internal/domain/<topic>/` に置く
5. 単体テストは対象パッケージ内に `_test.go` で。`internal/domain/score/score_test.go` を雛形に
6. E2E テストは `test/e2e/tests/<cmd>_<op>_test.go` に書く（`test/e2e/README.md` 参照）
7. `internal/cli/root.go` の `NewRootCmd` で `AddCommand`
8. プリセットに組み込むなら `internal/domain/preset/preset.go` の対応 phase に Step を追加し、`internal/usecase/run_preset.go` の switch に case を足す

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
- `go test -race -count=1 ./...` 全て pass
- `WP2EMDASH_E2E_TEST_ENABLED=true go test ./test/e2e/tests/...` 全て pass
- `golangci-lint run -c .golangci.yml` 全て pass
- `go build ./...` 成功
- スコアリング規則を変えるなら `legacy-bash/emdash-migration-audit.sh` の重みも合わせて更新（互換性維持）

## ロードマップ

[README.md の「ロードマップ」](README.md#ロードマップ) を参照。新機能は roadmap にある項目を優先します。
