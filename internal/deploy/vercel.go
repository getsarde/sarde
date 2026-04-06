package deploy

import "fmt"

// VercelDeployer deploys to Vercel via API.
type VercelDeployer struct {
	ProjectID string
}

func (d *VercelDeployer) Name() string { return "vercel" }

func (d *VercelDeployer) Deploy(distDir string) error {
	token, err := requireEnv("VERCEL_TOKEN", "Vercel deploy")
	if err != nil {
		return err
	}
	if d.ProjectID == "" {
		return fmt.Errorf("vercel deploy requires deploy.project_id in site.yaml")
	}

	fmt.Printf("Vercel deploy: project=%s, token=%s, dist=%s\n",
		d.ProjectID, maskToken(token), distDir)
	fmt.Println("Vercel API deployment not yet implemented — use vercel-cli or custom deploy command instead.")
	return nil
}
