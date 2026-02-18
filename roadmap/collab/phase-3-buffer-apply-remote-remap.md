# Phase 3: Buffer ApplyRemote and Remap

Scope: Implement deterministic remote edit application in `buffer` with explicit cursor/selection remap reporting and causality-aware options.

Documentation: `design/collab-editing-best-practices.md`, `research/collab.md`, `docs/buffer.md`

Legend: [ ] todo, [x] done.

## Remote API Surface
- [x] Introduce remote apply types and API — define stable contract for host/remote mutation flows
  - Repository: `flourish`
  - Component: `buffer`
  - Scope: add `RemoteEdit`, `ApplyRemoteOptions`, `ApplyRemoteResult`, `RemapReport`, `RemapPoint`, `RemapStatus`
  - Snippets: `func (b *Buffer) ApplyRemote(edits []RemoteEdit, opts ApplyRemoteOptions) (ApplyRemoteResult, bool)`
  - Tests: compile-level API exposure and status enum coverage — all statuses reachable in tests

## Deterministic Remap Algorithm
- [x] Implement ordered edit application semantics for remote batches — remove ambiguity for overlaps and sequencing
  - Repository: `flourish`
  - Component: `buffer`
  - Scope: apply edits in call order against evolving state, with normalization and deterministic final state
  - Snippets: loop applying each normalized edit with intermediate remap updates
  - Tests: overlap matrix tests (before/inside/after overlap) — deterministic final text and cursor/selection
  - Essence (simple): remote ranges are normalized with explicit clamp/error policy, then applied one-by-one against the latest text state.
  - Example: on `"abcdef"`, applying `[1,4)->"X"` then `[1,3)->"YZ"` deterministically yields `"aYZf"`; reversing order yields a different but still deterministic result.

- [x] Implement cursor and selection endpoint remap status calculation — make movement/clamping/invalidations explicit
  - Repository: `flourish`
  - Component: `buffer`
  - Scope: remap cursor, selection start, selection end with statuses `unchanged|moved|clamped|invalidated`
  - Snippets: remap helper returning `{Before, After, Status}`
  - Tests: focused remap tests for each status class — expected status and coordinates match
  - Essence (simple): each endpoint is remapped through every effective edit and reports whether it stayed, shifted, snapped, or became invalid.
  - Example: if cursor is inside deleted text, status is `clamped`; if selection endpoints collapse to one point, both become `invalidated` and selection is cleared.

- [x] Add causality/version handling policy in options — support host protocol bridging without hidden behavior
  - Repository: `flourish`
  - Component: `buffer`
  - Scope: enforce explicit behavior when `BaseVersion` mismatches (`reject` or policy-defined fallback)
  - Snippets: version gate at start of `ApplyRemote`
  - Tests: mismatch tests for each configured behavior — deterministic acceptance/rejection
  - Essence (simple): version mismatch behavior is now explicit and configurable instead of implicit.
  - Example: with `VersionMismatchReject`, mismatched remote batches return `changed=false`; with `VersionMismatchForceApply`, the same batch applies.

## Quality Gates
- [ ] Add property/fuzz tests for random remote edit sequences — protect against edge-case drift
  - Repository: `flourish`
  - Component: `buffer` tests
  - Scope: random batched edit generation with invariants for valid positions and stable policy outcomes
  - Snippets: fuzz target around `ApplyRemote` + invariant assertions
  - Tests: fuzz pass with no panics and invariant violations

- [ ] Update `docs/buffer.md` with `ApplyRemote` semantics and remap table — ensure host integrations are deterministic by contract
  - Repository: `flourish`
  - Component: docs
  - Scope: document ordering, overlap semantics, remap statuses, and version mismatch handling
  - Snippets: examples showing remap status outcomes
  - Tests: doc review against tests and API definitions
