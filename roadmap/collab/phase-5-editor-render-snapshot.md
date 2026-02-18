# Phase 5: Editor Render Snapshot API

Scope: Expose frame-stable render mapping snapshots in `editor` and require snapshot-token-based mapping calls for host cache safety.

Documentation: `design/collab-editing-best-practices.md`, `research/collab.md`, `docs/editor.md`

Legend: [ ] todo, [x] done.

## Snapshot API Surface
- [x] Add immutable snapshot types and APIs — provide host-safe mapping state per rendered frame
  - Repository: `flourish`
  - Component: `editor`
  - Scope: add `SnapshotToken`, `RowMap`, `RenderSnapshot`, `RenderSnapshot()`, `ScreenToDocWithSnapshot(...)`, `DocToScreenWithSnapshot(...)`
  - Snippets: `func (m Model) RenderSnapshot() RenderSnapshot`
  - Tests: compile-level API tests and non-empty snapshot coverage

## Snapshot Construction
- [x] Build snapshots from existing layout cache without host-visible mutation — keep frame mapping deterministic
  - Repository: `flourish`
  - Component: `editor` rendering/layout
  - Scope: capture viewport, row mapping, and visible-doc-col mapping into immutable snapshot values
  - Snippets: snapshot builder during/after `rebuildContent()`
  - Tests: same frame yields stable snapshot contents and token

- [x] Add token invalidation rules for relevant changes — prevent stale-cache reads
  - Repository: `flourish`
  - Component: `editor`
  - Scope: increment/rotate snapshot token on buffer version changes, viewport changes, wrap mode changes, virtual text/highlighter-affecting changes
  - Snippets: `nextSnapshotToken()` call sites in render invalidation path
  - Tests: token-change matrix test for each invalidating event

## Mapping Behavior
- [x] Implement snapshot-bound mapping functions with stale token rejection — force explicit cache lifecycle in hosts
  - Repository: `flourish`
  - Component: `editor`
  - Scope: `ScreenToDocWithSnapshot` and `DocToScreenWithSnapshot` must validate snapshot/token and mapping bounds
  - Snippets: early reject on token mismatch
  - Tests: stale snapshot tests return `ok=false`; fresh snapshots map identically to current frame

## Quality Gates
- [x] Extend viewport/hit-test tests for wrap and decorated lines under snapshot APIs — ensure parity with existing behavior
  - Repository: `flourish`
  - Component: `editor` tests
  - Scope: wrap-none and wrapped modes, line numbers, virtual text, and offscreen positions
  - Snippets: reuse `viewport_state_test` fixtures for snapshot variants
  - Tests: `go test ./editor` with snapshot parity assertions passing

- [x] Update `docs/editor.md` with snapshot lifecycle and cache contract — lock host usage pattern
  - Repository: `flourish`
  - Component: docs
  - Scope: add token, invalidation triggers, and recommended host cache keying
  - Snippets: host pseudocode caching snapshot by `Token`
  - Tests: doc review against implementation and tests
