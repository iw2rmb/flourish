# Scrollbar Implementation Roadmap (Vertical + Horizontal)

Scope: Implement editor-owned scrollbars for vertical and horizontal axes with deterministic rendering, correct hit-testing, manual mouse interactions, and stable host-facing viewport/snapshot behavior.

Documentation: `design/scrollbar.md`, `docs/editor.md`, `editor/render.go`, `editor/model.go`, `editor/update_mouse.go`, `editor/viewport_state.go`, `editor/snapshot.go`.

Legend: [ ] todo, [x] done.

## Phase 1: Public API Surface (Config + Style)
- [x] Add scrollbar config and style fields with normalization defaults — Exposes the feature cleanly and keeps zero-value behavior predictable.
  - Repository: `flourish`
  - Component: `editor` (`Config`, `Style`, constructor normalization)
  - Scope: Update `editor/config.go`, `editor/style.go`, and `editor/model.go:New(...)`.
  - Snippets:
    ```go
    // editor/config.go
    type ScrollbarMode int

    const (
    	ScrollbarAuto ScrollbarMode = iota
    	ScrollbarAlways
    	ScrollbarNever
    )

    type ScrollbarConfig struct {
    	Vertical   ScrollbarMode
    	Horizontal ScrollbarMode
    	MinThumb   int
    }

    type Config struct {
    	// ...
    	Scrollbar ScrollbarConfig
    }
    ```
    ```go
    // editor/style.go
    type Style struct {
    	// existing fields...
    	ScrollbarTrack  lipgloss.Style
    	ScrollbarThumb  lipgloss.Style
    	ScrollbarCorner lipgloss.Style
    }
    ```
    ```go
    // editor/model.go (inside New)
    if cfg.Scrollbar.MinThumb <= 0 {
    	cfg.Scrollbar.MinThumb = 1
    }
    ```
  - Tests: Add `editor/config_normalization_test.go` cases for `MinThumb`; add `editor/style` zero/default tests for new style fields — expect normalized defaults and no style regressions.

## Phase 2: Scrollbar Metrics + Content Area Reservation
- [x] Introduce resolved per-frame scrollbar metrics and use it for content width/height — Prevents text/overlay overlap and keeps both axes coherent.
  - Repository: `flourish`
  - Component: `editor` layout/viewport computations
  - Scope: Add `editor/scrollbar.go` (new helper), integrate calls in `editor/model.go` (`contentWidth`, `visibleRowCount`, `followCursorWithForce`), and anywhere height/width assumptions are currently raw viewport-frame based.
  - Snippets:
    ```go
    // editor/scrollbar.go
    type scrollbarMetrics struct {
    	showV, showH bool
    	innerWidth, innerHeight int
    	contentWidth, contentHeight int
    	totalRows, yOffset, vThumbPos, vThumbLen int
    	totalCols, xOffset, hThumbPos, hThumbLen int
    }
    ```
    ```go
    func (m *Model) resolveScrollbarMetrics(lines []string, layout wrapLayoutCache) scrollbarMetrics {
    	// 1) compute inner viewport size
    	// 2) iterate visibility (V/H) to fixed point
    	// 3) clamp offsets and derive thumb geometry
    	// 4) return resolved contentWidth/contentHeight
    }
    ```
  - Tests: Add `editor/scrollbar_metrics_test.go` for visibility, fixed-point coupling, and thumb math at start/mid/end offsets — expect deterministic geometry with clamped offsets.

## Phase 3: Render Pipeline Integration (Track/Thumb/Corner)
- [x] Render vertical and horizontal bars after content rows are built — Adds visible scrollbar chrome without disturbing text shaping.
  - Repository: `flourish`
  - Component: `editor` rendering
  - Scope: Update `editor/render.go` and `editor/model.go:View()` pipeline to apply scrollbar row/column painting; ensure content uses reserved area dimensions from metrics.
  - Snippets:
    ```go
    func (m *Model) renderRows(...) []string {
    	// existing line rendering...
    	// use metrics.contentWidth for content clipping decisions
    }
    ```
    ```go
    func paintVerticalScrollbar(rows []string, metrics scrollbarMetrics, st Style) []string {
    	// paint track + thumb in right reserved column
    }
    ```
    ```go
    func paintHorizontalScrollbar(rows []string, metrics scrollbarMetrics, st Style) []string {
    	// paint track + thumb in bottom reserved row
    	// paint corner when metrics.showV && metrics.showH
    }
    ```
  - Tests: Extend `editor/render_test.go` with cases for vertical-only, horizontal-only (`WrapNone`), and both-axis corner rendering — expect no text overlap and correct thumb placement.

## Phase 4: Mouse Interaction (Thumb Drag + Track Click + Wheel)
- [x] Add scrollbar hit-testing and drag state in mouse update flow — Enables direct manual navigation and paging behavior.
  - Repository: `flourish`
  - Component: `editor` input/update
  - Scope: Update `editor/model.go` state (drag axis/origin fields) and `editor/update_mouse.go`:
    - Detect press on vertical/horizontal scrollbar regions.
    - Track-click pages by visible span.
    - Thumb drag maps pointer delta to offset delta.
    - Release clears drag state.
    - Respect `ScrollPolicy` (`ScrollFollowCursorOnly` blocks manual interactions).
  - Snippets:
    ```go
    type scrollbarDragAxis int

    const (
    	dragNone scrollbarDragAxis = iota
    	dragVertical
    	dragHorizontal
    )
    ```
    ```go
    // update_mouse.go
    if m.cfg.ScrollPolicy == ScrollAllowManual {
    	if handled := m.handleScrollbarMouse(msg); handled {
    		return m, nil
    	}
    }
    ```
  - Tests: Add `editor/update_test.go` and/or `editor/scrollbar_mouse_test.go` cases for drag, page clicks, and policy blocking — expect offset changes only when manual scrolling is allowed.

## Phase 5: Mapping, Snapshot, and Overlay Consistency
- [ ] Keep host-facing state/mapping/snapshot coherent with scrollbar-reserved geometry — Prevents API inconsistencies and stale host caches.
  - Repository: `flourish`
  - Component: `editor` snapshot + viewport + hit-test + completion popup
  - Scope:
    - `editor/viewport_state.go`: `VisibleRows` reflects content area (excluding horizontal scrollbar row).
    - `editor/hittest.go`: scrollbar cells are not treated as text cells for internal scrollbar handling.
    - `editor/snapshot.go`: include scrollbar config in signature hashing.
    - `editor/completion_popup_render.go`: popup bounds use content area, not raw viewport area.
  - Snippets:
    ```go
    // snapshot signature additions
    scrollbarVMode ScrollbarMode
    scrollbarHMode ScrollbarMode
    scrollbarMinThumb int
    ```
    ```go
    // viewport_state.go
    return ViewportState{
    	TopVisualRow:   top,
    	VisibleRows:    metrics.contentHeight,
    	LeftCellOffset: left,
    	WrapMode:       m.cfg.WrapMode,
    }
    ```
  - Tests: Extend `editor/snapshot_test.go`, `editor/viewport_state_test.go`, `editor/completion_popup_render_test.go`, and `editor/hittest_test.go` — expect token invalidation on scrollbar config changes and correct content-area bounds.

## Phase 6: Documentation + End-to-End Verification
- [ ] Update docs and run full package verification — Keeps behavior discoverable and ensures integration safety.
  - Repository: `flourish`
  - Component: `docs` + `editor`
  - Scope:
    - Update `docs/editor.md` with scrollbar config, style fields, visibility rules, and `ScrollPolicy` interaction.
    - Cross-reference `design/scrollbar.md` and this roadmap.
    - Run targeted and full tests.
  - Snippets:
    ```bash
    go test ./editor -count=1
    go test ./... -count=1
    ```
  - Tests: Manual validation in `examples/simple` with small viewport and long lines, plus automated suite — expect stable rendering and no regressions in selection/cursor behavior.
