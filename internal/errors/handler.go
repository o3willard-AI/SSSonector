package errors

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"
)

// Handler manages error handling and reporting
type Handler struct {
	logger *log.Logger
	fixes  map[ErrorCode][]Fix
	docs   Documentation
}

// Fix represents a potential fix for an error
type Fix struct {
	Description string
	Command     string
	Automatic   bool
	Requires    []string
	Risk        string
}

// Documentation provides error documentation
type Documentation struct {
	Errors map[ErrorCode]*Document
}

// Document represents error documentation
type Document struct {
	Code        ErrorCode
	Title       string
	Description string
	Causes      []string
	Solutions   []Solution
	Examples    []Example
	URL         string
}

type Solution struct {
	Description string
	Steps       []string
	Commands    []string
	Warnings    []string
}

type Example struct {
	Scenario   string
	Error      string
	Resolution string
	Code       string
}

// NewHandler creates a new error handler
func NewHandler(logger *log.Logger) *Handler {
	return &Handler{
		logger: logger,
		fixes:  make(map[ErrorCode][]Fix),
		docs: Documentation{
			Errors: make(map[ErrorCode]*Document),
		},
	}
}

// RegisterFix registers a fix for a specific error code
func (h *Handler) RegisterFix(code ErrorCode, fix Fix) {
	if h.fixes[code] == nil {
		h.fixes[code] = make([]Fix, 0)
	}
	h.fixes[code] = append(h.fixes[code], fix)
}

// RegisterDocument registers documentation for a specific error code
func (h *Handler) RegisterDocument(doc *Document) {
	h.docs.Errors[doc.Code] = doc
}

// Handle processes an error and enriches it with context
func (h *Handler) Handle(err error) *Context {
	if err == nil {
		return nil
	}

	// If it's already a Context, use it
	if ctx, ok := err.(*Context); ok {
		return h.enrich(ctx)
	}

	// Create new context for standard error
	ctx := &Context{
		Code:      ErrSystemState,
		Category:  CategorySystem,
		Severity:  SeverityError,
		Message:   err.Error(),
		Timestamp: time.Now(),
	}

	return h.enrich(ctx)
}

// enrich adds additional context to an error
func (h *Handler) enrich(ctx *Context) *Context {
	// Add source location
	if ctx.Location == nil {
		ctx.Location = h.getLocation()
	}

	// Add fixes if available
	if fixes := h.fixes[ctx.Code]; len(fixes) > 0 {
		var steps []string
		for _, fix := range fixes {
			if fix.Automatic {
				steps = append(steps, fmt.Sprintf("Automatic fix: %s", fix.Description))
			} else {
				steps = append(steps, fmt.Sprintf("Manual fix: %s\nCommand: %s", fix.Description, fix.Command))
			}
		}
		ctx.Resolution = steps
	}

	// Add documentation reference
	if doc := h.docs.Errors[ctx.Code]; doc != nil {
		ctx.Reference = doc.URL
		if ctx.Suggestion == "" && len(doc.Solutions) > 0 {
			ctx.Suggestion = doc.Solutions[0].Description
		}
	}

	// Log the error
	h.logError(ctx)

	return ctx
}

// getLocation returns the source location of the error
func (h *Handler) getLocation() *Location {
	var frames []string
	for i := 2; i < 15; i++ { // Skip runtime frames
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			break
		}
		frames = append(frames, fmt.Sprintf("%s:%d %s", file, line, fn.Name()))
	}

	if len(frames) > 0 {
		parts := strings.Split(frames[0], " ")
		fileLine := strings.Split(parts[0], ":")
		return &Location{
			File:     fileLine[0],
			Line:     atoi(fileLine[1]),
			Function: parts[1],
			Stack:    frames,
		}
	}

	return nil
}

// logError logs the error with full context
func (h *Handler) logError(ctx *Context) {
	h.logger.Printf("[%s] %s: %s\nLocation: %s:%d\nDetails: %v\nSuggestion: %s\nResolution: %v\nReference: %s\n",
		ctx.Severity,
		ctx.Category,
		ctx.Message,
		ctx.Location.File,
		ctx.Location.Line,
		ctx.Details,
		ctx.Suggestion,
		ctx.Resolution,
		ctx.Reference,
	)
}

// Helper function to convert string to int
func atoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
