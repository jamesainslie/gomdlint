package output

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gomdlint/gomdlint/internal/app/service"
	"github.com/gomdlint/gomdlint/internal/domain/value"
)

// ThemedOutput provides themed output functionality for CLI commands.
// This separates presentation concerns from business logic.
type ThemedOutput struct {
	theme        value.Theme
	writer       io.Writer
	errorWriter  io.Writer
	themeService *service.ThemeService
	enableColors bool
}

// NewThemedOutput creates a new themed output with the specified theme.
func NewThemedOutput(ctx context.Context, themeConfig value.ThemeConfig, themeService *service.ThemeService) (*ThemedOutput, error) {
	result := themeService.CreateTheme(ctx, themeConfig)
	if !result.IsOk() {
		return nil, result.Error()
	}

	theme := result.Unwrap()

	return &ThemedOutput{
		theme:        theme,
		writer:       os.Stdout,
		errorWriter:  os.Stderr,
		themeService: themeService,
		enableColors: true,
	}, nil
}

// WithWriter sets the output writer.
func (to *ThemedOutput) WithWriter(writer io.Writer) *ThemedOutput {
	return &ThemedOutput{
		theme:        to.theme,
		writer:       writer,
		errorWriter:  to.errorWriter,
		themeService: to.themeService,
		enableColors: to.enableColors,
	}
}

// WithErrorWriter sets the error output writer.
func (to *ThemedOutput) WithErrorWriter(writer io.Writer) *ThemedOutput {
	return &ThemedOutput{
		theme:        to.theme,
		writer:       to.writer,
		errorWriter:  writer,
		themeService: to.themeService,
		enableColors: to.enableColors,
	}
}

// WithColors enables or disables color output.
func (to *ThemedOutput) WithColors(enable bool) *ThemedOutput {
	return &ThemedOutput{
		theme:        to.theme,
		writer:       to.writer,
		errorWriter:  to.errorWriter,
		themeService: to.themeService,
		enableColors: enable,
	}
}

// Success prints a success message with appropriate theming.
func (to *ThemedOutput) Success(format string, args ...interface{}) {
	symbol := to.theme.Symbol("success")
	message := fmt.Sprintf(format, args...)
	to.printWithSymbol(to.errorWriter, symbol, message, to.theme.Colors().Green)
}

// Error prints an error message with appropriate theming.
func (to *ThemedOutput) Error(format string, args ...interface{}) {
	symbol := to.theme.Symbol("error")
	message := fmt.Sprintf(format, args...)
	to.printWithSymbol(to.errorWriter, symbol, message, to.theme.Colors().Red)
}

// Warning prints a warning message with appropriate theming.
func (to *ThemedOutput) Warning(format string, args ...interface{}) {
	symbol := to.theme.Symbol("warning")
	message := fmt.Sprintf(format, args...)
	to.printWithSymbol(to.errorWriter, symbol, message, to.theme.Colors().Yellow)
}

// Info prints an info message with appropriate theming.
func (to *ThemedOutput) Info(format string, args ...interface{}) {
	symbol := to.theme.Symbol("info")
	message := fmt.Sprintf(format, args...)
	to.printWithSymbol(to.errorWriter, symbol, message, to.theme.Colors().Blue)
}

// Processing prints a processing message with appropriate theming.
func (to *ThemedOutput) Processing(format string, args ...interface{}) {
	symbol := to.theme.Symbol("processing")
	message := fmt.Sprintf(format, args...)
	to.printWithSymbol(to.errorWriter, symbol, message, to.theme.Colors().Cyan)
}

// FileFound prints a file found message.
func (to *ThemedOutput) FileFound(format string, args ...interface{}) {
	symbol := to.theme.Symbol("file_found")
	message := fmt.Sprintf(format, args...)
	to.printWithSymbol(to.errorWriter, symbol, message, "")
}

// FileSaved prints a file saved message.
func (to *ThemedOutput) FileSaved(format string, args ...interface{}) {
	symbol := to.theme.Symbol("file_saved")
	message := fmt.Sprintf(format, args...)
	to.printWithSymbol(to.errorWriter, symbol, message, to.theme.Colors().Green)
}

// Benchmark prints a benchmark message.
func (to *ThemedOutput) Benchmark(format string, args ...interface{}) {
	symbol := to.theme.Symbol("benchmark")
	message := fmt.Sprintf(format, args...)
	to.printWithSymbol(to.writer, symbol, message, to.theme.Colors().Magenta)
}

// Performance prints a performance message.
func (to *ThemedOutput) Performance(format string, args ...interface{}) {
	symbol := to.theme.Symbol("performance")
	message := fmt.Sprintf(format, args...)
	to.printWithSymbol(to.writer, symbol, message, to.theme.Colors().Blue)
}

// Winner prints a winner message.
func (to *ThemedOutput) Winner(format string, args ...interface{}) {
	symbol := to.theme.Symbol("winner")
	message := fmt.Sprintf(format, args...)
	to.printWithSymbol(to.writer, symbol, message, to.theme.Colors().Yellow)
}

// Results prints a results message.
func (to *ThemedOutput) Results(format string, args ...interface{}) {
	symbol := to.theme.Symbol("results")
	message := fmt.Sprintf(format, args...)
	to.printWithSymbol(to.writer, symbol, message, to.theme.Colors().Green)
}

// Search prints a search message.
func (to *ThemedOutput) Search(format string, args ...interface{}) {
	symbol := to.theme.Symbol("search")
	message := fmt.Sprintf(format, args...)
	to.printWithSymbol(to.errorWriter, symbol, message, "")
}

// Launch prints a launch message.
func (to *ThemedOutput) Launch(format string, args ...interface{}) {
	symbol := to.theme.Symbol("launch")
	message := fmt.Sprintf(format, args...)
	to.printWithSymbol(to.writer, symbol, message, to.theme.Colors().Magenta)
}

// Plain prints a message without any theming.
func (to *ThemedOutput) Plain(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprint(to.writer, message)
}

// PlainError prints a message to error writer without any theming.
func (to *ThemedOutput) PlainError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprint(to.errorWriter, message)
}

// printWithSymbol prints a message with symbol and optional color.
func (to *ThemedOutput) printWithSymbol(writer io.Writer, symbol, message, color string) {
	var output strings.Builder

	// Add color if enabled and available
	if to.enableColors && color != "" {
		output.WriteString(color)
	}

	// Add symbol with space if present
	if symbol != "" {
		output.WriteString(symbol)
		output.WriteString(" ")
	}

	// Add message
	output.WriteString(message)

	// Reset color if enabled and was applied
	if to.enableColors && color != "" {
		output.WriteString(to.theme.Colors().Reset)
	}

	// Ensure newline if not present
	if !strings.HasSuffix(message, "\n") {
		output.WriteString("\n")
	}

	fmt.Fprint(writer, output.String())
}

// Theme returns the current theme.
func (to *ThemedOutput) Theme() value.Theme {
	return to.theme
}

// UpdateTheme updates the theme configuration.
func (to *ThemedOutput) UpdateTheme(ctx context.Context, config value.ThemeConfig) error {
	result := to.themeService.CreateTheme(ctx, config)
	if !result.IsOk() {
		return result.Error()
	}

	to.theme = result.Unwrap()
	return nil
}
