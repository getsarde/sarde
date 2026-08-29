package links

import (
	"encoding/json"
	"strings"
)

type jsonReport struct {
	Summary  jsonSummary   `json:"summary"`
	Findings []jsonFinding `json:"findings"`
}

type jsonSummary struct {
	Links          int `json:"links"`
	Lanes          int `json:"lanes"`
	BrokenTargets  int `json:"broken_targets"`
	BrokenAnchors  int `json:"broken_anchors"`
	ExternalBroken int `json:"external_broken"`
	Warnings       int `json:"warnings"`
}

type jsonFinding struct {
	File     string  `json:"file"`
	Line     int     `json:"line"`
	Col      int     `json:"col"`
	Dest     string  `json:"dest"`
	Fragment string  `json:"fragment,omitempty"`
	Dim      jsonDim `json:"dim"`
	Type     string  `json:"type"`
	Policy   string  `json:"policy"`
	// Hint is fix advice for finding types that carry one (ambiguous_link).
	Hint string `json:"hint,omitempty"`
}

type jsonDim struct {
	Collection string `json:"collection"`
	Lang       string `json:"lang"`
	Version    string `json:"version"`
}

func writeJSONReport(sb *strings.Builder, findings []Finding, cov CoverageSummary, _ string) {
	var brokenTargets, brokenAnchors, externalBroken, warnCount int
	jFindings := make([]jsonFinding, 0, len(findings))

	for _, f := range findings {
		switch f.Type {
		case FindingBrokenTarget, FindingAmbiguousLink:
			brokenTargets++
		case FindingBrokenAnchor:
			brokenAnchors++
		case FindingExternalBroken:
			externalBroken++
		default:
			warnCount++
		}
		jFindings = append(jFindings, jsonFinding{
			File:     f.Ref.FromFile,
			Line:     f.Ref.Line,
			Col:      f.Ref.Col,
			Dest:     f.Ref.RawDest,
			Fragment: f.Ref.Fragment,
			Dim: jsonDim{
				Collection: f.Ref.Dim.Collection,
				Lang:       f.Ref.Dim.Lang,
				Version:    f.Ref.Dim.Version,
			},
			Type:   f.Type.String(),
			Policy: f.Policy,
			Hint:   f.Type.Hint(),
		})
	}

	report := jsonReport{
		Summary: jsonSummary{
			Links:          cov.TotalLinks,
			Lanes:          cov.TotalLanes,
			BrokenTargets:  brokenTargets,
			BrokenAnchors:  brokenAnchors,
			ExternalBroken: externalBroken,
			Warnings:       warnCount,
		},
		Findings: jFindings,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		sb.WriteString("{\"error\":\"failed to marshal report\"}\n")
		return
	}
	sb.Write(data)
	sb.WriteByte('\n')
}
