package parser

import (
	"context"
	"io"

	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// BlackfridayParser implements the Parser interface using Blackfriday
// This is a placeholder implementation for the plugin system
type BlackfridayParser struct {
	config ParserConfig
}

// NewBlackfridayParser creates a new Blackfriday parser instance
func NewBlackfridayParser() *BlackfridayParser {
	return &BlackfridayParser{
		config: ParserConfig{
			Extensions:      []string{".md", ".markdown", ".mdown", ".mkd"},
			StrictMode:      false,
			PreserveHTML:    true,
			EnableTables:    true,
			EnableFootnotes: false,
			EnableMath:      false,
			CustomOptions:   make(map[string]interface{}),
		},
	}
}

// Name returns the parser name
func (bp *BlackfridayParser) Name() string {
	return "blackfriday"
}

// Version returns the parser version
func (bp *BlackfridayParser) Version() string {
	return "2.1.0"
}

// SupportedExtensions returns supported file extensions
func (bp *BlackfridayParser) SupportedExtensions() []string {
	return bp.config.Extensions
}

// Parse parses markdown content using Blackfriday
// TODO: Implement actual Blackfriday integration when ready
func (bp *BlackfridayParser) Parse(ctx context.Context, content string, filename string) functional.Result[ParseResult] {
	// Placeholder implementation - delegate to CommonMark for now
	commonMarkParser := NewCommonMarkParser()
	return commonMarkParser.Parse(ctx, content, filename)
}

// ParseReader parses content from an io.Reader
func (bp *BlackfridayParser) ParseReader(ctx context.Context, reader io.Reader, filename string) functional.Result[ParseResult] {
	content, err := io.ReadAll(reader)
	if err != nil {
		return functional.Err[ParseResult](err)
	}

	return bp.Parse(ctx, string(content), filename)
}

// Configure updates the parser configuration
func (bp *BlackfridayParser) Configure(config ParserConfig) error {
	bp.config = config
	return nil
}

// GetConfig returns the current parser configuration
func (bp *BlackfridayParser) GetConfig() ParserConfig {
	return bp.config
}

// SupportsAsync returns whether the parser supports async operation
func (bp *BlackfridayParser) SupportsAsync() bool {
	return false // Blackfriday is synchronous
}

// SupportsStreaming returns whether the parser supports streaming
func (bp *BlackfridayParser) SupportsStreaming() bool {
	return false // Would need streaming implementation
}