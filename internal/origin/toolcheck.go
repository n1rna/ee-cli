package origin

import (
	"fmt"
	"os/exec"
)

var toolInstallHints = map[string]string{
	"gh":       "https://cli.github.com",
	"wrangler": "https://developers.cloudflare.com/workers/wrangler/install-and-update/",
}

// CheckTool verifies that a CLI tool is available on PATH.
func CheckTool(name string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		hint := toolInstallHints[name]
		if hint != "" {
			return fmt.Errorf("'%s' CLI not found on PATH. Install it from %s", name, hint)
		}
		return fmt.Errorf("'%s' CLI not found on PATH", name)
	}
	return nil
}

// RunCommand executes a command and returns its combined output.
func RunCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}
