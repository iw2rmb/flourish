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
- `Update(msg tea.Msg)`
- `View()`
- `Buffer()`
- `ViewportState()`
- `ScreenToDoc(x, y)`
- `DocToScreen(pos)`
- `RenderSnapshot()`
- `ScreenToDocWithSnapshot(snapshot, x, y)`
- `DocToScreenWithSnapshot(snapshot, pos)`

## Coordinate Model

- document columns are grapheme indices.
- rendering and hit-testing use terminal cell coordinates.
- tabs expand by `TabWidth` (default `4`).

## Rendering and Layout

- optional line number gutter (`ShowLineNums`).
- wrap modes:
- `WrapNone`: no soft wrap; horizontal scroll (`xOffset`) keeps cursor visible.
- `WrapWord`: wraps at word boundaries with fallback behavior.
- `WrapGrapheme`: wraps at grapheme boundaries.
- layout mapping preserves doc<->visual conversions, including wide glyphs.
- cursor rendering keeps visibility at soft-wrap boundaries:
- EOL cursor remains visible when a wrapped row exactly fills content width.
- trailing whitespace cursor cells are rendered with non-breaking spaces to avoid terminal elision.

## Input Behavior

Keyboard:
- movement, selection extension, editing, undo/redo, clipboard shortcuts.
- `ReadOnly=true` blocks text mutation, keeps movement/selection enabled.
- text mutation shortcuts include typing, enter, delete/backspace, cut, paste, undo/redo.

Mouse:
- click to move cursor.
- shift+click to extend selection.
- drag to update selection.
- hit-testing maps from viewport-local `(x,y)` cells to document positions.
- wheel scroll is controlled by `ScrollPolicy`.

Viewport integration:
- `ViewportState()` exposes top visual row, visible row count, wrap mode, and no-wrap horizontal offset.
- `ScreenToDoc` and `DocToScreen` provide stable host-side coordinate mapping.
- snapshot-bound mapping APIs (`RenderSnapshot`, `ScreenToDocWithSnapshot`, `DocToScreenWithSnapshot`) provide frame-stable mapping with stale-token rejection.
- `ScrollAllowManual` keeps wheel/manual viewport scrolling enabled.
- `ScrollFollowCursorOnly` ignores manual viewport scrolling and keeps viewport movement cursor-driven.

## Extension Points

`Config` supports optional hooks:
- `VirtualTextProvider` for per-line virtual deletions/insertions.
- `Highlighter` for per-line highlight spans.
- `GhostProvider` for inline ghost suggestions at cursor column.
- `VirtualOverlayStyleForKey` to resolve keyed overlay insertion styles (fallback: `Style.VirtualOverlay`).
- `GhostStyleForKey` to resolve keyed ghost insertion styles (fallback: `Style.Ghost`).
- `OnChange` for post-mutation change events.
- `OnIntent` for key-derived semantic intent batches (when intent mode is enabled).

Virtual text rules:
- deletions hide grapheme ranges from view.
- insertions are view-only and anchored to document grapheme columns.
- insertions can provide `StyleKey` for callback-based style resolution.
- ghost insertions are single-line and non-interactive.
- ghost suggestions can provide `StyleKey` for callback-based style resolution.
- cursor/selection remain document-based.

## Intent Mode

Intent mode lets hosts reuse editor key semantics while choosing mutation strategy.

Types:
- `MutationMode`: `MutateInEditor`, `EmitIntentsOnly`, `EmitIntentsAndMutate`.
- `IntentKind`: `IntentInsert`, `IntentDelete`, `IntentMove`, `IntentSelect`, `IntentUndo`, `IntentRedo`, `IntentPaste`.
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

Read-only behavior:
- `ReadOnly=true` still allows move/select intents.
- mutation intents (`insert/delete/undo/redo/paste`) are suppressed.

Clipboard behavior:
- copy remains local-only.
- cut emits delete intent semantics (no dedicated cut intent kind).
- paste emits `IntentPaste` with normalized newline text.

Minimal callback example:

```go
cfg := editor.Config{
    Text:         "hello",
    MutationMode: editor.EmitIntentsAndMutate,
    OnIntent: func(batch editor.IntentBatch) editor.IntentDecision {
        for _, in := range batch.Intents {
            // Example host transport hook for text intents.
            if in.Kind == editor.IntentInsert || in.Kind == editor.IntentDelete || in.Kind == editor.IntentPaste {
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
- mapping-affecting changes (buffer/version, viewport offsets/size, wrap mode, line-number gutter, focus/decoration context) -> different token.
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

`OnChange` receives:
- `ChangeEvent{ Change buffer.Change }`

Event contract:
- delta-first payload (no full text snapshot).
- `Change` includes version/cursor/selection before+after and ordered `AppliedEdits`.
- cursor/selection-only changes have `AppliedEdits=[]`.
- no-op updates emit no event.
- in intent modes, `OnChange` fires only if local apply actually mutates editor state.

Examples:
- move right once: `AppliedEdits=[]`, cursor changes from `(0,0)` to `(0,1)`, version increments.
- type `"X"` at `(0,2)`: one `AppliedEdit` with `InsertText="X"` and the exact before/after ranges.

Cross references:
- `docs/buffer.md`
- `design/collab-editing-best-practices.md`
- `research/collab.md`
