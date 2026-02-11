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

## Input Behavior

Keyboard:
- movement, selection extension, editing, undo/redo, clipboard shortcuts.
- `ReadOnly=true` blocks text mutation, keeps movement/selection enabled.

Mouse:
- click to move cursor.
- shift+click to extend selection.
- drag to update selection.
- hit-testing maps from viewport-local `(x,y)` cells to document positions.

## Extension Points

`Config` supports optional hooks:
- `VirtualTextProvider` for per-line virtual deletions/insertions.
- `Highlighter` for per-line highlight spans.
- `GhostProvider` for EOL ghost suggestions.
- `OnChange` for post-mutation change events.

Virtual text rules:
- deletions hide grapheme ranges from view.
- insertions are view-only and anchored to document grapheme columns.
- cursor/selection remain document-based.

## Change Events

`OnChange` receives:
- current version
- cursor
- selection state
- full text payload (v0)
