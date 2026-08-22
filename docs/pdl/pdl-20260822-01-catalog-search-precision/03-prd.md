# PRD: catalog search inclusion

Status: current on 2026-08-22
Owner: @ann
Audience: product and engineering

---

pdl_id: pdl-20260822-01-catalog-search-precision
artifact_id: PRD-20260822-01
Product key: 20260822-01
revision: 1
PRD status: accepted
authority_level: prd
Source User Contract: UC-20260822-01 rev.1 accepted 2026-08-22 by @ann
Accepted by: @ann
Accepted on: 2026-08-22
canonical_path: docs/pdl/pdl-20260822-01-catalog-search-precision/03-prd.md

## 1. Executive outcome

A catalog query lists only skills that wrote every typed keyword. App, CLI, and API show the same set. Letter-order hits that never wrote the keyword disappear.

## 2. Problem and opportunity

Today the CLI/API treat "letters appear in order somewhere" as a hit. `PDL` therefore lists about half the catalog, including skills that never wrote `pdl`. The App uses a different, contiguous rule and ignores tags. Users and agents cannot trust search as "find the skills that contain this text."

The opportunity is a single inclusion rule that matches the accepted mental model: search answers which skills wrote this text.

## 3. Target users and jobs

- Person in the App: type a keyword, see only matching skills, open one.
- Person or agent on the CLI: `skm skills -q` / `--tag` / filters, then `skills get`.
- Agent or other client on the API: `q=` returns the same set as the other surfaces.

Job: identify the skill that wrote this keyword. Not: browse a guessed neighborhood.

## 4. Product principles and constraints

- Inclusion over ranking. A non-hit must be absent.
- Written fields only: name, slug, directory name, tags, summary/description, declared aliases.
- Contiguous, case-insensitive. No typo repair. Agents rewrite the keyword.
- One rule on App, CLI, and API.
- Existing filters only subtract hits.
- Digest stays an inventory printout, not lookup.

## 5. Scope by release or priority

This delivery (P0):

- Repair inclusion on CLI, API, and App so they share one rule (PRD-20260822-01-001 … 006).
- Empty query still lists the filtered catalog (unchanged).
- Empty query result is empty, not a widened fallback.
- Historical `docs/PRD.md` §9.3.2 is stale for this behavior; this package is canonical.

Out of this delivery: semantic search, typo repair, body full-text, provider-name match, digest redesign.

## 6. User stories mapped to UC IDs

| ID | Story | UC |
|---|---|---|
| PRD-20260822-01-001 | As a user I type `PDL` and see only skills that wrote `pdl`. | UC-20260822-01-001, UC-20260822-01-005 |
| PRD-20260822-01-002 | As a user I type several keywords and see only skills that wrote all of them. | UC-20260822-01-002 |
| PRD-20260822-01-003 | As a user I type a keyword no skill wrote and see an empty list I can refine. | UC-20260822-01-003, UC-20260822-01-006 |
| PRD-20260822-01-004 | As a user I use App, CLI, or API with the same query and filters and get the same skills. | UC-20260822-01-004 |
| PRD-20260822-01-005 | As an author I declare an alias or write an abbreviation in a written field and that text is findable. | UC-20260822-01-005 |
| PRD-20260822-01-006 | As a user I apply provider/status/tag/conflict filters and never gain skills that failed inclusion. | UC-20260822-01-001 |

## 7. Functional capabilities

- Evaluate each keyword against written fields as a contiguous, case-insensitive piece.
- Require every keyword (AND).
- Rank among hits only (name-quality before weaker written-field hits is allowed; it must not resurrect non-hits).
- Expose the same inclusion through App search, CLI `-q`, and API `q`.
- Keep `--tag` as an exact tag filter; it does not replace query inclusion.

## 8. Success metrics and signals

- `PDL` / `pdl` on the current catalog does not list skills that lack a written `pdl`.
- `jira` does not list `job-iteration-report-app` unless that skill writes `jira`.
- The same query+filters return the same skill set from App, CLI, and API.
- Empty unknown keyword → zero rows, no second-pass widen.

## 9. Risks and dependencies

- Agents or docs that relied on letter-skipping (`brauth` → `browserauth` when `brauth` is not written) will get empty. Accepted: they type a written fragment or an LLM-repaired keyword.
- App today misses tag-only aliases; unifying on written fields will add tag hits in the App. Accepted: that is the skill authoring the alias.
- Tests that require letter-order false positives must change with the repair.

## 10. Non-goals

Same as the User Contract §10. No new user-visible behavior beyond that contract.

## 11. Product acceptance criteria mapped to UC IDs

- Given tag `pdl` on `product-development-lifecycle`, when `PDL` is typed on App, CLI, and API, then that skill is present and skills that did not write `pdl` are absent. (UC-20260822-01-001, 004, 005; PRD-20260822-01-001, 004)
- Given `product development`, then only skills that wrote both keywords are listed. (UC-20260822-01-002; PRD-20260822-01-002)
- Given a keyword no skill wrote, then the list is empty and is not auto-widened. (UC-20260822-01-003, 006; PRD-20260822-01-003)
- Given extra filters, then the list is a subset of hits, never a superset. (UC-20260822-01-001; PRD-20260822-01-006)

## 12. Open scope or priority decisions

None.

## 13. Change history

| Date | Change | Accepting user |
|---|---|---|
| 2026-08-22 | Drafted from UC-20260822-01 rev.1 accepted | |
| 2026-08-22 | Explicitly accepted | @ann |
