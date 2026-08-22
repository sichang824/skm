# Grill Me: catalog search precision

Status: current on 2026-08-22
Owner: @ann
Audience: product

---

pdl_id: pdl-20260822-01-catalog-search-precision
artifact_id: GRILL-20260822-01
revision: 3
status: draft
authority_level: user-decision
accepted_ref: UC-20260822-01 rev.1 accepted by @ann on 2026-08-22
supersedes: none
canonical_path: docs/pdl/pdl-20260822-01-catalog-search-precision/01-grill-me.md

## Already accepted in user language

1. Searching for a skill should return the intended skill, not a large pile of unrelated names.
2. Finding a named skill is the primary job. Dumping the whole catalog (digest) is not the default and must not be used often, because it burns context.
3. Agent workflow: search first, `get` the hit, digest only as a rare overview.
4. The keyword the user typed must actually appear in the skill they see. A skill that does not contain that keyword must not be shown.
5. An abbreviation or alias matches only when the author wrote that text in a written field (`auth` in `authz` / `browserauth` / `authentication`, or tag `pdl`).
6. Showing skills that never contain the keyword is a defect, not a ranking or recall strategy. One keyword producing ~80 hits that do not contain that keyword is a bug.
7. App, CLI, and API use the same inclusion rules. Same query, same set.
8. The product does not repair typos. Agents rewrite the intended keyword before they search.

## Round 1 — situation and counterexample

Situation: the user (or an agent acting for them) wants the Product Development Lifecycle skill. They type `PDL`, the alias printed on that skill.

Today:

- `-q PDL` keeps any skill whose name or summary contains the letters p, then d, then l, in order, with any gaps. About half the catalog stays. The right skill is first, but the rest are noise.
- `--tag pdl` keeps only `product-development-lifecycle`. That is the result the user expected from "search".
- Typing a longer fragment (`product-development`) is already much tighter (3 rows).

Counterexample the current CLI still treats as success: `skm skills -q jira` includes `job-iteration-report-app` because j-i-r-a appear in order. A test locks this in.

Provisional statements — Round 1, confirmed in Round 2:

- P1. Inclusion is the product. A skill that does not contain the keyword must not appear, even last.
- P2. `PDL` may select a skill only if that skill writes `pdl` (name, tag, alias, or other accepted written field).
- P3. Digest is an inventory printout, not lookup.
- P4. This class of miss is a bug: the filter accepted "letters exist in order somewhere" as "the keyword appeared".
- P5. App, CLI, and API share one inclusion rule.

## Round 2 — user correction (2026-08-22)

The user rejected the "weak strategy / tune ranking" frame.

Restated: they give a keyword. That keyword will be present in one or a few skills. Skills that do not contain it must not be listed. `Auth` is allowed when it is a typo, an abbreviation, or an alias the author wrote. `PDL` listing `karpathy-guidelines` / `coding` / `browser` is not a recall tradeoff — those skills do not contain `pdl`. Because this search is keyword-and-rule based, not semantic, that outcome is a bug.

Closed:

- D1 → only skills that contain the keyword (as written, or as typo/abbrev/declared alias of written text). No hidden "maybe" pile of skills that lack the keyword.
- D2 → typo / abbreviation / declared alias of written text is in. Phantom letter-order matches across a field that never wrote the keyword are out.
- D3 → App, CLI, and API: one inclusion rule.

## Round 3 — written fields and typos (2026-08-22)

Closed:

- D4 → written fields are name, slug, directory name, tags, summary/description, and aliases the skill declares. Provider name and body prose are out.
- D5 → contiguous piece of a written field only. No typo repair. Large-language-model callers fix input before they search.

Draft User Contract: `02-user-contract.md` (in review). Open product decisions: none.

## Open product decisions (blocking)

None. Waiting for explicit User Contract acceptance.

## Not asked (engineering, deferred)

Storage engine, FTS, score formula internals, whether scores appear in JSON, default page size numbers. Those wait for the contract.

## Later rounds (not started)

Defaults and control (always-on filters, "show more"), empty and huge-result states, App vs CLI copy, what happens when two skills share an alias, conflict copies in results.

## Change history

| Date | Change |
|---|---|
| 2026-08-22 | Round 1 opened from user report: `-q PDL` too noisy; search outranks digest. |
| 2026-08-22 | Round 2: inclusion is "keyword appears in the skill"; phantom hits are a bug; App/CLI/API share one rule. D4–D5 opened. |
| 2026-08-22 | Round 3: D4/D5 closed — written fields only; no typo repair. Contract drafted. |
