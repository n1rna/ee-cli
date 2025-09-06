package schema

import (
	"testing"
	"time"
)

func TestNewEntity(t *testing.T) {
	name := "test-entity"
	description := "Test entity description"
	
	entity := NewEntity(name, description)
	
	if entity.ID == "" {
		t.Error("Entity ID should not be empty")
	}
	
	if entity.Name != name {
		t.Errorf("Expected name %s, got %s", name, entity.Name)
	}
	
	if entity.Description != description {
		t.Errorf("Expected description %s, got %s", description, entity.Description)
	}
	
	if !entity.Local {
		t.Error("New entity should be local by default")
	}
	
	if entity.Remote != "" {
		t.Error("New entity should not have remote URL by default")
	}
	
	// Check that timestamps are recent (within last minute)
	now := time.Now()
	if now.Sub(entity.CreatedAt) > time.Minute {
		t.Error("CreatedAt timestamp should be recent")
	}
	
	if now.Sub(entity.UpdatedAt) > time.Minute {
		t.Error("UpdatedAt timestamp should be recent")
	}
}

func TestNewSchema(t *testing.T) {
	variables := []Variable{
		{
			Name:     "DATABASE_URL",
			Title:    "Database Connection URL",
			Type:     "url",
			Required: true,
		},
		{
			Name:     "DEBUG",
			Type:     "boolean",
			Default:  "false",
			Required: false,
		},
	}
	
	extends := []string{"base-schema"}
	
	schema := NewSchema("api-schema", "API service schema", variables, extends)
	
	if schema.ID == "" {
		t.Error("Schema ID should not be empty")
	}
	
	if schema.Name != "api-schema" {
		t.Errorf("Expected name 'api-schema', got %s", schema.Name)
	}
	
	if len(schema.Variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(schema.Variables))
	}
	
	if len(schema.Extends) != 1 {
		t.Errorf("Expected 1 extended schema, got %d", len(schema.Extends))
	}
	
	// Test NewSchema with nil extends
	schema2 := NewSchema("simple-schema", "Simple schema", variables, nil)
	if len(schema2.Extends) != 0 {
		t.Errorf("Expected 0 extended schemas for nil extends, got %d", len(schema2.Extends))
	}
}

func TestNewProject(t *testing.T) {
	schemaID := "550e8400-e29b-41d4-a716-446655440000"
	
	project := NewProject("my-api", "My API project", schemaID)
	
	if project.ID == "" {
		t.Error("Project ID should not be empty")
	}
	
	if project.Name != "my-api" {
		t.Errorf("Expected name 'my-api', got %s", project.Name)
	}
	
	if project.Schema != schemaID {
		t.Errorf("Expected schema ID %s, got %s", schemaID, project.Schema)
	}
	
	if project.Environments == nil {
		t.Error("Environments map should be initialized")
	}
	
	if len(project.Environments) != 0 {
		t.Errorf("Expected 0 environments initially, got %d", len(project.Environments))
	}
}

func TestProjectAddEnvironment(t *testing.T) {
	project := NewProject("my-api", "My API project", "schema-id")
	
	project.AddEnvironment("development")
	
	if len(project.Environments) != 1 {
		t.Errorf("Expected 1 environment after adding, got %d", len(project.Environments))
	}
	
	env, exists := project.Environments["development"]
	if !exists {
		t.Error("Development environment should exist")
	}
	
	if env.Name != "development" {
		t.Errorf("Expected environment name 'development', got %s", env.Name)
	}
}

func TestSchemaReference(t *testing.T) {
	// Test reference-based SchemaReference
	refSchema := SchemaReference{
		Ref: "#/schemas/550e8400-e29b-41d4-a716-446655440000",
	}
	
	if !refSchema.IsReference() {
		t.Error("Should be detected as reference")
	}
	
	if refSchema.IsInline() {
		t.Error("Should not be detected as inline")
	}
	
	// Test inline SchemaReference
	inlineSchema := SchemaReference{
		Variables: map[string]Variable{
			"PORT": {
				Name: "PORT",
				Type: "number",
				Default: "3000",
			},
		},
	}
	
	if !inlineSchema.IsInline() {
		t.Error("Should be detected as inline")
	}
	
	if inlineSchema.IsReference() {
		t.Error("Should not be detected as reference")
	}
}

func TestNewConfigSheet(t *testing.T) {
	schemaRef := SchemaReference{
		Ref: "#/schemas/api-schema",
	}
	
	values := map[string]string{
		"DATABASE_URL": "postgresql://localhost:5432/myapi",
		"DEBUG":        "true",
	}
	
	sheet := NewConfigSheet("my-config", "My configuration", schemaRef, values)
	
	if sheet.ID == "" {
		t.Error("ConfigSheet ID should not be empty")
	}
	
	if sheet.Name != "my-config" {
		t.Errorf("Expected name 'my-config', got %s", sheet.Name)
	}
	
	if !sheet.Schema.IsReference() {
		t.Error("Schema should be reference-based")
	}
	
	if len(sheet.Values) != 2 {
		t.Errorf("Expected 2 values, got %d", len(sheet.Values))
	}
	
	if sheet.Values["DATABASE_URL"] != values["DATABASE_URL"] {
		t.Error("DATABASE_URL value mismatch")
	}
	
	if len(sheet.Extends) != 0 {
		t.Errorf("Expected 0 extends initially, got %d", len(sheet.Extends))
	}
}

func TestConfigSheetTypes(t *testing.T) {
	// Test standalone config sheet
	standaloneSheet := NewConfigSheet("standalone", "Standalone sheet", 
		SchemaReference{Ref: "#/schemas/test"}, nil)
	
	if !standaloneSheet.IsStandalone() {
		t.Error("Should be detected as standalone")
	}
	
	if standaloneSheet.IsProjectEnvironment() {
		t.Error("Should not be detected as project environment")
	}
	
	// Test project environment config sheet
	projectSheet := NewConfigSheetForProject("project-dev", "Project dev sheet",
		SchemaReference{Ref: "#/schemas/test"}, "project-uuid", "development", nil)
	
	if projectSheet.IsStandalone() {
		t.Error("Should not be detected as standalone")
	}
	
	if !projectSheet.IsProjectEnvironment() {
		t.Error("Should be detected as project environment")
	}
	
	if projectSheet.Project != "project-uuid" {
		t.Errorf("Expected project UUID 'project-uuid', got %s", projectSheet.Project)
	}
	
	if projectSheet.Environment != "development" {
		t.Errorf("Expected environment 'development', got %s", projectSheet.Environment)
	}
}

func TestIndex(t *testing.T) {
	index := NewIndex()
	
	if index.NameToID == nil {
		t.Error("NameToID map should be initialized")
	}
	
	if index.Summaries == nil {
		t.Error("Summaries map should be initialized")
	}
	
	// Test adding entity to index
	entity := NewEntity("test-schema", "Test schema description")
	index.AddEntity(entity)
	
	// Test name-to-UUID resolution
	resolvedUUID, exists := index.ResolveUUID("test-schema")
	if !exists {
		t.Error("Should resolve name to UUID")
	}
	
	if resolvedUUID != entity.ID {
		t.Errorf("Expected UUID %s, got %s", entity.ID, resolvedUUID)
	}
	
	// Test UUID-to-UUID resolution
	resolvedUUID2, exists := index.ResolveUUID(entity.ID)
	if !exists {
		t.Error("Should resolve UUID to UUID")
	}
	
	if resolvedUUID2 != entity.ID {
		t.Errorf("Expected UUID %s, got %s", entity.ID, resolvedUUID2)
	}
	
	// Test getting summary
	summary, exists := index.GetSummary("test-schema")
	if !exists {
		t.Error("Should get summary by name")
	}
	
	if summary.Name != entity.Name {
		t.Errorf("Expected name %s, got %s", entity.Name, summary.Name)
	}
	
	// Test listing summaries
	summaries := index.ListSummaries()
	if len(summaries) != 1 {
		t.Errorf("Expected 1 summary, got %d", len(summaries))
	}
	
	// Test removing entity
	index.RemoveEntity("test-schema")
	
	_, exists = index.ResolveUUID("test-schema")
	if exists {
		t.Error("Entity should be removed from index")
	}
}