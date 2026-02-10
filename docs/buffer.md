# `buffer` package — current state

Source: `buffer/`

Design targets:
- `design/spec.md` (architecture layering)
- `design/api.md` (public API draft)

Roadmap:
- Phase 1: `roadmap/phase-1-buffer-foundation.md`
- Phase 2: `roadmap/phase-2-buffer-movement-selection.md`

## Coordinates and ranges

- `Pos` is `(Row, Col)` in **runes**, 0-based.
- `Range` is a **half-open** interval in document coordinates: `[Start, End)`.

Helpers:
- `ComparePos(a, b)` orders positions in document order.
- `NormalizeRange(r)` ensures `Start <= End`.
- `ClampPos(p, rowCount, lineLen)` clamps to `0 <= Row < rowCount` and `0 <= Col <= lineLen(Row)`.
- `ClampRange(r, rowCount, lineLen)` clamps both endpoints.

## Buffer state

`Buffer` stores:
- text as logical lines split on `\n` (each line is `[]rune`)
- `Cursor` position (clamped)
- optional `Selection` (normalized; empty selection is treated as inactive)
- `Version` counter

## Versioning

- `Version()` starts at 0.
- `SetCursor` increments version only when the clamped cursor position changes.
- `SetSelection` increments version only when the effective selection changes (including clearing an existing selection).
- `ClearSelection` increments version only when it clears a non-empty active selection.
- `Move` increments version only when it changes cursor and/or selection.

## Movement + selection

`Buffer.Move(Move)` updates cursor and selection using rune-accurate document coordinates.

Types (from `design/api.md`):
- `MoveUnit`: `MoveRune`, `MoveWord`, `MoveLine`, `MoveDoc`
- `MoveDir`: `DirLeft`, `DirRight`, `DirUp`, `DirDown`, `DirHome`, `DirEnd`
- `Move`: `{Unit, Dir, Extend}`

Rules:
- `Extend=false` clears selection.
- `Extend=true` keeps a stable selection anchor across repeated extend moves until the selection is cleared.
- Word movement uses portable v0 semantics: skip whitespace, then skip non-whitespace (single-line; newline is a hard boundary).
