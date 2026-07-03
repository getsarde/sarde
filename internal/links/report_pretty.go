package links

import (
	"fmt"
	"strings"

	"github.com/getsarde/sarde/internal/devlog"
)

func writePrettyReport(sb *strings.Builder, findings []Finding, cov CoverageSummary, _ string) {
	sb.WriteString(devlog.FormatLog("links", prettySummary(cov, findings)))
	sb.WriteByte('\n')
	if len(findings) == 0 {
		return
	}
	sb.WriteByte('\n')

	groups := groupByFile(findings)
	for _, g := range groups {
		sb.WriteString(devlog.Bold(g.File))
		sb.WriteByte('\n')

		deduped := deduplicateFindings(g.Findings)
		for _, d := range deduped {
			label := policyLabel(d.finding.Policy)
			typeLabel := d.finding.Type.Label()
			dest := d.finding.Ref.RawDest
			if d.finding.Ref.Fragment != "" && !strings.Contains(dest, "#") {
				dest += "#" + d.finding.Ref.Fragment
			}

			line := fmt.Sprintf("  %s  %-16s %s", label, typeLabel, devlog.Dim(dest))
			if d.count > 1 {
				line += " " + devlog.Dim(fmt.Sprintf("(x%d)", d.count))
			}

			dimStr := formatDim(d.finding.Ref.Dim)
			if dimStr != "" {
				line += "  " + devlog.Dim(dimStr)
			}

			sb.WriteString(line)
			sb.WriteByte('\n')
		}
		sb.WriteByte('\n')
	}
}

func policyLabel(policy string) string {
	switch policy {
	case "error":
		return devlog.Red("ERROR")
	case "warn":
		return devlog.Yellow(" WARN")
	default:
		return "     "
	}
}

func formatDim(dim DimKey) string {
	var parts []string
	if dim.Lang != "" {
		parts = append(parts, dim.Lang)
	}
	if dim.Version != "" {
		parts = append(parts, dim.Version)
	}
	if len(parts) == 0 {
		return ""
	}
	return "[" + strings.Join(parts, "/") + "]"
}

type dedupedFinding struct {
	finding Finding
	count   int
}

func prettySummary(cov CoverageSummary, findings []Finding) string {
	var brokenTargets, brokenAnchors, externalBroken, warnCount int
	for _, f := range findings {
		switch f.Type {
		case FindingBrokenTarget:
			brokenTargets++
		case FindingBrokenAnchor:
			brokenAnchors++
		case FindingExternalBroken:
			externalBroken++
		default:
			warnCount++
		}
	}
	return fmt.Sprintf("%s %s %s %s %s %d broken targets, %d broken anchors, %d broken external, %d warnings",
		devlog.Green("checked"), devlog.Bold(fmt.Sprint(cov.TotalLinks)),
		devlog.Green("links across"), devlog.Bold(fmt.Sprint(cov.TotalLanes)),
		devlog.Green("lanes:"),
		brokenTargets, brokenAnchors, externalBroken, warnCount)
}

func deduplicateFindings(findings []Finding) []dedupedFinding {
	if len(findings) == 0 {
		return nil
	}
	var result []dedupedFinding
	current := dedupedFinding{finding: findings[0], count: 1}
	for i := 1; i < len(findings); i++ {
		f := findings[i]
		if f.Ref.RawDest == current.finding.Ref.RawDest && f.Type == current.finding.Type {
			current.count++
		} else {
			result = append(result, current)
			current = dedupedFinding{finding: f, count: 1}
		}
	}
	result = append(result, current)
	return result
}
