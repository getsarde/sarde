---
title: "About the Minimal Theme"
toc: false
---

The Minimal preset removes nearly all color, letting content speak for itself. Uses system fonts for maximum compatibility and fast loading.

## Token Values

| Token | Light | Dark |
|-------|-------|------|
| `primary` | `#18181b` | `#fafafa` |
| `bg` | `#ffffff` | `#09090b` |
| `bg-surface` | `#fafafa` | `#18181b` |
| `text` | `#18181b` | `#fafafa` |
| `text-muted` | `#a1a1aa` | `#71717a` |
| `border` | `#e4e4e7` | `#27272a` |
| `font-sans` | system-ui | (inherited) |
| `font-mono` | ui-monospace, SF Mono | (inherited) |
| `radius` | `0.375rem` | (inherited) |

```go
theme := sarde.Theme{Preset: "minimal"}
```

> The minimal preset is perfect for personal blogs, technical writing, and content-first sites.
