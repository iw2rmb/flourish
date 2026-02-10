# `buffer` package — current state

Source: `buffer/`

Design targets:
- `design/spec.md` (architecture layering)
- `design/api.md` (public API draft)

Roadmap:
- Phase 1: `roadmap/phase-1-buffer-foundation.md`
- Phase 2: `roadmap/phase-2-buffer-movement-selection.md`
- Phase 3: `roadmap/phase-3-buffer-editing-apply.md`

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

## Editing

All editing operations are rune-accurate and follow selection-first semantics:
- If a selection is active, insertion replaces the selection (including inserting `""`, which deletes the selection).
- If a selection is active, backspace/delete delete the selection.
- Otherwise, backspace deletes the rune before the cursor (joining lines at SOL).
- Otherwise, delete deletes the rune at the cursor (joining lines at EOL).

Implemented:
- `InsertText(s string)` accepts `\n` and updates the cursor to the end of inserted text.
- `InsertRune(r rune)` inserts one rune.
- `InsertNewline()` inserts `\n`.
- `DeleteBackward()`, `DeleteForward()`, `DeleteSelection()`.

## Deterministic apply

`Apply(edits ...TextEdit)` applies edits sequentially, interpreting each edit’s range against the buffer state at the time the edit is applied.

Current semantics:
- Edit ranges are clamped into current document bounds.
- Empty range + non-empty text inserts.
- Cursor moves to the end of the last effective edit.
- Selection is cleared if any edit applies.
