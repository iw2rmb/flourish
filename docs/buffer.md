# Package `buffer`

The `buffer` package implements a pure document model for text editing.
It has no Bubble Tea, rendering, or terminal dependencies.

## Overview

The package stores text as logical lines split by `\n`.
Each line is stored as grapheme clusters.
All columns are grapheme indices.

Core state:
- text
- cursor position
- selection
- undo/redo history
- version counter

## Coordinates

- `Pos` is `(Row, GraphemeCol)`, both 0-based.
- `Range` is half-open: `[Start, End)`.
- Empty ranges are valid values; active selections treat empty as inactive.

Helpers:
- `ComparePos(a, b)`
- `NormalizeRange(r)`
- `ClampPos(p, rowCount, lineLen)`
- `ClampRange(r, rowCount, lineLen)`

## Conversion APIs (Baseline)

`buffer` now exposes canonical coordinate conversions with explicit policy.

Types:
- `OffsetClampMode`: `OffsetError`, `OffsetClamp`
- `NewlineMode`: `NewlineAsSingleRune`
- `ConvertPolicy`: `{ ClampMode, NewlineMode }`
- `GapBias`: `GapBiasLeft`, `GapBiasRight`
- `Gap`: `{ RuneOffset, Bias }`

Methods:
- `PosFromByteOffset(off, policy) (Pos, bool)`
- `ByteOffsetFromPos(pos, policy) (int, bool)`
- `PosFromRuneOffset(off, policy) (Pos, bool)`
- `RuneOffsetFromPos(pos, policy) (int, bool)`
- `GapFromPos(pos, bias) (Gap, bool)`
- `PosFromGap(gap, policy) (Pos, bool)`

Behavior:
- Offsets are document-global over `Text()`.
- Newline separators between lines count as one byte and one rune (`\n`).
- `OffsetError` rejects out-of-range offsets and invalid positions.
- `OffsetClamp` clamps out-of-range offsets/positions to valid document bounds.
- In-range byte/rune offsets that are not at grapheme boundaries are rejected.
- `PosFromGap` maps through gap rune offset conversion with the supplied policy.
- Unicode fixture examples now covered by tests:
- `"a"`: offset `1` maps to `(Row:0, GraphemeCol:1)`.
- `"é"`: byte offset `1` is inside one grapheme and is rejected.
- `"e\u0301"`: rune offset `1` is inside one grapheme and is rejected.
- `"👨‍👩‍👧‍👦"`: interior byte/rune offsets are rejected; only start/end map.
- `"a\nb"`: newline boundary maps from offset `2` to `(Row:1, GraphemeCol:0)`.

## Editing Semantics

Insertion:
- `InsertText` inserts text at cursor or replaces active selection.
- `InsertGrapheme` inserts one grapheme cluster.
- `InsertNewline` inserts `\n`.

Deletion:
- with active selection: `DeleteBackward` and `DeleteForward` delete selection.
- without selection:
- `DeleteBackward` removes one grapheme cluster before cursor, joining lines at SOL.
- `DeleteForward` removes one grapheme cluster at cursor, joining lines at EOL.

Apply:
- `Apply(edits ...TextEdit)` applies edits in order.
- each edit range is interpreted against the current buffer state at apply time.
- cursor moves to the end of the last effective edit.

## Change Model

`buffer` now emits structured mutation payloads via:
- `LastChange() (Change, bool)`

Types:
- `ChangeSource`: `ChangeSourceLocal`, `ChangeSourceRemote`
- `SelectionState`: `{ Active, Range }`
- `AppliedEdit`: `{ RangeBefore, RangeAfter, InsertText, DeletedText }`
- `Change`: version/cursor/selection before/after plus ordered `AppliedEdits`

Rules:
- only effective mutations create a new `Change`.
- no-op calls do not increment version and do not replace the previous change.
- text mutation calls (`Insert*`, `Delete*`, `Apply`) populate `AppliedEdits` in apply order.
- cursor/selection-only state changes emit a `Change` with empty `AppliedEdits`.
- `Undo`/`Redo` emit one deterministic replacement `AppliedEdit` representing the text transition.

Example:
- inserting `"X"` at `(0,1)` reports one `AppliedEdit`:
- `RangeBefore=[(0,1)->(0,1))`
- `RangeAfter=[(0,1)->(0,2))`
- `InsertText="X"`
- `DeletedText=""`

## Movement and Selection

- `Move(Move)` supports grapheme, word, line, and document movement.
- `Extend=true` keeps a stable anchor and updates selection end.
- word movement is single-line and treats newline as a hard boundary.

## Versioning

`Version()` increments only on effective state changes:
- cursor changes
- selection changes
- text mutations
- successful undo/redo

No-op operations do not increment version.

## Undo/Redo

- bounded by `Options.HistoryLimit` (default `1000`).
- one undo step per public text mutation call.
- undo/redo restore text, cursor, and selection (including selection direction).
- new text mutations clear redo stack.
