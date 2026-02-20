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
- `CompletionState()`
- `SetCompletionState(state)`
- `ClearCompletion()`
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

- optional custom gutter via `Config.Gutter` callbacks.
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
- `Gutter.Width` to resolve gutter width in terminal cells for the current frame.
- `Gutter.Cell` to resolve per-row gutter segments (`[]GutterSegment`) and click mapping.
- `GutterStyleForKey` to resolve keyed gutter segment styles (fallback: `Style.Gutter`).
- `VirtualTextProvider` for per-line virtual deletions/insertions.
- `Highlighter` for per-line highlight spans.
- `GhostProvider` for inline ghost suggestions at cursor column.
- `VirtualOverlayStyleForKey` to resolve keyed overlay insertion styles (fallback: `Style.VirtualOverlay`).
- `GhostStyleForKey` to resolve keyed ghost insertion styles (fallback: `Style.Ghost`).
- `OnChange` for post-mutation change events.
- `OnIntent` for key-derived semantic intent batches (when intent mode is enabled).
- `CompletionFilter` for host-defined completion item filtering and ordering.
- `CompletionStyleForKey` for keyed completion row/segment style overrides.
- `OnCompletionIntent` for completion semantic intent batches.

Completion foundation config:
- `CompletionKeyMap` controls completion-specific key bindings.
- `CompletionInputMode` default is `CompletionInputQueryOnly`.
- `CompletionMaxVisibleRows` default is `8` when `<=0`.
- `CompletionMaxWidth` default is `60` when `<=0`.

Completion state model:
- `SetCompletionState` stores a cloned completion state and recomputes filtered visibility/selection.
- `CompletionState` returns a cloned state snapshot (no shared mutable slices).
- `ClearCompletion` resets completion to zero value state.

Minimal host flow (`SetCompletionState` + `CompletionFilter`):

```go
cfg := editor.Config{
    CompletionFilter: func(ctx editor.CompletionFilterContext) editor.CompletionFilterResult {
        indices := make([]int, len(ctx.Items))
        for i := range ctx.Items {
            indices[i] = i
        }
        return editor.CompletionFilterResult{VisibleIndices: indices, SelectedIndex: 0}
    },
}

// Host opens popup and provides item list.
m = m.SetCompletionState(editor.CompletionState{
    Visible: true,
    Anchor:  m.Buffer().Cursor(),
    Items: []editor.CompletionItem{
        {ID: "println", InsertText: "println()"},
        {ID: "print", InsertText: "print()"},
    },
})
```

Completion input and acceptance (Phase 2):
- completion key handling runs before regular editor key handling when popup is visible.
- `CompletionKeyMap.Trigger` opens completion at the current cursor anchor and resets query to `""`.
- default `CompletionKeyMap.Trigger` binds both `ctrl+space` and `ctrl+@` (NUL alias used by some terminals/runtime key decoders).
- when popup is visible, `Next`/`Prev`/`PageNext`/`PagePrev` move completion selection and do not move the cursor.
- `Dismiss` closes popup without document mutation.
- `Accept` applies selected completion deterministically:
  use `CompletionItem.Edits` when non-empty, otherwise insert `CompletionItem.InsertText` at `CompletionState.Anchor`.
- after successful local accept apply, popup is cleared.
- `CompletionKeyMap.AcceptTab=false` keeps `Tab` out of completion accept path and falls through to normal tab handling.
- `CompletionInputQueryOnly`: typing/backspace updates `CompletionState.Query` only and does not mutate document text.
- `CompletionInputMutateDocument`: typing/backspace follows normal document mutation and keeps popup visible.
- mutate-document query recompute uses buffer text in range `[Anchor, Cursor)` only when cursor stays on `Anchor.Row` and `Cursor.GraphemeCol >= Anchor.GraphemeCol`; otherwise query resets to `""`.
- `ReadOnly=true` suppresses local document mutation; mutate-document input behaves as query-only.
- while completion is visible, ghost suggestions are suppressed for both rendering and ghost-accept key paths.

Completion filtering and item styling (Phase 3):
- `CompletionFilter` executes synchronously when completion query/items/context change.
- filter context includes `Query`, `Items`, `Cursor`, `DocID`, and current buffer version.
- callback results sanitize invalid/duplicate indices and clamp `SelectedIndex` into visible range.
- default filter (nil callback): case-insensitive `contains` over flattened `Prefix+Label+Detail` text with stable source ordering.
- completion filter is also recomputed while popup is visible when cursor/doc version context changes.
- completion row style precedence is implemented as `segment StyleKey -> item StyleKey -> Style.CompletionItem`, with selected rows based on `Style.CompletionSelected`.
- completion segment truncation helpers preserve segment order and allow partial tail segment rendering with terminal-cell-safe clipping.

Completion popup rendering and placement (Phase 4):
- `Model.View()` renders completion popup rows as an editor-owned overlay on top of the viewport output.
- overlay composition uses `github.com/rmhubbert/bubbletea-overlay` `Composite(...)` directly (no overlay model ownership).
- popup anchor uses `CompletionState.Anchor` projected through `DocToScreen`.
- vertical placement prefers below the anchor row, then flips above when below-space is insufficient.
- when anchor is offscreen (`DocToScreen` not visible), popup render is suppressed while completion state remains intact.
- popup width is measured from rendered completion rows, then clamped by `CompletionMaxWidth` and viewport bounds.
- popup row count is clamped by `CompletionMaxVisibleRows`, available vertical space, and visible completion count.
- runnable host integration example: `examples/completion-popup/main.go`.

Completion intents and host control (Phase 5):
- `OnCompletionIntent` emits completion semantic batches for trigger/navigate/accept/dismiss/query actions.
- completion intent payloads are typed:
- `CompletionTriggerIntentPayload{Anchor}`
- `CompletionNavigateIntentPayload{Delta, Selected, ItemIndex}`
- `CompletionAcceptIntentPayload{ItemID, ItemIndex, VisibleIndex, InsertText, Edits}`
- `CompletionDismissIntentPayload{}`
- `CompletionQueryIntentPayload{Query}`
- callback order for mutate-document completion keys is deterministic: `OnCompletionIntent` first, then `OnIntent`.
- `MutationMode` gates only local document mutation, not completion callback emission.
- in `EmitIntentsOnly`, completion UI state still updates for non-document actions (trigger/navigate/dismiss/query-only input).
- in `CompletionInputMutateDocument`, typing/backspace emits both completion query intent and document insert/delete intent.
- in `EmitIntentsOnly`, completion-driven document edits are emitted via `OnIntent` and skipped locally.
- in `EmitIntentsAndMutate`, completion-driven document edits are emitted and then locally applied only when `IntentDecision.ApplyLocally=true`.
- when local completion accept apply runs, popup is cleared; if local apply is skipped, popup remains host-controlled via `SetCompletionState`.

Emit-only host flow example:

```go
cfg := editor.Config{
    MutationMode: editor.EmitIntentsOnly,
    OnCompletionIntent: func(batch editor.CompletionIntentBatch) {
        sendCompletionToRemote(batch)
    },
    OnIntent: func(batch editor.IntentBatch) editor.IntentDecision {
        sendDocumentToRemote(batch)
        return editor.IntentDecision{ApplyLocally: false}
    },
}
```

Emit-and-mutate host flow example:

```go
cfg := editor.Config{
    MutationMode: editor.EmitIntentsAndMutate,
    OnCompletionIntent: func(batch editor.CompletionIntentBatch) {
        auditCompletion(batch)
    },
    OnIntent: func(batch editor.IntentBatch) editor.IntentDecision {
        ack := replicate(batch)
        return editor.IntentDecision{ApplyLocally: ack}
    },
}
```

Virtual text rules:
- deletions hide grapheme ranges from view.
- insertions are view-only and anchored to document grapheme columns.
- insertions can provide `StyleKey` for callback-based style resolution.
- ghost insertions are single-line and non-interactive.
- ghost suggestions can provide `StyleKey` for callback-based style resolution.
- cursor/selection remain document-based.

Gutter rules:
- gutter is disabled when `Gutter.Width` is nil (or resolves to `<=0`).
- gutter content is provided as `GutterCell.Segments`; each segment can provide `StyleKey` or direct `Style`.
- `Gutter.Cell` receives `LineText` (raw unwrapped document line text).
- gutter click mapping uses `GutterCell.ClickCol` (default `0`, clamped per row).
- use `InvalidateGutter()` when host-side gutter dependencies changed broadly outside editor updates.
- use `InvalidateGutterRows(...)` for row-scoped gutter dependency changes; only targeted rows are rerendered.
- `LineNumberGutter()` provides built-in line-number behavior.
- `LineNumberWidth(lineCount)` and `LineNumberSegment(ctx)` expose reusable line-number pieces for custom gutters.
- line-number gutter style keys are `line_num` and `line_num_active`.
- segment text is normalized to the resolved gutter width per row.

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
- `design/completion.md`
- `design/collab-editing-best-practices.md`
- `research/collab.md`
