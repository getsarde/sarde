package deploy

import "fmt"

// CloudflareDeployer deploys to Cloudflare Pages via API.
type CloudflareDeployer struct {
	ProjectName string
}

func (d *CloudflareDeployer) Name() string { return "cloudflare-pages" }

func (d *CloudflareDeployer) Deploy(distDir string) error {
	token, err := requireEnv("CLOUDFLARE_API_TOKEN", "Cloudflare deploy")
	if err != nil {
		return err
	}
	accountID, err := requireEnv("CLOUDFLARE_ACCOUNT_ID", "Cloudflare deploy")
	if err != nil {
		return err
	}
	if d.ProjectName == "" {
		return fmt.Errorf("cloudflare deploy requires deploy.project_name in site.yaml")
	}

	fmt.Printf("Cloudflare Pages deploy: account=%s, project=%s, token=%s, dist=%s\n",
		accountID, d.ProjectName, maskToken(token), distDir)
	fmt.Println("Cloudflare Pages API deployment not yet implemented — use wrangler or custom deploy command instead.")
	return nil
}
