# P1: Expand evidence coverage and field-level observation states

## Problem

The legacy audit model collapses failed individual probes into zero values and stores several areas only as aggregate counts/booleans. Consumers cannot always distinguish not detected from not observable, or trace plugin/theme/dynamic findings to specific evidence.

## Goal

Extend schema 1.x with field-level observation state and richer technical evidence while preserving legacy audit and EmDash migration compatibility.

## Scope

- Complete active/inactive plugin and parent/child theme evidence
- Field-level state for CPTs, taxonomies, shortcodes, custom/serialized metadata
- DETECTED/LIKELY/UNKNOWN dynamic-feature evidence
- MIME, oversized, missing, orphan-candidate and duplicate-candidate media evidence
- SEO URL inventory and content MATCH/CHANGED/MISSING/NEW/UNKNOWN projection
- Per-phase timing and HTTP-agent schema/security hardening

## Non-goals

CMS recommendations, pricing, estimates, proposals, TechVit consulting logic, SEO guarantees, vulnerability scanning, and destructive migration actions.

## Acceptance criteria

- Important signals link to typed evidence.
- Failed probes never become confirmed zero/false.
- Signal ordering and score breakdown are deterministic.
- Legacy audit JSON remains readable.
- Agent schemas expose no arbitrary configuration or secrets.

## Tests

Golden fixtures for inventory/SEO/media, partial permissions, malformed payloads, deterministic scoring, schema compatibility, content state projection, and HTTP-agent failures.
