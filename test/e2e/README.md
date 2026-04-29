# E2E Tests

`wp2emdash` の E2E テストは、実際に CLI バイナリをビルドして実行し、出力ファイルと stdout を検証する。

## 実行方法

```bash
make test-e2e
make test-e2e E2E_RUN=TestAuditCommand
```

通常の `make test` では E2E は実行しない。`WP2EMDASH_E2E_TEST_ENABLED=true` が有効なときだけ `test/e2e/tests` が走る。

## 構成

```text
test/e2e/
├── setup.go            # バイナリ build・stub tool 配置・実行 helper
├── testdata/wp-site/   # WordPress fixture
├── tests/
│   ├── main_test.go    # E2E 有効化フラグの入口
│   ├── audit_cmd_test.go
│   ├── media_scan_cmd_test.go
│   └── run_cmd_test.go
```

## 方針

- `wp`, `wrangler`, `git` はテスト専用 stub を PATH 先頭に置く
- `audit` は `wp-content/` の実ファイルと fake `wp` 出力の両方を使って検証する
- `run --preset` もバイナリ経由で通し、CLI wiring の崩れを検知する
