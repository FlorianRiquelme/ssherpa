---
phase: quick-4
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/tui/model.go
  - internal/tui/keys.go
autonomous: true

must_haves:
  truths:
    - "User can type any letter (a, e, p, d, u, s, q, g, etc.) into search input when focused"
    - "Pressing Esc while search is focused blurs the input but keeps the filter text and filtered results"
    - "Action keys (a, e, p, d, u, s, q, ?, g, G, ctrl+u, ctrl+d) work on filtered results after Esc blur"
    - "Pressing Esc when search is already blurred AND filter text exists clears the filter and shows all hosts"
    - "Pressing / re-focuses search input to edit the existing filter text"
  artifacts:
    - path: "internal/tui/model.go"
      provides: "Two-phase search: typing mode vs filter-active mode"
    - path: "internal/tui/keys.go"
      provides: "Updated ClearSearch help text"
  key_links:
    - from: "searchFocused block"
      to: "else (list mode) block"
      via: "Esc blurs without clearing, list mode handles action keys on filtered results"
      pattern: "searchFocused.*false.*filterHosts"
---

<objective>
Implement two-phase search UX: Phase 1 (typing mode) lets all letter keys flow to the search input; Phase 2 (filter-active mode) keeps the filter visible but blurs input so action keys work on filtered results.

Purpose: Quick task 3 broke search typing by adding action key handlers inside the searchFocused block. This fix reverts that approach and implements the correct two-phase behavior where Esc transitions between modes.
Output: Updated model.go with correct search key handling, updated keys.go with accurate help text.
</objective>

<execution_context>
@/Users/florianriquelme/.claude/get-shit-done/workflows/execute-plan.md
@/Users/florianriquelme/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@internal/tui/model.go
@internal/tui/keys.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Revert searchFocused action keys and implement two-phase Esc behavior</name>
  <files>internal/tui/model.go</files>
  <action>
  In `internal/tui/model.go`, make these changes to the Update() method:

  1. **Remove `m.statusMsg = ""` at line 957** — this was added by quick-3 and is not needed in typing mode. The list-mode block already clears status on key press.

  2. **Change the ClearSearch (Esc) handler in the searchFocused block (lines 961-967)**: Instead of clearing search text + blurring + filtering, ONLY blur the input:
     ```go
     case key.Matches(msg, m.keys.ClearSearch):
         // Esc: exit typing mode, keep filter active
         m.searchInput.Blur()
         m.searchFocused = false
         return m, nil
     ```
     Do NOT call `m.searchInput.SetValue("")` or `m.filterHosts()`. The filter text stays, results stay filtered.

  3. **Remove ALL action key cases from the searchFocused block (lines 998-1104)**. Remove these cases entirely:
     - `m.keys.Quit` (lines 998-1000)
     - `m.keys.Help` (lines 1002-1011)
     - `m.keys.AddServer` (lines 1013-1020)
     - `m.keys.EditServer` (lines 1022-1032)
     - `m.keys.AssignProject` (lines 1034-1044)
     - `m.keys.DeleteServer` (lines 1046-1056)
     - `m.keys.Undo` (lines 1058-1073)
     - `m.keys.SignIn` (lines 1075-1080)
     - `m.keys.GoToTop` (lines 1082-1084)
     - `m.keys.GoToBottom` (lines 1086-1088)
     - `m.keys.HalfPageUp` (lines 1090-1096)
     - `m.keys.HalfPageDown` (lines 1098-1104)

     After removal, the searchFocused block should only contain: ClearSearch (Esc), Connect (Enter), searchNavKeys (up/down arrows), Details (Tab/i), and default (text input + re-filter).

  4. **Add Esc handler in the list-mode (else) block**: Add a new case at the TOP of the list-mode switch (before the Search "/" case, after `m.statusMsg = ""`) that handles Esc when there's active filter text:
     ```go
     case key.Matches(msg, m.keys.ClearSearch):
         // Esc in list mode: if filter text exists, clear it
         if m.searchInput.Value() != "" {
             m.searchInput.SetValue("")
             m.filterHosts()
         }
     ```
     This is the "clear filter" action for Phase 2.

  5. **Revert searchNavKeys** in the `New()` function (line 138): Change from `key.NewBinding(key.WithKeys("up", "down", "j", "k"))` to `key.NewBinding(key.WithKeys("up", "down"))`. Remove "j" and "k" since those letters need to be typeable in search mode. (j/k navigation works in list mode via the default delegate.)

  6. **Update "no matches" text** in the View() method (around line 1726): Replace the static message with context-sensitive text:
     ```go
     var noMatchHint string
     if m.searchFocused {
         noMatchHint = "Press Esc to exit search"
     } else {
         noMatchHint = "Press Esc to clear filter"
     }
     mainContent = noMatchesStyle.Render(fmt.Sprintf(
         "No matches for \"%s\"\n\n%s",
         m.searchInput.Value(),
         noMatchHint,
     ))
     ```
  </action>
  <verify>Run `go build ./...` to confirm compilation. Run `go test ./internal/tui/...` to confirm no test regressions.</verify>
  <done>The searchFocused block only handles Esc (blur), Enter (connect), arrows (navigate), Tab/i (details), and default (type + filter). All action keys are removed. Esc in list mode with filter text clears the filter. searchNavKeys uses only "up" and "down".</done>
</task>

<task type="auto">
  <name>Task 2: Update ClearSearch help text in keys.go</name>
  <files>internal/tui/keys.go</files>
  <action>
  In `internal/tui/keys.go`:

  1. **Update ClearSearch help text** (line 139): Change from `"clear search"` to `"exit search"` since Esc now blurs rather than clears:
     ```go
     ClearSearch: key.NewBinding(
         key.WithKeys("esc"),
         key.WithHelp("esc", "exit search"),
     ),
     ```

  2. **Update SearchKeyMap ShortHelp**: The SearchKeyMap is used for the help footer shown during search mode. Verify it references the updated ClearSearch binding (it should already since it's assigned from `keys.ClearSearch` in model.go New()). No code change needed here, just verify the binding propagates correctly — since SearchKeyMap.ClearSearch is assigned by value from keys.ClearSearch in New(), the updated help text will automatically appear.
  </action>
  <verify>Run `go build ./...` to confirm compilation. Grep for "clear search" in both files to confirm no stale references remain.</verify>
  <done>ClearSearch help text says "exit search" instead of "clear search". Help footer during search mode shows updated text.</done>
</task>

</tasks>

<verification>
1. `go build ./...` compiles without errors
2. `go test ./internal/tui/...` passes all existing tests
3. Manual verification: Launch the TUI, press `/`, type "apache" — all letters appear in search input (no action keys intercepted)
4. Manual verification: Press Esc — input blurs, filter text stays, results remain filtered, action keys now work
5. Manual verification: Press Esc again — filter clears, all hosts shown
6. Manual verification: Press `/` again — re-focuses search with empty input, ready to type
</verification>

<success_criteria>
- Typing mode allows ALL letter keys to reach search input (no a/e/p/d/u/s/q/g/G interception)
- Esc in typing mode blurs input but preserves filter text and filtered results
- Action keys work on filtered results after Esc blur (via existing list-mode handlers)
- Esc in list mode with active filter clears the filter
- Help text accurately reflects Esc behavior ("exit search" not "clear search")
- No matches text is context-sensitive (typing mode vs filter-active mode)
</success_criteria>

<output>
After completion, create `.planning/quick/4-implement-two-phase-search-esc-blurs-but/4-SUMMARY.md`
</output>
