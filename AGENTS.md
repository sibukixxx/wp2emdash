# AGENTS.md

このリポジトリで作業するエージェント（Claude Code / Codex / Cursor 等）向けの正典ガイド。
人間向けの導入は [`README.md`](README.md)、設計原則の詳細は [`CONTRIBUTING.md`](CONTRIBUTING.md) を参照。

## プロジェクト概要

`wp2emdash` は WordPress → EmDash 移行を **Unix 思想で扱う Go 製 orchestrator**。
`wp-cli` / `wrangler` / `rclone` などを薄くラップし、JSON / Markdown を出力する小さなコマンド群として実装する。

- 言語: Go 1.22+
- モジュール名: `github.com/rokubunnoni-inc/wp2emdash`
- CLI フレームワーク: `github.com/spf13/cobra`
- バイナリ: `bin/wp2emdash`（`make build` で生成）

## 5 つの設計原則（不変）

PR・実装の判断軸。これに反する変更は原則リジェクト。

1. **1 コマンド = 1 責務** — `migrate-all` のような全部入りは作らない
2. **JSON / Markdown 出力** — 他ツールに pipe できる形にする
3. **dry-run 既定** — 破壊的操作は `--apply` か `--confirm-domain <d>` を必須に
4. **外部ツールは薄くラップ** — `wp` / `wrangler` / `rclone` を再実装しない
5. **`.env` を生成・上書きしない** — 認証情報の置き場を強制しない

## ディレクトリ構成

```
cmd/wp2emdash/main.go       エントリポイント（薄い）
internal/
  cli/                       cobra コマンド定義（1 ファイル = 1 サブコマンド）
  shell/                     os/exec の薄いラッパ（DryRun 対応）
  wordpress/                 wp-cli 経由の audit、ファイルシステム集計
  media/                     uploads スキャナ・manifest 生成
  score/                     スコアリング規則（純粋関数中心）
  report/                    JSON / Markdown レポート生成
  preset/                    フェーズプリセット定義
legacy-bash/                 v0 相当の bash 実装（参照用 / fallback）
```

`internal/cli/` は **薄く保つ**。フラグの読み取りと、`internal/<topic>/` のロジック呼び出しのみ。
ビジネスロジックを `cli/` に書くのは禁止。

## ビルド・テスト・品質ゲート

`Makefile` を正典として使う（global の Build/Test Command Detection ルールに準拠）。

```bash
make build      # bin/wp2emdash を生成
make test       # go test ./...
make vet        # go vet ./...
make lint       # 現状は vet のエイリアス
make dist       # darwin/linux × amd64/arm64 の static binary を dist/ に
make clean      # bin/ wp2emdash-output/ coverage.out を削除
```

PR 通過の最低条件：

- [ ] `go vet ./...` が 0 警告
- [ ] `go test ./...` が全 pass（CI は `-race -count=1`）
- [ ] `go build ./...` が成功
- [ ] スコアリング規則を変えたら `legacy-bash/emdash-migration-audit.sh` の重みも合わせて更新

## 新しいサブコマンドの追加手順

1. `internal/cli/<name>.go` に cobra コマンドを定義（薄く）
2. ロジック本体は `internal/<topic>/` 以下のパッケージに置く
3. 単体テストは対象パッケージ内に `_test.go` で書く（雛形: `internal/score/score_test.go`）
4. `internal/cli/root.go` の `NewRootCmd` で `root.AddCommand(...)`
5. プリセットに組み込むなら `internal/preset/preset.go` の対応 phase に Step を追加し、
   `internal/cli/run.go` の `runStep` switch に case を足す

## Go 実装ルール（このプロジェクト固有）

global の `rules/backend/go/{coding,design,testing}.md` に加え、本プロジェクトでは：

- **`shell.Runner` を経由する** — `os/exec` を直接呼ばない。dry-run / verbose / 引数記録のため
- **JSON 出力は `--json` フラグで切替** — human-readable がデフォルト、機械可読は opt-in
- **出力先は `--out` で受ける** — デフォルトは `wp2emdash-output/`、ハードコード禁止
- **副作用は `--apply` ガード** — ファイル/ネットワーク書き込みは dry-run 既定
- **`internal/score` は純粋関数で保つ** — I/O に依存させない（テスト容易性のため）
- **エラーは `%w` でラップ**（`%v` ではない）
- **`internal/cli` の panic は flag missing のみ**（`mustString` / `mustBool`）。それ以外は error を返す

### スコアリング規則を変える場合

スコアの重みは Go と bash の **両方を同時に更新**：

- `internal/score/score.go`（および `score_test.go`）
- `legacy-bash/emdash-migration-audit.sh`

片方だけ更新する PR は受けない（互換性が崩れるため）。

## テスト方針

global の TDD ルールに準拠（Red → Green → Refactor）。Go 固有の追加事項：

- **テーブル駆動テスト** を基本形にする
- **`t.Helper()` / `t.Cleanup()`** をヘルパー・リソース解放で使う
- **テストは振る舞い**：`internal/score` ならスコア値、`internal/cli` なら stdout/stderr/exit code
- **外部コマンドのテスト** は `shell.Runner` をフェイク化して引数を検証

## コミット規約

`rules/core/commit.md` を継承。本リポジトリでも同じ：

- Conventional Commits（英語 type + 日本語 subject）
- 絵文字なし、`Co-Authored-By` なし
- 構造的変更（`[STRUCTURAL]`）と動作的変更（`[BEHAVIORAL]`）を同一コミットに混ぜない

例：

```
feat(media): mediaカテゴリにsync wrapperを追加
fix(score): WooCommerce検出のplugin name 一致条件を修正
docs(readme): preset一覧をv0.3に更新
```

## やってはいけないこと

- ❌ `migrate-all` のような統合コマンドを作る
- ❌ `internal/cli/` にビジネスロジックを書く
- ❌ `os/exec` を直接呼ぶ（必ず `shell.Runner` 経由）
- ❌ `.env` を生成・編集する
- ❌ デフォルトで destructive な動作をする（dry-run を外すには明示フラグ）
- ❌ `legacy-bash/` のスコア重みを Go 側と片方だけ更新する
- ❌ 一度マージされた何か（migration / 公開 API）を後方互換なしに変える

## 参考リソース

- [README.md](README.md) — 機能一覧・プリセット・ロードマップ
- [CONTRIBUTING.md](CONTRIBUTING.md) — 設計原則・PR フロー
- [legacy-bash/README.md](legacy-bash/README.md) — bash 版の挙動と互換性メモ
