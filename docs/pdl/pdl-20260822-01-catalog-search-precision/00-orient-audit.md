# Orient audit: catalog search precision

Status: current on 2026-08-22
Owner: @ann
Audience: product and engineering

---

pdl_id: pdl-20260822-01-catalog-search-precision
artifact_id: AUDIT-20260822-01
revision: 1
status: draft
authority_level: evidence
accepted_ref:
supersedes: none
canonical_path: docs/pdl/pdl-20260822-01-catalog-search-precision/00-orient-audit.md

## Existing artifacts and revisions

| Artifact | Path | Status | What it says about search |
|---|---|---|---|
| Historical PRD | `docs/PRD.md` v0.1, 2026-04-15, Draft | Not accepted | Goal is a searchable catalog. §9.3.2: search by skill name, plus provider / category / tag / status / conflict filters. No fuzzy ranks, no subsequence, no score floor, no limit. |
| SKILL.md (agent contract) | `SKILL.md` | Current operating text | Documents exact > prefix > substring > subsequence and field weights. Until this package, it also auto-injected `skm skills --digest` on skill load. |
| README | `README.md` | Current | Example: `skm skills -q jira`. |
| Frontend API contract | `docs/frontend-api-contract.md` | Current | `GET /api/skills?q=` exists; no ranking formula. |
| PDL for search | none before this package | — | Search shipped inside the 2026-08-19 exec commit (`7bcc227`), not as its own product cycle. |
| Digest | commit `5d681b0` | Shipped | Compact TSV for LLM context. Not a second search engine. Can combine with `-q`. |

No accepted User Contract, UX Behavior Contract, ADR, or Engineering Spec covers catalog search.

## Current user-visible behavior

Evidence from `/tmp` against `~/.skm/app.db` on 2026-08-22. Catalog size: 153 skills.

### CLI / API (`skm skills -q`, `GET /api/skills?q=`)

Implementation: `backend/internal/service/catalog_service.go` (`filterSkillsByQuery`, `skillQueryScore`, `fuzzyMatchRank`, `isRuneSubsequence`). Tests in `backend/internal/service/skill_query_test.go`.

1. SQL applies `--provider` / `--category` / `--status` / `--conflict` and `--sort`.
2. Query tokens are whitespace-split, lowercased, AND-combined.
3. Each token scores `rank × field weight`. Any token with score 0 drops the skill. Score > 0 keeps it. No limit.
4. Ranks: exact 4, prefix 3, substring 2, subsequence 1.
5. Weights: name 8, each tag 6, slug / directoryName 5, category / provider name 4, summary 2.
6. Subsequence is ordered runes with unlimited gaps and no compactness penalty. `brauth` matches `browserauth`. `jira` matches `job-iteration-report-app` (test keeps this as a hit, last place).
7. Body markdown is not searched.
8. Scores are not shown in CLI or JSON.

Observed:

| Query | Result |
|---|---|
| `-q PDL` | 82 skills (~54% of catalog). First row is `product-development-lifecycle` (tag exact `pdl`). Next rows include `karpathy-guidelines`, `platform-report-download`, `shipping-and-launch`, `spec-driven-development`, `browser`, `coding`. |
| `--tag pdl` | 1 skill: Workspace `product-development-lifecycle`. |
| `-q product-development` | 3 skills. Intended skill first. |
| `-q "product lifecycle"` | 26 skills. Still loose because `product` / `lifecycle` are common subsequences. |
| `-q "skm CLI"` | 72 skills. Token `skm` plus token `cli` still subsequence-match most English summaries. |

`--tag` is exact, case-insensitive. It is the only precise alias path today. There is no `--limit` on skill list (only on `skills execs`).

### App UI

`frontend/src/pages/SkillsPage.tsx` loads the full list and filters client-side with `String.includes` on name / summary / provider / category / directoryName. It does not send `q`, does not search tags, and does not use subsequence. Typing `PDL` in the App is stricter than the CLI: a hit needs a contiguous `pdl` in those fields.

Human and agent therefore see two different products named "search".

### Agent digest path

`skm skills --digest` is a display fold over the same list (or the `-q` subset). It does not rank. On this catalog it is ~150 TSV lines if unfiltered, or ~72 lines for `-q PDL`. The skm skill previously injected the full digest on every load, which occupies a large share of the agent context before any user task starts.

## Contradictions

1. Historical PRD: search by name. Shipped CLI: four-level fuzzy including subsequence across name, tags, slug, directory, category, provider, and summary.
2. Tests treat `jira` → `job-iteration-report-app` as a required hit. The user's PDL example treats that class of hit as failure.
3. App substring search vs CLI subsequence search.
4. Agent skill treated digest as the default discovery fact; the user now treats targeted search as the default and digest as rare.
5. Short aliases work when stored as tags (`--tag pdl`), but `-q` on the same alias floods the list. The product does not tell the user these are different tools.

## Missing gates

- No User Contract for "I typed X, what must appear, what must not".
- No accepted ranking or empty/too-many result behavior.
- No decision on whether typo-style subsequence is in or out.
- No decision that CLI, API, and App must share one ranking.
- No score visibility, relevance floor, or default cap.
- Digest vs search roles were never contracted.

## Stale candidates

- `docs/PRD.md` §9.3.2 and §14.3 once a new contract exists.
- `SKILL.md` search paragraph (algorithm description) after the contract changes inclusion rules.
- `skill_query_test.go` cases that require subsequence false positives, if the contract drops them.

## Preserved upstream decisions

Keep unless the user reopens them:

- Catalog is searchable. Filters by provider, category, tag, status, and conflict remain.
- Multi-word queries are AND.
- Case-insensitive matching, including Chinese runes.
- Field priority name > tags > slug/directory > category/provider > summary (weights may be retuned; the order is the current shipped intent).
- `skills get` / `skills exec` remain the way to read or run a skill after it has been identified.
- Digest remains a compact listing format, not a second catalog.

## Recommended mode and next gate

Mode: product repair. Search already exists and is wrong for the user's job: find one intended skill without wading through half the catalog.

Next gate: Grill Me on three product questions (recorded in `01-grill-me.md`). Do not change ranking code until those answers are accepted into a User Contract.

Agent-facing discovery rules (search first, do not inject digest) are a separate operating-doc change requested in this same conversation and do not wait on the ranking contract.
