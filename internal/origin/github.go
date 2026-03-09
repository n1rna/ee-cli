package origin

import (
	"fmt"
	"sort"
	"strings"
)

// GitHub implements the Origin interface for GitHub secrets via the gh CLI.
type GitHub struct {
	name string
	cfg  Config
}

// NewGitHub creates a new GitHub origin driver.
func NewGitHub(name string, cfg Config) *GitHub {
	return &GitHub{name: name, cfg: cfg}
}

// Name returns the display name.
func (g *GitHub) Name() string {
	return g.name
}

// CheckPrerequisites verifies gh is installed and authenticated.
func (g *GitHub) CheckPrerequisites() error {
	if err := CheckTool("gh"); err != nil {
		return err
	}

	out, err := RunCommand("gh", "auth", "status")
	if err != nil {
		return fmt.Errorf("'gh' is not authenticated. Run 'gh auth login' first.\n%s", string(out))
	}

	return nil
}

// Push pushes secrets to GitHub.
func (g *GitHub) Push(env string, values map[string]string, mode PushMode, dryRun bool) (*PushResult, error) {
	if mode == "" {
		mode = ModeBundled
	}

	switch mode {
	case ModeBundled:
		return g.pushBundled(env, values, dryRun)
	case ModeIndividual:
		return g.pushIndividual(env, values, dryRun)
	default:
		return nil, fmt.Errorf("unsupported push mode: %q", mode)
	}
}

// pushBundled pushes all secrets as a single multi-line KEY=VALUE secret.
func (g *GitHub) pushBundled(env string, values map[string]string, dryRun bool) (*PushResult, error) {
	secretName := g.cfg.SecretName
	if secretName == "" {
		secretName = fmt.Sprintf("ENV_%s", strings.ToUpper(env))
	}

	// Build bundled content: sorted KEY=VALUE lines
	body := buildDotenvBody(values)

	if dryRun {
		return &PushResult{SecretsCount: 1}, nil
	}

	args := g.baseArgs()
	args = append(args, "secret", "set", secretName, "--body", body)

	out, err := RunCommand("gh", args...)
	if err != nil {
		return &PushResult{SecretsCount: 0, Errors: []error{
			fmt.Errorf("failed to set secret %s: %s", secretName, string(out)),
		}}, nil
	}

	return &PushResult{SecretsCount: 1}, nil
}

// pushIndividual pushes each secret as a separate GitHub secret.
func (g *GitHub) pushIndividual(env string, values map[string]string, dryRun bool) (*PushResult, error) {
	result := &PushResult{}

	keys := sortedKeys(values)
	for _, key := range keys {
		if dryRun {
			result.SecretsCount++
			continue
		}

		args := g.baseArgs()
		args = append(args, "secret", "set", key, "--body", values[key])

		out, err := RunCommand("gh", args...)
		if err != nil {
			result.Errors = append(result.Errors,
				fmt.Errorf("failed to set secret %s: %s", key, string(out)))
			continue
		}
		result.SecretsCount++
	}

	return result, nil
}

// baseArgs returns the common gh CLI arguments for this origin.
func (g *GitHub) baseArgs() []string {
	var args []string

	if g.cfg.Repo != "" {
		args = append(args, "-R", g.cfg.Repo)
	}
	if g.cfg.Environment != "" {
		args = append(args, "--env", g.cfg.Environment)
	}

	return args
}

// buildDotenvBody creates a multi-line KEY=VALUE string from a map.
func buildDotenvBody(values map[string]string) string {
	keys := sortedKeys(values)
	var lines []string
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", key, values[key]))
	}
	return strings.Join(lines, "\n")
}

// sortedKeys returns the keys of a map sorted alphabetically.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
