# flourish

Flourish is a Go text editing library for Bubble Tea TUIs.
It provides a pure document buffer package and an editor component package.

## Features

- `buffer` package with grapheme-based coordinates and half-open ranges.
- cursor movement by grapheme, word, line, and document units.
- selection model with stable anchor behavior.
- text editing operations with selection-first semantics.
- bounded undo/redo history.
- deterministic `Apply` API for host-driven edits.
- `editor` Bubble Tea component with viewport integration.
- soft wrap (`WrapWord`, `WrapGrapheme`) and no-wrap horizontal scrolling.
- mouse hit-testing and drag selection in terminal cell coordinates.
- optional clipboard integration.
- optional virtual text, highlighting, ghost suggestions, and change events.

## Documentation

- package docs index: `docs/index.md`
- buffer package docs: `docs/buffer.md`
- editor package docs: `docs/editor.md`
