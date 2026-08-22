# User Contract: catalog search inclusion

Status: current on 2026-08-22
Owner: @ann
Audience: product and engineering

---

pdl_id: pdl-20260822-01-catalog-search-precision
artifact_id: UC-20260822-01
Contract key: 20260822-01
revision: 1
Contract status: accepted
authority_level: user-contract
Accepted by: @ann
Accepted on: 2026-08-22
Supersedes: none
Upstream principles: keyword search; one catalog; search before digest
canonical_path: docs/pdl/pdl-20260822-01-catalog-search-precision/02-user-contract.md

## 1. Product intent

- Target user: a person or an agent looking up skills in SKM by keywords they believe were written into those skills.
- Problem: a keyword that does not appear in a skill is still listed. That is a defect. Ranking those extras last does not make them valid.
- Desired outcome: every listed skill contains every typed keyword in a field the skill itself wrote. Skills that do not contain a keyword are absent, not demoted.
- Product mental model: search answers "which skills wrote this text?", not "which skills could be stretched to look like this text?". Agents may rewrite a typo before they search. The product does not guess or repair typing.

## 2. Vocabulary

User-facing nouns and verbs:

- Keyword: one word the user or agent typed.
- Query: the full typed string; one or more keywords.
- Written field: text the skill authored and that search may read.
- Hit: a skill that contains every keyword of the query in a written field.
- Empty result: no skill contains every keyword.

Forbidden on Consumer surfaces: subsequence, token, rank, score, FTS, haystack, needle, Levenshtein. Do not tell the user a miss was "low relevance".

## 3. Entry points and scope

The user meets this capability at:

- the App skill list search box;
- `skm skills -q` / `--query`;
- `GET /api/skills?q=`.

It belongs to the skill catalog, not to a single provider. The same query must yield the same set of skills on App, CLI, and API. Filters already chosen (provider, category, tag, status, conflict) still apply after inclusion.

Roles: human and agent. Both use the same inclusion rule.

## 4. Defaults and control hierarchy

- Root control: the query. With no query, the full filtered catalog is listed (unchanged).
- A skill is a hit only when every keyword appears, case-insensitively, as a contiguous piece of at least one written field.
- Written fields: name, slug, directory name, tags, summary/description, and any alias the skill declares (today: tags, and alias words the author wrote in those fields).
- Not written fields for this contract: provider name, body prose below frontmatter, and category unless the skill itself wrote that category value.
- Several keywords: all must hit (AND).
- Typo repair is off. The product does not insert, drop, or reorder letters to force a match. An agent that knows the intended keyword types the repaired form.
- No "maybe" or "wider match" group of skills that failed inclusion.
- Filters do not add skills that failed inclusion. They only remove hits.

## 5. Primary journeys

1. The user types a keyword they expect to find in a skill (`PDL`, `auth`, `product-development`).
2. The product lists only hits, better matches first among hits.
3. Completion: every row is a skill that wrote that keyword; the user can open one.

Agent path: search with the intended keyword, then open the hit. The agent does not dump the catalog to compensate for a noisy query.

## 6. State and action matrix

| UC ID | State | What the user sees | What the system does | User actions | Must not happen |
|---|---|---|---|---|---|
| UC-20260822-01-001 | Query entered | Hits only | Keeps a skill iff every keyword is a contiguous piece of a written field | Refine query; open a hit; apply filters | A skill that did not write a keyword appears, even last |
| UC-20260822-01-002 | Several keywords | Hits that contain all of them | AND across keywords; each keyword is tested on written fields | Add or remove words | One matching word pulls in a skill that lacks the other |
| UC-20260822-01-003 | Empty | An empty list and a clear "no skill contains this" meaning | Returns zero skills | Change keywords; clear query | Invented or letter-skipped fills |
| UC-20260822-01-004 | Same query, three surfaces | The same skill set in App, CLI, and API | One inclusion rule | Use any surface | App substring-only vs CLI letter-skipping vs API drift |
| UC-20260822-01-005 | Alias / abbreviation written by the author | The skill that wrote it | `pdl` hits a skill whose tag or name/summary contains `pdl`; `auth` hits `authz` / `browserauth` / `authentication` | Type the written form or a contiguous piece of it | `pdl` hits a skill that never wrote `pdl` |
| UC-20260822-01-006 | Typo not written in any skill | Empty (or only skills that still contain that exact misspelling) | No repair | Agent or user retypes the intended keyword | Guessing `autth` → `auth` |

## 7. Failure and recovery

- Recoverable: empty result. The user or agent types a keyword that actually appears.
- Not recoverable by the product: a keyword the catalog does not contain. The product must not widen the match.
- No automatic retry or silent second-pass fuzzy search.
- No notification beyond the empty or hit list.
- Filters that leave zero hits show empty, not an unfiltered fallback.

## 8. Data, privacy, safety, cost, and destructive behavior

Search reads catalog metadata already stored for listing. It does not write skills, does not send queries off-machine beyond the user's SKM, and is not destructive. Cost: listing only hits; the product must not spend context or screen space on skills that failed inclusion.

## 9. Edge cases and lifecycle

- Concurrent searches: each query is independent.
- Conflict copies: a copy is a hit only if that copy's own written fields contain the keyword.
- Disabled / invalid skills: inclusion is unchanged; existing status filters still hide them when asked.
- After this repair, letter-order matches that are not contiguous text in a written field must disappear. They are not a compatibility path.

## 10. Non-goals and future boundaries

- Semantic or embedding search.
- Typo repair, spellcheck, or "did you mean".
- Searching `SKILL.md` body beyond summary/description.
- Matching against provider name.
- Digest as a lookup tool (digest remains an inventory printout).
- Changing `--tag` exact filter semantics beyond sharing the same catalog.

## 11. Acceptance examples

Happy: Given the catalog contains `product-development-lifecycle` with tag `pdl`, when the user types `PDL` on App, CLI, or API, then that skill is listed and `karpathy-guidelines`, `coding`, and `browser` are not.

Abbreviation written in the skill: Given a skill named `browserauth` or tagged `auth`, when the user types `auth`, then that skill is listed.

AND: Given a skill that writes both `product` and `development` in written fields and another that writes only `product`, when the user types `product development`, then only skills that contain both keywords are listed.

Empty: Given no skill writes `zzzz-not-a-skill`, when the user types that keyword, then the list is empty.

Control: Given the same keyword on App, CLI, and API, when no extra filters differ, then the three surfaces list the same skills.

Must not: Given `jira` and a skill named `job-iteration-report-app` that does not write `jira`, when the user types `jira`, then that skill is absent.

## 12. Open product decisions

None.

## 13. Change history

| Date | Changed UC IDs | Reason | Accepting user | Downstream reopened |
|---|---|---|---|---|
| 2026-08-22 | UC-20260822-01-001 … 006 | Draft from Grill rounds 1–3 | | |
| 2026-08-22 | UC-20260822-01-001 … 006 | Explicitly accepted | @ann | PRD and UX opened |
