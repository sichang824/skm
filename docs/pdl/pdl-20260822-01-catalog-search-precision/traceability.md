# Traceability: catalog search inclusion

Status: current on 2026-08-22
Owner: @ann
Audience: product and engineering

---

pdl_id: pdl-20260822-01-catalog-search-precision
artifact_id: TRACE-20260822-01
revision: 2
status: draft
authority_level: implementation-plan
canonical_path: docs/pdl/pdl-20260822-01-catalog-search-precision/traceability.md

| UC ID | PRD IDs | UX IDs | ADRs | Spec | Task IDs | Tests | Runtime/user evidence | Status |
|---|---|---|---|---|---|---|---|---|
| UC-20260822-01-001 | PRD-001, 006 | UX-001, 003 | ADR-001 | SPEC-001 | WP-01, WP-02 | skill_query_test ListSkills written hits | `/tmp` `-q PDL` → 1 skill | evidenced |
| UC-20260822-01-002 | PRD-002 | UX-001 | ADR-001 | SPEC-001 | WP-01/S-A | TestListSkillsQueryRequiresEveryKeyword | pending live multi-word | evidenced (unit) |
| UC-20260822-01-003 | PRD-003 | UX-002 | ADR-001 | SPEC-001 | WP-01/S-A | unknown empty unit | `-q zzzz-not-a-skill` → 0 | evidenced |
| UC-20260822-01-004 | PRD-004 | UX-001 | ADR-001 | SPEC-001 | WP-02, WP-04 | SkillsPage.spec sends `q` | CLI uses same ListSkills as API | evidenced (shape) |
| UC-20260822-01-005 | PRD-001, 005 | UX-001 | ADR-001 | SPEC-001 | WP-01/S-A | tag/abbrev unit | `-q PDL` hits tagged skill | evidenced |
| UC-20260822-01-006 | PRD-003 | UX-002 | ADR-001 | SPEC-001 | WP-01/S-A | letter-order rejected | `jira` does not list `job-iteration-report-app` | evidenced |
