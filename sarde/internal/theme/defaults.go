package theme

// DefaultTokens returns the hardcoded default light-mode tokens.
// This is layer 0 — the absolute fallback ensuring every expected
// CSS variable exists even with no theme.yaml.
func DefaultTokens() map[string]string {
	return map[string]string{
		"bg":         "#ffffff",
		"bg-surface": "#f8fafc",
		"bg-code":    "#1e293b",
		"text":       "#1e293b",
		"text-muted": "#64748b",
		"border":     "#e2e8f0",
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
		"text-accent":     "var(--sd-accent-high, var(--sd-accent, #818cf8))",
		"text-invert":     "var(--sd-black)",
		"text-success":    "var(--sd-green-high)",
		"text-warning":    "var(--sd-amber-high)",
		"text-danger":     "var(--sd-red-high)",
		// Semantic backgrounds
		"bg":              "var(--sd-black)",
		"bg-surface":      "var(--sd-gray-9)",
		"bg-inline-code":  "var(--sd-gray-8)",
		"bg-accent-subtle": "var(--sd-accent-low, rgba(99, 102, 241, 0.15))",
		"hover":           "var(--sd-gray-8)",
		// Borders
		"border":          "var(--sd-gray-8)",
		"border-muted":    "var(--sd-gray-7)",
		"border-accent":   "var(--sd-accent-high, var(--sd-accent, #818cf8))",
		"hairline":        "var(--sd-gray-8)",
		// Code
		"code-bg":         "var(--sd-gray-9)",
		"code-border":     "var(--sd-gray-7)",
		// Shadows
		"shadow-sm":       "0 1px 3px rgba(0, 0, 0, 0.25)",
		"shadow-md":       "0 4px 6px -1px rgba(0, 0, 0, 0.3)",
		"shadow-lg":       "0 10px 15px -3px rgba(0, 0, 0, 0.35)",
		"shadow-accent":   "0 4px 14px -2px rgba(129, 140, 248, 0.2)",
		// Glass
		"glass-bg":        "rgba(0, 0, 0, 0.6)",
		"glass-border":    "rgba(255, 255, 255, 0.08)",
		// Aside variants
		"aside-note-bg":   "rgba(59, 130, 246, 0.1)",
		"aside-tip-bg":    "rgba(34, 197, 94, 0.1)",
		"aside-caution-bg": "rgba(245, 158, 11, 0.1)",
		"aside-danger-bg": "rgba(239, 68, 68, 0.1)",
	}
}
