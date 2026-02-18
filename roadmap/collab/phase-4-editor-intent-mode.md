# Phase 4: Editor Intent Mode

Scope: Add an intent-emission mode to `editor.Model` so hosts can decide mutation strategy (local apply, remote send, or both) without duplicating key semantics.

Documentation: `design/collab-editing-best-practices.md`, `research/collab.md`, `docs/editor.md`

Legend: [ ] todo, [x] done.

## Intent API Surface
- [x] Introduce intent mode types and config hooks — separate intent capture from mutation execution
  - Repository: `flourish`
  - Component: `editor`
  - Scope: add `MutationMode`, `IntentKind`, `Intent`, `IntentBatch`, `IntentDecision`, `OnIntent` config hook
  - Snippets: `OnIntent func(IntentBatch) IntentDecision`
  - Tests: compile-level API tests and default behavior tests — existing mutate behavior remains default

## Input Pipeline Integration
- [x] Refactor key handling to produce typed intents before mutation — guarantee single source of key semantics
  - Repository: `flourish`
  - Component: `editor` input/update path
  - Scope: update key processing to emit intent batches for insert/delete/move/select/undo/redo/paste
  - Snippets: internal `buildIntentsFromKey(msg)` helper
  - Tests: per-key intent emission tests including selection-aware delete and paste

- [x] Implement mode-specific execution behavior — support `MutateInEditor`, `EmitIntentsOnly`, `EmitIntentsAndMutate`
  - Repository: `flourish`
  - Component: `editor`
  - Scope: gate mutation execution path by `MutationMode` and `IntentDecision.ApplyLocally`
  - Snippets: switch on mode after intent generation
  - Tests: parity tests comparing final buffer state across modes for equivalent local-apply decisions

## Event and Change Integration
- [x] Ensure `OnChange` emission remains coherent with intent execution decisions — avoid duplicate or missing change events
  - Repository: `flourish`
  - Component: `editor`
  - Scope: fire `OnChange` only when actual local mutation occurs; no event on intents-only path
  - Snippets: version-before/version-after guard around local mutation
  - Tests: intent-only mode emits intents without change events; mutate modes emit both correctly

## Quality Gates
- [x] Update `docs/editor.md` and examples to document intent mode behavior and host responsibilities — make integration deterministic
  - Repository: `flourish`
  - Component: docs + examples
  - Scope: add intent mode section and minimal host callback example
  - Snippets: callback example mapping insert intent to host transport
  - Tests: manual example run confirms intent payload and local-apply decisions
