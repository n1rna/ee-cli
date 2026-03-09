// Package output provides formatted terminal output for ee.
// This centralizes all printing and formatting logic away from command modules.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/pterm/pterm"
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

// PrintChange prints a change notification (e.g., "Field: old -> new")
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
