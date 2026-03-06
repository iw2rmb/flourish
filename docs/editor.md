# Package `editor`

The `editor` package provides a Bubble Tea text editor component built on `buffer`.
It handles input, viewport management, and rendering.

## Overview

`Model` is the main component.
It owns an internal `*buffer.Buffer` and renders it with Bubble Tea viewport semantics.

Primary API:
- `New(Config) Model`
- `SetSize(width, height)`
- `Focus()`, `Blur()`, `Focused()`
- `InvalidateGutter()`
- `InvalidateGutterRows(rows ...int)`
- `InvalidateStyles()`
- `Update(msg tea.Msg)`
- `View() tea.View`
- `Buffer()`
- `ViewportState()`
- `ScreenToDoc(x, y)`
- `DocToScreen(pos)`
- `LinkAt(pos)`
- `LinkAtScreen(x, y)`
- `RenderSnapshot()`
- `ScreenToDocWithSnapshot(snapshot, x, y)`
- `DocToScreenWithSnapshot(snapshot, pos)`

## Coordinate Model

- document columns are grapheme indices.
- rendering and hit-testing use terminal cell coordinates.
- tabs expand by `TabWidth` (default `4`).

## Rendering and Layout

- optional custom gutter via `Config.Gutter` callbacks.
- editor-owned scrollbars are configured with `Config.Scrollbar`.
- wrap modes:
- `WrapNone`: no soft wrap; horizontal scroll (`xOffset`) keeps cursor visible.
- `WrapWord`: wraps at word boundaries; when no boundary fits, it falls back to width-based breaks.
- `WrapGrapheme`: wraps at grapheme boundaries.
- layout mapping preserves doc<->visual conversions, including wide glyphs.
- visual-token whitespace/grapheme metadata is computed during `BuildVisualLine` and reused by row rendering.
- cursor rendering keeps visibility at soft-wrap boundaries:
- on non-full wrapped rows, EOL cursor is rendered one cell after the last glyph.
- EOL cursor remains visible when a wrapped row exactly fills content width.
- trailing whitespace cursor cells are rendered with non-breaking spaces to avoid terminal elision.

## Input Behavior

Keyboard:
- Bubble Tea v2 key input is handled via `tea.KeyPressMsg`.
- `ReadOnly=true` blocks text mutation, keeps movement/selection enabled.

## Keyboard

Default keyboard shortcuts (can be overridden via `Config.KeyMap` and `Config.GhostAccept`):

| Context | Shortcut | Action |
| --- | --- | --- |
| Document | `left` | Move cursor left by one grapheme. |
| Document | `right` | Move cursor right by one grapheme. |
| Document | `up` | Move cursor up one row (preferred-column aware). |
| Document | `down` | Move cursor down one row (preferred-column aware). |
| Document | `alt+up` or `ctrl+up` | Move cursor to previous empty row (or document start when none). |
| Document | `alt+down` or `ctrl+down` | Move cursor to next empty row (or document end when none). |
| Document | `shift+left` | Extend selection left by one grapheme. |
| Document | `shift+right` | Extend selection right by one grapheme. |
| Document | `shift+up` | Extend selection to cursor position moved one row up (same cursor movement semantics as `up`, preferred-column aware). |
| Document | `shift+down` | Extend selection to cursor position moved one row down (same cursor movement semantics as `down`, preferred-column aware). |
| Document | `alt+left` or `ctrl+left` | Move cursor to previous word boundary (same row). |
| Document | `alt+right` or `ctrl+right` | Move cursor to next word boundary (same row). |
| Document | `alt+shift+left` | Extend selection to previous word boundary. |
| Document | `alt+shift+right` | Extend selection to next word boundary. |
| Document | `alt+shift+up` | Extend selection to previous empty row (or document start when none), using the same cursor movement semantics as paragraph-up movement. |
| Document | `alt+shift+down` | Extend selection to next empty row (or document end when none), using the same cursor movement semantics as paragraph-down movement. |
| Document | `pgup` | Move cursor up by current visible row count. |
| Document | `pgdown` | Move cursor down by current visible row count. |
| Document | `home` or `ctrl+a` | Move cursor to line start. |
| Document | `end` or `ctrl+e` | Move cursor to line end. |
| Document | `backspace` or `ctrl+h` | Delete backward (or delete active selection). |
| Document | `delete` | Delete forward (or delete active selection). |
| Document | `enter` | Insert newline. |
| Document | `tab` | Insert tab (`\t`). |
| Document | `space` | Insert a space. |
| Document | printable key text | Insert typed text (`alt`-modified text is ignored). |
| Document | `ctrl+z` | Undo. |
| Document | `ctrl+y` or `ctrl+shift+z` | Redo. |
| Ghost suggestion (visible) | `tab` | Accept ghost suggestion when `GhostAccept.AcceptTab=true`. |
| Ghost suggestion (visible) | `right` | Accept ghost suggestion when `GhostAccept.AcceptRight=true`. |

Terminal note:
- some terminals (including macOS Terminal defaults) reserve combinations like `shift+up/down` and other modified arrows for terminal-level selection/scrollback and may not forward them to Bubble Tea apps.
- run hosts in alt-screen (for demos/examples) or remap terminal shortcuts and/or editor `KeyMap` bindings when these keys are not delivered.
- WezTerm defaults bind `ctrl+shift+arrow` for pane navigation, which prevents delivery to TUI apps unless you disable/rebind those defaults (`wezterm show-keys --lua` helps inspect active mappings).
- WezTerm key protocol delivery depends on `enable_kitty_keyboard` (default is `false` in WezTerm docs).
- Zed terminal defaults bind `shift+up/down` to terminal scroll history; forward with `terminal::SendKeystroke` if you need those keys inside TUI apps.

WezTerm Lua example (`~/.wezterm.lua`):

```lua
local wezterm = require 'wezterm'

return {
  enable_kitty_keyboard = true,
}
```

Mouse:
- Bubble Tea v2 typed mouse messages are handled via `tea.MouseClickMsg`, `tea.MouseMotionMsg`, `tea.MouseReleaseMsg`, and `tea.MouseWheelMsg`.
- click to move cursor.
- shift+click to extend selection.
- drag to update selection.
- hit-testing maps from viewport-local `(x,y)` cells to document positions.
- wheel scroll is controlled by `ScrollPolicy`.
- left-click on scrollbar track pages by one visible span on the corresponding axis.
- left-button thumb drag updates vertical `TopVisualRow` or no-wrap horizontal `LeftCellOffset`.
- `ScrollFollowCursorOnly` blocks manual scrollbar interactions (press/drag) in addition to wheel scrolling.

Viewport integration:
- `ViewportState()` exposes top visual row, visible row count, wrap mode, and no-wrap horizontal offset.
- `ViewportState().VisibleRows` reports content-area rows (excludes reserved horizontal scrollbar row when horizontal scrollbar is visible).
- `ScreenToDoc` and `DocToScreen` provide stable host-side coordinate mapping.
- snapshot-bound mapping APIs (`RenderSnapshot`, `ScreenToDocWithSnapshot`, `DocToScreenWithSnapshot`) provide frame-stable mapping with stale-token rejection.
- snapshot token signatures include scrollbar config (`Vertical`, `Horizontal`, `MinThumb`) to invalidate stale host caches when scrollbar policy changes.
- `LinkAt` and `LinkAtScreen` resolve configured hyperlink spans to host-facing targets.
- `ScrollAllowManual` keeps wheel/manual viewport scrolling enabled.
- `ScrollFollowCursorOnly` ignores manual viewport scrolling and keeps viewport movement cursor-driven.
- `View()` returns Bubble Tea v2 `tea.View`; host models can either return `m.editor.View()` directly or compose with `tea.NewView(...)` using `m.editor.View().Content`.

## Extension Points

`Config` supports optional hooks:
- `Gutter.Width` to resolve gutter width in terminal cells for the current frame.
- `Gutter.Cell` to resolve per-row gutter segments (`[]GutterSegment`) and click mapping.
- `RowMarkProvider` to resolve per-row inserted/updated/deleted markers in a dedicated marker lane.
- `RowMarkWidth` to control marker lane width (defaults to `1` when provider is set).
- `RowMarkSymbols` to override marker glyphs for inserted/updated bars and deleted-row arrows.
- `GutterStyleForKey` to resolve keyed gutter segment styles (fallback: `Style.Gutter`).
- `RowStyleForRow` for per-visual-row content-area overrides (box styles allowed; output is clamped to one line and content width).
- `TokenStyleForToken` for per-token style overrides (`IsHighlighted`, `IsActiveRow`, and token metadata).
- `VirtualTextProvider` for per-line virtual deletions/insertions.
- `Highlighter` for per-line highlight spans.
- `LinkProvider` for per-line hyperlink spans (`LinkSpan`) over raw line text.
- `GhostProvider` for inline ghost suggestions at cursor column.
- `VirtualOverlayStyleForKey` to resolve keyed overlay insertion styles (fallback: `Style.VirtualOverlay`).
- `GhostStyleForKey` to resolve keyed ghost insertion styles (fallback: `Style.Ghost`).
- `OnChange` for post-mutation change events.
- `OnIntent` for key-derived semantic intent batches (when intent mode is enabled).

Scrollbar config:
- `Scrollbar.Vertical` and `Scrollbar.Horizontal` use `ScrollbarMode` (`ScrollbarAuto`, `ScrollbarAlways`, `ScrollbarNever`).
- `Scrollbar.MinThumb` defaults to `1` when `<=0`.
- per-frame scrollbar metrics resolve axis visibility and reserve content area dimensions (`contentWidth`/`contentHeight`) used by layout, cursor-follow, and viewport state.
- scrollbar chrome is painted in `Model.View()` on top of `viewport.View()` output.
- vertical scrollbar paints track/thumb in the rightmost inner viewport column for content rows only.
- horizontal scrollbar paints track/thumb in the reserved bottom inner row (content area only), clears the reserved row first, and paints `ScrollbarCorner` when both axes are visible.
- scrollbar cells render as styled spaces (`" "`) using `Style.ScrollbarTrack`, `Style.ScrollbarThumb`, and `Style.ScrollbarCorner`.
- scrollbar interactions are manual-scroll operations: page clicks and thumb dragging are active only when `ScrollPolicy==ScrollAllowManual`.
- runnable host integration example: `examples/scrollbar/main.go`.

Scrollbar style fields:
- `Style.ScrollbarTrack`
- `Style.ScrollbarThumb`
- `Style.ScrollbarCorner`

Row marker style fields:
- `Style.RowMarkInserted`
- `Style.RowMarkUpdated`
- `Style.RowMarkDeleted`

Scrollbar design and implementation roadmap:
- `design/scrollbar.md`
- `roadmap/scrollbar.md`

Virtual text rules:
- deletions hide grapheme ranges from view.
- insertions are view-only and anchored to document grapheme columns.
- insertions can provide `StyleKey` for callback-based style resolution.
- ghost insertions are single-line and non-interactive.
- ghost suggestions can provide `StyleKey` for callback-based style resolution.
- cursor/selection remain document-based.
- cursor/selection-only editor updates rerender only dirty logical rows (old/new cursor rows plus old/new selection coverage).
- text mutations attempt dirty-line incremental rebuild first, and fall back to full rebuild when wrap-row shape changes.
- `VirtualTextProvider` is treated as row-local for cursor/selection movement: non-dirty rows are expected to remain unchanged.
- per-line visible text/mapping derived from virtual deletions is computed once per layout line and reused by both `Highlighter` and `LinkProvider` in the same frame.
- row/token style callbacks are composed as base text style -> row style -> role/highlight/link styles -> token style callback.
- cursor and selection styles still take precedence over token style callbacks.

Hyperlink rules:
- `LinkProvider` receives both raw line text and visible text (after virtual deletions).
- hyperlink spans are interpreted in raw grapheme columns and sanitized to non-overlapping ranges.
- rendered hyperlink spans emit OSC8 links and apply `Style.Link` by default.
- `LinkAt` / `LinkAtScreen` return `LinkHit{Row, StartGraphemeCol, EndGraphemeCol, Target}` for host navigation handling.

Gutter rules:
- gutter is disabled when `Gutter.Width` is nil (or resolves to `<=0`).
- gutter content is provided as `GutterCell.Segments`; each segment can provide `StyleKey` or direct `Style`.
- `Gutter.Cell` receives `LineText` (raw unwrapped document line text).
- gutter click mapping uses `GutterCell.ClickCol` (default `0`, clamped per row).
- row-marker lane is appended to the right side of the configured gutter width.
- `RowMarkProvider` receives `RowMarkContext` with row, segment index, focus/cursor state, and doc metadata.
- marker precedence per visual row is: `DeletedAbove` (segment `0` only), `DeletedBelow` (segment `0` only), `Inserted`, then `Updated`.
- deleted markers are rendered only on the first wrapped segment (`SegmentIndex==0`); inserted/updated markers render on all wrapped segments.
- marker styles use `Style.RowMarkInserted`, `Style.RowMarkUpdated`, and `Style.RowMarkDeleted`.
- runnable host integration example: `examples/row-marks/main.go`.
- use `InvalidateGutter()` when host-side gutter dependencies changed broadly outside editor updates.
- use `InvalidateGutterRows(...)` for row-scoped gutter dependency changes; only targeted rows are rerendered.
- use `InvalidateStyles()` when `RowStyleForRow` / `TokenStyleForToken` depend on host state that changed outside editor updates.
- `LineNumberGutter()` provides built-in line-number behavior.
- `LineNumberWidth(lineCount)` and `LineNumberSegment(ctx)` expose reusable line-number pieces for custom gutters.
- line-number gutter style keys are `line_num` and `line_num_active`.
- segment text is normalized to the resolved gutter width per row.

## Intent Mode

Intent mode lets hosts reuse editor key semantics while choosing mutation strategy.

Types:
- `MutationMode`: `MutateInEditor`, `EmitIntentsOnly`, `EmitIntentsAndMutate`.
- `IntentKind`: `IntentInsert`, `IntentDelete`, `IntentMove`, `IntentSelect`, `IntentUndo`, `IntentRedo`.
- `Intent`: `{ Kind, Before, Payload }`.
- `IntentBatch`: one or more intents produced from one key input.
- `IntentDecision`: `{ ApplyLocally bool }`.

Config:
- `MutationMode` (default `MutateInEditor`).
- `OnIntent func(IntentBatch) IntentDecision`.

Mode behavior:
- `MutateInEditor`: keeps existing behavior. No host intent callback required.
- `EmitIntentsOnly`: emits intents and skips local apply.
- `EmitIntentsAndMutate`: emits intents and applies locally only when `IntentDecision.ApplyLocally=true` (default true when `OnIntent` is nil).
- `IntentUndo` is emitted only when undo history exists (`CanUndo()==true`).
- `IntentRedo` is emitted only when redo history exists (`CanRedo()==true`).

Read-only behavior:
- `ReadOnly=true` still allows move/select intents.
- mutation intents (`insert/delete/undo/redo`) are suppressed.

Move/select payloads:
- `MoveIntentPayload.Move.Count` and `SelectIntentPayload.Move.Count` repeat the move operation.
- default arrow/word/home/end moves use `Count=1` (zero value also means `1`).
- default `pgup`/`pgdown` emit `MoveLine` with `Count=visible row count`.

Host paste behavior:
- editor no longer owns clipboard mechanics (`ctrl+c`/`ctrl+x`/`ctrl+v` are not editor bindings).
- handle `tea.PasteMsg` in the host model and choose the mutation path (local buffer apply, remote transport, or both).
- normalize line endings from paste payloads before apply when needed.

Minimal callback example:

```go
cfg := editor.Config{
    Text:         "hello",
    MutationMode: editor.EmitIntentsAndMutate,
    OnIntent: func(batch editor.IntentBatch) editor.IntentDecision {
        for _, in := range batch.Intents {
            // Example host transport hook for text intents.
            if in.Kind == editor.IntentInsert || in.Kind == editor.IntentDelete {
                sendToRemote(in)
            }
        }
        return editor.IntentDecision{ApplyLocally: true}
    },
}
```

## Render Snapshot Lifecycle

`RenderSnapshot` captures immutable mapping state for the currently rendered frame:
- `Token`: frame identity for host cache keying.
- `BufferVersion`: current buffer version.
- `Viewport`: camera state (`TopVisualRow`, `VisibleRows`, `LeftCellOffset`, `WrapMode`).
- `Rows`: visible row mapping (`ScreenRow`, `DocRow`, doc grapheme span, and per-cell doc column map).
- `Rows`: visible row mapping (`ScreenRow`, `DocRow`, `SegmentIndex`, doc grapheme span, and per-cell doc column map).
  `SegmentIndex` is zero-based within a wrapped doc row (`0` is the first segment, `>0` are continuations).

Token contract:
- same frame/state -> same token.
- mapping-affecting changes (buffer/version, viewport offsets/size, wrap mode, gutter callbacks/width, explicit gutter invalidation, focus/decoration context) -> different token.
- snapshot-bound mapping methods return `ok=false` when token is stale.

Host usage pattern:

```go
snap := ed.RenderSnapshot()
cache[snap.Token] = snap

if pos, ok := ed.ScreenToDocWithSnapshot(snap, mouseX, mouseY); ok {
    // safe mapping for the same frame
    _ = pos
} else {
    // snapshot stale: refresh and retry
    snap = ed.RenderSnapshot()
}
```

## Change Events

`OnChange` receives a `buffer.Change` directly.

Event contract:
- delta-first payload (no full text snapshot).
- The change includes version/cursor/selection before+after and ordered `AppliedEdits`.
- cursor/selection-only changes have `AppliedEdits=[]`.
- no-op updates emit no event.
- in intent modes, `OnChange` fires only if local apply actually mutates editor state.

Examples:
- move right once: `AppliedEdits=[]`, cursor changes from `(0,0)` to `(0,1)`, version increments.
- type `"X"` at `(0,2)`: one `AppliedEdit` with `InsertText="X"` and the exact before/after ranges.

Cross references:
- `docs/buffer.md`
- `docs/completions.md`
- `design/scrollbar.md`
- `design/collab-editing-best-practices.md`
- `roadmap/scrollbar.md`
- `research/collab.md`
