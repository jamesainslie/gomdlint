package parser

import (
	"context"
	"io"
	"strings"

	"github.com/gomdlint/gomdlint/internal/app/service"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// GomdlintParserAdapter adapts the existing ParserService to the Parser interface
type GomdlintParserAdapter struct {
	parser *service.ParserService
	config ParserConfig
}

// NewGomdlintParserAdapter creates a new adapter for the existing gomdlint parser
func NewGomdlintParserAdapter() Parser {
	return &GomdlintParserAdapter{
		parser: service.NewParserService(),
		config: DefaultParserConfig(),
	}
}

// Name returns the parser name
func (gpa *GomdlintParserAdapter) Name() string {
	return "gomdlint"
}

// Version returns the parser version
func (gpa *GomdlintParserAdapter) Version() string {
	return "1.0.0" // Would typically be derived from build info
}

// SupportedExtensions returns supported file extensions
func (gpa *GomdlintParserAdapter) SupportedExtensions() []string {
	return []string{".md", ".markdown", ".mdown", ".mkd", ".text"}
}

// Parse processes markdown content and returns tokens
func (gpa *GomdlintParserAdapter) Parse(ctx context.Context, content string, filename string) functional.Result[ParseResult] {
	// Use the existing ParserService
	tokensResult := gpa.parser.ParseDocument(ctx, content, filename)
	if tokensResult.IsErr() {
		return functional.Err[ParseResult](tokensResult.Error())
	}

	tokens := tokensResult.Unwrap()
	lines := strings.Split(content, "\n")

	// Extract front matter if present
	frontMatter := gpa.extractFrontMatter(content)

	result := ParseResult{
		Tokens: tokens,
		Lines:  lines,
		AST:    nil, // gomdlint parser doesn't expose AST
		Metadata: map[string]interface{}{
			"parser":     "gomdlint",
			"filename":   filename,
			"line_count": len(lines),
			"char_count": len(content),
		},
		FrontMatter: frontMatter,
		Errors:      []ParseError{},
		Warnings:    []ParseWarning{},
	}

	return functional.Ok(result)
}

// ParseReader processes markdown from a reader
func (gpa *GomdlintParserAdapter) ParseReader(ctx context.Context, reader io.Reader, filename string) functional.Result[ParseResult] {
	content, err := io.ReadAll(reader)
	if err != nil {
		return functional.Err[ParseResult](err)
	}

	return gpa.Parse(ctx, string(content), filename)
}

// Configure updates parser configuration
func (gpa *GomdlintParserAdapter) Configure(config ParserConfig) error {
	gpa.config = config

	// Apply configuration to the underlying parser
	// Note: The existing ParserService doesn't expose much configuration,
	// so this is mostly for compatibility with the interface
	return nil
}

// GetConfig returns current configuration
func (gpa *GomdlintParserAdapter) GetConfig() ParserConfig {
	return gpa.config
}

// SupportsAsync returns whether async parsing is supported
func (gpa *GomdlintParserAdapter) SupportsAsync() bool {
	return false // The existing parser is synchronous
}

// SupportsStreaming returns whether streaming is supported
func (gpa *GomdlintParserAdapter) SupportsStreaming() bool {
	return false // The existing parser processes full content
}

// extractFrontMatter extracts front matter from content
func (gpa *GomdlintParserAdapter) extractFrontMatter(content string) functional.Option[map[string]interface{}] {
	// Simple YAML front matter detection
	lines := strings.Split(content, "\n")
	if len(lines) < 3 {
		return functional.None[map[string]interface{}]()
	}

	if !strings.HasPrefix(lines[0], "---") {
		return functional.None[map[string]interface{}]()
	}

	// Find ending delimiter
	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "---") || strings.HasPrefix(lines[i], "...") {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		return functional.None[map[string]interface{}]()
	}

	// Extract front matter content
	frontMatterLines := lines[1:endIdx]
	frontMatterContent := strings.Join(frontMatterLines, "\n")

	// Basic key-value extraction (simplified)
	frontMatter := make(map[string]interface{})
	for _, line := range frontMatterLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove quotes if present
			if len(value) >= 2 &&
				((strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
					(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'"))) {
				value = value[1 : len(value)-1]
			}

			frontMatter[key] = value
		}
	}

	if len(frontMatter) == 0 {
		// Store raw content as fallback
		frontMatter["raw"] = frontMatterContent
	}

	return functional.Some(frontMatter)
}
