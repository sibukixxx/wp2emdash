# wp2emdash

Small Go migration/orchestration CLI. It wraps existing tools such as `wp`, `wrangler`, and `rclone` and emits inspectable JSON/Markdown instead of reimplementing those tools.

## Source of truth
- User-facing behavior: `README.md` / localized READMEs
- Design details: `CONTRIBUTING.md`

## Commands
- `make build`
- `make test`
- `make vet`
- `make lint`
- `make dist` — release artifacts; run only when the task needs distribution

## Shared rules
- Keep wrappers thin; do not reimplement external CLIs or hide their failure modes behind a new platform.
- Preserve machine-readable output contracts and stable exit/error behavior.
- Auto/destructive operations must keep existing permission/safety boundaries.
- Generated migration/output directories are outputs, not hand-edited source.
- A race failure is a real failure; CI tests with the race detector.

## Change-dependent checks
- Go/CLI behavior: `make test && make vet`.
- Output/schema/scoring behavior: add/update focused tests and verify machine-readable output.
- Release packaging: use `make dist` only when release artifacts are part of the task.

## Done
- Applicable tests/vet pass.
- Changed CLI/output contracts are covered by tests/docs.
- Destructive/external actions were not performed merely to prove completion.
