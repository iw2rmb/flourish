# `editor` package — current state

Source: `editor/`

Design targets:
- `design/spec.md` (architecture layering)
- `design/api.md` (public API draft)

Roadmap:
- Phase 5: `roadmap/phase-5-editor-skeleton-rendering.md`
- Phase 6: `roadmap/phase-6-editor-keys-selection-scroll.md`
- Phase 7: `roadmap/phase-7-editor-mouse-clipboard.md`
- Phase 8: `roadmap/phase-8-editor-visual-line-mapping.md`
- Phase 9: `roadmap/phase-9-editor-ghost-highlight-events.md`
- Phase 10: `roadmap/phase-10-editor-horizontal-scrolling.md`
- Phase 11: `roadmap/phase-11-editor-soft-wrapping.md`

## What exists (Phase 11)

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
  - logical buffer lines rendered as visual rows from a wrap layout cache
  - `WrapNone` horizontal scrolling for long lines (internal `xOffset` in terminal cells)
  - `WrapWord` soft wrapping with word-boundary preference and punctuation fallback heuristic
  - `WrapGrapheme` soft wrapping at grapheme boundaries
  - optional line numbers gutter (`Config.ShowLineNums`)
  - selection styling (`Style.Selection`) for the active selection range (half-open `[Start, End)`)
  - cursor styling on the active line when focused (`Style.Cursor`)
  - virtual text transforms via `Config.VirtualTextProvider` (virtual deletions/insertions)
  - EOL-only ghost suggestions via `Config.GhostProvider` (rendered with `Style.Ghost`, accepted via `Config.GhostAccept`)
  - optional per-line highlighting via `Config.Highlighter` (spans over visible doc text after deletions)

Rendering uses `bubbles/viewport` for vertical scrolling and width/height clipping.

## Wrapping + horizontal scrolling

- `Config.WrapMode` exists (default: `WrapNone`).
- Layout is computed as wrapped segments (`StartCol`, `EndCol`, `Cells`) per logical line and flattened into visual rows.
- Grapheme boundaries and widths are Unicode-aware (`rivo/uniseg`) and measured in terminal cells (`go-runewidth`); tabs use tab-stop expansion by `Config.TabWidth`.
- For `WrapNone`, the editor maintains an internal horizontal offset `xOffset` (cells) so the cursor stays visible on long lines.
- Horizontal scrolling clips each logical line by cells to `[xOffset:xOffset+contentWidth)`, where `contentWidth = viewportWidth - gutterWidth`.
- Horizontal scrolling is updated on key-driven cursor moves and edits; it is not adjusted while mouse-dragging a selection.
- For `WrapWord` and `WrapGrapheme`, `xOffset` is not used; viewport Y offset and cursor-follow operate on wrapped visual rows.

## Virtual text + visual mapping

- `Config.VirtualTextProvider` can return per-line view-only transforms:
  - virtual deletions hide raw rune ranges (they do not render and are skipped by hit-testing)
  - virtual insertions add rendered cells anchored at a raw document column (clicks inside insertions map to the anchor column)
- `Config.DocID` (optional) is forwarded into hook contexts for caching (`VirtualTextContext.DocID`, `GhostContext.DocID`).
- Cursor and selection remain document-based; selection styling applies only to visible doc-backed cells.
- Tabs expand by `Config.TabWidth` (default: 4) and all horizontal mapping is in terminal-cell coordinates (grapheme-aware).
- Virtual insertions render using their role:
  - `VirtualRoleGhost` uses `Style.Ghost`
  - `VirtualRoleOverlay` uses `Style.VirtualOverlay`

## Key handling

- Default keymap: `DefaultKeyMap()` (arrow movement, shift+arrows selection, ctrl/alt word movement fallbacks, backspace/delete/enter, undo/redo, copy/cut/paste).
- `Config.ReadOnly=true` ignores buffer mutations but still allows movement and selection.
- Clipboard integration is optional via `Config.Clipboard`. If nil, copy/cut/paste are disabled.
- Ghost acceptance:
  - at EOL only
  - applies `Ghost.Edits` via `buffer.Apply(...)`
  - default accept keys: Tab and Right (configurable via `Config.GhostAccept`)

## Change events

- `Config.OnChange` fires after every buffer mutation triggered via `Update`.
- Event payload includes the full buffer text (v0), cursor position, and selection state.

## Scrolling behavior

- The viewport follows the cursor after key-driven movement/edits to keep the cursor row visible.
- Manual mouse wheel scrolling is preserved (cursor-follow does not override wheel scrolling).
- Under `WrapNone`, horizontal scrolling follows the cursor to keep its visual cell column visible.
- Under `WrapWord`/`WrapGrapheme`, vertical follow maps cursor doc position to wrapped visual row/column.

## Mouse handling

Hit-testing maps viewport-local mouse coordinates `(X,Y)` to document positions:
- `Y` maps to `buffer.Pos.Row` using `viewport.YOffset`.
- `X` maps to `buffer.Pos.Col` using terminal **cell** coordinates (grapheme-aware; wide graphemes map multiple cells to one doc position; tabs expand by `Config.TabWidth`, default 4).
- Clicking in the line number gutter maps to column 0 (start of line).
- Positions are clamped into document bounds.
- Under `WrapNone`, hit-testing accounts for the horizontal scroll offset (`xOffset`).
- Under soft-wrap modes, hit-testing uses the wrap layout cache:
  - `(x,y)` maps to logical line + wrapped segment + doc column
  - click past segment end maps to segment end column
  - viewport `YOffset` is interpreted in wrapped visual rows

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
