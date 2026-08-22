# Product acceptance report: catalog search inclusion

Status: current on 2026-08-22
Owner: @ann
Audience: product and engineering

---

pdl_id: pdl-20260822-01-catalog-search-precision
artifact_id: ACCEPT-20260822-01
revision: 1
status: awaiting_acceptance
authority_level: evidence
canonical_path: docs/pdl/pdl-20260822-01-catalog-search-precision/08-product-acceptance-report.md

## 1. What the user can do now

Type a keyword on CLI, API, or App search. Only skills that wrote that keyword in name, slug, directory, tags, or summary appear. Skills that never wrote it are absent.

## 2. Verified primary journey

Given the live catalog (`~/.skm/app.db`, 2026-08-22), `skm skills -q PDL` listed only `product-development-lifecycle`. `karpathy-guidelines`, `coding`, and `browser` were absent. Same set as `--tag pdl`.

## 3. Verified control and default behavior

Empty unknown keyword `zzzz-not-a-skill` → `Total skills: 0`. App search sends `q` (SkillsPage.spec). Filters still subtract.

## 4. Verified failure, recovery, and human-action behavior

No second-pass widen. Recovery is retype a written keyword. `jira` listed `jira` and `browserauth` (the latter writes `jira` in its summary). `job-iteration-report-app` was not listed.

## 5. Data, privacy, safety, cost

Read-only catalog query. No new export or destructive path.

## 6. Traceability summary

See `traceability.md`. UC-001…006 have CLI/API or unit evidence. App same-set vs live API was verified by request shape tests, not a running desktop session.

## 7. Engineering verification summary

- `GOWORK=off go test ./... -count=1` in `backend/` — pass
- `pnpm test -- --run src/pages/SkillsPage.spec.tsx` — pass
- Full frontend `pnpm test -- --run` — 6 tests pass; pre-existing AddonsPage `requestAnimationFrame` unhandled error remains (not this delivery)
- Live CLI: `-q PDL` → 1 skill; `-q zzzz-not-a-skill` → 0

## 8. Removed or superseded paths

`isRuneSubsequence` / subsequence rank; provider name and category as query fields; App `includes` text matcher.

## 9. Structural ownership outcome

Catalog `ListSkills` owns inclusion. App → `GET /api/skills?q=`. No shim.

## 10. Intentional remaining scope

WP-05 release not authorized. Desktop App not exercised in this session. Independent review verdict: PASS.

## 11. Release-readiness status

Not released. Implementation complete for WP-01…03; observable CLI evidence recorded.

## 12. Product decision

Awaiting explicit user acceptance of this report.
