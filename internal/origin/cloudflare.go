package origin

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Cloudflare implements the Origin interface for Cloudflare Workers secrets via the wrangler CLI.
type Cloudflare struct {
	name string
	cfg  Config
}

// NewCloudflare creates a new Cloudflare origin driver.
func NewCloudflare(name string, cfg Config) *Cloudflare {
	return &Cloudflare{name: name, cfg: cfg}
}

// Name returns the display name.
func (c *Cloudflare) Name() string {
	return c.name
}

// CheckPrerequisites verifies wrangler is installed and authenticated.
func (c *Cloudflare) CheckPrerequisites() error {
	if err := CheckTool("wrangler"); err != nil {
		return err
	}

	out, err := RunCommand("wrangler", "whoami")
	if err != nil {
		return fmt.Errorf(
			"'wrangler' is not authenticated. Run 'wrangler login' first.\n%s",
			string(out),
		)
	}

	return nil
}

// Push pushes secrets to Cloudflare Workers.
func (c *Cloudflare) Push(
	env string, values map[string]string, mode PushMode, dryRun bool,
) (*PushResult, error) {
	if c.cfg.Worker == "" {
		return nil, fmt.Errorf("cloudflare origin requires 'worker' field in config")
	}

	if mode == "" {
		mode = ModeIndividual
	}

	switch mode {
	case ModeBundled:
		return c.pushBulk(values, dryRun)
	case ModeIndividual:
		return c.pushIndividual(values, dryRun)
	default:
		return nil, fmt.Errorf("unsupported push mode: %q", mode)
	}
}

// pushBulk pushes all secrets at once using wrangler secret bulk.
func (c *Cloudflare) pushBulk(values map[string]string, dryRun bool) (*PushResult, error) {
	if dryRun {
		return &PushResult{SecretsCount: len(values)}, nil
	}

	// Build JSON object for bulk upload
	jsonData, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secrets: %w", err)
	}

	args := []string{"secret", "bulk", "--name", c.cfg.Worker}
	cmd := exec.Command("wrangler", args...)
	cmd.Stdin = strings.NewReader(string(jsonData))

	out, err := cmd.CombinedOutput()
	if err != nil {
		return &PushResult{SecretsCount: 0, Errors: []error{
			fmt.Errorf("failed to bulk upload secrets: %s", string(out)),
		}}, nil
	}

	return &PushResult{SecretsCount: len(values)}, nil
}

// pushIndividual pushes each secret one at a time using wrangler secret put.
func (c *Cloudflare) pushIndividual(values map[string]string, dryRun bool) (*PushResult, error) {
	result := &PushResult{}

	keys := sortedKeys(values)
	for _, key := range keys {
		if dryRun {
			result.SecretsCount++
			continue
		}

		args := []string{"secret", "put", key, "--name", c.cfg.Worker}
		cmd := exec.Command("wrangler", args...)
		cmd.Stdin = strings.NewReader(values[key])

		out, err := cmd.CombinedOutput()
		if err != nil {
			result.Errors = append(result.Errors,
				fmt.Errorf("failed to set secret %s: %s", key, string(out)))
			continue
		}
		result.SecretsCount++
	}

	return result, nil
}
