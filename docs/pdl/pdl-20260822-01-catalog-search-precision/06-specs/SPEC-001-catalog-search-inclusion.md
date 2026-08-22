# SPEC-001: Catalog search inclusion

Status: current on 2026-08-22
Owner: @ann
Audience: engineering

---

pdl_id: pdl-20260822-01-catalog-search-precision
artifact_id: SPEC-001
Spec key: 20260822-01
revision: 1
Implementation state: planned
authority_level: spec
Product sources: UC-20260822-01 rev.1 accepted; PRD-20260822-01 rev.1 accepted; UX-20260822-01 rev.1 accepted
ADR coverage: ADR-001 rev.1 accepted 2026-08-22 by @ann
Owners: @ann
canonical_path: docs/pdl/pdl-20260822-01-catalog-search-precision/06-specs/SPEC-001-catalog-search-inclusion.md

## 1. Delivery scope and non-goals

Repair inclusion for catalog search so every listed skill contains every keyword as contiguous text in a written field. CLI, HTTP `q`, and App search share `CatalogService.ListSkills`.

Non-goals (UC §10): semantic search, typo repair, body full-text, provider-name match, digest redesign, restyling `/api/skills` to `/v1` or adding pagination.

## 2. Upstream traceability

| Behavior | UC | PRD | UX | ADR |
|---|---|---|---|---|
| Contiguous inclusion, no phantom rows | 001, 005 | 001 | 001 | ADR-001 |
| Multi-keyword AND | 002 | 002 | 001 | ADR-001 |
| Empty means none | 003, 006 | 003 | 002 | ADR-001 |
| Same set on App, CLI, API | 004 | 004 | 001 | ADR-001 |
| Filters only subtract | 001 | 006 | 003 | ADR-001 |
| No typo repair | 006 | 003 | 002 | ADR-001 |

## 3. Current problem and superseded behavior

`fuzzyMatchRank` accepts an in-order rune subsequence over a whole field. `skillQueryFields` also searches provider name and category. Tests require `jira` → `job-iteration-report-app`. The App ignores `q` and uses `String.includes` on a different field set.

Superseded: subsequence inclusion; query match on provider name; App-local query matching. Those paths are deleted in this delivery.

## 4. Ownership and dependency boundaries

Owner: `CatalogService.ListSkills` query filter.

Dependents: `skm skills` (`runSkillsList`), `GET /api/skills` (`SkillHandler.List`), App `SkillsPage` via `api.getSkills({ q })`.

The App must not import or duplicate inclusion. Provider/status chips may filter the returned hit set locally.

## 5. Public and internal contracts

HTTP (existing shape; rest-api-standard `/v1` + pagination not applied — pre-existing list, not restyled here):

`GET /api/skills?q=&provider=&category=&tag=&status=&conflict=&sort=&grouped=`

`q` (trimmed, empty = no text filter): whitespace-split keywords, AND. A skill is kept if each keyword, compared case-insensitively, is an exact field, a prefix of a field, or a contiguous substring of a field, on at least one written field:

- name
- slug
- directory name
- each tag (declared aliases live here)
- summary / description

Not searched: provider name, category, body markdown, raw markdown.

Among hits only, keep today's weight × rank order (exact 4, prefix 3, substring 2) and field weights name 8, tag 6, slug/directory 5, summary 2. Remove the subsequence rank. `--sort` remains the tie-breaker.

`grouped=true` after inclusion (ADR-001).

CLI `-q` / `--query` is the same `Query` field. `--tag` stays an exact tag filter and is not a substitute for `q`.

## 6. State, data, transaction, concurrency, and idempotency semantics

Read-only over the current catalog snapshot. Each request is independent. No write, cache of old letter-order results, or second-pass widen on empty.

## 7. Authorization, privacy, safety, and audit

Unchanged catalog read path. Queries are not logged as a new product audit stream.

## 8. Failure and recovery realization

Empty `q` hits → empty list / `Total skills: 0` / App empty list. No auto-retry. Catalog load errors stay existing error states (UX: not this repair).

## 9. Migration, removal, and compatibility boundary

Delete in this delivery:

- `isRuneSubsequence` and `fuzzyRankSubsequence` as inclusion
- provider name and category on `skillQueryFields`
- App search branch that reads skill text (`matchesSkillFilters` query part)
- tests that require letter-order false positives

No flag, shim, or dual matcher. Agents type a written fragment or a repaired keyword.

## 10. Frontend / backend split

Backend: one filter used by CLI and HTTP.

App: when the search field is non-empty, `getSkills` includes `q` (and `grouped: true` as today). Do not flash the unfiltered catalog after that request is in flight (UX-20260822-01 loading). When the field is cleared, fetch without `q`.

Client provider/status filters remain subtractive on the response.

## 11. Refactor / module-boundary outcome

- Final owner: catalog list query.
- Direction: App → HTTP `q` → `ListSkills`. Not App → local text match.
- Deleted: subsequence inclusion; App text matcher; provider/category query fields.
- Boundary test: ListSkills and App request shape; no second matcher.

## 12. TDD scenarios mapped to upstream IDs

Write or invert tests first, then change production code.

| Test | Expect | Upstream |
|---|---|---|
| `pdl` vs skill tagged `pdl` and `karpathy-guidelines` | only the tagged skill | UC-001, 005 |
| `jira` vs `job-iteration-report-app` with no written `jira` | absent | UC-001 |
| `jbr` vs `jira-browser` | absent unless `jbr` is written | UC-006 |
| `auth` vs `browserauth` / `authentication` / tag `auth` | present | UC-005 |
| `product development` AND | both keywords required | UC-002 |
| `PDL` case | same as `pdl` | UC-001 |
| `zzzz-not-a-skill` | empty | UC-003 |
| query does not match provider name only | absent | UC-001, D4 |
| App: non-empty search calls `getSkills` with `q` | no local text includes | UC-004, ADR-001 |
| existing tests that require subsequence hits | invert or delete | UC §9 |

Chinese contiguous match (`报销` / `报销助手`) stays.

## 13. E2E and product-observable verification

Against a real or fixture catalog:

- CLI `-q PDL` does not list `coding`, `browser`, or `karpathy-guidelines` unless they write `pdl`.
- App search `PDL` shows the same names as CLI/API for the same filters.
- Empty unknown keyword → empty, no suggestions.

## 14. Documentation sync

- `SKILL.md`: replace the subsequence matching paragraph with contiguous written-field inclusion; keep search-first / digest-rare.
- `README.md` `-q` example: still valid; drop any subsequence claim.
- `docs/frontend-api-contract.md`: document `q` fields and AND.
- Historical `docs/PRD.md` §9.3.2: point at this package; do not copy the contract.

## 15. Exit criteria

- UC-001…006 observable on CLI, API, and App.
- ADR-001: no App text matcher; no subsequence inclusion.
- Superseded tests and functions gone.
- Docs in §14 match the shipped rule.

## 16. Intentional remaining boundaries

- `--tag` exact filter unchanged.
- Digest format unchanged.
- List pagination / `/v1` restyle not in this delivery.
- Body and provider name remain unsearched.
