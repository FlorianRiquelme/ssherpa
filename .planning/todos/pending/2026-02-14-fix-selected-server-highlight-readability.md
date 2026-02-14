---
created: 2026-02-14T20:51:59.563Z
title: Fix selected server highlight readability
area: ui
files:
  - internal/tui/list.go
---

## Problem

The highlight coloring of the selected server in the TUI list view makes the text unreadable. The current foreground/background color combination doesn't provide enough contrast when a server row is selected/focused.

## Solution

Adjust the selected item style in the list view to use a color combination with better contrast. Options:
- Change the highlight background to a darker/lighter shade
- Change the foreground text color when selected to ensure readability
- Use bold or inverse styling for the selected row
- Test with both light and dark terminal themes (AdaptiveColor is already used in the project)
