package preflight

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
)

// printText outputs the report in text format
func (r *Report) printText() error {
	const textTemplate = `
SSSonector Pre-Flight Check Report
=================================
Time: {{.Timestamp}}
System: {{.System}}
Version: {{.Version}}

Check Results:
{{range .Results}}
[{{.Status}}] {{.Name}}
{{if .Details}}  Details: {{.Details}}{{end}}
{{if .Error}}  Error: {{.Error}}{{end}}
{{if .FixPossible}}  Fix: {{.FixCommand}}{{end}}
{{if .Duration}}  Duration: {{.Duration}}{{end}}
{{end}}

Summary:
--------
Total Checks: {{.Summary.Total}}
Passed: {{.Summary.Passed}}
Failed: {{.Summary.Failed}}
Warnings: {{.Summary.Warnings}}
Fixable Issues: {{.Summary.FixableCount}}

{{if .Fixes}}
Available Fixes:
---------------
{{range .Fixes}}
- {{.}}
{{end}}
{{end}}
`

	tmpl, err := template.New("report").Parse(textTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	return tmpl.Execute(os.Stdout, r)
}

// printJSON outputs the report in JSON format
func (r *Report) printJSON() error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}

// printMarkdown outputs the report in Markdown format
func (r *Report) printMarkdown() error {
	var b strings.Builder

	// Header
	fmt.Fprintf(&b, "# SSSonector Pre-Flight Check Report\n\n")
	fmt.Fprintf(&b, "## System Information\n\n")
	fmt.Fprintf(&b, "- **Time**: %s\n", r.Timestamp)
	fmt.Fprintf(&b, "- **System**: %s\n", r.System)
	fmt.Fprintf(&b, "- **Version**: %s\n\n", r.Version)

	// Results
	fmt.Fprintf(&b, "## Check Results\n\n")
	for _, result := range r.Results {
		// Status icon
		var icon string
		switch result.Status {
		case StatusPass:
			icon = "✅"
		case StatusFail:
			icon = "❌"
		case StatusWarning:
			icon = "⚠️"
		}

		fmt.Fprintf(&b, "### %s %s\n\n", icon, result.Name)
		if result.Details != "" {
			fmt.Fprintf(&b, "**Details**: %s\n\n", result.Details)
		}
		if result.Error != nil {
			fmt.Fprintf(&b, "**Error**: %s\n\n", result.Error)
		}
		if result.FixPossible {
			fmt.Fprintf(&b, "**Fix Command**:\n```bash\n%s\n```\n\n", result.FixCommand)
		}
		if result.Duration > 0 {
			fmt.Fprintf(&b, "**Duration**: %s\n\n", result.Duration)
		}
	}

	// Summary
	fmt.Fprintf(&b, "## Summary\n\n")
	fmt.Fprintf(&b, "| Metric | Count |\n")
	fmt.Fprintf(&b, "|--------|-------|\n")
	fmt.Fprintf(&b, "| Total Checks | %d |\n", r.Summary.Total)
	fmt.Fprintf(&b, "| Passed | %d |\n", r.Summary.Passed)
	fmt.Fprintf(&b, "| Failed | %d |\n", r.Summary.Failed)
	fmt.Fprintf(&b, "| Warnings | %d |\n", r.Summary.Warnings)
	fmt.Fprintf(&b, "| Fixable Issues | %d |\n\n", r.Summary.FixableCount)

	// Available Fixes
	if len(r.Fixes) > 0 {
		fmt.Fprintf(&b, "## Available Fixes\n\n")
		for _, fix := range r.Fixes {
			fmt.Fprintf(&b, "- %s\n", fix)
		}
		fmt.Fprintf(&b, "\n")
	}

	// Write to stdout
	_, err := fmt.Print(b.String())
	return err
}
