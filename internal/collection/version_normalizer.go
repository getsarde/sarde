package collection

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/getsarde/sarde/internal/engine"
)

// normalizeRootVersionPages assigns Version=lastVersion to pages at the
// collection root (Version=="") in a versioned collection. The latest version's
// content lives at the root without a vN/ directory; this normalization ensures
// every page in the collection has a concrete Version value.
func normalizeRootVersionPages(pages []*engine.Page, lastVersion string) {
	if lastVersion == "" {
		return
	}
	for _, p := range pages {
		if p.Version != "" {
			continue
		}
		p.Version = lastVersion
		if parts := strings.SplitN(p.LangRelPath, "/", 2); len(parts) == 2 {
			p.VersionRelPath = parts[1]
		}
	}
}

func rawDigest(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8])
}

func fmDigest(fmMap map[string]interface{}) string {
	if len(fmMap) == 0 {
		return ""
	}
	b, err := json.Marshal(fmMap)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(b)
	return fmt.Sprintf("%x", h[:8])
}
