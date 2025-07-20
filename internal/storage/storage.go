// Package storage handles all file system operations for menv.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/n1rna/menv/internal/config"
	"github.com/n1rna/menv/internal/schema"
	"gopkg.in/yaml.v3"
)

const (
	schemasDir  = "schemas"
	projectsDir = "projects"
)

// Storage handles all file system operations for menv
type Storage struct {
	config      *config.Config
	schemaCache map[string]*schema.Schema
	configCache map[string]*schema.ConfigSheet
}

// GetBaseDir returns the current base directory
func (s *Storage) GetBaseDir() string {
	return s.config.BaseDir
}

// getSchemasPath returns the path to schemas directory
func (s *Storage) getSchemasPath() string {
	return filepath.Join(s.config.BaseDir, schemasDir)
}

// getProjectsPath returns the path to projects directory
func (s *Storage) getProjectsPath() string {
	return filepath.Join(s.config.BaseDir, projectsDir)
}

// getProjectPath returns the path to a specific project directory
func (s *Storage) getProjectPath(projectName string) string {
	return filepath.Join(s.getProjectsPath(), projectName)
}

// getConfigSheetPath returns the path to a specific config sheet
func (s *Storage) getConfigSheetPath(projectName, envName string) string {
	return filepath.Join(s.getProjectPath(projectName), fmt.Sprintf("%s.yaml", envName))
}

// NewStorage creates a new storage instance with the given configuration.
func NewStorage(cfg *config.Config) (*Storage, error) {
	if cfg == nil {
		var err error
		cfg, err = config.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	s := &Storage{
		config:      cfg,
		schemaCache: make(map[string]*schema.Schema),
		configCache: make(map[string]*schema.ConfigSheet),
	}

	if err := cfg.EnsureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to initialize directories: %w", err)
	}

	return s, nil
}

// LoadSchema with caching and inheritance support
func (s *Storage) LoadSchema(name string) (*schema.Schema, error) {
	// Check cache first
	if cached, ok := s.schemaCache[name]; ok {
		return cached, nil
	}

	schemaPath := filepath.Join(s.getSchemasPath(), fmt.Sprintf("%s.yaml", name))
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema schema.Schema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	// Cache the schema
	s.schemaCache[name] = &schema
	return &schema, nil
}

// LoadConfigSheet with caching and inheritance support
func (s *Storage) LoadConfigSheet(projectName, envName string) (*schema.ConfigSheet, error) {
	cacheKey := fmt.Sprintf("%s:%s", projectName, envName)

	// Check cache first
	if cached, ok := s.configCache[cacheKey]; ok {
		return cached, nil
	}

	configPath := s.getConfigSheetPath(projectName, envName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var configSheet schema.ConfigSheet
	if err := yaml.Unmarshal(data, &configSheet); err != nil {
		return nil, fmt.Errorf("failed to parse config sheet: %w", err)
	}

	// Cache the config sheet
	s.configCache[cacheKey] = &configSheet
	return &configSheet, nil
}

// SaveSchema with cache invalidation
func (s *Storage) SaveSchema(schema *schema.Schema) error {
	if schema.Name == "" {
		return fmt.Errorf("schema name cannot be empty")
	}

	schemaPath := filepath.Join(s.getSchemasPath(), fmt.Sprintf("%s.yaml", schema.Name))
	file, err := os.Create(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to create schema file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("warning: failed to close file: %v\n", err)
		}
	}()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(schema); err != nil {
		return fmt.Errorf("failed to encode schema: %w", err)
	}

	// Invalidate cache
	delete(s.schemaCache, schema.Name)
	return nil
}

// SaveConfigSheet with cache invalidation
func (s *Storage) SaveConfigSheet(configSheet *schema.ConfigSheet) error {
	if configSheet.ProjectName == "" || configSheet.EnvName == "" {
		return fmt.Errorf("project name and environment name cannot be empty")
	}

	projectPath := s.getProjectPath(configSheet.ProjectName)
	if err := os.MkdirAll(projectPath, 0750); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	configPath := s.getConfigSheetPath(configSheet.ProjectName, configSheet.EnvName)
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("warning: failed to close file: %v\n", err)
		}
	}()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(configSheet); err != nil {
		return fmt.Errorf("failed to encode config sheet: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("%s:%s", configSheet.ProjectName, configSheet.EnvName)
	delete(s.configCache, cacheKey)
	return nil
}

// ListSchemas returns all available schema names
func (s *Storage) ListSchemas() ([]string, error) {
	entries, err := os.ReadDir(s.getSchemasPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read schemas directory: %w", err)
	}

	schemas := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			schemas = append(schemas, strings.TrimSuffix(entry.Name(), ".yaml"))
		}
	}

	return schemas, nil
}

// ListProjects returns all available project names
func (s *Storage) ListProjects() ([]string, error) {
	entries, err := os.ReadDir(s.getProjectsPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	projects := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			projects = append(projects, entry.Name())
		}
	}

	return projects, nil
}

// ListEnvironments returns all available environments for a project
func (s *Storage) ListEnvironments(projectName string) ([]string, error) {
	projectPath := s.getProjectPath(projectName)
	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read project directory: %w", err)
	}

	envs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			envs = append(envs, strings.TrimSuffix(entry.Name(), ".yaml"))
		}
	}

	return envs, nil
}

// DeleteProject removes a project and all its configurations
func (s *Storage) DeleteProject(projectName string) error {
	projectPath := s.getProjectPath(projectName)
	if err := os.RemoveAll(projectPath); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}

// DeleteEnvironment removes a specific environment configuration from a project
func (s *Storage) DeleteEnvironment(projectName, envName string) error {
	configPath := s.getConfigSheetPath(projectName, envName)
	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("failed to delete environment: %w", err)
	}
	// Invalidate cache
	cacheKey := fmt.Sprintf("%s:%s", projectName, envName)
	delete(s.configCache, cacheKey)
	return nil
}

// DeleteSchema removes a schema
func (s *Storage) DeleteSchema(name string) error {
	schemaPath := filepath.Join(s.getSchemasPath(), fmt.Sprintf("%s.yaml", name))
	if err := os.Remove(schemaPath); err != nil {
		return fmt.Errorf("failed to delete schema file: %w", err)
	}
	// Invalidate cache
	delete(s.schemaCache, name)
	return nil
}
