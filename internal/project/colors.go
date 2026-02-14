package project

import (
	"fmt"
	"hash/fnv"
	"math"

	"github.com/charmbracelet/lipgloss"
)

// ProjectColor generates a deterministic, visually distinct color for a project ID.
// Uses FNV-1a hash to generate a hue, then converts to hex color strings suitable
// for lipgloss.AdaptiveColor.
//
// Returns different colors for light and dark terminals:
//   - Light terminal: 40% lightness (darker, more readable on white)
//   - Dark terminal: 70% lightness (lighter, more readable on black)
//
// Saturation is fixed at 65% for vibrant but professional colors.
func ProjectColor(projectID string) lipgloss.AdaptiveColor {
	// Use FNV-1a hash for deterministic hue generation
	h := fnv.New32a()
	h.Write([]byte(projectID))
	hash := h.Sum32()

	// Convert hash to hue (0-359 degrees)
	hue := float64(hash%360) // Modulo 360 for valid hue range

	// Fixed saturation and different lightness for light/dark terminals
	saturation := 0.65

	lightColor := hslToHex(hue, saturation, 0.40) // Darker for light terminals
	darkColor := hslToHex(hue, saturation, 0.70)  // Lighter for dark terminals

	return lipgloss.AdaptiveColor{
		Light: lightColor,
		Dark:  darkColor,
	}
}

// hslToHex converts HSL color values to a hex color string.
// h: hue (0-359), s: saturation (0-1), l: lightness (0-1)
// Returns hex color string like "#5A67D8"
func hslToHex(h, s, l float64) string {
	// Normalize hue to 0-1 range
	h = h / 360.0

	var r, g, b float64

	if s == 0 {
		// Achromatic (gray)
		r, g, b = l, l, l
	} else {
		// Chromatic color
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q

		r = hueToRGB(p, q, h+1.0/3.0)
		g = hueToRGB(p, q, h)
		b = hueToRGB(p, q, h-1.0/3.0)
	}

	// Convert to 0-255 range and format as hex
	rByte := uint8(math.Round(r * 255))
	gByte := uint8(math.Round(g * 255))
	bByte := uint8(math.Round(b * 255))

	return fmt.Sprintf("#%02X%02X%02X", rByte, gByte, bByte)
}

// hueToRGB is a helper function for HSL to RGB conversion
func hueToRGB(p, q, t float64) float64 {
	// Normalize t to 0-1 range
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}

	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}
