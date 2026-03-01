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
- host-facing viewport state and doc<->screen coordinate mapping APIs.
- soft wrap (`WrapWord`, `WrapGrapheme`) and no-wrap horizontal scrolling.
- mouse hit-testing and drag selection in terminal cell coordinates.
- host-controlled paste handling via Bubble Tea v2 `tea.PasteMsg`.
- optional virtual text, highlighting, ghost suggestions, and change events.
- semver-enabled runtime version API via `flourish.Version()`.
- semver release flow with `vMAJOR.MINOR.PATCH` git tags.

## Bubble Tea v2 Integration

Use v2 module paths:

```go
import tea "charm.land/bubbletea/v2"
import "github.com/iw2rmb/flourish/editor"
```

`editor.Model.View()` returns `tea.View`, so host models can return it directly
or compose via `tea.NewView(...)`.

Host-side paste flow:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.PasteMsg:
		b := m.ed.Buffer()
		b.InsertTextAt(b.Cursor(), string(msg))
	}
	var cmd tea.Cmd
	m.ed, cmd = m.ed.Update(msg)
	return m, cmd
}
```

## Documentation

- package docs index: `docs/README.md`
- buffer package docs: `docs/buffer.md`
- editor package docs: `docs/editor.md`
- versioning docs: `docs/versioning.md`

## Examples

- `go run ./examples/simple`
- `go run ./examples/wrap-modes`
- `go run ./examples/scrollbar`
- `go run ./examples/inline-suggestions`
- `go run ./examples/virtual-text`
- `go run ./examples/highlighter`
- `go run ./examples/on-change`

## Versioning

- current version source of truth: `VERSION`
- runtime version API: `flourish.Version()`, `flourish.VersionTag()`
- local semver tooling: `scripts/semver.sh`
