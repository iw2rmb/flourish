# Phase 1: Buffer Conversion APIs

Scope: Add canonical, deterministic position/offset conversion APIs in `flourish/buffer`, including byte/rune/pos and gap anchor conversions, with full Unicode and bounds policy coverage.

Documentation: `design/collab-editing-best-practices.md`, `research/collab.md`, `docs/buffer.md`

Legend: [ ] todo, [x] done.

## API Surface
- [x] Introduce conversion policy and conversion APIs in `buffer` — standardize all host coordinate conversions in one place
  - Repository: `flourish`
  - Component: `buffer`
  - Scope: add `ConvertPolicy`, `OffsetClampMode`, `NewlineMode`, `PosFromByteOffset`, `ByteOffsetFromPos`, `PosFromRuneOffset`, `RuneOffsetFromPos`, `GapFromPos`, `PosFromGap`
  - Snippets: `func (b *Buffer) PosFromByteOffset(off int, p ConvertPolicy) (Pos, bool)`
  - Tests: compile-level API exposure + behavior tests for each function signature — all new APIs callable from unit tests

## Conversion Engine
- [ ] Implement deterministic byte/rune walking across multi-line grapheme-backed content — ensure identical results for repeated calls
  - Repository: `flourish`
  - Component: `buffer`
  - Scope: implement internal helpers for newline accounting, line traversal, and clamp/error behavior; ensure conversion behavior is independent of call site
  - Snippets: internal helper pattern `offsetToPos(off, unit, policy)`
  - Tests: table-driven tests for in-range and out-of-range offsets — expected `Pos` and `ok` results match policy

- [ ] Implement gap conversion model bound to rune offsets and explicit bias — remove host-side ad-hoc anchor math
  - Repository: `flourish`
  - Component: `buffer`
  - Scope: define `Gap` and `GapBias`; implement forward and reverse mapping between `Pos` and insertion gap
  - Snippets: `func (b *Buffer) GapFromPos(pos Pos, bias GapBias) (Gap, bool)`
  - Tests: round-trip tests `Pos -> Gap -> Pos`; boundary tests at BOF/EOF and line breaks — stable results

## Quality Gates
- [ ] Add Unicode fixture coverage (ASCII, multibyte UTF-8, combining marks, ZWJ emoji, multiline boundaries) — lock correctness for collaborative contexts
  - Repository: `flourish`
  - Component: `buffer` tests
  - Scope: add dedicated conversion test file and reusable fixtures for text corpus
  - Snippets: fixture rows containing `"a"`, `"é"`, `"e\u0301"`, `"👨‍👩‍👧‍👦"`, `"\n"` boundaries
  - Tests: run `go test ./buffer` — all fixture cases pass with exact expected coordinates/offsets

- [ ] Update `docs/buffer.md` with conversion contracts and policy semantics — make host integration rules explicit
  - Repository: `flourish`
  - Component: docs
  - Scope: document round-trip rules, clamp behavior, newline treatment, and failure behavior
  - Snippets: conversion behavior examples for valid and out-of-range offsets
  - Tests: doc review against implemented APIs — no undocumented behavior remains
