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
