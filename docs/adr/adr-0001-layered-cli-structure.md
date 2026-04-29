# レイヤード CLI 構造の採用

## 状態

accepted

## 背景

`wp2emdash` は WordPress 監査、media manifest 生成、report 生成などの責務を持つ CLI であり、機能追加のたびに `cobra` コマンドへロジックを書き込むと保守しづらくなる。E2E テスト導入後は、CLI wiring と実処理を分離しておかないと、テスト対象の切り分けも難しくなる。

## 決定

`wp2emdash` は `cmd -> internal/cli -> internal/usecase -> internal/infra` の流れを標準構造とし、純粋なデータ型とルールは `internal/domain` に置く。

- `cmd/` は起動のみ
- `internal/cli/` は flag 解析と出力整形のみ
- `internal/usecase/` はコマンドごとの orchestration
- `internal/infra/` は `wp-cli` や filesystem など外部依存の adapter
- `internal/domain/` は audit/media/preset/score の純粋な型と規則

## 影響

- 新規サブコマンド追加時の配置規則が明確になる
- CLI テストとユースケーステスト、E2E テストの責務分離がしやすくなる
- `internal/cli/` にビジネスロジックを戻す変更は避ける必要がある
