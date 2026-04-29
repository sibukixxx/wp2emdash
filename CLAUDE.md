# CLAUDE.md

Claude Code 向けのプロジェクト指示。本リポジトリの正典は [`AGENTS.md`](AGENTS.md) で、
**まずそちらを読むこと**。本ファイルは Claude Code 固有の補足のみ扱う。

## 最初に読む順序

1. [`AGENTS.md`](AGENTS.md) — 設計原則・ディレクトリ構成・新コマンド追加手順
2. [`README.md`](README.md) — 機能一覧・プリセット・ユーザー視点
3. [`CONTRIBUTING.md`](CONTRIBUTING.md) — PR フロー・品質ゲート

## このプロジェクトの一行要約

`wp2emdash` は WordPress → EmDash 移行を **小さなコマンド群**として提供する Go 製 CLI。
`wp-cli` / `wrangler` / `rclone` を薄くラップし、JSON / Markdown を吐く。

## ビルド・テスト

global の Build/Test Command Detection ルールどおり、`Makefile` 優先：

```bash
make build      # bin/wp2emdash
make test       # go test ./...
make vet        # go vet ./...
```

CI は `go test -race -count=1 ./...`。race detector で落ちる変更は受け付けない。

## このリポジトリで特に守ること

global ルール + `AGENTS.md` の上に、Claude Code として特に意識すべき項目：

- **設計原則 5 つを判断軸にする**（1 コマンド 1 責務 / JSON 出力 / dry-run 既定 /
  外部ツール薄ラップ / `.env` 触らない）。原則と矛盾する依頼は確認してから進める
- **`internal/cli/` は薄く保つ**。ロジックを書きたくなったら `internal/<topic>/` に出す
- **`os/exec` を直接呼ばない**。必ず `internal/shell.Runner` 経由（dry-run のため）
- **スコアリングを変えるときは Go と `legacy-bash/` を同時更新**（互換性維持）
- **subagent を反射で呼ばない**（global ルール）。本プロジェクトは規模も小さく、
  1 レスポンスで完結する作業がほとんど。`Explore` agent はモノレポ横断調査時のみ
- **コミットは Conventional Commits + 日本語 subject、絵文字・Co-Authored-By なし**
  （global の `rules/core/commit.md` どおり）

## auto mode / 権限

- このリポジトリでの作業は基本 **auto mode** で問題ない（純粋な Go CLI、外部書き込みなし）
- ただし以下は **必ずユーザー確認**：
  - `legacy-bash/` の編集（互換性に影響）
  - `go.mod` の依存追加・更新
  - `Makefile` / `.github/workflows/` の変更
  - リリース（`make dist` / タグ打ち）

## トレーシング・観測性

このリポジトリは現状 **CLI ツール**であり LLM を呼ばないため、
global の `ai-agent-o11y.md` ルールは適用対象外。将来 LLM-driven な auto-fix 機能を入れる場合のみ再検討する。

## z-ai/

global ルールどおり `z-ai/` は gitignore 済み（このリポジトリでも有効）。
作業計画・進捗メモはそこに置くこと。
