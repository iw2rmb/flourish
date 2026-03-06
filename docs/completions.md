# Editor Completions

This document describes the current completion implementation in `editor`.

## Overview

Completions are editor-owned popup state plus host-controlled item sourcing/filtering.
The host opens/updates completion with `SetCompletionState`.
The editor handles navigation, query updates, popup rendering, and optional local document mutation.

Primary API:
- `CompletionState()`
- `SetCompletionState(state)`
- `ClearCompletion()`

## Config Surface

Completion config fields in `editor.Config`:
- `CompletionKeyMap` for completion-specific key bindings.
- `CompletionInputMode` (`CompletionInputQueryOnly` or `CompletionInputMutateDocument`).
- `CompletionMaxVisibleRows` (defaults to `8` when `<=0`).
- `CompletionMaxWidth` (defaults to `60` when `<=0`).
- `CompletionFilter` for host-defined filtering/ranking.
- `CompletionStyleForKey` for keyed completion row/segment style overrides.
- `OnCompletionIntent` for completion semantic intent batches.

## State Model

- `SetCompletionState` stores a cloned completion state and recomputes filtered visibility/selection.
- `CompletionState` returns a cloned state snapshot (no shared mutable slices).
- `ClearCompletion` resets completion to zero value state.

## Keyboard

Default completion shortcuts:

| Context | Shortcut | Action |
| --- | --- | --- |
| Completion popup | `ctrl+space` or `ctrl+@` | Trigger/open completion at cursor anchor. |
| Completion popup (visible) | `down` | Select next completion item. |
| Completion popup (visible) | `up` | Select previous completion item. |
| Completion popup (visible) | `pgdown` | Select next completion page. |
| Completion popup (visible) | `pgup` | Select previous completion page. |
| Completion popup (visible) | `enter` | Accept selected completion item. |
| Completion popup (visible) | `tab` | Accept selected item when `CompletionKeyMap.AcceptTab=true`. |
| Completion popup (visible) | `esc` | Dismiss completion popup. |

## Input and Acceptance Behavior

- completion key handling runs before regular editor key handling when popup is visible.
- `CompletionKeyMap.Trigger` opens completion at current cursor anchor and resets query to `""`.
- default `CompletionKeyMap.Trigger` binds both `ctrl+space` and `ctrl+@` (NUL alias used by some terminals/runtime key decoders).
- when popup is visible, `Next`/`Prev`/`PageNext`/`PagePrev` move completion selection and do not move cursor.
- `Dismiss` closes popup without document mutation.
- `Accept` applies selected completion deterministically:
  use `CompletionItem.Edits` when non-empty; otherwise insert `CompletionItem.InsertText` at `CompletionState.Anchor`.
- after successful local accept apply, popup is cleared.
- `CompletionKeyMap.AcceptTab=false` keeps `Tab` out of completion accept path and falls through to normal tab handling.
- `CompletionInputQueryOnly`: typing/backspace updates `CompletionState.Query` only and does not mutate document text.
- `CompletionInputMutateDocument`: typing/backspace follows normal document mutation and keeps popup visible.
- mutate-document query recompute uses buffer text in range `[Anchor, Cursor)` only when cursor stays on `Anchor.Row` and `Cursor.GraphemeCol >= Anchor.GraphemeCol`; otherwise query resets to `""`.
- mutate-document query updates use `buffer.TextInRange` and avoid temporary full-buffer clones.
- cursor movement keeps popup open only while cursor stays within the token span anchored at `CompletionState.Anchor`; leaving that span (or row) clears popup state.
- `ReadOnly=true` suppresses local document mutation; mutate-document input behaves as query-only.
- while completion is visible, ghost suggestions are suppressed for both rendering and ghost-accept key paths.

## Filtering and Styling

- `CompletionFilter` executes synchronously when completion query/items/context change.
- filter context includes `Query`, `Items`, `Cursor`, `DocID`, and current buffer version.
- `CompletionFilterContext.Items` is passed directly from current completion state (no defensive deep copy); treat it as read-only.
- callback results sanitize invalid/duplicate indices and clamp `SelectedIndex` into visible range.
- default filter (nil callback): case-insensitive `contains` over flattened `Prefix+Label+Detail` text with stable source ordering.
- completion filter is also recomputed while popup is visible when cursor/doc version context changes.
- completion row style precedence: `segment StyleKey -> item StyleKey -> Style.CompletionItem`; selected rows use `Style.CompletionSelected`.
- completion segment truncation preserves segment order and allows partial tail segment rendering with terminal-cell-safe clipping.

## Rendering and Placement

- `Model.View()` renders completion popup rows as an editor-owned overlay on top of viewport output.
- overlay composition uses editor helper `compositeTopLeft` backed by Lip Gloss v2 layers/compositor.
- popup anchor uses `CompletionState.Anchor` projected through `DocToScreen`.
- vertical placement prefers below anchor row, then flips above when below-space is insufficient.
- when anchor is offscreen (`DocToScreen` not visible), popup render is suppressed while completion state remains intact.
- popup width is measured from rendered completion rows, then clamped by `CompletionMaxWidth` and content-area width (excluding gutter and reserved vertical scrollbar column).
- popup X is clamped to content-area bounds, so overlay never paints into reserved scrollbar chrome.
- completion popup segment cell widths are precomputed per item and reused for width measurement.
- popup row count is clamped by `CompletionMaxVisibleRows`, available vertical space, and visible completion count.
- runnable host integration example: `examples/completion-popup/main.go`.

## Intents and Host Control

- `OnCompletionIntent` emits completion semantic batches for trigger/navigate/accept/dismiss/query actions.
- completion intent payload types:
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

m = m.SetCompletionState(editor.CompletionState{
    Visible: true,
    Anchor:  m.Buffer().Cursor(),
    Items: []editor.CompletionItem{
        {ID: "println", InsertText: "println()"},
        {ID: "print", InsertText: "print()"},
    },
})
```

## Cross References

- `docs/editor.md`
- `docs/buffer.md`
- `design/completion.md`
