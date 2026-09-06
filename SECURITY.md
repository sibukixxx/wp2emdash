# Security policy

Please report suspected vulnerabilities privately through GitHub Security Advisories rather than a public issue. Include affected versions, reproduction steps, impact, and any suggested mitigation.

`wp2emdash` is an assessment and migration orchestration client, not a vulnerability scanner. Do not use it to bypass authentication, brute-force paths, or exploit a site.

Prefer `WP2EMDASH_AGENT_TOKEN` over `--agent-token` so tokens do not appear in shell history. Never place WordPress salts, database credentials, passwords, or API keys in assessment fixtures or artifacts. The documented agent response schemas contain inventory and media facts only; operators of an HTTP agent remain responsible for server-side authentication, authorization, rate limiting, logging redaction, TLS, and read-only enforcement.
