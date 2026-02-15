# ssherpa demo recording
# Generate: vhs scripts/demo.tape

Output demo.gif
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20

# Launch ssherpa
Type "ssherpa"
Sleep 500ms
Enter

# Wait for TUI to load
Sleep 1.5s

# Navigate the server list
Down 3
Sleep 400ms
Up 1
Sleep 400ms

# Search for a server
Type "/"
Sleep 300ms
Type "prod"
Sleep 800ms

# Clear search
Escape
Sleep 500ms

# Show detail view
Type "d"
Sleep 1.5s

# Return to list
Escape
Sleep 500ms

# Exit
Type "q"
Sleep 500ms
