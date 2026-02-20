# Phase 6: Core Conversion Helpers (UTF-16 and Rune/Grapheme)

Scope: Add conversion helpers in `buffer` for UTF-16 and rune↔grapheme mapping as core APIs independent of completion features.

Documentation: `design/completion.md`, `docs/buffer.md`, `research/collab.md`

Legend: [ ] todo, [x] done.

## UTF-16 Conversions
- [x] Add UTF-16 offset conversion APIs at `buffer` root surface — support protocol/tooling integrations that use UTF-16 code units
  - Repository: `flourish`
  - Component: `buffer`
  - Scope: add exported helpers for `Pos <-> UTF16Offset` with policy-driven clamp/error behavior
  - Snippets: `func (b *Buffer) PosFromUTF16Offset(off int, p ConvertPolicy) (Pos, bool)`
  - Tests: round-trip tests with surrogate pairs and multiline boundaries

- [x] Enforce grapheme-boundary safety for UTF-16 conversions — keep behavior aligned with existing byte/rune contracts
  - Repository: `flourish`
  - Component: `buffer`
  - Scope: reject interior grapheme offsets in `OffsetError`; clamp deterministically in `OffsetClamp`
  - Snippets: shared conversion walker over grapheme clusters + unit width function
  - Tests: rejection tests for offsets inside ZWJ and combining-mark clusters

## Rune/Grapheme Helpers
- [x] Add line-scoped rune↔grapheme helpers — remove host-side ad-hoc mapping logic
  - Repository: `flourish`
  - Component: `buffer`
  - Scope: add helpers for rune-offset to grapheme-col and reverse with explicit clamp mode
  - Snippets: `GraphemeColFromRuneOffsetInLine(line string, runeOff int, clamp OffsetClampMode)`
  - Tests: boundary tests for ASCII, multibyte, combining marks, and family emoji

## Quality Gates
- [x] Update `docs/buffer.md` with UTF-16 and rune/grapheme contracts — keep conversion policy centralized and explicit
  - Repository: `flourish`
  - Component: docs
  - Scope: document semantics, examples, and failure modes for all conversion units
  - Snippets: examples showing surrogate pair counting as 2 UTF-16 code units
  - Tests: doc review against implemented helper behavior and test fixtures
