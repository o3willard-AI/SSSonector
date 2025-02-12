package preflight

import (
	"fmt"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/errors"
)

// Options configures the pre-flight checker
type Options struct {
	ConfigPath string
	Mode       string
	Fix        bool
	Format     string
	Quiet      bool
	Verbose    bool
	CheckOnly  string
}

// Checker performs pre-flight validation
type Checker struct {
	opts         *Options
	errorHandler *errors.Handler
	validators   []Validator
}

// Result represents a validation result
type Result struct {
	Name        string
	Status      Status
	Details     string
	Error       error
	FixPossible bool
	FixCommand  string
	Duration    time.Duration
}

// Status represents validation status
type Status string

const (
	StatusPass    Status = "PASS"
	StatusFail    Status = "FAIL"
	StatusWarning Status = "WARNING"
)

// Report contains all validation results
type Report struct {
	Timestamp time.Time
	System    string
	Version   string
	Results   []Result
	Summary   Summary
	Fixes     []string
}

// Summary provides report statistics
type Summary struct {
	Total        int
	Passed       int
	Failed       int
	Warnings     int
	FixableCount int
}

// NewChecker creates a new pre-flight checker
func NewChecker(opts *Options, errorHandler *errors.Handler) *Checker {
	c := &Checker{
		opts:         opts,
		errorHandler: errorHandler,
		validators:   make([]Validator, 0),
	}

	// Register validators
	c.registerValidators()

	return c
}

// RunChecks executes all validation checks
func (c *Checker) RunChecks() (*Report, error) {
	report := &Report{
		Timestamp: time.Now(),
		System:    getSystemInfo(),
		Version:   "2.0.0", // TODO: Get from version package
		Results:   make([]Result, 0),
	}

	// Run validators
	for _, v := range c.validators {
		// Skip if not requested
		if c.opts.CheckOnly != "" && c.opts.CheckOnly != v.Name() {
			continue
		}

		start := time.Now()
		result := v.Check()
		result.Duration = time.Since(start)

		report.Results = append(report.Results, result)

		// Apply fixes if enabled
		if c.opts.Fix && result.Status == StatusFail && result.FixPossible {
			if err := v.Fix(); err != nil {
				return nil, fmt.Errorf("fix failed for %s: %v", v.Name(), err)
			}
			// Re-run check after fix
			result = v.Check()
			result.Duration = time.Since(start)
			report.Results[len(report.Results)-1] = result
		}
	}

	// Calculate summary
	report.Summary = c.calculateSummary(report.Results)

	// Collect fixes
	report.Fixes = c.collectFixes(report.Results)

	return report, nil
}

// Print outputs the report in the specified format
func (r *Report) Print(format string) error {
	switch format {
	case "text":
		return r.printText()
	case "json":
		return r.printJSON()
	case "markdown":
		return r.printMarkdown()
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// Passed returns true if all checks passed
func (r *Report) Passed() bool {
	return r.Summary.Failed == 0
}

func (c *Checker) calculateSummary(results []Result) Summary {
	var s Summary
	s.Total = len(results)
	for _, r := range results {
		switch r.Status {
		case StatusPass:
			s.Passed++
		case StatusFail:
			s.Failed++
			if r.FixPossible {
				s.FixableCount++
			}
		case StatusWarning:
			s.Warnings++
		}
	}
	return s
}

func (c *Checker) collectFixes(results []Result) []string {
	var fixes []string
	for _, r := range results {
		if r.Status == StatusFail && r.FixPossible {
			fixes = append(fixes, fmt.Sprintf("%s: %s", r.Name, r.FixCommand))
		}
	}
	return fixes
}

func getSystemInfo() string {
	// TODO: Implement system info collection
	return "Linux x86_64"
}
