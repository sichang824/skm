# UX Behavior Contract: catalog search inclusion

Status: current on 2026-08-22
Owner: @ann
Audience: product and engineering

---

pdl_id: pdl-20260822-01-catalog-search-precision
artifact_id: UX-20260822-01
UX key: 20260822-01
revision: 1
UX status: accepted
authority_level: ux
Source User Contract: UC-20260822-01 rev.1 accepted 2026-08-22 by @ann
Source PRD: PRD-20260822-01 rev.1 accepted 2026-08-22 by @ann
Design evidence: N/A — this repair changes which rows belong in the existing list; it does not change navigation, layout, or control composition
Accepted by: @ann
canonical_path: docs/pdl/pdl-20260822-01-catalog-search-precision/04-ux-behavior-contract.md

## 1. User mental model and vocabulary

The search box and `-q` mean: show skills that contain this text. Empty means none of them do. The product does not offer a second "similar" pile.

Words the user sees: search, keyword, no matching skills. The user does not see rank, score, or fuzzy.

## 2. Information architecture and entry points

- App: the existing skill-list search field. Results replace the list in place.
- CLI: `skm skills -q`. The table is the result.
- API: clients render the same set; this contract does not invent a new screen.

No new page, tab, or mode.

## 3. Control hierarchy and progressive disclosure

- Root: query text. Clearing it restores the filtered catalog.
- Dependent: existing filters (provider, category, tag, status, conflict). They only hide hits.
- No control for "wider match", "include similar", or typo suggestions.
- Digest is not an App control and is not a search empty-state action.

## 4. Primary and alternate flows

Happy: type a keyword → list shows only hits → open a skill.

Alternate: add a second keyword → list shrinks to skills that contain both.

Empty: type a keyword no skill wrote → empty meaning → edit or clear the query.

Filter: after hits appear, apply a filter → fewer or equal rows; never new names that failed the query.

## 5. State model

| UX ID | Trigger | Visible state | Available actions | Persistence | Related UC/PRD |
|---|---|---|---|---|---|
| UX-20260822-01-001 | Query with at least one hit | Only those skills, same set as the other surfaces | Open; refine query; filter | Query stays until cleared | UC-20260822-01-001, 004; PRD-20260822-01-001, 004 |
| UX-20260822-01-002 | Query with no hit | Empty list; meaning is "no skill contains this" | Edit query; clear query | Query stays | UC-20260822-01-003, 006; PRD-20260822-01-003 |
| UX-20260822-01-003 | Query plus filters leave zero rows | Empty; meaning is "no remaining match", not "search failed" | Change filters; edit query | Query and filters stay | UC-20260822-01-001; PRD-20260822-01-006 |
| UX-20260822-01-004 | Query cleared | Filtered catalog without a query | Type again | Cleared | UC-20260822-01-001 |

## 6. Feedback and notification behavior

No toast, badge, or interrupt for a successful search or an empty search. The list is the feedback. Do not say "showing low relevance matches" or count phantom hits.

## 7. Failure, recovery, and user actions

Catalog unavailable is an existing load failure, not part of this repair. Search-empty is not a failure. Recovery is edit or clear. The product must not auto-retry with a looser query.

## 8. Loading, empty, partial, stale, and unavailable states

- Loading: existing list-loading behavior; do not flash the unfiltered catalog after a query has been sent.
- Empty: UX-20260822-01-002 / 003.
- Partial: not used. A query either has its full hit set or is still loading.
- Stale: switching surfaces or repeating the same query+filters must not show a different set.
- Unavailable: unchanged existing error.

## 9. Responsive and accessibility behavior

Unchanged list and search-field behavior. Empty state text must be readable by the same assistive path as the list. CLI empty is a zero-row table plus the existing total line at zero.

## 10. Consumer versus Pro/Support detail boundary

Consumers do not see why a row scored, which field matched, or that an old letter-order rule existed. Support/diagnostics may log field names; that text is not shown in the App list or default CLI table.

## 11. UX acceptance scenarios

- Type `PDL` in the App search field: only skills that wrote `pdl` appear; no second group. (UX-20260822-01-001; UC-20260822-01-001, 005)
- Same keyword in CLI and API: same names. (UX-20260822-01-001; UC-20260822-01-004)
- Type a keyword no skill wrote: empty, no suggestions. (UX-20260822-01-002; UC-20260822-01-003, 006)
- Clear the query: full filtered list returns. (UX-20260822-01-004)

## 12. Reviewable design evidence

N/A. No new hierarchy, navigation, or interaction composition.

## 13. Open UX decisions

None.

## 14. Change history

| Date | Change | Accepting user |
|---|---|---|
| 2026-08-22 | Drafted from accepted UC-20260822-01 | |
| 2026-08-22 | Explicitly accepted | @ann |
