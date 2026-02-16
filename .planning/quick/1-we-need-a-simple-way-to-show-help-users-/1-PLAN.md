---
phase: quick-1
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - cmd/ssherpa/main.go
  - internal/tui/help_overlay.go
  - internal/tui/model.go
  - internal/tui/keys.go
  - internal/tui/styles.go
autonomous: true

must_haves:
  truths:
    - "User can see the full list of 1Password fields and their effects by running ssherpa --fields"
    - "User can press ? in the TUI list view to see a help overlay with field reference"
    - "User can dismiss the help overlay with Esc or ? to return to the server list"
  artifacts:
    - path: "internal/tui/help_overlay.go"
      provides: "Help overlay view with 1Password field reference table"
    - path: "cmd/ssherpa/main.go"
      provides: "--fields CLI flag that prints field reference and exits"
  key_links:
    - from: "cmd/ssherpa/main.go"
      to: "internal/tui/help_overlay.go"
      via: "shared field reference content"
      pattern: "fieldsFlag|--fields"
    - from: "internal/tui/model.go"
      to: "internal/tui/help_overlay.go"
      via: "ViewHelp mode renders overlay"
      pattern: "ViewHelp|showingHelp"
---

<objective>
Add a 1Password field reference that users can access from (1) a `--fields` CLI flag and (2) a `?` help overlay inside the TUI.

Purpose: Users who create 1Password entries for ssherpa need to know which fields are recognized, which are required, what format to use, and what each field does. Currently this info only appears briefly during initial setup.

Output: A `--fields` CLI flag that prints the field reference to stdout, and a `?` key in the TUI that opens a scrollable help overlay showing the same reference.
</objective>

<execution_context>
@/Users/florianriquelme/.claude/get-shit-done/workflows/execute-plan.md
@/Users/florianriquelme/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@internal/backend/onepassword/mapping.go (source of truth for 1Password field names)
@internal/tui/model.go (TUI model with ViewMode and key handling)
@internal/tui/keys.go (key bindings definition)
@internal/tui/wizard.go (existing partial field template in renderOnePasswordSetup)
@cmd/ssherpa/main.go (CLI entry point with existing flags)
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add --fields CLI flag and help overlay view</name>
  <files>
    internal/tui/help_overlay.go
    cmd/ssherpa/main.go
  </files>
  <action>
1. Create `internal/tui/help_overlay.go` with a `RenderFieldReference()` function that returns a formatted string containing the complete 1Password field reference. The reference must include:

   - A title: "1Password Field Reference"
   - The item-level properties:
     - Title: display name for the server (becomes the alias in ssherpa)
     - Category: must be "Server"
     - Tag: must include "ssherpa" (case-insensitive)
   - A table of all recognized fields from mapping.go:
     | Field | Required | Default | Description |
     |-------|----------|---------|-------------|
     | hostname | yes | - | Server hostname or IP address |
     | user | yes | - | SSH username |
     | port | no | 22 | SSH port number |
     | identity_file | no | SSH default | Path to SSH private key file |
     | proxy_jump | no | - | Bastion/jump host for ProxyJump |
     | project_tags | no | - | Comma-separated project tags (e.g. "web,api") |
     | remote_project_path | no | - | Remote path to cd into on connect |
     | forward_agent | no | - | Enable SSH agent forwarding (noted, not yet mapped) |
     | extra_config | no | - | Additional SSH config directives (noted, not yet mapped) |
   - A minimal example showing a 1Password entry with required + one optional field
   - A note that fields are case-insensitive

   Use lipgloss styles consistent with the project's style palette (accentColor, secondaryColor, etc.) for the styled version. Also provide a plain-text version (`RenderFieldReferencePlain()`) for CLI output (no ANSI codes, clean for piping).

2. Also create in the same file: a `HelpOverlay` struct with `View()` method that wraps the styled field reference in a bordered overlay (similar to pickerBorderStyle but wider, ~70 chars). Include a `viewport.Model` so the content is scrollable when the terminal is small. The overlay should show footer text: "Esc or ?: close | arrow keys: scroll".

3. In `cmd/ssherpa/main.go`:
   - Add a `fieldsFlag` boolean flag: `flag.Bool("fields", false, "Show 1Password field reference")`
   - Handle it right after the `--version` check (before any backend initialization): if `*fieldsFlag`, call `tui.RenderFieldReferencePlain()`, print to stdout, and `os.Exit(0)`.
  </action>
  <verify>
    Run `go build -o /dev/null ./cmd/ssherpa/` to verify compilation. Then `go run ./cmd/ssherpa/ --fields` should print the field reference table to stdout without launching the TUI.
  </verify>
  <done>
    `ssherpa --fields` prints a complete, readable 1Password field reference to stdout and exits. The help_overlay.go file exports HelpOverlay, RenderFieldReference, and RenderFieldReferencePlain.
  </done>
</task>

<task type="auto">
  <name>Task 2: Wire ? key to toggle help overlay in TUI</name>
  <files>
    internal/tui/keys.go
    internal/tui/model.go
    internal/tui/styles.go
  </files>
  <action>
1. In `internal/tui/keys.go`:
   - Add a `Help` field to `KeyMap`: `Help key.Binding`
   - Initialize it in `DefaultKeyMap()`: `key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help"))`
   - Add `k.Help` to the end of `ShortHelp()` return slice (so users can see it in the footer)
   - Add `k.Help` to the last group in `FullHelp()` alongside Undo and SignIn

2. In `internal/tui/model.go`:
   - Add `showingHelp bool` and `helpOverlay *HelpOverlay` fields to Model struct (alongside similar fields like showingPicker, showingKeyPicker)
   - In the `ViewList` key handling (the non-search-focused else branch), add a case for `key.Matches(msg, m.keys.Help)`:
     - If `m.showingHelp` is true, close it (set false, nil the overlay)
     - Otherwise, create a new HelpOverlay with `NewHelpOverlay(m.width, m.height)` and set `m.showingHelp = true`
   - Also handle `?` when `m.showingHelp` is true: at the TOP of the ViewList key handling (before search-focused check), add:
     ```
     if m.showingHelp {
       switch {
       case key.Matches(msg, m.keys.Help) || key.Matches(msg, m.keys.ClearSearch):
         m.showingHelp = false
         m.helpOverlay = nil
         return m, nil
       default:
         // Route arrow/scroll keys to help overlay viewport
         if m.helpOverlay != nil {
           m.helpOverlay.viewport, _ = m.helpOverlay.viewport.Update(msg)
         }
         return m, nil
       }
     }
     ```
   - In the `View()` method, in the `ViewList` case, AFTER the key picker overlay check and BEFORE `return baseView`: add a check for `m.showingHelp && m.helpOverlay != nil` that centers the overlay using `lipgloss.Place` (same pattern as picker overlay).
   - Handle `tea.WindowSizeMsg` for the help overlay: if `m.showingHelp && m.helpOverlay != nil`, update its viewport dimensions.

3. In `internal/tui/styles.go`:
   - Add a `helpOverlayStyle` similar to `pickerBorderStyle` but with Width(72) for the wider content.
  </action>
  <verify>
    Run `go build -o /dev/null ./cmd/ssherpa/` to verify compilation. Run `go vet ./...` to check for issues.
  </verify>
  <done>
    Pressing `?` in the TUI list view opens a scrollable help overlay showing the 1Password field reference. Pressing `?` or `Esc` dismisses it. The `?` key appears in the help footer.
  </done>
</task>

</tasks>

<verification>
1. `go build -o /dev/null ./cmd/ssherpa/` compiles without errors
2. `go vet ./...` passes
3. `go run ./cmd/ssherpa/ --fields` prints field reference to stdout
4. Existing tests pass: `go test ./...`
</verification>

<success_criteria>
- `ssherpa --fields` prints the complete 1Password field reference and exits
- `?` key in TUI list view opens a help overlay with field reference
- `Esc` or `?` dismisses the overlay
- Help footer shows `?` as available key
- All existing tests pass
- No regressions in TUI behavior
</success_criteria>

<output>
After completion, create `.planning/quick/1-we-need-a-simple-way-to-show-help-users-/1-SUMMARY.md`
</output>
