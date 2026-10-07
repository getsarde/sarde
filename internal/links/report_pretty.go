package links

import (
	"fmt"
	"strings"

	"github.com/getsarde/sarde/internal/devlog"
)

// writePrettyReport prints the summary line, then one line per finding
// (duplicates within a file collapsed to "(xN)"), led by a path:line:col
// location that terminals can open.
func writePrettyReport(sb *strings.Builder, findings []Finding, cov CoverageSummary, _ string) {
	sb.WriteString(devlog.FormatLog("links", prettySummary(cov, findings)))
	sb.WriteByte('\n')

	for _, g := range groupByFile(findings) {
		for _, d := range deduplicateFindings(g.Findings) {
			ref := d.finding.Ref
			dest := ref.RawDest
			if ref.Fragment != "" && !strings.Contains(dest, "#") {
				dest += "#" + ref.Fragment
			}

			line := fmt.Sprintf("  %s  %s  %-16s %s",
				policyLabel(d.finding.Policy), findingLocation(ref), d.finding.Type.Label(), devlog.Dim(dest))
			if d.count > 1 {
				line += " " + devlog.Dim(fmt.Sprintf("(x%d)", d.count))
			}

			dimStr := formatDim(ref.Dim)
			if dimStr != "" {
				line += "  " + devlog.Dim(dimStr)
			}
			if hint := d.finding.Type.Hint(); hint != "" {
				line += "  " + devlog.Dim("("+hint+")")
			}

			sb.WriteString(line)
			sb.WriteByte('\n')
		}
	}
}

// findingLocation renders a ref's source position as path[:line[:col]].
func findingLocation(ref LinkRef) string {
	loc := devlog.DisplayPath(ref.FromFile)
	if ref.Line > 0 {
		loc += fmt.Sprintf(":%d", ref.Line)
		if ref.Col > 0 {
			loc += fmt.Sprintf(":%d", ref.Col)
		}
	}
	return loc
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

// prettySummary reports the link count and the findings split by policy:
// "checked 812 links across 1 lane: no issues" or "...: 1 error, 2 warnings".
func prettySummary(cov CoverageSummary, findings []Finding) string {
	outcome := "no issues"
	if len(findings) > 0 {
		c := CountFindings(findings)
		var parts []string
		if c.Errors > 0 {
			parts = append(parts, devlog.Red(plural(c.Errors, "error", "errors")))
		}
		if c.Warnings > 0 {
			parts = append(parts, devlog.Yellow(plural(c.Warnings, "warning", "warnings")))
		}
		outcome = strings.Join(parts, ", ")
	}
	return fmt.Sprintf("%s %s %s %s %s",
		devlog.Green("checked"), devlog.Bold(fmt.Sprint(cov.TotalLinks)),
		devlog.Green(pluralWord(cov.TotalLinks, "link", "links")+" across"),
		devlog.Bold(fmt.Sprint(cov.TotalLanes)),
		devlog.Green(pluralWord(cov.TotalLanes, "lane", "lanes")+":")+" "+outcome)
}

// plural formats a count with the matching noun form: "1 error", "2 errors".
func plural(n int, one, many string) string {
	return fmt.Sprintf("%d %s", n, pluralWord(n, one, many))
}

func pluralWord(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
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
