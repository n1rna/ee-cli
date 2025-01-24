// internal/schema/types_test.go
package schema

import (
    "testing"
)

func TestValidateSchema(t *testing.T) {
    validator := NewValidator()

    tests := []struct {
        name    string
        schema  Schema
        wantErr bool
    }{
        {
            name: "valid schema",
            schema: Schema{
                Name: "test-schema",
                Variables: []Variable{
                    {
                        Name:     "BASE_URL",
                        Type:     "string",
                        Regex:    "^https?://.*$",
                        Default:  "http://localhost:8000",
                        Required: true,
                    },
                },
            },
            wantErr: false,
        },
        {
            name: "invalid regex",
            schema: Schema{
                Name: "invalid-regex",
                Variables: []Variable{
                    {
                        Name:  "TEST_VAR",
                        Type:  "string",
                        Regex: "[Invalid Regex",
                    },
                },
            },
            wantErr: true,
        },
        {
            name: "invalid type",
            schema: Schema{
                Name: "invalid-type",
                Variables: []Variable{
                    {
                        Name: "TEST_VAR",
                        Type: "invalid_type",
                    },
                },
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validator.ValidateSchema(&tt.schema)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}

func TestValidateConfigSheet(t *testing.T) {
    validator := NewValidator()

    schema := &Schema{
        Name: "test-schema",
        Variables: []Variable{
            {
                Name:     "BASE_URL",
                Type:     "string",
                Regex:    "^https?://.*$",
                Default:  "http://localhost:8000",
                Required: true,
            },
            {
                Name:     "DEBUG",
                Type:     "boolean",
                Required: true,
            },
        },
    }

    tests := []struct {
        name        string
        configSheet ConfigSheet
        wantErr    bool
    }{
        {
            name: "valid config",
            configSheet: ConfigSheet{
                ProjectName: "test-project",
                EnvName:    "development",
                Schema:     "test-schema",
                Values: map[string]string{
                    "BASE_URL": "http://localhost:8000",
                    "DEBUG":    "true",
                },
            },
            wantErr: false,
        },
        {
            name: "missing required value",
            configSheet: ConfigSheet{
                ProjectName: "test-project",
                EnvName:    "development",
                Schema:     "test-schema",
                Values: map[string]string{
                    "BASE_URL": "http://localhost:8000",
                },
            },
            wantErr: true,
        },
        {
            name: "invalid URL format",
            configSheet: ConfigSheet{
                ProjectName: "test-project",
                EnvName:    "development",
                Schema:     "test-schema",
                Values: map[string]string{
                    "BASE_URL": "invalid-url",
                    "DEBUG":    "true",
                },
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validator.ValidateConfigSheet(&tt.configSheet, schema)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateConfigSheet() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}