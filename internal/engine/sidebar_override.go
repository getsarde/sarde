package engine

import "sort"

// SidebarOverride holds sidebar.yaml overrides for one path-keyed node
// (section or page). Unset fields fall through to the next precedence layer
// (frontmatter, then inferred defaults).
type SidebarOverride struct {
	Label       string
	Description string
	Order       *int  // nil = unset (0 is a valid explicit value)
	Collapsed   *bool
	Icon        string
	Badge       Badge
	Hidden      *bool // nil = unset; false un-hides a frontmatter-hidden page
	Attrs       map[string]string
}

// TabOverride holds sidebar.yaml overrides for one docs tab (keyed by slug).
type TabOverride struct {
	Label       string
	Description string
	Icon        string
	Order       *int
}

// MarkOverrideMatched records that a sidebar.yaml override key was consulted
// during nav-tree building. A key is only reported unmatched when no lane
// (language/version combination) ever matched it. Nav-tree assembly is
// serial, so no locking is needed.
func (s *SidebarConfig) MarkOverrideMatched(key string) {
	if s == nil {
		return
	}
	if s.matchedOverrides == nil {
		s.matchedOverrides = make(map[string]bool)
	}
	s.matchedOverrides[key] = true
}

// MarkTabMatched records that a sidebar.yaml tab override key was consulted.
func (s *SidebarConfig) MarkTabMatched(key string) {
	if s == nil {
		return
	}
	if s.matchedTabs == nil {
		s.matchedTabs = make(map[string]bool)
	}
	s.matchedTabs[key] = true
}

// UnmatchedOverrideKeys returns the sorted override keys that no lane matched.
func (s *SidebarConfig) UnmatchedOverrideKeys() []string {
	if s == nil || len(s.Overrides) == 0 {
		return nil
	}
	var keys []string
	for key := range s.Overrides {
		if !s.matchedOverrides[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

// UnmatchedTabKeys returns the sorted tab override keys that no lane matched.
func (s *SidebarConfig) UnmatchedTabKeys() []string {
	if s == nil || len(s.TabOverrides) == 0 {
		return nil
	}
	var keys []string
	for key := range s.TabOverrides {
		if !s.matchedTabs[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}
