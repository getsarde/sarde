package theme

// DefaultTokens returns the hardcoded default light-mode tokens.
// This is layer 0 — the absolute fallback ensuring every expected
// CSS variable exists even with no theme.yaml.
func DefaultTokens() map[string]string {
	return map[string]string{
		"primary":    "#6366f1",
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
func DefaultDarkTokens() map[string]string {
	return map[string]string{
		"bg":         "#0f172a",
		"bg-surface": "#1e293b",
		"bg-code":    "#0f172a",
		"text":       "#e2e8f0",
		"text-muted": "#94a3b8",
		"border":     "#334155",
	}
}
