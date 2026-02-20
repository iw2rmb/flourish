# Phase 3: Completion Filtering and Item Styling

Scope: Add host-provided completion filtering/ranking callback and segment-based item styling with deterministic fallback and truncation behavior.

Documentation: `design/completion.md`, `docs/editor.md`

Legend: [ ] todo, [x] done.

## Filtering
- [x] Implement `CompletionFilter` callback execution contract — allow host ranking/filtering without async/editor-side I/O
  - Repository: `flourish`
  - Component: `editor`
  - Scope: call filter on query/item/context changes; sanitize indices; clamp selected index
  - Snippets: `CompletionFilterResult{VisibleIndices, SelectedIndex}`
  - Tests: callback tests for invalid indices, empty result, and selected-index clamping

- [x] Implement default filtering when callback is nil — preserve useful behavior without host customization
  - Repository: `flourish`
  - Component: `editor`
  - Scope: case-insensitive contains over flattened item text with stable source ordering
  - Snippets: `strings.Contains(strings.ToLower(flat), strings.ToLower(query))`
  - Tests: default filter ordering and matching tests with mixed-case labels/details

## Styling
- [x] Add completion row styles and keyed resolver integration — match existing gutter/ghost style-key architecture
  - Repository: `flourish`
  - Component: `editor` rendering styles
  - Scope: add `Style.CompletionItem` and `Style.CompletionSelected`; resolve by segment key, then item key, then default style
  - Snippets: style precedence `segment -> item -> default`
  - Tests: style fallback tests and selected-row style composition tests

- [x] Enforce segment-safe truncation and layout stability — prevent style callbacks from breaking deterministic layout
  - Repository: `flourish`
  - Component: `editor` completion row rendering
  - Scope: truncate by terminal cell width while preserving segment order and allowing partial tail segment render
  - Snippets: segment iterator with cell-budget tracking
  - Tests: truncation tests with wide glyphs and long detail columns
