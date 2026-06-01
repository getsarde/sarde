package links

import (
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
)

// FindingType classifies what kind of issue a report finding represents.
type FindingType int

const (
	FindingBrokenTarget FindingType = iota
	FindingBrokenAnchor
	FindingRelativeLink
	FindingLocalLink
	FindingSameSite
	FindingExternalBroken
	FindingUnverifiedInternal
)

func (ft FindingType) String() string {
	switch ft {
	case FindingBrokenTarget:
		return "broken_target"
	case FindingBrokenAnchor:
		return "broken_anchor"
	case FindingRelativeLink:
		return "relative_link"
	case FindingLocalLink:
		return "local_link"
	case FindingSameSite:
		return "same_site"
	case FindingExternalBroken:
		return "external_broken"
	case FindingUnverifiedInternal:
		return "unverified_internal"
	default:
		return "unknown"
	}
}

func (ft FindingType) Label() string {
	switch ft {
	case FindingBrokenTarget:
		return "broken target"
	case FindingBrokenAnchor:
		return "broken anchor"
	case FindingRelativeLink:
		return "relative link"
	case FindingLocalLink:
		return "local link"
	case FindingSameSite:
		return "same site"
	case FindingExternalBroken:
		return "external broken"
	case FindingUnverifiedInternal:
		return "unverified internal"
	default:
		return "unknown"
	}
}

// Finding is a single reportable issue derived from a LinkRef.
type Finding struct {
	Type   FindingType
	Ref    LinkRef
	Policy string // "error" | "warn"
}

// LinkCheckConfig holds resolved policy values for report generation.
// Avoids importing internal/config in this package.
type LinkCheckConfig struct {
	OnBroken             string
	OnBrokenAnchor       string
	OnRelativeLinks      string
	OnLocalLinks         string
	SameSitePolicy       string
	ReportFormat         string
	Exclude              []string
	OnExternalBroken     string
	OnUnverifiedInternal string
}

// ReportInput bundles everything GenerateReport needs.
type ReportInput struct {
	Graph    *LinkGraph
	Coverage CoverageSummary
	Config   LinkCheckConfig
	SiteURL  string
}

// ReportResult is what GenerateReport returns to the builder.
type ReportResult struct {
	HasErrors bool
	Output    string
	Findings  []Finding
	Summary   string
}

// GenerateReport applies policy filters over the link graph and returns a
// formatted report. The caller (builder.go) uses HasErrors to decide whether
// to return a build error.
func GenerateReport(in ReportInput) ReportResult {
	refs := in.Graph.Refs()
	findings := classifyRefs(refs, in.Config, in.SiteURL)

	hasErrors := false
	for _, f := range findings {
		if f.Policy == "error" {
			hasErrors = true
			break
		}
	}

	summary := buildSummaryLine(in.Coverage, findings)

	var sb strings.Builder
	switch in.Config.ReportFormat {
	case "json":
		writeJSONReport(&sb, findings, in.Coverage, summary)
	case "github-actions":
		writeGitHubActionsReport(&sb, findings, in.Coverage, summary, hasErrors)
	default:
		writePrettyReport(&sb, findings, in.Coverage, summary)
	}

	return ReportResult{
		HasErrors: hasErrors,
		Output:    sb.String(),
		Findings:  findings,
		Summary:   summary,
	}
}

func classifyRefs(refs []LinkRef, cfg LinkCheckConfig, siteURL string) []Finding {
	var findings []Finding
	for _, ref := range refs {
		if shouldExcludeRef(ref.RawDest, cfg.Exclude) {
			continue
		}

		switch ref.Status {
		case StatusBrokenTarget:
			if cfg.OnBroken == "ignore" {
				continue
			}
			findings = append(findings, Finding{
				Type: FindingBrokenTarget, Ref: ref, Policy: cfg.OnBroken,
			})

		case StatusBrokenAnchor:
			if cfg.OnBrokenAnchor == "ignore" {
				continue
			}
			findings = append(findings, Finding{
				Type: FindingBrokenAnchor, Ref: ref, Policy: cfg.OnBrokenAnchor,
			})

		case StatusOK:
			if ref.Kind == KindRelative {
				if cfg.OnRelativeLinks == "ignore" {
					continue
				}
				findings = append(findings, Finding{
					Type: FindingRelativeLink, Ref: ref, Policy: cfg.OnRelativeLinks,
				})
			}

		case StatusExternal:
			if isLocalHref(ref.RawDest) {
				if cfg.OnLocalLinks == "ignore" {
					continue
				}
				findings = append(findings, Finding{
					Type: FindingLocalLink, Ref: ref, Policy: cfg.OnLocalLinks,
				})
				continue
			}
			if siteURL != "" && isSameSiteHref(ref.RawDest, siteURL) {
				policy := cfg.SameSitePolicy
				if policy == "" || policy == "ignore" {
					continue
				}
				findings = append(findings, Finding{
					Type: FindingSameSite, Ref: ref, Policy: policy,
				})
			}

		case StatusExternalBroken:
			policy := cfg.OnExternalBroken
			if policy == "" {
				policy = "warn"
			}
			if policy == "ignore" {
				continue
			}
			findings = append(findings, Finding{
				Type: FindingExternalBroken, Ref: ref, Policy: policy,
			})

		case StatusUnverified:
			policy := cfg.OnUnverifiedInternal
			if policy == "" {
				policy = "warn"
			}
			if policy == "ignore" {
				continue
			}
			findings = append(findings, Finding{
				Type: FindingUnverifiedInternal, Ref: ref, Policy: policy,
			})
		}
	}
	return findings
}

func buildSummaryLine(cov CoverageSummary, findings []Finding) string {
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
	return fmt.Sprintf("checked %d links across %d lanes — %d broken targets, %d broken anchors, %d broken external, %d warnings",
		cov.TotalLinks, cov.TotalLanes, brokenTargets, brokenAnchors, externalBroken, warnCount)
}

func shouldExcludeRef(rawDest string, patterns []string) bool {
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, rawDest); matched {
			return true
		}
	}
	return false
}

func isLocalHref(href string) bool {
	u, err := url.Parse(href)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" || host == "::1"
}

func isSameSiteHref(href, siteURL string) bool {
	u, err := url.Parse(href)
	if err != nil || u.Host == "" {
		return false
	}
	su, err := url.Parse(siteURL)
	if err != nil || su.Host == "" {
		return false
	}
	return strings.EqualFold(u.Scheme, su.Scheme) && strings.EqualFold(u.Host, su.Host)
}

// groupByFile groups findings by source file, sorted alphabetically.
func groupByFile(findings []Finding) []fileGroup {
	m := make(map[string][]Finding)
	for _, f := range findings {
		m[f.Ref.FromFile] = append(m[f.Ref.FromFile], f)
	}
	groups := make([]fileGroup, 0, len(m))
	for file, ff := range m {
		groups = append(groups, fileGroup{File: file, Findings: ff})
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].File < groups[j].File
	})
	return groups
}

type fileGroup struct {
	File     string
	Findings []Finding
}
