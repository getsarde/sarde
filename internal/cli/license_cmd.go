package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/getsarde/sarde/internal/license"
	"github.com/spf13/cobra"
)

var licenseCmd = &cobra.Command{
	Use:   "license",
	Short: "Manage premium plugin licenses",
}

var licenseInstallCmd = &cobra.Command{
	Use:   "install <license-file>",
	Short: "Install a premium plugin license",
	Long: `Copy a license file into the license directory and verify it.

By default the license is installed to the user home directory
(~/.sarde/licenses/) so one purchase covers every local project. Use
--project to install into the current project's .sarde/licenses/ instead
(useful for CI).`,
	Args: cobra.ExactArgs(1),
	RunE: runLicenseInstall,
}

var licenseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed licenses and their status",
	RunE:  runLicenseList,
}

func init() {
	licenseInstallCmd.Flags().Bool("project", false, "Install into the project's .sarde/licenses/ instead of the user home")
	licenseCmd.AddCommand(licenseInstallCmd)
	licenseCmd.AddCommand(licenseListCmd)
	rootCmd.AddCommand(licenseCmd)
}

func runLicenseInstall(cmd *cobra.Command, args []string) error {
	f, err := license.Load(args[0])
	if err != nil {
		return err
	}
	if f.Slug == "" {
		return fmt.Errorf("license file has no plugin slug")
	}

	var destDir string
	if project, _ := cmd.Flags().GetBool("project"); project {
		projectDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		destDir = filepath.Join(projectDir, ".sarde", "licenses")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolving home directory: %w", err)
		}
		destDir = filepath.Join(home, ".sarde", "licenses")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	destPath := filepath.Join(destDir, f.Slug+".license")
	if err := os.WriteFile(destPath, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", destPath, err)
	}

	fmt.Printf("Installed license for %q (licensee: %s) to %s\n", f.Slug, f.Licensee, destPath)
	if err := license.Verify(f, license.PublicKey, f.Slug, "", time.Now()); err != nil {
		fmt.Printf("Warning: license does not verify: %v\n", err)
		fmt.Println("The plugin will stay inactive until a valid license is installed.")
	} else {
		fmt.Println("License verified successfully.")
	}
	return nil
}

func runLicenseList(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	dirs := []string{filepath.Join(projectDir, ".sarde", "licenses")}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".sarde", "licenses"))
	}

	found := false
	fmt.Printf("%-20s %-30s %-12s %s\n", "PLUGIN", "LICENSEE", "EXPIRES", "STATUS")
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".license") {
				continue
			}
			found = true
			path := filepath.Join(dir, e.Name())
			f, err := license.Load(path)
			if err != nil {
				fmt.Printf("%-20s %-30s %-12s %s\n", strings.TrimSuffix(e.Name(), ".license"), "-", "-", "unreadable")
				continue
			}
			expires := f.Expires
			if expires == "" {
				expires = "never"
			}
			status := "valid"
			if err := license.Verify(f, license.PublicKey, f.Slug, "", time.Now()); err != nil {
				status = err.Error()
			}
			fmt.Printf("%-20s %-30s %-12s %s\n", f.Slug, f.Licensee, expires, status)
		}
	}
	if !found {
		fmt.Println("No licenses installed.")
	}
	return nil
}
