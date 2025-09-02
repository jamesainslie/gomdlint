package parser

import (
	"context"
	"io"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// Parser defines the interface for markdown parsers
type Parser interface {
	// Metadata
	Name() string
	Version() string
	SupportedExtensions() []string

	// Parsing
	Parse(ctx context.Context, content string, filename string) functional.Result[ParseResult]
	ParseReader(ctx context.Context, reader io.Reader, filename string) functional.Result[ParseResult]

	// Configuration
	Configure(config ParserConfig) error
	GetConfig() ParserConfig

	// Capabilities
	SupportsAsync() bool
	SupportsStreaming() bool
}

// ParseResult contains the output of parsing
type ParseResult struct {
	Tokens      []value.Token
	AST         interface{} // Parser-specific AST
	Metadata    map[string]interface{}
	FrontMatter functional.Option[map[string]interface{}]
	Errors      []ParseError
	Warnings    []ParseWarning
}

// ParserConfig for parser configuration
type ParserConfig struct {
	Extensions      []string
	StrictMode      bool
	PreserveHTML    bool
	EnableTables    bool
	EnableFootnotes bool
	EnableMath      bool
	CustomOptions   map[string]interface{}
}

// ParseError represents parsing errors
type ParseError struct {
	Line    int
	Column  int
	Message string
	Code    string
}

// ParseWarning represents parsing warnings
type ParseWarning struct {
	Line    int
	Column  int
	Message string
	Code    string
}

// AsyncParser extends Parser with async capabilities
type AsyncParser interface {
	Parser
	ParseAsync(ctx context.Context, content string, filename string) <-chan ParseResult
}

// StreamingParser extends Parser with streaming capabilities
type StreamingParser interface {
	Parser
	ParseStream(ctx context.Context, reader io.Reader, filename string) <-chan StreamChunk
}

// StreamChunk represents a chunk of parsed content for streaming
type StreamChunk struct {
	Tokens   []value.Token
	Offset   int
	Size     int
	Error    error
	Complete bool
}

// ParserFactory creates parser instances
type ParserFactory interface {
	CreateParser(parserType string, config ParserConfig) (Parser, error)
	SupportedParsers() []string
}

// ParserRegistry manages available parsers
type ParserRegistry interface {
	RegisterParser(parser Parser) error
	UnregisterParser(name string) error
	GetParser(name string) (Parser, error)
	ListParsers() []string
	GetParserInfo(name string) (*ParserInfo, error)
}

// ParserInfo contains metadata about a parser
type ParserInfo struct {
	Name                string
	Version             string
	SupportedExtensions []string
	SupportsAsync       bool
	SupportsStreaming   bool
	Description         string
	Author              string
}

// ConfigurableParser indicates a parser that can be dynamically configured
type ConfigurableParser interface {
	Configure(config ParserConfig) error
	GetConfig() ParserConfig
	ValidateConfig(config ParserConfig) error
	GetDefaultConfig() ParserConfig
}

// CacheableParser indicates a parser that supports caching
type CacheableParser interface {
	EnableCaching(enabled bool)
	ClearCache()
	GetCacheStats() map[string]interface{}
}