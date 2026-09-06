# Assessment architecture

`wp2emdash assess` is a read-only aggregation command for a full WordPress migration assessment. It reuses the existing local WP-CLI, SSH, and HTTP-agent audit adapters and never calls migration, sync, deployment, or database-write commands.

## Stable artifacts

- `summary.json`: stable umbrella interface
- `inventory.json`: observed facts and per-section observation status
- `signals.json`: evidence-backed signals and deterministic score breakdown
- `technical-report.md`: technical human-readable rendering

Every JSON artifact has `schema_version: "1.0"`. Additive fields may be introduced in a minor release. Removing a field, changing its meaning/type, or changing an enum requires a new major schema version. Consumers must ignore unknown fields and must not infer a recommendation from the score.

## Facts, signals, and score

Facts are measurements returned by source adapters. Signals identify migration-relevant technical characteristics and include evidence. The technical score is only a deterministic aggregation of scored signals; it is not a CMS recommendation, estimate, price, or proposal.

Observation status distinguishes `observed`, `not_detected`, `unavailable`, `unknown`, and `error`. In schema 1.0 a section with a probe warning is `unavailable`; its numeric and boolean values must not be interpreted as confirmed zero/false. This preserves compatibility with the legacy audit model while fixing the ambiguity at the stable assessment boundary.

## Public versus full assessment

A full assessment requires local WP-CLI access, SSH, or a compatible authenticated read-only HTTP agent. Public pages cannot reveal drafts, private content, database metadata, complete plugin state, or filesystem evidence. A public-surface collector is therefore a separate future command and must label its result `Public Surface Only`.

## Security

Agent tokens should be supplied with `WP2EMDASH_AGENT_TOKEN`; command-line tokens are supported for compatibility but can appear in shell history. Agent responses are limited to the audit/media schemas. Passwords, auth salts, database credentials, and arbitrary configuration data are not part of those schemas. Error bodies are capped before being included in errors.

## Public OSS boundary and non-goals

This repository may inventory WordPress, detect technical features, produce migration signals, extract SEO/redirect evidence, fingerprint content, verify migrations, and expose machine-readable diagnostics. It does not contain TechVit consulting rules, customer recommendations, destination recommendations, estimates, pricing, proposals, customer-facing sales reports, CRM/lead automation, SEO ranking guarantees, or vulnerability exploitation.
