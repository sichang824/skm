# Implementation handoff: catalog search inclusion

Status: current on 2026-08-22
Owner: @ann
Audience: executors and reviewers

---

pdl_id: pdl-20260822-01-catalog-search-precision
artifact_id: HANDOFF-20260822-01
revision: 1
status: authorized
authority_level: implementation-plan
canonical_path: docs/pdl/pdl-20260822-01-catalog-search-precision/07-implementation-handoff.md

Bound to: `todo.md` TODO-1 rev.1
product_contract_accepted_ref: UC-20260822-01 rev.1
todo_baseline_accepted_ref: TODO-1 rev.1 accepted by @ann on 2026-08-22
implementation_authorized_ref: user "开始实现" on 2026-08-22

## In scope

WP-01 catalog inclusion, WP-02 App `q`, WP-03 docs, WP-04 verification evidence.

## Forbidden

User-visible behavior not in UC-20260822-01. Subsequence inclusion. Typo repair. Second App matcher. Dual/shim path. WP-05 release.

## Required tests

SPEC-001 §12. Red then green. `make test` before claiming WP-04.

## Completion evidence

CLI `-q PDL` from `/tmp` lists only skills that wrote `pdl`. App search calls `getSkills` with `q`. Same set on CLI and API.
