package icons

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	swarmicons "github.com/frostybee/go-swarm-icons"
	"github.com/frostybee/go-swarm-icons/lucide"
)

var manager *swarmicons.IconManager

var managerMu sync.Mutex

var (
	collectionsMu sync.RWMutex
	collections   = make(map[string]*iconifyCollection)
)

type iconifyCollection struct {
	Prefix string    `json:"prefix"`
	Info   *iconInfo `json:"info,omitempty"`
}

type iconInfo struct {
	Name    string          `json:"name"`
	License *iconifyLicense `json:"license,omitempty"`
}

type iconifyLicense struct {
	Title string `json:"title"`
	SPDX  string `json:"spdx"`
	URL   string `json:"url"`
}

func init() {
	ensureManager()
}

func ensureManager() {
	managerMu.Lock()
	defer managerMu.Unlock()
	if manager != nil {
		return
	}

	cfg := swarmicons.NewConfig().
		AddProvider("lucide", lucide.Provider()).
		DefaultPrefix("lucide").
		DefaultAttributes(map[string]string{
			"xmlns": "http://www.w3.org/2000/svg",
		}).
		IgnoreNotFound()

	var err error
	manager, err = cfg.Build()
	if err != nil {
		panic("icons: " + err.Error())
	}

	sprites = swarmicons.NewSpriteCollector()

	collectionsMu.Lock()
	collections["lucide"] = &iconifyCollection{
		Prefix: "lucide",
		Info: &iconInfo{
			Name:    "Lucide",
			License: &iconifyLicense{Title: "ISC License", SPDX: "ISC", URL: "https://github.com/lucide-icons/lucide/blob/main/LICENSE"},
		},
	}
	collectionsMu.Unlock()
}

func extractLicenseInfo(prefix string, data []byte) {
	var col iconifyCollection
	if err := json.Unmarshal(data, &col); err == nil {
		col.Prefix = prefix
		collectionsMu.Lock()
		collections[prefix] = &col
		collectionsMu.Unlock()
	}
}

// ---------------------------------------------------------------------------
// Dynamic provider registration (called from build init)
// ---------------------------------------------------------------------------

// LoadCollection loads an additional Iconify JSON collection from raw bytes
// and registers it as a provider under its prefix.
func LoadCollection(data []byte) error {
	var meta struct {
		Prefix string    `json:"prefix"`
		Info   *iconInfo `json:"info,omitempty"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}
	if meta.Prefix == "" {
		return fmt.Errorf("icon collection missing prefix")
	}

	provider := swarmicons.NewJsonCollectionFromBytes(data)
	manager.Register(meta.Prefix, provider)

	collectionsMu.Lock()
	collections[meta.Prefix] = &iconifyCollection{Prefix: meta.Prefix, Info: meta.Info}
	collectionsMu.Unlock()
	return nil
}

// LoadIconDirectory registers a DirectoryProvider that loads SVG files from
// dirPath. Bare icon names resolve from this directory FIRST (project overrides win).
// A missing directory is not an error.
func LoadIconDirectory(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil
	}
	provider, err := swarmicons.NewDirectoryProvider(dirPath, swarmicons.WithRecursive(false))
	if err != nil {
		return err
	}

	managerMu.Lock()
	defer managerMu.Unlock()
	manager.Register("local", provider)
	return nil
}
