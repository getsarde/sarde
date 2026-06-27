package deploy

import "fmt"

// NetlifyDeployer deploys to Netlify via API.
type NetlifyDeployer struct {
	SiteID string
}

func (d *NetlifyDeployer) Name() string { return "netlify" }

func (d *NetlifyDeployer) Deploy(distDir string) error {
	token, err := requireEnv("NETLIFY_AUTH_TOKEN", "Netlify deploy")
	if err != nil {
		return err
	}
	if d.SiteID == "" {
		return fmt.Errorf("netlify deploy requires deploy.site_id in sarde.yaml")
	}

	fmt.Printf("Netlify deploy: site_id=%s, token=%s, dist=%s\n", d.SiteID, maskToken(token), distDir)
	fmt.Println("Netlify API deployment not yet implemented — use netlify-cli or custom deploy command instead.")
	return nil
}
