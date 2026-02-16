---
phase: quick-3
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/tui/model.go
autonomous: true
must_haves:
  truths:
    - "Pressing 'a' while search results are visible opens the Add Server form"
    - "Pressing 'e' while search results are visible opens the Edit Server form for the selected server"
    - "Pressing 'p' while search results are visible opens the Project Picker for the selected server"
    - "Pressing 'd' while search results are visible opens the Delete Confirmation for the selected server"
    - "Pressing 'u' while search results are visible triggers undo"
    - "Pressing 's' while search results are visible triggers 1Password sign-in"
    - "Pressing '?' while search results are visible toggles the help overlay"
    - "Pressing 'q' while search results are visible quits the application"
    - "Pressing 'g' while search results are visible jumps to the top of the list"
    - "Pressing 'G' while search results are visible jumps to the bottom of the list"
    - "Pressing ctrl+u/ctrl+d while search results are visible scrolls half page up/down"
    - "Pressing 'j'/'k' while search results are visible navigates the list (same as arrow keys)"
    - "Typing normal characters that are NOT action keys still filters the search input"
    - "Search bar retains focus and filter text after action keys are processed"
  artifacts:
    - path: "internal/tui/model.go"
      provides: "Search-mode action key handling in Update()"
      contains: "key.Matches(msg, m.keys.AddServer)"
  key_links:
    - from: "internal/tui/model.go (searchFocused block)"
      to: "internal/tui/model.go (list mode block)"
      via: "Shared action key handlers"
      pattern: "searchFocused.*keys\\.(AddServer|EditServer|AssignProject|DeleteServer)"
---

<objective>
Fix search mode swallowing action shortcut keys ('a', 'e', 'p', 'd', 'u', 's', '?', 'q', 'j', 'k', 'g', 'G', ctrl+u, ctrl+d) by treating them as text input instead of commands.

Purpose: When users search for servers and find results, they expect to interact with those results using the same keyboard shortcuts as in normal list mode. Currently, pressing any action key while search is focused just types that character into the search bar.

Output: Updated `internal/tui/model.go` where the `searchFocused` switch block handles all action and navigation keys before falling through to the default text-input case.
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
  <name>Task 1: Add action and navigation key handlers to search-focused switch block</name>
  <files>internal/tui/model.go</files>
  <action>
In the `Update()` method, find the `if m.searchFocused {` block (around line 955). The current switch statement only handles: ClearSearch (Esc), Connect (Enter), searchNavKeys (up/down arrows), and Details (Tab/i). Everything else falls to `default` which sends keys to `m.searchInput.Update(msg)` as text input.

Add explicit cases for ALL action keys BEFORE the `default` case. The cases must mirror the list-mode handlers (lines 1017-1170) but with search mode awareness:

1. **Quit** (`q`): `key.Matches(msg, m.keys.Quit)` -- return `m, tea.Quit`

2. **Help** (`?`): `key.Matches(msg, m.keys.Help)` -- toggle help overlay (same as list mode lines 1154-1163)

3. **AddServer** (`a`): `key.Matches(msg, m.keys.AddServer)` -- create new server form, set `m.viewMode = ViewAdd` (same as list mode lines 1066-1073)

4. **EditServer** (`e`): `key.Matches(msg, m.keys.EditServer)` -- get selected item, create edit form, set `m.viewMode = ViewEdit` (same as list mode lines 1075-1086)

5. **AssignProject** (`p`): `key.Matches(msg, m.keys.AssignProject)` -- get selected item, create picker, set `m.showingPicker = true` (same as list mode lines 1052-1064)

6. **DeleteServer** (`d`): `key.Matches(msg, m.keys.DeleteServer)` -- get selected item, create delete confirm, set `m.viewMode = ViewDelete` (same as list mode lines 1088-1099)

7. **Undo** (`u`): `key.Matches(msg, m.keys.Undo)` -- pop undo buffer and restore (same as list mode lines 1101-1121)

8. **SignIn** (`s`): `key.Matches(msg, m.keys.SignIn)` -- trigger 1Password sync if backend available (same as list mode lines 1123-1128)

9. **GoToTop** (`g`/Home): `key.Matches(msg, m.keys.GoToTop)` -- `m.list.Select(0)` (same as list mode lines 1130-1132)

10. **GoToBottom** (`G`/End): `key.Matches(msg, m.keys.GoToBottom)` -- `m.list.Select(len(m.list.Items()) - 1)` (same as list mode lines 1134-1136)

11. **HalfPageUp** (ctrl+u): `key.Matches(msg, m.keys.HalfPageUp)` -- half-page scroll up (same as list mode lines 1138-1144)

12. **HalfPageDown** (ctrl+d): `key.Matches(msg, m.keys.HalfPageDown)` -- half-page scroll down (same as list mode lines 1146-1152)

Also update the existing `searchNavKeys` case to include `j` and `k` keys alongside `up`/`down`. The simplest approach: change the `searchNavKeys` binding initialization (around line 138) to include `"j"`, `"k"` in addition to `"up"`, `"down"`:
```go
searchNavKeys := key.NewBinding(key.WithKeys("up", "down", "j", "k"))
```
This way j/k navigate the list instead of being typed into the search box.

IMPORTANT: Do NOT clear search or blur the search input when action keys are pressed. The search bar should stay focused with the current filter text visible. The user is performing an action on a FILTERED result, not leaving search mode.

IMPORTANT: For action keys that switch viewMode (Add, Edit, Delete, AssignProject), the search state will naturally be preserved. When the user returns from that view, they should still see their filtered results. Do NOT add any `m.searchFocused = false` or `m.searchInput.Blur()` calls for these actions.

Also add `m.statusMsg = ""` at the top of the searchFocused block (before the switch), matching the pattern in the list mode block (line 1007), so status messages clear on action.
  </action>
  <verify>
Run `go build ./...` from the project root to confirm compilation succeeds. Then run `go vet ./...` to check for issues. Then manually verify the switch block has all the new cases by reading the updated code.
  </verify>
  <done>
All action keys (a, e, p, d, u, s, ?, q) and navigation keys (j, k, g, G, ctrl+u, ctrl+d) are handled as commands in search mode, matching their list-mode behavior. Only characters that do NOT match any key binding are passed to the search input as text. The app compiles cleanly.
  </done>
</task>

<task type="auto">
  <name>Task 2: Verify search text input is not broken for non-action characters</name>
  <files>internal/tui/model.go</files>
  <action>
After Task 1, verify that the `default` case in the search-focused switch block still correctly handles non-action characters (letters like 'b', 'c', 'f', 'h', 'l', 'm', 'n', 'o', 'r', 't', 'v', 'w', 'x', 'y', 'z', numbers, symbols like '-', '.', '_') by passing them to `m.searchInput.Update(msg)` and calling `m.filterHosts()`.

Review the complete switch statement to ensure:
1. No duplicate key handling (each key binding appears exactly once)
2. The `default` case remains as the last case and passes unmatched keys to search input
3. No key binding conflicts (e.g., 'g' is GoToTop but 'g' typed fast might conflict -- verify that `key.Matches` uses the exact binding which is just 'g', so single 'g' press will always match GoToTop. This is the correct behavior since search is for filtering, not typing -- the always-on search bar decision from Phase 03-02 means the search input captures text BEFORE entering search mode via '/', and action keys should work on results)

Run `go build ./...` and `go vet ./...` to confirm everything is clean.
  </action>
  <verify>
Run `go build ./...` and `go vet ./...` -- both must pass with zero errors and zero warnings. Read the final switch block to confirm all cases are present and the default case handles text input correctly.
  </verify>
  <done>
The search-focused switch block correctly routes action keys to their handlers and non-action keys to the search text input. Build and vet pass cleanly. No regressions introduced.
  </done>
</task>

</tasks>

<verification>
1. `go build ./...` passes -- confirms compilation
2. `go vet ./...` passes -- confirms no static analysis issues
3. Manual code review: the `searchFocused` block has cases for ClearSearch, Connect, Details, searchNavKeys (up/down/j/k), Quit, Help, AddServer, EditServer, AssignProject, DeleteServer, Undo, SignIn, GoToTop, GoToBottom, HalfPageUp, HalfPageDown, and a default case for text input
4. The `searchNavKeys` binding includes j/k in addition to up/down
</verification>

<success_criteria>
- All 14 action/navigation key bindings are handled in search mode (not swallowed as text input)
- Non-action characters still filter the search input correctly
- Search state (focus, filter text) is preserved when action keys trigger view changes
- `go build ./...` and `go vet ./...` pass cleanly
</success_criteria>

<output>
After completion, create `.planning/quick/3-fix-search-results-not-responding-to-sho/3-SUMMARY.md`
</output>
