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
	"time"

	"github.com/pterm/pterm"

	"github.com/n1rna/ee-cli/internal/entities"
	"github.com/n1rna/ee-cli/internal/storage"
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

// printf is a helper that handles fmt.Fprintf errors
func (p *Printer) printf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(p.writer, format, args...)
}

// Success prints a success message
func (p *Printer) Success(message string) {
	if !p.quiet {
		pterm.Success.Println(message)
	}
}

// Error prints an error message
func (p *Printer) Error(message string) {
	pterm.Error.Println(message)
}

// Warning prints a warning message
func (p *Printer) Warning(message string) {
	if !p.quiet {
		pterm.Warning.Println(message)
	}
}

// Info prints an informational message
func (p *Printer) Info(message string) {
	if !p.quiet {
		pterm.Info.Println(message)
	}
}

// Printf prints a formatted message
func (p *Printer) Printf(format string, args ...interface{}) {
	if !p.quiet {
		pterm.Printf(format, args...)
	}
}

// Println prints a line
func (p *Printer) Println(message string) {
	if !p.quiet {
		pterm.Println(message)
	}
}

// Sprintf returns a formatted string (for building complex messages)
func (p *Printer) Sprintf(format string, args ...interface{}) string {
	return pterm.Sprintf(format, args...)
}

// Debug prints a debug message
func (p *Printer) Debug(message string) {
	if !p.quiet {
		pterm.Debug.Println(message)
	}
}

// Fatal prints an error message and exits
func (p *Printer) Fatal(message string) {
	pterm.Fatal.Println(message)
}

// PrintChange prints a change notification (e.g., "Field: old → new")
func (p *Printer) PrintChange(field, oldValue, newValue string) {
	if !p.quiet {
		pterm.Printf("  %s: %s → %s\n",
			pterm.LightYellow(field),
			pterm.Gray(oldValue),
			pterm.LightGreen(newValue))
	}
}

// PrintUpdate prints an update notification
func (p *Printer) PrintUpdate(message string) {
	if !p.quiet {
		pterm.Printf("  %s\n", pterm.LightBlue(message))
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
func (p *Printer) PrintSchemaList(summaries []storage.EntitySummary) error {
	switch p.format {
	case FormatTable:
		return p.printSchemaListTable(summaries)
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
func (p *Printer) PrintConfigSheetList(summaries []storage.EntitySummary) error {
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
	// Print header info
	pterm.DefaultSection.Println("Schema: " + s.Name)
	pterm.Println(pterm.Gray("ID: " + s.ID))
	if s.Description != "" {
		pterm.Println(pterm.LightCyan(s.Description))
	}
	pterm.Println(pterm.Gray("Created: " + s.CreatedAt.Format(time.RFC3339)))
	pterm.Println(pterm.Gray("Updated: " + s.UpdatedAt.Format(time.RFC3339)))

	if len(s.Extends) > 0 {
		pterm.Println(pterm.Gray("Extends: " + strings.Join(s.Extends, ", ")))
	}

	pterm.Println()
	pterm.DefaultHeader.Println("Variables")

	if len(s.Variables) == 0 {
		pterm.Info.Println("No variables defined")
		return nil
	}

	// Build table data
	tableData := pterm.TableData{
		{"NAME", "TYPE", "REQUIRED", "DEFAULT", "REGEX"},
	}

	for _, variable := range s.Variables {
		required := "No"
		if variable.Required {
			required = "Yes"
		}

		tableData = append(tableData, []string{
			variable.Name,
			variable.Type,
			required,
			variable.Default,
			variable.Regex,
		})
	}

	return pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

// printSchemaListTable prints a list of schemas in table format
func (p *Printer) printSchemaListTable(summaries []storage.EntitySummary) error {
	if len(summaries) == 0 {
		pterm.Info.Println("No schemas found")
		return nil
	}

	// Sort by name
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	// Build table data
	tableData := pterm.TableData{
		{"NAME", "DESCRIPTION", "CREATED"},
	}

	for _, summary := range summaries {
		desc := summary.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		tableData = append(tableData, []string{
			summary.Name,
			desc,
			summary.CreatedAt.Format("2006-01-02"),
		})
	}

	return pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

// printConfigSheetTable prints a config sheet in table format
func (p *Printer) printConfigSheetTable(cs *entities.ConfigSheet) error {
	// Print header info
	pterm.DefaultSection.Println("Config Sheet: " + cs.Name)
	pterm.Println(pterm.Gray("ID: " + cs.ID))
	if cs.Description != "" {
		pterm.Println(pterm.LightCyan(cs.Description))
	}

	if cs.Schema.IsReference() {
		pterm.Println(pterm.Gray("Schema Reference: " + cs.Schema.Ref))
	} else if cs.Schema.IsInline() {
		pterm.Println(pterm.Gray(pterm.Sprintf("Schema: Inline (%d variables)", len(cs.Schema.Variables))))
	}

	pterm.Println(pterm.Gray("Created: " + cs.CreatedAt.Format(time.RFC3339)))
	pterm.Println(pterm.Gray("Updated: " + cs.UpdatedAt.Format(time.RFC3339)))

	if len(cs.Extends) > 0 {
		pterm.Println(pterm.Gray("Extends: " + strings.Join(cs.Extends, ", ")))
	}

	pterm.Println()
	pterm.DefaultHeader.Println("Values")
	return p.printValuesTable(cs.Values)
}

// printConfigSheetListTable prints a list of config sheets in table format
func (p *Printer) printConfigSheetListTable(summaries []storage.EntitySummary) error {
	if len(summaries) == 0 {
		pterm.Info.Println("No config sheets found")
		return nil
	}

	// Sort by name
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	// Build table data
	tableData := pterm.TableData{
		{"NAME", "DESCRIPTION", "CREATED"},
	}

	for _, summary := range summaries {
		desc := summary.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		tableData = append(tableData, []string{
			summary.Name,
			desc,
			summary.CreatedAt.Format("2006-01-02"),
		})
	}

	return pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

// printValuesTable prints variable values in table format
func (p *Printer) printValuesTable(values map[string]string) error {
	if len(values) == 0 {
		pterm.Info.Println("No values defined")
		return nil
	}

	// Sort keys
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build table data
	tableData := pterm.TableData{
		{"VARIABLE", "VALUE"},
	}

	for _, key := range keys {
		value := values[key]
		// Truncate long values
		if len(value) > 80 {
			value = value[:77] + "..."
		}
		tableData = append(tableData, []string{key, value})
	}

	return pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
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
		p.printf("export %s=\"%s\"\n", key, value)
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
		p.printf("%s=\"%s\"\n", key, value)
	}

	return nil
}
