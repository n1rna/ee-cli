// Package command implements the ee auth command for CLI authentication
package command

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/n1rna/ee-cli/internal/config"
	"github.com/n1rna/ee-cli/internal/output"
)

// AuthCommand handles the ee auth command
type AuthCommand struct {
	server *http.Server
	token  chan string
}

// TokenResponse represents the response from token exchange
type TokenResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// NewAuthCommand creates a new ee auth command
func NewAuthCommand(groupId string) *cobra.Command {
	ac := &AuthCommand{
		token: make(chan string, 1),
	}

	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate the CLI with your ee account",
		Long: `Authenticate the CLI with your ee account.

This command will open your browser to log in to your ee account.
After successful authentication, the CLI will store your credentials
for use with remote operations.

Examples:
  # Authenticate with default API endpoint
  ee auth

  # Authenticate with custom API endpoint
  ee auth --api-url https://api.ee.dev

  # Logout (remove stored credentials)
  ee auth logout
`,
		RunE:    ac.Run,
		GroupID: groupId,
	}

	cmd.Flags().StringP("api-url", "u", "", "API endpoint URL (default: from EE_API_URL or config)")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress non-error output")

	// Add logout subcommand
	cmd.AddCommand(&cobra.Command{
		Use:   "logout",
		Short: "Remove stored authentication credentials",
		RunE:  ac.Logout,
	})

	return cmd
}

// Run executes the auth command
func (c *AuthCommand) Run(cmd *cobra.Command, args []string) error {
	// Set up printer
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.FormatTable, quiet)

	// Get API URL from flag or config
	apiURL, _ := cmd.Flags().GetString("api-url")
	if apiURL == "" {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		apiURL = cfg.API.BaseURL
	}

	if apiURL == "" {
		apiURL = "http://localhost:8000" // Default fallback
	}

	printer.Info("Starting CLI authentication...")

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	printer.Info(fmt.Sprintf("Listening for authentication on port %d", port))

	// Set up HTTP server for callback
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", c.handleCallback)
	c.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start server in background
	go func() {
		if err := c.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			printer.Error(fmt.Sprintf("Server error: %v", err))
		}
	}()

	// Build login URL with CLI auth parameters
	loginURL := fmt.Sprintf("%s/auth/login?cliauth=true&redirect=%s",
		apiURL, callbackURL)

	printer.Info("\n🔐 Opening browser for authentication...")
	printer.Info(fmt.Sprintf("If browser doesn't open, visit: %s", loginURL))

	// Open browser
	if err := openBrowser(loginURL); err != nil {
		printer.Warning(fmt.Sprintf("Could not open browser: %v", err))
		printer.Info(fmt.Sprintf("\nPlease manually open: %s", loginURL))
	}

	// Wait for token with timeout
	printer.Info("\nWaiting for authentication...")
	select {
	case tokenCode := <-c.token:
		printer.Success("✓ Authentication code received")

		// Shutdown server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.server.Shutdown(ctx)

		// Exchange token code for access token
		printer.Info("Exchanging token code for access token...")
		accessToken, expiresAt, err := c.exchangeToken(apiURL, tokenCode)
		if err != nil {
			return fmt.Errorf("failed to exchange token: %w", err)
		}

		// Store token
		if err := c.storeToken(accessToken, apiURL, expiresAt); err != nil {
			return fmt.Errorf("failed to store token: %w", err)
		}

		printer.Success("✓ Successfully authenticated!")
		printer.Info(fmt.Sprintf("Token expires at: %s", expiresAt.Format(time.RFC3339)))
		printer.Info("\nYou can now use remote operations (push, pull)")

		return nil

	case <-time.After(5 * time.Minute):
		// Shutdown server on timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.server.Shutdown(ctx)

		return fmt.Errorf("authentication timeout - no response received within 5 minutes")
	}
}

// handleCallback handles the OAuth callback from the browser
func (c *AuthCommand) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No authentication code provided", http.StatusBadRequest)
		return
	}

	// Send token to channel
	select {
	case c.token <- code:
		// Success response
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Authentication Successful</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        }
        .container {
            background: white;
            padding: 3rem;
            border-radius: 1rem;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            text-align: center;
            max-width: 400px;
        }
        .success-icon {
            font-size: 4rem;
            margin-bottom: 1rem;
        }
        h1 {
            color: #2d3748;
            margin-bottom: 0.5rem;
        }
        p {
            color: #718096;
            margin-bottom: 2rem;
        }
        .button {
            background: #667eea;
            color: white;
            padding: 0.75rem 2rem;
            border-radius: 0.5rem;
            text-decoration: none;
            display: inline-block;
            transition: background 0.2s;
        }
        .button:hover {
            background: #5a67d8;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="success-icon">✅</div>
        <h1>Authentication Successful!</h1>
        <p>You can now close this window and return to your terminal.</p>
        <button class="button" onclick="window.close()">Close Window</button>
    </div>
    <script>
        // Auto-close after 3 seconds
        setTimeout(() => window.close(), 3000);
    </script>
</body>
</html>
`))
	default:
		http.Error(w, "Token already received", http.StatusConflict)
	}
}

// exchangeToken exchanges the token code for an access token
func (c *AuthCommand) exchangeToken(apiURL, tokenCode string) (string, time.Time, error) {
	exchangeURL := fmt.Sprintf("%s/api/v2/user/cli/exchange", apiURL)

	reqBody := []byte(fmt.Sprintf(`{"token_code":"%s"}`, tokenCode))
	req, err := http.NewRequest("POST", exchangeURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", time.Time{}, fmt.Errorf("exchange failed (status %d): %s",
			resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return tokenResp.AccessToken, tokenResp.ExpiresAt, nil
}

// storeToken stores the access token in the config directory
func (c *AuthCommand) storeToken(token, apiURL string, expiresAt time.Time) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Ensure config directory exists
	if err := cfg.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to create config directories: %w", err)
	}

	// Store token in a credentials file
	credsPath := filepath.Join(cfg.BaseDir, "credentials.json")
	creds := map[string]interface{}{
		"access_token": token,
		"api_url":      apiURL,
		"expires_at":   expiresAt.Format(time.RFC3339),
		"created_at":   time.Now().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Write with restricted permissions
	if err := os.WriteFile(credsPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}

// Logout removes stored credentials
func (c *AuthCommand) Logout(cmd *cobra.Command, args []string) error {
	quiet, _ := cmd.Flags().GetBool("quiet")
	printer := output.NewPrinter(output.FormatTable, quiet)

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	credsPath := filepath.Join(cfg.BaseDir, "credentials.json")

	// Check if credentials file exists
	if _, err := os.Stat(credsPath); os.IsNotExist(err) {
		printer.Info("No stored credentials found")
		return nil
	}

	// Remove credentials file
	if err := os.Remove(credsPath); err != nil {
		return fmt.Errorf("failed to remove credentials: %w", err)
	}

	printer.Success("✓ Successfully logged out")
	printer.Info("Run 'ee auth' to authenticate again")

	return nil
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

// LoadCredentials loads stored credentials from the config directory
func LoadCredentials() (token, apiURL string, err error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return "", "", fmt.Errorf("failed to load config: %w", err)
	}

	credsPath := filepath.Join(cfg.BaseDir, "credentials.json")

	// Check if credentials file exists
	if _, err := os.Stat(credsPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("not authenticated: run 'ee auth' to authenticate")
	}

	// Read credentials
	data, err := os.ReadFile(credsPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read credentials: %w", err)
	}

	var creds map[string]interface{}
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", "", fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Check expiration
	if expiresAtStr, ok := creds["expires_at"].(string); ok {
		expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
		if err == nil && time.Now().After(expiresAt) {
			return "", "", fmt.Errorf("authentication expired: run 'ee auth' to re-authenticate")
		}
	}

	token, _ = creds["access_token"].(string)
	apiURL, _ = creds["api_url"].(string)

	if token == "" {
		return "", "", fmt.Errorf("invalid credentials: run 'ee auth' to re-authenticate")
	}

	return token, apiURL, nil
}
