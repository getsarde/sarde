package links

import (
	"fmt"
	"os"
	"strings"
)

func writeGitHubActionsReport(sb *strings.Builder, findings []Finding, cov CoverageSummary, summary string, hasErrors bool) {
	// Write to $GITHUB_OUTPUT if available.
	if outputFile := os.Getenv("GITHUB_OUTPUT"); outputFile != "" {
		val := "false"
		if hasErrors {
			val = "true"
		}
		_ = appendToFile(outputFile, "link_validation_failed="+val+"\n")
	}

	// Write Markdown summary to $GITHUB_STEP_SUMMARY if available.
	if summaryFile := os.Getenv("GITHUB_STEP_SUMMARY"); summaryFile != "" {
		md := buildMarkdownSummary(findings, cov, summary)
		_ = appendToFile(summaryFile, md)
	}

	// Always produce human-readable output (without colors for CI logs).
	writePlainReport(sb, findings, cov, summary)
}

func buildMarkdownSummary(findings []Finding, cov CoverageSummary, summary string) string {
	var sb strings.Builder
	sb.WriteString("## Link Validation\n\n")

	if len(findings) == 0 {
		sb.WriteString(summary)
		sb.WriteByte('\n')
		return sb.String()
	}

	sb.WriteString("| File | Dest | Type | Policy |\n")
	sb.WriteString("|------|------|------|--------|\n")
	for _, f := range findings {
		dest := f.Ref.RawDest
		if f.Ref.Fragment != "" && !strings.Contains(dest, "#") {
			dest += "#" + f.Ref.Fragment
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			f.Ref.FromFile, dest, f.Type.String(), f.Policy))
	}
	sb.WriteString("\n**")
	sb.WriteString(summary)
	sb.WriteString("**\n")
	return sb.String()
}

func writePlainReport(sb *strings.Builder, findings []Finding, cov CoverageSummary, summary string) {
	sb.WriteString("link check: ")
	sb.WriteString(summary)
	sb.WriteByte('\n')

	if len(findings) == 0 {
		return
	}
	sb.WriteByte('\n')

	groups := groupByFile(findings)
	for _, g := range groups {
		sb.WriteString(g.File)
		sb.WriteByte('\n')
		for _, f := range g.Findings {
			dest := f.Ref.RawDest
			if f.Ref.Fragment != "" && !strings.Contains(dest, "#") {
				dest += "#" + f.Ref.Fragment
			}
			policyTag := strings.ToUpper(f.Policy)
			sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", policyTag, f.Type.Label(), dest))
		}
		sb.WriteByte('\n')
	}
}

func appendToFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
