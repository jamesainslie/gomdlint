package parser

import (
	"context"
	"io"
	"strings"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// NoneParser is a passthrough parser that doesn't parse markdown
// It creates basic text tokens for simple rule testing
type NoneParser struct {
	config ParserConfig
}

// NewNoneParser creates a new none parser instance
func NewNoneParser() *NoneParser {
	return &NoneParser{
		config: ParserConfig{
			Extensions:      []string{".md", ".markdown", ".mdown", ".mkd", ".txt"},
			StrictMode:      false,
			PreserveHTML:    true,
			EnableTables:    false,
			EnableFootnotes: false,
			EnableMath:      false,
			CustomOptions:   make(map[string]interface{}),
		},
	}
}

// Name returns the parser name
func (np *NoneParser) Name() string {
	return "none"
}

// Version returns the parser version
func (np *NoneParser) Version() string {
	return "1.0.0"
}

// SupportedExtensions returns supported file extensions
func (np *NoneParser) SupportedExtensions() []string {
	return np.config.Extensions
}

// Parse creates basic text tokens from content lines
func (np *NoneParser) Parse(ctx context.Context, content string, filename string) functional.Result[ParseResult] {
	lines := strings.Split(content, "\n")
	var tokens []value.Token

	for i, line := range lines {
		lineNumber := i + 1

		// Check context cancellation
		select {
		case <-ctx.Done():
			return functional.Err[ParseResult](ctx.Err())
		default:
		}

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Create simple text token
		startPos := value.NewPosition(lineNumber, 1)
		endPos := value.NewPosition(lineNumber, len(line)+1)

		properties := map[string]interface{}{
			"line_content": line,
			"trimmed":      strings.TrimSpace(line),
		}

		token := value.NewToken(value.TokenTypeText, line, startPos, endPos)
		
		// Add properties to token
		for key, val := range properties {
			token = token.WithProperty(key, val)
		}

		tokens = append(tokens, token)
	}

	result := ParseResult{
		Tokens: tokens,
		AST:    nil, // No AST for none parser
		Metadata: map[string]interface{}{
			"parser":   "none",
			"filename": filename,
			"size":     len(content),
			"lines":    len(lines),
			"tokens":   len(tokens),
		},
		FrontMatter: functional.None[map[string]interface{}](),
		Errors:      []ParseError{},
		Warnings:    []ParseWarning{},
	}

	return functional.Ok(result)
}

// ParseReader parses content from an io.Reader
func (np *NoneParser) ParseReader(ctx context.Context, reader io.Reader, filename string) functional.Result[ParseResult] {
	content, err := io.ReadAll(reader)
	if err != nil {
		return functional.Err[ParseResult](err)
	}

	return np.Parse(ctx, string(content), filename)
}

// Configure updates the parser configuration
func (np *NoneParser) Configure(config ParserConfig) error {
	np.config = config
	return nil
}

// GetConfig returns the current parser configuration
func (np *NoneParser) GetConfig() ParserConfig {
	return np.config
}

// SupportsAsync returns whether the parser supports async operation
func (np *NoneParser) SupportsAsync() bool {
	return false
}

// SupportsStreaming returns whether the parser supports streaming
func (np *NoneParser) SupportsStreaming() bool {
	return false
}