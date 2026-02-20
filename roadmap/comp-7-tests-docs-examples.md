# Phase 7: Completion Hardening (Tests, Docs, Examples)

Scope: Finalize completion feature quality with full test matrix, documentation updates, and runnable examples for host integration patterns.

Documentation: `design/completion.md`, `docs/editor.md`, `docs/buffer.md`, `docs/README.md`

Legend: [ ] todo, [x] done.

## Test Matrix Completion
- [x] Add completion behavior tests across editor subsystems — lock contracts for rendering, key handling, and mutation semantics
  - Repository: `flourish`
  - Component: `editor` tests
  - Scope: add tests for popup visibility transitions, navigation, accept, dismiss, query/mutate routing, ghost suppression, wrap/no-wrap rendering
  - Snippets: table-driven test cases keyed by `CompletionInputMode` and `MutationMode`
  - Tests: run `go test ./editor` — all completion scenarios pass deterministically

- [x] Add conversion helper tests for UTF-16 and rune/grapheme APIs — ensure protocol-grade correctness
  - Repository: `flourish`
  - Component: `buffer` tests
  - Scope: add fixtures and round-trip tests for surrogate pairs, combining marks, ZWJ clusters, and multiline offsets
  - Snippets: fixture rows with expected UTF-16/rune/grapheme boundaries
  - Tests: run `go test ./buffer` — round-trip and interior-boundary rejection tests pass

## Documentation
- [x] Update docs to reflect actual completion and conversion behavior — keep host-facing docs synchronized with code
  - Repository: `flourish`
  - Component: docs
  - Scope: update `docs/editor.md`, `docs/buffer.md`, and `docs/README.md` with completion APIs, intent hooks, sizing/input controls, and conversion helpers
  - Snippets: minimal host example for `SetCompletionState` + `CompletionFilter`
  - Tests: doc review for cross-reference integrity and API parity

## Examples
- [x] Add dedicated completion popup example app — provide executable integration reference for hosts
  - Repository: `flourish`
  - Component: `examples`
  - Scope: add `examples/completion-popup/main.go` demonstrating item styling, filter callback, Tab disable, and input-routing mode
  - Snippets: config with `CompletionInputMutateDocument`, `CompletionMaxVisibleRows`, `CompletionMaxWidth`
  - Tests: run `go run ./examples/completion-popup` — popup interaction matches documented contracts
