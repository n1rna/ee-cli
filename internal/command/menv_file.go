// Package command provides utilities for managing .ee project files
// These utilities support the new project-based workflow as specified in docs/entities.md
package command

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// EasyEnvFile represents the structure of a .ee project configuration file
type EasyEnvFile struct {
	Remote  string `json:"remote,omitempty"`  // Remote API base URL
	Project string `json:"project,omitempty"` // Project UUID
}

// DefaultEasyEnvFile returns a default .ee file structure
func DefaultEasyEnvFile() *EasyEnvFile {
	return &EasyEnvFile{}
}

// LoadEasyEnvFile loads a .ee file from the specified directory
// If no directory is specified, uses the current working directory
func LoadEasyEnvFile(dir string) (*EasyEnvFile, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	menvPath := filepath.Join(dir, ".ee")

	// Check if .ee file exists
	if _, err := os.Stat(menvPath); os.IsNotExist(err) {
		return nil, fmt.Errorf(".ee file not found in %s", dir)
	}

	// Read the file
	data, err := os.ReadFile(menvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read .ee file: %w", err)
	}

	// Parse JSON
	var menvFile EasyEnvFile
	if err := json.Unmarshal(data, &menvFile); err != nil {
		return nil, fmt.Errorf("failed to parse .ee file: %w", err)
	}

	return &menvFile, nil
}

// SaveEasyEnvFile saves a .ee file to the specified directory
// If no directory is specified, uses the current working directory
func SaveEasyEnvFile(menvFile *EasyEnvFile, dir string) error {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	menvPath := filepath.Join(dir, ".ee")

	// Marshal to JSON with proper formatting
	data, err := json.MarshalIndent(menvFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal .ee file: %w", err)
	}

	// Write to file
	if err := os.WriteFile(menvPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write .ee file: %w", err)
	}

	return nil
}

// EasyEnvFileExists checks if a .ee file exists in the specified directory
func EasyEnvFileExists(dir string) bool {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return false
		}
	}

	menvPath := filepath.Join(dir, ".ee")
	_, err := os.Stat(menvPath)
	return err == nil
}

// GetCurrentProject returns the project UUID from the .ee file in the current directory
// Returns empty string if no .ee file exists or no project is specified
func GetCurrentProject() (string, error) {
	menvFile, err := LoadEasyEnvFile("")
	if err != nil {
		return "", err
	}

	return menvFile.Project, nil
}

// GetCurrentRemote returns the remote URL from the .ee file in the current directory
// Returns empty string if no .ee file exists or no remote is specified
func GetCurrentRemote() (string, error) {
	menvFile, err := LoadEasyEnvFile("")
	if err != nil {
		return "", err
	}

	return menvFile.Remote, nil
}

// SetCurrentProject sets the project UUID in the .ee file in the current directory
func SetCurrentProject(projectUUID string) error {
	var menvFile *EasyEnvFile

	// Try to load existing .ee file, or create new one
	if EasyEnvFileExists("") {
		var err error
		menvFile, err = LoadEasyEnvFile("")
		if err != nil {
			return fmt.Errorf("failed to load existing .ee file: %w", err)
		}
	} else {
		menvFile = DefaultEasyEnvFile()
	}

	// Update project UUID
	menvFile.Project = projectUUID

	// Save the file
	return SaveEasyEnvFile(menvFile, "")
}

// SetCurrentRemote sets the remote URL in the .ee file in the current directory
func SetCurrentRemote(remoteURL string) error {
	var menvFile *EasyEnvFile

	// Try to load existing .ee file, or create new one
	if EasyEnvFileExists("") {
		var err error
		menvFile, err = LoadEasyEnvFile("")
		if err != nil {
			return fmt.Errorf("failed to load existing .ee file: %w", err)
		}
	} else {
		menvFile = DefaultEasyEnvFile()
	}

	// Update remote URL
	menvFile.Remote = remoteURL

	// Save the file
	return SaveEasyEnvFile(menvFile, "")
}

// ValidateEasyEnvFile validates the structure and content of a .ee file
func ValidateEasyEnvFile(menvFile *EasyEnvFile) error {
	if menvFile == nil {
		return fmt.Errorf(".ee file cannot be nil")
	}

	// Project UUID should be a valid UUID format if specified
	if menvFile.Project != "" {
		// Basic UUID format validation (can be enhanced later)
		if len(menvFile.Project) != 36 {
			return fmt.Errorf("invalid project UUID format: %s", menvFile.Project)
		}
	}

	// Remote URL should be a valid HTTP/HTTPS URL if specified
	if menvFile.Remote != "" {
		// Basic URL format validation (can be enhanced later)
		if !(len(menvFile.Remote) > 7 && (menvFile.Remote[:7] == "http://" || menvFile.Remote[:8] == "https://")) {
			return fmt.Errorf("invalid remote URL format: %s", menvFile.Remote)
		}
	}

	return nil
}

// CreateEasyEnvFile creates a new .ee file in the specified directory
func CreateEasyEnvFile(projectUUID, remoteURL, dir string) error {
	menvFile := &EasyEnvFile{
		Project: projectUUID,
		Remote:  remoteURL,
	}

	// Validate the file
	if err := ValidateEasyEnvFile(menvFile); err != nil {
		return fmt.Errorf("invalid .ee file: %w", err)
	}

	// Check if .ee already exists
	if EasyEnvFileExists(dir) {
		return fmt.Errorf(".ee file already exists in directory")
	}

	// Save the file
	return SaveEasyEnvFile(menvFile, dir)
}

// UpdateEasyEnvFile updates an existing .ee file or creates a new one
func UpdateEasyEnvFile(projectUUID, remoteURL, dir string) error {
	var menvFile *EasyEnvFile

	// Try to load existing .ee file, or create new one
	if EasyEnvFileExists(dir) {
		var err error
		menvFile, err = LoadEasyEnvFile(dir)
		if err != nil {
			return fmt.Errorf("failed to load existing .ee file: %w", err)
		}
	} else {
		menvFile = DefaultEasyEnvFile()
	}

	// Update fields if provided
	if projectUUID != "" {
		menvFile.Project = projectUUID
	}
	if remoteURL != "" {
		menvFile.Remote = remoteURL
	}

	// Validate the updated file
	if err := ValidateEasyEnvFile(menvFile); err != nil {
		return fmt.Errorf("invalid updated .ee file: %w", err)
	}

	// Save the file
	return SaveEasyEnvFile(menvFile, dir)
}

// DeleteEasyEnvFile removes the .ee file from the specified directory
func DeleteEasyEnvFile(dir string) error {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	menvPath := filepath.Join(dir, ".ee")

	// Check if file exists
	if !EasyEnvFileExists(dir) {
		return fmt.Errorf(".ee file does not exist in %s", dir)
	}

	// Remove the file
	if err := os.Remove(menvPath); err != nil {
		return fmt.Errorf("failed to remove .ee file: %w", err)
	}

	return nil
}
