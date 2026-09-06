# P2: Add scoped public audit, benchmarks, and CLI polish

## Problem

Public URLs provide useful preflight evidence but cannot support a full assessment. The project also lacks benchmark scenarios, a release workflow, and fully consistent automation-oriented flags.

## Goal

Add a deliberately limited public-surface audit and reproducible operational tooling without blurring it with full WP-CLI/SSH/Agent assessment.

## Scope

- robots-respecting `public-audit` labeled `Public Surface Only`
- HTTPS, redirects, robots, sitemap, canonical, metadata, structured data, URL inventory, public REST/forms/dynamic hints
- conservative crawl/rate limits and local HTTP test fixtures
- small/medium/media-heavy benchmarks for runtime, memory and scan/report cost
- consistent `--format`, `--quiet`, stdout/stderr and exit-code documentation
- schema golden tests, release policy/workflow, changelog and contributor polish

## Non-goals

Authenticated/private discovery, vulnerability exploitation, full-assessment claims, CMS recommendations, pricing, estimates, proposals, CRM, and sales automation.

## Acceptance criteria

- Public output always states its limited scope and honors robots.txt.
- Tests have no live-site dependency.
- README contains only measured performance results.
- CLI behavior remains backward-compatible and CI-friendly.
- Reproducible release builds are documented and automated.

## Tests

Robots allow/disallow, redirects, sitemap/malformed HTML, crawl limits, no-auth-bypass regressions, benchmark smoke tests, stdout/stderr/exit-code matrix, schema golden files, and cross-builds.
