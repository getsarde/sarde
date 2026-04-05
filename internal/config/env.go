package config

import (
	"os"
	"strconv"
	"strings"
)

// applyEnvOverrides reads CODEROO_-prefixed environment variables and applies
// them to the config. Only the most commonly overridden fields are supported.
// This is layer 5 (highest priority) of the config cascade.
func applyEnvOverrides(cfg *SiteConfig, prefix string) {
	if prefix == "" {
		prefix = "CODEROO"
	}

	// String fields
	if v, ok := lookupEnv(prefix, "SITE_TITLE"); ok {
		cfg.Site.Title = v
	}
	if v, ok := lookupEnv(prefix, "SITE_DESCRIPTION"); ok {
		cfg.Site.Description = v
	}
	if v, ok := lookupEnv(prefix, "SITE_URL"); ok {
		cfg.Site.URL = v
	}
	if v, ok := lookupEnv(prefix, "SITE_LANGUAGE"); ok {
		cfg.Site.Language = v
	}
	if v, ok := lookupEnv(prefix, "THEME_NAME"); ok {
		cfg.Theme.Name = v
	}
	if v, ok := lookupEnv(prefix, "THEME_PRESET"); ok {
		cfg.Theme.Preset = v
	}
	if v, ok := lookupEnv(prefix, "BUILD_OUTPUT"); ok {
		cfg.Build.Output = v
	}
	if v, ok := lookupEnv(prefix, "BUILD_BASE_PATH"); ok {
		cfg.Build.BasePath = v
	}
	if v, ok := lookupEnv(prefix, "ANALYTICS_PROVIDER"); ok {
		cfg.Analytics.Provider = v
	}
	if v, ok := lookupEnv(prefix, "ANALYTICS_SITE_ID"); ok {
		cfg.Analytics.SiteID = v
	}
	if v, ok := lookupEnv(prefix, "SEARCH_PROVIDER"); ok {
		cfg.Search.Provider = v
	}

	// Bool fields
	if v, ok := lookupEnvBool(prefix, "BUILD_DRAFTS"); ok {
		cfg.Build.Drafts = BoolPtr(v)
	}
	if v, ok := lookupEnvBool(prefix, "BUILD_MINIFY"); ok {
		cfg.Build.Minify = BoolPtr(v)
	}
	if v, ok := lookupEnvBool(prefix, "BUILD_CLEAN"); ok {
		cfg.Build.Clean = BoolPtr(v)
	}
	if v, ok := lookupEnvBool(prefix, "BUILD_PARALLEL"); ok {
		cfg.Build.Parallel = BoolPtr(v)
	}
	if v, ok := lookupEnvBool(prefix, "MARKDOWN_UNSAFE"); ok {
		cfg.Markdown.Unsafe = BoolPtr(v)
	}
	if v, ok := lookupEnvBool(prefix, "SEARCH_ENABLED"); ok {
		cfg.Search.Enabled = BoolPtr(v)
	}
	if v, ok := lookupEnvBool(prefix, "THEME_DARK"); ok {
		cfg.Theme.Dark = BoolPtr(v)
	}

	// Int fields
	if v, ok := lookupEnvInt(prefix, "SERVER_PORT"); ok {
		cfg.Server.Port = v
	}
	if v, ok := lookupEnvInt(prefix, "TOC_MIN_LEVEL"); ok {
		cfg.TOC.MinLevel = v
	}
	if v, ok := lookupEnvInt(prefix, "TOC_MAX_LEVEL"); ok {
		cfg.TOC.MaxLevel = v
	}
}

func lookupEnv(prefix, key string) (string, bool) {
	return os.LookupEnv(prefix + "_" + key)
}

func lookupEnvBool(prefix, key string) (bool, bool) {
	v, ok := os.LookupEnv(prefix + "_" + key)
	if !ok {
		return false, false
	}
	switch strings.ToLower(v) {
	case "true", "1", "yes", "on":
		return true, true
	case "false", "0", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func lookupEnvInt(prefix, key string) (int, bool) {
	v, ok := os.LookupEnv(prefix + "_" + key)
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}
