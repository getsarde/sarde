package theme

// DefaultTokens returns the hardcoded default light-mode tokens.
// This is layer 0 — the absolute fallback ensuring every expected
// CSS variable exists even with no theme.yaml.
func DefaultTokens() map[string]string {
	return map[string]string{
		"bg":         "oklch(1 0 0)",
		"bg-surface": "oklch(0.979 0.004 248)",
		"bg-code":    "oklch(0.228 0.023 255)",
		"text":       "oklch(0.228 0.023 255)",
		"text-muted": "oklch(0.493 0.025 252)",
		"border":     "oklch(0.916 0.009 253)",
		"font-sans":  "'Inter', system-ui, -apple-system, sans-serif",
		"font-mono":  "'JetBrains Mono', ui-monospace, monospace",
		"radius":     "0.5rem",
	}
}

// DefaultDarkTokens returns the hardcoded default dark-mode token overrides.
// These mirror the semantic remaps previously in dark.css, making all dark-mode
// values reachable via theme.dark_overrides in sarde.yaml.
func DefaultDarkTokens() map[string]string {
	return map[string]string{
		// Semantic text
		"text":            "var(--sd-gray-3)",
		"text-muted":      "var(--sd-gray-4)",
		"text-subtle":     "var(--sd-gray-5)",
		"text-accent":     "var(--sd-accent-high, var(--sd-accent, oklch(0.71 0.19 264)))",
		"text-invert":     "var(--sd-black)",
		"text-success":    "var(--sd-green-high)",
		"text-warning":    "var(--sd-amber-high)",
		"text-danger":     "var(--sd-red-high)",
		// Semantic backgrounds
		"bg":              "#1a1a1c",
		"bg-nav":          "#19191b",
		"bg-sidebar":      "#18181b",
		"bg-surface":      "var(--sd-gray-9)",
		"bg-inline-code":  "var(--sd-gray-8)",
		"bg-accent-subtle": "var(--sd-accent-low, oklch(0.60 0.22 264 / 0.15))",
		"hover":           "var(--sd-gray-8)",
		// Borders
		"border":          "var(--sd-gray-8)",
		"border-muted":    "var(--sd-gray-7)",
		"border-accent":   "var(--sd-accent-high, var(--sd-accent, oklch(0.71 0.19 264)))",
		"hairline":        "var(--sd-gray-8)",
		// Code
		"code-bg":         "var(--sd-gray-9)",
		"code-border":     "var(--sd-gray-7)",
		// Shadows
		"shadow-sm":       "0 1px 3px oklch(0 0 0 / 0.25)",
		"shadow-md":       "0 4px 6px -1px oklch(0 0 0 / 0.3)",
		"shadow-lg":       "0 10px 15px -3px oklch(0 0 0 / 0.35)",
		"shadow-accent":   "0 4px 14px -2px oklch(0.71 0.19 264 / 0.2)",
		// Glass
		"glass-bg":        "oklch(0 0 0 / 0.6)",
		"glass-border":    "oklch(1 0 0 / 0.08)",
		// Aside variants
		"aside-note-bg":    "oklch(0.62 0.20 231 / 0.1)",
		"aside-tip-bg":     "oklch(0.62 0.17 155 / 0.1)",
		"aside-info-bg":    "oklch(0.52 0.14 196 / 0.1)",
		"aside-caution-bg": "oklch(0.75 0.16 75 / 0.1)",
		"aside-danger-bg":  "oklch(0.55 0.20 25 / 0.1)",
		"aside-success-bg": "oklch(0.62 0.17 155 / 0.1)",
	}
}
