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
- text-version counter

## Coordinates

- `Pos` is `(Row, GraphemeCol)`, both 0-based.
- `Range` is half-open: `[Start, End)`.
- Empty ranges are valid values; active selections treat empty as inactive.

Helpers:
- `ComparePos(a, b)`
- `NormalizeRange(r)`
- `ClampPos(p, rowCount, lineLen)`
- `ClampRange(r, rowCount, lineLen)`

## Conversion APIs

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
- `PosFromUTF16Offset(off, policy) (Pos, bool)`
- `UTF16OffsetFromPos(pos, policy) (int, bool)`
- `GapFromPos(pos, bias) (Gap, bool)`
- `PosFromGap(gap, policy) (Pos, bool)`
- `GraphemeColFromRuneOffsetInLine(line, runeOff, clamp) (int, bool)`
- `RuneOffsetFromGraphemeColInLine(line, graphemeCol, clamp) (int, bool)`

Behavior:
- Offsets are document-global over `Text()`.
- Newline separators between lines count as one byte, one rune, and one UTF-16 code unit (`\n`).
- `OffsetError` rejects out-of-range offsets and invalid positions.
- `OffsetClamp` clamps out-of-range offsets/positions to valid document bounds.
- In-range byte/rune/UTF-16 offsets that are not at grapheme boundaries are rejected.
- `PosFromGap` maps through gap rune offset conversion with the supplied policy.
- `UTF16Offset*` counts UTF-16 code units (supplementary runes count as `2`).
- Line-scoped rune/grapheme helpers apply the same clamp contract with no newline handling.
- Unicode fixture examples now covered by tests:
- `"a"`: offset `1` maps to `(Row:0, GraphemeCol:1)`.
- `"é"`: byte offset `1` is inside one grapheme and is rejected.
- `"e\u0301"`: rune offset `1` is inside one grapheme and is rejected.
- `"👨‍👩‍👧‍👦"`: interior byte/rune offsets are rejected; only start/end map.
- `"a\nb"`: newline boundary maps from offset `2` to `(Row:1, GraphemeCol:0)`.
- `"😀"`: UTF-16 offsets are `0` at BOF and `2` at EOF.

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

Remote apply (Phase 3 API surface):
- `ApplyRemote(edits []RemoteEdit, opts ApplyRemoteOptions) (ApplyRemoteResult, bool)` applies remote edits in call order.
- remote edit payload type: `RemoteEdit { Range, Text, OpID }` (`OpID` is metadata only).
- options:
- `ApplyRemoteOptions.BaseVersion` is compared to current `Version()`.
- `ApplyRemoteOptions.VersionMismatchMode` controls mismatch behavior:
- `VersionMismatchReject` (default): if `BaseVersion != Version()`, reject and return `changed=false`.
- `VersionMismatchForceApply`: apply anyway, even when base version mismatches.
- `ApplyRemoteOptions.ClampPolicy.ClampMode` controls range endpoint handling for each edit:
- `OffsetError`: reject the whole call if any endpoint is out of bounds.
- `OffsetClamp`: clamp endpoints into bounds before each edit.
- result:
- on effective mutation (`changed=true`): `Change.Source` is `ChangeSourceRemote`; `Remap` reports cursor/selection endpoint remaps.
- on no-op/reject/invalid options (`changed=false`): return zero-value `ApplyRemoteResult`.

Deterministic ordering and overlap:
- edits are interpreted against evolving state in explicit list order.
- overlap resolution follows edit order only (same list -> same final result).
- order changes output when ranges overlap.
- example:
- on `"abcdef"`, `[1,4)->"X"` then `[1,3)->"YZ"` yields `"aYZf"`.
- reversing order (`[1,3)->"YZ"` then `[1,4)->"X"`) yields `"aXef"`.

Remap statuses:
- `Remap` contains `Cursor`, `SelStart`, and `SelEnd` as `RemapPoint { Before, After, Status }`.
- status enum: `RemapUnchanged`, `RemapMoved`, `RemapClamped`, `RemapInvalidated`.
- insertion at an endpoint uses right-bias (`off >= insertPos` shifts by inserted rune length).

| Status | Meaning | Typical Trigger | Host Action Hint |
| --- | --- | --- | --- |
| `RemapUnchanged` | Endpoint position is unaffected by effective edits. | Edits happen strictly after the endpoint, or net offset delta is zero. | Keep cursor/selection endpoint where it is. |
| `RemapMoved` | Endpoint shifts because edits before it changed document length. | Insertion before endpoint, or replacement before endpoint with non-zero net delta. | Move endpoint to reported `After`. |
| `RemapClamped` | Endpoint falls inside a replaced/deleted range and snaps to replacement boundary. | Cursor or selection endpoint lies in removed range. | Use `After`; do not try to preserve old interior position. |
| `RemapInvalidated` | Selection endpoint pair collapsed after remap and selection was cleared. | `SelStart.After == SelEnd.After` after applying batch. | Clear selection UI state. |

Examples:
- cursor clamped: cursor at grapheme `2` in `"abc"`, apply remote delete `[0,3)->""` -> cursor becomes `0`, status `RemapClamped`.
- selection invalidated: selection `[1,3)` in `"abcd"`, apply delete `[0,4)->""` -> selection is cleared, both endpoints report `RemapInvalidated`.
- version mismatch policy:
- with `VersionMismatchReject`, mismatched `BaseVersion` returns `changed=false` and leaves state unchanged.
- with `VersionMismatchForceApply`, the same mismatched batch can still apply and produce remote change/remap output.

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

`TextVersion()` increments only on effective text mutations:
- `Insert*`, `Delete*`, `Apply`, `ApplyRemote`
- undo/redo when the restored text differs

`TextVersion()` does not change for cursor-only and selection-only mutations.

## Undo/Redo

- bounded by `Options.HistoryLimit` (default `1000`).
- one undo step per public text mutation call.
- undo/redo restore text, cursor, and selection (including selection direction).
- new text mutations clear redo stack.
