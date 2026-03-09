// Package origin provides drivers for pushing secrets to remote origins (GitHub, Cloudflare).
package origin

import "fmt"

// PushMode defines how secrets are pushed to the origin.
type PushMode string

const (
	// ModeBundled pushes all secrets as a single multi-line KEY=VALUE secret.
	// This is compatible with ee-action which reads a bundled secret.
	ModeBundled PushMode = "bundled"

	// ModeIndividual pushes each secret as a separate key-value pair.
	ModeIndividual PushMode = "individual"
)

// Config represents an origin configuration in the .ee project file.
type Config struct {
	Type string   `json:"type"`           // "github" or "cloudflare"
	Mode PushMode `json:"mode,omitempty"` // "bundled" or "individual"

	// GitHub-specific fields
	Repo        string `json:"repo,omitempty"`        // e.g. "owner/repo", defaults to current repo
	SecretName  string `json:"secret_name,omitempty"` // bundled mode secret name (default: "ENV_<ENV>")
	Environment string `json:"environment,omitempty"` // GitHub environment name

	// Cloudflare-specific fields
	Worker string `json:"worker,omitempty"` // Cloudflare Worker name
}

// PushResult contains the result of a push operation.
type PushResult struct {
	SecretsCount int
	Errors       []error
}

// Origin is the interface that all origin drivers must implement.
type Origin interface {
	// Push pushes key-value pairs to the origin.
	Push(env string, values map[string]string, mode PushMode, dryRun bool) (*PushResult, error)

	// CheckPrerequisites verifies the CLI tool is installed and authenticated.
	CheckPrerequisites() error

	// Name returns the display name of this origin.
	Name() string
}

// New creates a new origin driver based on the config type.
func New(name string, cfg Config) (Origin, error) {
	switch cfg.Type {
	case "github":
		return NewGitHub(name, cfg), nil
	case "cloudflare":
		return NewCloudflare(name, cfg), nil
	default:
		return nil, fmt.Errorf(
			"unsupported origin type: %q (supported: github, cloudflare)",
			cfg.Type,
		)
	}
}

// DefaultMode returns the default push mode for the given origin type.
func DefaultMode(originType string) PushMode {
	switch originType {
	case "cloudflare":
		return ModeIndividual
	default:
		return ModeBundled
	}
}
