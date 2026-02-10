# `editor` package — current state

Source: `editor/`

Design targets:
- `design/spec.md` (architecture layering)
- `design/api.md` (public API draft)

Roadmap:
- Phase 5: `roadmap/phase-5-editor-skeleton-rendering.md`

## What exists (Phase 5)

The `editor` package provides a Bubble Tea component:
- `editor.New(editor.Config)` constructs a value-type `editor.Model` that owns an internal `*buffer.Buffer`.
- `SetSize(width, height)` sets the viewport size.
- `Focus()`, `Blur()`, `Focused()` control cursor rendering and active line number styling.
- `View()` renders:
  - logical buffer lines (no soft wrap yet)
  - optional line numbers gutter (`Config.ShowLineNums`)
  - cursor styling on the active line when focused

Rendering uses `bubbles/viewport` for vertical scrolling and width/height clipping.

## Manual demo

Run:
- `go run ./cmd/flourish-demo`

Demo notes:
- Editing and movement are handled by the demo app by mutating `editor.Model.Buffer()` directly.
- The editor itself does not yet handle keybindings (planned in Phase 6).

