# `editor` package — current state

Source: `editor/`

Design targets:
- `design/spec.md` (architecture layering)
- `design/api.md` (public API draft)

Roadmap:
- Phase 5: `roadmap/phase-5-editor-skeleton-rendering.md`
- Phase 6: `roadmap/phase-6-editor-keys-selection-scroll.md`
- Phase 7: `roadmap/phase-7-editor-mouse-clipboard.md`

## What exists (Phase 7)

The `editor` package provides a Bubble Tea component:
- `editor.New(editor.Config)` constructs a value-type `editor.Model` that owns an internal `*buffer.Buffer`.
- `SetSize(width, height)` sets the viewport size.
- `Focus()`, `Blur()`, `Focused()` control cursor rendering and active line number styling.
- `Update(msg tea.Msg)` handles:
  - keybindings (movement, selection, editing, undo/redo, clipboard)
  - window resize
  - mouse wheel scrolling (via `bubbles/viewport`)
  - mouse click/shift+click/drag selection
- `View()` renders:
  - logical buffer lines (no soft wrap yet)
  - optional line numbers gutter (`Config.ShowLineNums`)
  - selection styling (`Style.Selection`) for the active selection range (half-open `[Start, End)`)
  - cursor styling on the active line when focused (`Style.Cursor`)

Rendering uses `bubbles/viewport` for vertical scrolling and width/height clipping.

## Key handling

- Default keymap: `DefaultKeyMap()` (arrow movement, shift+arrows selection, ctrl/alt word movement fallbacks, backspace/delete/enter, undo/redo, copy/cut/paste).
- `Config.ReadOnly=true` ignores buffer mutations but still allows movement and selection.
- Clipboard integration is optional via `Config.Clipboard`. If nil, copy/cut/paste are disabled.

## Scrolling behavior

- The viewport follows the cursor after key-driven movement/edits to keep the cursor row visible.
- Manual mouse wheel scrolling is preserved (cursor-follow does not override wheel scrolling).

## Mouse handling

Hit-testing maps viewport-local mouse coordinates `(X,Y)` to document positions:
- `Y` maps to `buffer.Pos.Row` using `viewport.YOffset`.
- `X` maps to `buffer.Pos.Col` (v0: runes are treated as 1-cell).
- Clicking in the line number gutter maps to column 0 (start of line).
- Positions are clamped into document bounds.

Behavior:
- click: set cursor and clear selection
- shift+click: extend selection from the existing anchor (or the current cursor if no selection)
- drag: while left button is down, update selection end and cursor

Note: mouse coordinates are assumed to be relative to the editor's viewport. If the editor is rendered inside a larger layout, the parent model should translate mouse coordinates before forwarding the message.

## Manual demo

Run:
- `go run ./cmd/flourish-demo`

Demo notes:
- The demo relies on the editor's internal key handling.
- Quit: `ctrl+q`.
