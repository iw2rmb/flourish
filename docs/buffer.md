# `buffer` package — current state

Source: `buffer/`

Design targets:
- `design/spec.md` (architecture layering)
- `design/api.md` (public API draft)

Roadmap:
- Phase 1: `roadmap/phase-1-buffer-foundation.md`

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
