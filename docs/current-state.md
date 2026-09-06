# Current implementation assessment

This inventory was produced from command registration and use-case/adapter code, not from README claims.

## Architecture

The executable is a thin Cobra entry point. `internal/cli` owns flags and output, `internal/usecase` orchestrates commands, `internal/domain` contains models and pure comparison/scoring rules, and `internal/infra` owns WP-CLI, SSH, filesystem, HTTP-agent, rclone, and EmDash CLI adapters. Presets dispatch through a step registry. Unit tests live beside packages and CLI E2E tests use deterministic stub executables and a WordPress filesystem fixture.

## Command status

| Command | Status | Code evidence / limitation |
| --- | --- | --- |
| `doctor` | IMPLEMENTED | Registered; checks external tools through use case |
| `audit` | IMPLEMENTED | Local WP-CLI, SSH, and HTTP-agent adapters; warnings tolerate partial probes |
| `db plan` | IMPLEMENTED | Generates JSON/Markdown plan; does not mutate DB |
| `media scan` | IMPLEMENTED | Local, SSH, and agent modes; hashing and histogram options |
| `report` | IMPLEMENTED | Re-renders legacy audit Markdown from summary |
| `run` | PARTIAL | Registry works; several preset steps remain explicit `todo` placeholders |
| `secrets check` | IMPLEMENTED | Checks environment presence; does not emit secret values or write `.env` |
| `seo extract-meta` | IMPLEMENTED | WP-CLI extraction for core/Yoast/Rank Math/AIOSEO; no public-page canonical crawler |
| `seo extract-redirects` | IMPLEMENTED | `.htaccess`, Redirection, and Safe Redirect Manager extraction |
| `seo url-map` | IMPLEMENTED | Deterministic old/new URL comparison |
| `content snapshot wordpress` | IMPLEMENTED | Hash/structure snapshot through WP-CLI/SSH |
| `content snapshot emdash` | IMPLEMENTED | Snapshot through EmDash CLI adapter |
| `content verify` | IMPLEMENTED | Identity/content/relationship comparison and CI policy gate |
| `assess` | IMPLEMENTED (this change) | Read-only aggregation with schema 1.0 artifacts |
| `public-audit` | PLANNED | Kept separate because public evidence is not a full assessment |

No registered command was classified DEAD/UNUSED. The `legacy-bash` implementation is intentionally retained as compatibility/fallback reference rather than registered Go CLI code.

## Technical debt and gaps

- The legacy audit schema uses scalar zero values after failed probes. `assess` repairs the stable consumer boundary with section status, but field-level status remains a future schema enhancement.
- The legacy score retains level/estimate fields for JSON compatibility. The bundled policy no longer contains pricing; `assess` exposes only technical score and breakdown.
- Plugin inventory stores category booleans/count, not the complete active/inactive slug evidence.
- Dynamic-feature detection is largely aggregate/code-pattern evidence and cannot yet express all DETECTED/LIKELY/UNKNOWN cases independently.
- Media inventory lacks MIME distribution, missing/orphan/duplicate candidates, and oversized-file evidence.
- Theme inventory lacks parent/child relationship and named template evidence.
- HTTP agent is a client-side adapter; server-side authentication, rate limiting, logging, and secret filtering cannot be guaranteed by this repository alone.
- No release workflow, changelog, benchmark scenarios, or public-surface audit exists.
- Tests cover the core audit, score, media, SEO, content, agent failures, and E2E commands, but not every partial-permission combination or schema golden file.

## Follow-up issue plan

P1 should add field-level observation status, complete plugin/theme/dynamic evidence, media evidence, SEO URL inventory, content result states, agent hardening documentation/tests, and phase timing. P2 should add a robots-respecting public-surface audit, benchmark fixtures, CLI `--quiet/--format` consistency, schema golden compatibility tests, release policy, and contributor polish.
