# Development Commands

`wp2emdash` 開発時に使う標準コマンド一覧。

## Build

- `make build` - `bin/wp2emdash` を生成する
- `make run` - バイナリを build して `--help` を実行する
- `make dist` - `darwin/linux x amd64/arm64` の配布バイナリを生成する

## Test

- `make test` - 単体テストを `-race -count=1` で実行する
- `make test TEST_RUN=TestName` - 特定のテスト関数だけ実行する
- `make test-e2e` - CLI の E2E テストを実行する
- `make test-e2e E2E_RUN=TestAuditCommand` - 特定の E2E シナリオだけ実行する
- `make test-all` - 単体テストと E2E の両方を実行する

## Quality

- `make vet` - `go vet ./...` を実行する
- `make golangci` - `golangci-lint` を実行する
- `make lint` - `vet` と `golangci-lint` をまとめて実行する
- `make fmt` - `gofmt` で Go ファイルを整形する
- `make fix` - `gofmt` と `golangci-lint --fix` を実行する

## Cleanup

- `make clean` - `bin/`, `wp2emdash-output/`, `coverage.out` を削除する
