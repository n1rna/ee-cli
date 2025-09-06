// Package command implements the ee remote command for managing remote URLs
package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

// RemoteCommand handles the ee remote command
type RemoteCommand struct{}

// NewRemoteCommand creates a new ee remote command
func NewRemoteCommand() *cobra.Command {
	rc := &RemoteCommand{}

	cmd := &cobra.Command{
		Use:   "remote [url]",
		Short: "Manage remote URL for current project",
		Long: `Manage remote URL configuration for the current project.

Without arguments, shows the current remote URL.
With a URL argument, sets the remote URL in the .ee file.

Remote URL format: company@ee.dev/project-name
This will be expanded to: http://api-server/-/company/projects/project-id

Examples:
  # Show current remote URL
  ee remote

  # Set remote URL
  ee remote company@ee.dev/my-project

  # Set remote URL with explicit server
  ee remote http://localhost:8000/-/company/projects/project-id

  # Clear remote URL
  ee remote --unset
`,
		Args: cobra.MaximumNArgs(1),
		RunE: rc.Run,
	}

	cmd.Flags().Bool("unset", false, "Remove remote URL configuration")

	return cmd
}

// Run executes the remote command
func (rc *RemoteCommand) Run(cmd *cobra.Command, args []string) error {
	unset, _ := cmd.Flags().GetBool("unset")

	// Check if we're in a project directory with .ee file
	if !EasyEnvFileExists("") {
		return fmt.Errorf(".ee file not found in current directory. Run 'ee init' first")
	}

	// Load current .ee file
	menvFile, err := LoadEasyEnvFile("")
	if err != nil {
		return fmt.Errorf("failed to load .ee file: %w", err)
	}

	// Handle different operations
	if unset {
		// Clear remote URL
		return rc.unsetRemote(menvFile)
	} else if len(args) == 0 {
		// Show current remote URL
		return rc.showRemote(menvFile)
	} else {
		// Set new remote URL
		newRemote := args[0]
		return rc.setRemote(menvFile, newRemote)
	}
}

// showRemote displays the current remote URL
func (rc *RemoteCommand) showRemote(menvFile *EasyEnvFile) error {
	if menvFile.Remote == "" {
		fmt.Println("No remote URL configured")
		fmt.Println("Use 'ee remote <url>' to set a remote URL")
	} else {
		fmt.Printf("Remote URL: %s\n", menvFile.Remote)
	}
	return nil
}

// setRemote sets a new remote URL
func (rc *RemoteCommand) setRemote(menvFile *EasyEnvFile, remoteURL string) error {
	// Validate and potentially expand the remote URL
	expandedURL, err := rc.expandRemoteURL(remoteURL)
	if err != nil {
		return fmt.Errorf("invalid remote URL: %w", err)
	}

	// Update the ee file
	menvFile.Remote = expandedURL

	// Save the updated .ee file
	if err := SaveEasyEnvFile(menvFile, ""); err != nil {
		return fmt.Errorf("failed to save .ee file: %w", err)
	}

	fmt.Printf("✅ Remote URL set to: %s\n", expandedURL)

	if remoteURL != expandedURL {
		fmt.Printf("(Expanded from: %s)\n", remoteURL)
	}

	return nil
}

// unsetRemote removes the remote URL
func (rc *RemoteCommand) unsetRemote(menvFile *EasyEnvFile) error {
	if menvFile.Remote == "" {
		fmt.Println("No remote URL configured")
		return nil
	}

	oldRemote := menvFile.Remote
	menvFile.Remote = ""

	// Save the updated .ee file
	if err := SaveEasyEnvFile(menvFile, ""); err != nil {
		return fmt.Errorf("failed to save .ee file: %w", err)
	}

	fmt.Printf("✅ Remote URL removed (was: %s)\n", oldRemote)
	return nil
}

// expandRemoteURL expands shorthand remote URLs to full HTTP URLs
func (rc *RemoteCommand) expandRemoteURL(remoteURL string) (string, error) {
	// If it's already a full HTTP URL, return as-is
	if len(remoteURL) > 7 && (remoteURL[:7] == "http://" || remoteURL[:8] == "https://") {
		return remoteURL, nil
	}

	// Handle shorthand format: company@server/project
	// Example: company@ee.dev/my-project
	// Should expand to: https://ee.dev/-/company/projects/my-project

	// TODO: Implement proper URL expansion logic
	// For now, we'll store the shorthand format and expand it when making API calls
	// This allows flexibility in the future for different server configurations

	if len(remoteURL) == 0 {
		return "", fmt.Errorf("remote URL cannot be empty")
	}

	// Basic validation - should contain @ and /
	// More sophisticated validation will be added when we implement the HTTP client

	return remoteURL, nil
}
