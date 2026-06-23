package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/frostybee/go-swarm-icons/npm"
	"github.com/frostybee/sarde/internal/config"
	"github.com/spf13/cobra"
)

// defaultSetsDir is where downloaded Iconify sets land when icons.sets_dir is
// not configured and --dest is not given.
const defaultSetsDir = "icon-sets"

var iconsCmd = &cobra.Command{
	Use:   "icons",
	Short: "Manage icon sets",
}

var iconsAddCmd = &cobra.Command{
	Use:   "add <prefix> [prefix...]",
	Short: "Download Iconify icon sets into the project",
	Long: "Fetch icon sets from the npm registry and save them as JSON files. " +
		"Sets are written to icons.sets_dir (or --dest). " +
		"After downloading, reference icons as :icon[prefix:name] in your content.",
	Args: cobra.MinimumNArgs(1),
	RunE: runIconsAdd,
}

var iconsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Iconify icon sets available for download",
	Long:  "Fetch the full Iconify collection list and display it as a table with local download status.",
	RunE:  runIconsList,
}

func init() {
	iconsAddCmd.Flags().StringP("dest", "d", "", "Destination directory (overrides icons.sets_dir)")
	iconsListCmd.Flags().StringP("search", "s", "", "Filter by prefix, name, or category (case-insensitive)")
	iconsListCmd.Flags().IntP("page-size", "n", 30, "Rows per page (0 = no pagination)")

	iconsCmd.AddCommand(iconsAddCmd)
	iconsCmd.AddCommand(iconsListCmd)
	rootCmd.AddCommand(iconsCmd)
}

func runIconsAdd(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(nil)

	dest, _ := cmd.Flags().GetString("dest")
	if dest == "" {
		dest = resolveSetsDir(cmd, projectDir)
	} else if !filepath.IsAbs(dest) {
		dest = filepath.Join(projectDir, dest)
	}

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	dl := npm.NewDownloader()
	var downloaded, failed int
	for _, prefix := range args {
		fmt.Printf("Downloading %s... ", prefix)
		content, version, err := dl.DownloadSet(prefix)
		if err != nil {
			fmt.Printf("failed (%v)\n", err)
			failed++
			continue
		}
		outPath := filepath.Join(dest, prefix+".json")
		if err := os.WriteFile(outPath, content, 0o644); err != nil {
			fmt.Printf("failed (%v)\n", err)
			failed++
			continue
		}
		fmt.Printf("ok (%s, %s)\n", version, humanSize(len(content)))
		downloaded++
	}

	relDest := dest
	if rel, err := filepath.Rel(projectDir, dest); err == nil {
		relDest = rel
	}
	fmt.Printf("\n%d set(s) downloaded, %d failed. Output: %s\n", downloaded, failed, relDest)
	if downloaded > 0 {
		fmt.Printf("If not already set, add `icons.sets_dir: %q` to sarde.yaml,\n", filepath.ToSlash(relDest))
		fmt.Println("then reference downloaded icons as :icon[prefix:name] in your content.")
	}
	return nil
}

func runIconsList(cmd *cobra.Command, args []string) error {
	search, _ := cmd.Flags().GetString("search")
	pageSize, _ := cmd.Flags().GetInt("page-size")

	dl := npm.NewDownloader()
	collections, err := dl.FetchCollections()
	if err != nil {
		return fmt.Errorf("fetching collection list: %w", err)
	}

	projectDir := projectDirFromArgs(nil)
	setsDir := resolveSetsDir(cmd, projectDir)

	prefixes := make([]string, 0, len(collections))
	for p := range collections {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)

	lower := strings.ToLower(search)

	type row struct {
		prefix, name, category, license, dl string
		total                               int
	}
	var rows []row
	var totalIcons int
	for _, p := range prefixes {
		c := collections[p]
		if lower != "" {
			haystack := strings.ToLower(p + " " + c.Name + " " + c.Category)
			if !strings.Contains(haystack, lower) {
				continue
			}
		}
		mark := ""
		if _, err := os.Stat(filepath.Join(setsDir, p+".json")); err == nil {
			mark = "*"
		}
		name := c.Name
		if len(name) > 34 {
			name = name[:31] + "..."
		}
		license := c.License
		if len(license) > 19 {
			license = license[:16] + "..."
		}
		rows = append(rows, row{p, name, c.Category, license, mark, c.Total})
		totalIcons += c.Total
	}

	printHeader := func() {
		fmt.Printf("%-20s %-35s %6s %-16s %-20s %s\n", "PREFIX", "NAME", "ICONS", "CATEGORY", "LICENSE", "DL")
		fmt.Println(strings.Repeat("-", 110))
	}

	reader := bufio.NewReader(os.Stdin)
	totalPages := 0
	if pageSize > 0 {
		totalPages = (len(rows) + pageSize - 1) / pageSize
	}
	printHeader()
	for i, r := range rows {
		fmt.Printf("%-20s %-35s %6d %-16s %-20s %s\n", r.prefix, r.name, r.total, r.category, r.license, r.dl)
		if pageSize > 0 && (i+1)%pageSize == 0 && i+1 < len(rows) {
			page := (i + 1) / pageSize
			fmt.Printf("\n--- Page %d/%d. Continue? (yes/no) [yes]: ", page, totalPages)
			line, _ := reader.ReadString('\n')
			line = strings.TrimSpace(strings.ToLower(line))
			if line == "no" || line == "n" {
				break
			}
			fmt.Println()
			printHeader()
		}
	}

	fmt.Printf("\n%d set(s) shown, %d total icons. Status checked against %s\n",
		len(rows), totalIcons, setsDir)
	return nil
}

// resolveSetsDir returns the absolute directory where icon sets live: the
// configured icons.sets_dir when set, otherwise defaultSetsDir, both resolved
// relative to projectDir.
func resolveSetsDir(cmd *cobra.Command, projectDir string) string {
	dir := defaultSetsDir
	configPath, _ := cmd.Flags().GetString("config")
	if cfg, err := config.Resolve(config.ResolveOptions{ConfigPath: configPath}); err == nil && cfg.Icons.SetsDir != "" {
		dir = cfg.Icons.SetsDir
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(projectDir, dir)
	}
	return dir
}

// humanSize formats a byte count as a short human-readable string (e.g. "12.3 KB").
func humanSize(n int) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := int64(n) / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
