# ADR-001: Single owner for catalog search inclusion

Status: current on 2026-08-22
Owner: @ann
Audience: product and engineering

---

pdl_id: pdl-20260822-01-catalog-search-precision
artifact_id: ADR-001
ADR: ADR-001
revision: 1
Decision status: accepted
authority_level: adr
Decision owners: @ann
Product sources: UC-20260822-01, PRD-20260822-01-004, UX-20260822-01-001
Supersedes: none
canonical_path: docs/pdl/pdl-20260822-01-catalog-search-precision/05-adrs/ADR-001-single-search-inclusion-owner.md

## 1. Context

Three surfaces list skills: App, CLI, and HTTP `GET /api/skills?q=`. CLI and API already call `CatalogService.ListSkills`. The App loads the full grouped list and applies a second, different inclusion rule in the page (`includes` on name/summary/provider/category/directory; no tags; whole query as one string).

The accepted contract requires the same hit set on all three surfaces. Two algorithms cannot stay.

## 2. Product and engineering constraints

- A skill is a hit only when every keyword is a contiguous, case-insensitive piece of a written field (UC-20260822-01-001, 002, 005).
- App, CLI, and API must return the same set (UC-20260822-01-004).
- Letter-order matching and typo repair are deleted, not wrapped (UC §9, §10).
- Existing `GET /api/skills` query shape stays. This delivery does not introduce `/v1`, pagination, or snake_case restyle.

## 3. Decision drivers

- One inclusion rule cannot drift.
- Frontend must not re-implement catalog membership.
- Grouped copies that are not hits must not reappear as related rows.

## 4. Considered alternatives

1. **Copy the Go rule into the App.** Rejected: two owners; the current bug is this split.
2. **Keep App client-side search; change only CLI/API.** Rejected: violates UC-20260822-01-004.
3. **CatalogService is the only inclusion owner; App sends `q`.** Accepted.

## 5. Decision

`CatalogService` (CLI and HTTP already share it) is the only place that decides whether a skill is a hit for `q`.

The App search field sends `q` on `GET /api/skills`. It must not test query text against skill fields locally. Provider/status chips may stay on the client only if they subtract from that hit set.

When `q` is present and `grouped=true`, grouping runs after inclusion. A related copy that is not a hit is absent, including as a nested child. A child hit whose parent is not a hit appears as its own row.

## 6. Ownership and contract boundaries

- Final owner: catalog list query (`ListSkills` / `q`).
- Allowed: CLI and HTTP handlers pass `q` through; App passes `q` through `getSkills`.
- Forbidden: a second inclusion implementation in the App, a compatibility letter-order path, or a client fallback that lists non-hits when the query is empty of hits.
- Public contract: existing `GET /api/skills?q=`; semantics of `q` become the accepted inclusion rule. Document fields in the Spec.
- Delete: `isRuneSubsequence` as an inclusion path; App `matchesSkillFilters` search branch that reads skill text.

## 7. Consequences

Positive: one set; tag aliases work in the App; phantom hits disappear.

Negative: `brauth` no longer finds `browserauth` unless that text is written. Accepted.

Risks: App grouping UX must use the post-inclusion list. Tests that require letter-order hits must be rewritten as negatives.

Migration: delete the old paths in this delivery. No shim, flag, or dual rank.

## 8. Product impact check

This ADR does not change User Contract, PRD, or UX behavior. It assigns ownership so those artifacts can be implemented without drift. If a surface needs a different set, reopen the User Contract.

## 9. Verification implications

- Service tests: `pdl` does not include skills that did not write `pdl`; `jira` does not include `job-iteration-report-app`; AND; tags; case-insensitive contiguous match.
- Tests that require subsequence hits are removed or inverted.
- App tests: a non-empty search calls `getSkills` with `q` and does not keep a local text matcher.
- CLI and HTTP use the same `ListSkills` filters.

## 10. Related artifacts

- SPEC-001 (this package)
- `docs/frontend-api-contract.md` (stale until Spec documentation sync)
