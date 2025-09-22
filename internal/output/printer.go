// Package output provides formatted terminal output for ee entities.
// This centralizes all printing and formatting logic away from command modules.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/n1rna/ee-cli/internal/entities"
)

// Format represents different output formats
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
	FormatCSV   Format = "csv"
)

// Printer handles formatted output to the terminal
type Printer struct {
	writer io.Writer
	format Format
	quiet  bool
}

// NewPrinter creates a new printer with the specified format
func NewPrinter(format Format, quiet bool) *Printer {
	return &Printer{
		writer: os.Stdout,
		format: format,
		quiet:  quiet,
	}
}

// NewPrinterWithWriter creates a new printer with a custom writer
func NewPrinterWithWriter(writer io.Writer, format Format, quiet bool) *Printer {
	return &Printer{
		writer: writer,
		format: format,
		quiet:  quiet,
	}
}

// Success prints a success message
func (p *Printer) Success(message string) {
	if !p.quiet {
		fmt.Fprintf(p.writer, "✓ %s\n", message)
	}
}

// Error prints an error message
func (p *Printer) Error(message string) {
	fmt.Fprintf(p.writer, "✗ %s\n", message)
}

// Warning prints a warning message
func (p *Printer) Warning(message string) {
	if !p.quiet {
		fmt.Fprintf(p.writer, "⚠ %s\n", message)
	}
}

// Info prints an informational message
func (p *Printer) Info(message string) {
	if !p.quiet {
		fmt.Fprintf(p.writer, "ℹ %s\n", message)
	}
}

// PrintSchema prints a schema in the specified format
func (p *Printer) PrintSchema(s *entities.Schema) error {
	switch p.format {
	case FormatTable:
		return p.printSchemaTable(s)
	case FormatJSON:
		return p.printJSON(s)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// PrintSchemaList prints a list of schema summaries
func (p *Printer) PrintSchemaList(summaries []entities.EntitySummary) error {
	switch p.format {
	case FormatTable:
		return p.printSchemaListTable(summaries)
	case FormatJSON:
		return p.printJSON(summaries)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// PrintProject prints a project in the specified format
func (p *Printer) PrintProject(proj *entities.Project) error {
	switch p.format {
	case FormatTable:
		return p.printProjectTable(proj)
	case FormatJSON:
		return p.printJSON(proj)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// PrintProjectList prints a list of project summaries
func (p *Printer) PrintProjectList(summaries []entities.EntitySummary) error {
	switch p.format {
	case FormatTable:
		return p.printProjectListTable(summaries)
	case FormatJSON:
		return p.printJSON(summaries)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// PrintConfigSheet prints a config sheet in the specified format
func (p *Printer) PrintConfigSheet(cs *entities.ConfigSheet) error {
	switch p.format {
	case FormatTable:
		return p.printConfigSheetTable(cs)
	case FormatJSON:
		return p.printJSON(cs)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// PrintConfigSheetList prints a list of config sheet summaries
func (p *Printer) PrintConfigSheetList(summaries []entities.EntitySummary) error {
	switch p.format {
	case FormatTable:
		return p.printConfigSheetListTable(summaries)
	case FormatJSON:
		return p.printJSON(summaries)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// PrintValues prints environment variable values
func (p *Printer) PrintValues(values map[string]string) error {
	switch p.format {
	case FormatTable:
		return p.printValuesTable(values)
	case FormatJSON:
		return p.printJSON(values)
	default:
		return fmt.Errorf("unsupported format: %s", p.format)
	}
}

// printSchemaTable prints a schema in table format
func (p *Printer) printSchemaTable(s *entities.Schema) error {
	fmt.Fprintf(p.writer, "Schema: %s\n", s.Name)
	fmt.Fprintf(p.writer, "ID: %s\n", s.ID)
	if s.Description != "" {
		fmt.Fprintf(p.writer, "Description: %s\n", s.Description)
	}
	fmt.Fprintf(p.writer, "Created: %s\n", s.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(p.writer, "Updated: %s\n", s.UpdatedAt.Format(time.RFC3339))

	if len(s.Extends) > 0 {
		fmt.Fprintf(p.writer, "Extends: %s\n", strings.Join(s.Extends, ", "))
	}

	fmt.Fprintf(p.writer, "\nVariables:\n")
	if len(s.Variables) == 0 {
		fmt.Fprintf(p.writer, "  No variables defined\n")
		return nil
	}

	w := tabwriter.NewWriter(p.writer, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  NAME\tTYPE\tREQUIRED\tDEFAULT\tREGEX\n")
	fmt.Fprintf(w, "  ----\t----\t--------\t-------\t-----\n")

	for _, variable := range s.Variables {
		required := "No"
		if variable.Required {
			required = "Yes"
		}

		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\n",
			variable.Name,
			variable.Type,
			required,
			variable.Default,
			variable.Regex,
		)
	}

	return w.Flush()
}

// printSchemaListTable prints a list of schemas in table format
func (p *Printer) printSchemaListTable(summaries []entities.EntitySummary) error {
	if len(summaries) == 0 {
		fmt.Fprintf(p.writer, "No schemas found\n")
		return nil
	}

	// Sort by name
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	w := tabwriter.NewWriter(p.writer, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "NAME\tDESCRIPTION\tCREATED\n")
	fmt.Fprintf(w, "----\t-----------\t-------\n")

	for _, summary := range summaries {
		desc := summary.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n",
			summary.Name,
			desc,
			summary.CreatedAt.Format("2006-01-02"),
		)
	}

	return w.Flush()
}

// printProjectTable prints a project in table format
func (p *Printer) printProjectTable(proj *entities.Project) error {
	fmt.Fprintf(p.writer, "Project: %s\n", proj.Name)
	fmt.Fprintf(p.writer, "ID: %s\n", proj.ID)
	if proj.Description != "" {
		fmt.Fprintf(p.writer, "Description: %s\n", proj.Description)
	}
	fmt.Fprintf(p.writer, "Schema: %s\n", proj.Schema)
	fmt.Fprintf(p.writer, "Created: %s\n", proj.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(p.writer, "Updated: %s\n", proj.UpdatedAt.Format(time.RFC3339))

	fmt.Fprintf(p.writer, "\nEnvironments:\n")
	if len(proj.Environments) == 0 {
		fmt.Fprintf(p.writer, "  No environments defined\n")
		return nil
	}

	// Sort environment names
	envNames := make([]string, 0, len(proj.Environments))
	for name := range proj.Environments {
		envNames = append(envNames, name)
	}
	sort.Strings(envNames)

	for _, name := range envNames {
		fmt.Fprintf(p.writer, "  - %s\n", name)
	}

	return nil
}

// printProjectListTable prints a list of projects in table format
func (p *Printer) printProjectListTable(summaries []entities.EntitySummary) error {
	if len(summaries) == 0 {
		fmt.Fprintf(p.writer, "No projects found\n")
		return nil
	}

	// Sort by name
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	w := tabwriter.NewWriter(p.writer, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "NAME\tDESCRIPTION\tCREATED\n")
	fmt.Fprintf(w, "----\t-----------\t-------\n")

	for _, summary := range summaries {
		desc := summary.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n",
			summary.Name,
			desc,
			summary.CreatedAt.Format("2006-01-02"),
		)
	}

	return w.Flush()
}

// printConfigSheetTable prints a config sheet in table format
func (p *Printer) printConfigSheetTable(cs *entities.ConfigSheet) error {
	fmt.Fprintf(p.writer, "Config Sheet: %s\n", cs.Name)
	fmt.Fprintf(p.writer, "ID: %s\n", cs.ID)
	if cs.Description != "" {
		fmt.Fprintf(p.writer, "Description: %s\n", cs.Description)
	}

	if cs.Project != "" {
		fmt.Fprintf(p.writer, "Project: %s\n", cs.Project)
	}
	if cs.Environment != "" {
		fmt.Fprintf(p.writer, "Environment: %s\n", cs.Environment)
	}

	if cs.Schema.IsReference() {
		fmt.Fprintf(p.writer, "Schema Reference: %s\n", cs.Schema.Ref)
	} else if cs.Schema.IsInline() {
		fmt.Fprintf(p.writer, "Schema: Inline (%d variables)\n", len(cs.Schema.Variables))
	}

	fmt.Fprintf(p.writer, "Created: %s\n", cs.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(p.writer, "Updated: %s\n", cs.UpdatedAt.Format(time.RFC3339))

	if len(cs.Extends) > 0 {
		fmt.Fprintf(p.writer, "Extends: %s\n", strings.Join(cs.Extends, ", "))
	}

	fmt.Fprintf(p.writer, "\nValues:\n")
	return p.printValuesTable(cs.Values)
}

// printConfigSheetListTable prints a list of config sheets in table format
func (p *Printer) printConfigSheetListTable(summaries []entities.EntitySummary) error {
	if len(summaries) == 0 {
		fmt.Fprintf(p.writer, "No config sheets found\n")
		return nil
	}

	// Sort by name
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	w := tabwriter.NewWriter(p.writer, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "NAME\tDESCRIPTION\tCREATED\n")
	fmt.Fprintf(w, "----\t-----------\t-------\n")

	for _, summary := range summaries {
		desc := summary.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n",
			summary.Name,
			desc,
			summary.CreatedAt.Format("2006-01-02"),
		)
	}

	return w.Flush()
}

// printValuesTable prints variable values in table format
func (p *Printer) printValuesTable(values map[string]string) error {
	if len(values) == 0 {
		fmt.Fprintf(p.writer, "  No values defined\n")
		return nil
	}

	// Sort keys
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	w := tabwriter.NewWriter(p.writer, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  VARIABLE\tVALUE\n")
	fmt.Fprintf(w, "  --------\t-----\n")

	for _, key := range keys {
		value := values[key]
		// Truncate long values
		if len(value) > 80 {
			value = value[:77] + "..."
		}
		fmt.Fprintf(w, "  %s\t%s\n", key, value)
	}

	return w.Flush()
}

// printJSON prints any object as JSON
func (p *Printer) printJSON(obj interface{}) error {
	encoder := json.NewEncoder(p.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(obj)
}

// PrintEnvironmentExport prints environment variables in export format
func (p *Printer) PrintEnvironmentExport(values map[string]string) error {
	// Sort keys for consistent output
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := values[key]
		// Escape quotes in value
		value = strings.ReplaceAll(value, "\"", "\\\"")
		fmt.Fprintf(p.writer, "export %s=\"%s\"\n", key, value)
	}

	return nil
}

// PrintDotEnv prints environment variables in .env format
func (p *Printer) PrintDotEnv(values map[string]string) error {
	// Sort keys for consistent output
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := values[key]
		// Escape quotes and newlines in value
		value = strings.ReplaceAll(value, "\\", "\\\\")
		value = strings.ReplaceAll(value, "\"", "\\\"")
		value = strings.ReplaceAll(value, "\n", "\\n")
		fmt.Fprintf(p.writer, "%s=\"%s\"\n", key, value)
	}

	return nil
}