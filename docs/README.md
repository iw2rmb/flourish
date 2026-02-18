# Flourish Documentation

Packages:
- `docs/buffer.md` — `buffer` package behavior and contracts.
- `docs/editor.md` — `editor` package behavior and integration contracts.

Examples:
- `examples/simple/main.go` — baseline editor setup.
- `examples/wrap-modes/main.go` — toggles `WrapNone`, `WrapWord`, `WrapGrapheme` via `ctrl+n`, `ctrl+w`, `ctrl+g`.
- `examples/inline-suggestions/main.go` — inline ghost suggestion provider and accept flow.
- `examples/virtual-text/main.go` — virtual deletions/insertions overlay behavior.
- `examples/highlighter/main.go` — line highlighter integration.
- `examples/on-change/main.go` — delta-backed `OnChange` event reporting (`buffer.Change` payload).
- `examples/intent-mode/main.go` — intent emission with host-controlled local-apply decisions.
