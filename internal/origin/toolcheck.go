package origin

import (
	"fmt"
	"os/exec"
	"strings"
)

var toolInstallHints = map[string]string{
	"gh":       "https://cli.github.com",
	"wrangler": "https://developers.cloudflare.com/workers/wrangler/install-and-update/",
}

// npmRunnableTools maps a logical tool name to the npm package/binary that can
// be executed through a JavaScript package runner (bunx/npx). These tools are
// commonly installed as a project dependency rather than globally on PATH, so
// when they are missing from PATH we fall back to running them via a runner,
// which resolves the project-local install (node_modules/.bin) or fetches the
// package on the fly.
var npmRunnableTools = map[string]string{
	"wrangler": "wrangler",
}

// jsRunners lists the supported JavaScript package runners in preference order.
// bunx is tried first (it is faster and also resolves node_modules), then npx.
var jsRunners = []string{"bunx", "npx"}

// lookPath is a package-level indirection over exec.LookPath so tests can
// simulate which executables are available.
var lookPath = exec.LookPath

// runnerArgv builds the argument vector for invoking pkg through the given
// runner. npx is passed -y so it never prompts before installing on the fly.
func runnerArgv(runner, pkg string) []string {
	if runner == "npx" {
		return []string{"npx", "-y", pkg}
	}
	return []string{runner, pkg}
}

// ResolveTool returns the argument vector used to invoke a logical tool name.
// It prefers a binary found directly on PATH; for npm-based tools it falls back
// to a JavaScript runner (bunx/npx) when the binary is not on PATH.
func ResolveTool(name string) ([]string, error) {
	if _, err := lookPath(name); err == nil {
		return []string{name}, nil
	}

	if pkg, ok := npmRunnableTools[name]; ok {
		for _, runner := range jsRunners {
			if _, err := lookPath(runner); err == nil {
				return runnerArgv(runner, pkg), nil
			}
		}
	}

	return nil, notFoundError(name)
}

// notFoundError builds a helpful "not found" error, mentioning the runner
// fallback for npm-based tools.
func notFoundError(name string) error {
	hint := toolInstallHints[name]
	if _, runnable := npmRunnableTools[name]; runnable {
		msg := fmt.Sprintf(
			"'%s' not found on PATH and no JavaScript runner (%s) is available to run it",
			name, strings.Join(jsRunners, " or "),
		)
		if hint != "" {
			msg += fmt.Sprintf(". Install it from %s, or install bun/node so it can run via %s",
				hint, strings.Join(jsRunners, "/"))
		}
		return fmt.Errorf("%s", msg)
	}

	if hint != "" {
		return fmt.Errorf("'%s' CLI not found on PATH. Install it from %s", name, hint)
	}
	return fmt.Errorf("'%s' CLI not found on PATH", name)
}

// CheckTool verifies that a CLI tool can be resolved, either directly on PATH
// or via a JavaScript runner fallback for npm-based tools.
func CheckTool(name string) error {
	_, err := ResolveTool(name)
	return err
}

// ToolCommand builds an *exec.Cmd for a logical tool name, resolving PATH and
// runner fallbacks. Callers may further configure the returned command (for
// example, setting Stdin) before running it.
func ToolCommand(name string, args ...string) (*exec.Cmd, error) {
	argv, err := ResolveTool(name)
	if err != nil {
		return nil, err
	}
	full := append(argv, args...)
	return exec.Command(full[0], full[1:]...), nil
}

// RunCommand executes a resolved tool command and returns its combined output.
func RunCommand(name string, args ...string) ([]byte, error) {
	cmd, err := ToolCommand(name, args...)
	if err != nil {
		return nil, err
	}
	return cmd.CombinedOutput()
}
