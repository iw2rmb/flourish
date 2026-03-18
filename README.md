# flourish

Flourish is a Go text editing library for Bubble Tea TUIs.
It provides a pure document buffer package and an editor component package.

Documentation: [docs/README.md](docs/README.md)

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
- conditional row/token style callbacks for active-row and token-state rendering.


## Integration

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
