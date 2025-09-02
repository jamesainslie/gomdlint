package parser

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// CommonMarkParser implements the Parser interface for CommonMark
type CommonMarkParser struct {
	config ParserConfig
}

// NewCommonMarkParser creates a new CommonMark parser instance
func NewCommonMarkParser() *CommonMarkParser {
	return &CommonMarkParser{
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
func (cmp *CommonMarkParser) Name() string {
	return "commonmark"
}

// Version returns the parser version
func (cmp *CommonMarkParser) Version() string {
	return "1.0.0"
}

// SupportedExtensions returns supported file extensions
func (cmp *CommonMarkParser) SupportedExtensions() []string {
	return cmp.config.Extensions
}

// Parse parses markdown content and returns tokens
func (cmp *CommonMarkParser) Parse(ctx context.Context, content string, filename string) functional.Result[ParseResult] {
	// Remove front matter if present
	frontMatter, bodyContent := cmp.extractFrontMatter(content)

	lines := strings.Split(bodyContent, "\n")
	var tokens []value.Token
	var errors []ParseError
	var warnings []ParseWarning

	// Simple tokenization approach
	for i, line := range lines {
		lineNumber := i + 1

		// Skip empty lines in token generation
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return functional.Err[ParseResult](ctx.Err())
		default:
		}

		token, parseErr := cmp.parseLine(line, lineNumber)
		if parseErr != nil {
			errors = append(errors, *parseErr)
			continue
		}

		if token != nil {
			tokens = append(tokens, *token)
		}
	}

	result := ParseResult{
		Tokens: tokens,
		AST:    nil, // CommonMark doesn't provide AST
		Metadata: map[string]interface{}{
			"parser":    "commonmark",
			"filename":  filename,
			"size":      len(content),
			"lines":     len(lines),
			"tokens":    len(tokens),
		},
		FrontMatter: frontMatter,
		Errors:      errors,
		Warnings:    warnings,
	}

	return functional.Ok(result)
}

// ParseReader parses content from an io.Reader
func (cmp *CommonMarkParser) ParseReader(ctx context.Context, reader io.Reader, filename string) functional.Result[ParseResult] {
	content, err := io.ReadAll(reader)
	if err != nil {
		return functional.Err[ParseResult](fmt.Errorf("failed to read content: %w", err))
	}

	return cmp.Parse(ctx, string(content), filename)
}

// Configure updates the parser configuration
func (cmp *CommonMarkParser) Configure(config ParserConfig) error {
	cmp.config = config
	return nil
}

// GetConfig returns the current parser configuration
func (cmp *CommonMarkParser) GetConfig() ParserConfig {
	return cmp.config
}

// SupportsAsync returns whether the parser supports async operation
func (cmp *CommonMarkParser) SupportsAsync() bool {
	return false // CommonMark parser is synchronous
}

// SupportsStreaming returns whether the parser supports streaming
func (cmp *CommonMarkParser) SupportsStreaming() bool {
	return false // CommonMark parser doesn't support streaming
}

// parseLine parses a single line and returns a token
func (cmp *CommonMarkParser) parseLine(line string, lineNumber int) (*value.Token, *ParseError) {
	trimmed := strings.TrimSpace(line)

	// Determine token type based on line content
	var tokenType value.TokenType
	properties := make(map[string]interface{})

	switch {
	case cmp.isATXHeading(line):
		tokenType = value.TokenTypeATXHeading
		properties["level"] = cmp.getATXHeadingLevel(line)
		properties["text"] = cmp.getATXHeadingText(line)

	case cmp.isListItem(line):
		tokenType = value.TokenTypeListItem
		marker, isOrdered := cmp.getListMarker(line)
		properties["marker"] = marker
		properties["ordered"] = isOrdered
		properties["text"] = cmp.getListItemText(line)

	case cmp.isFencedCodeBlock(line):
		tokenType = value.TokenTypeCodeFenced
		language, info := cmp.getCodeFenceInfo(line)
		properties["language"] = language
		properties["info"] = info

	case cmp.isBlockquote(line):
		tokenType = value.TokenTypeBlockquote
		properties["text"] = cmp.getBlockquoteText(line)

	case cmp.isHorizontalRule(line):
		tokenType = value.TokenTypeHorizontalRule

	default:
		tokenType = value.TokenTypeParagraph
		properties["text"] = trimmed
	}

	// Calculate positions
	startPos := value.NewPosition(lineNumber, 1)
	endPos := value.NewPosition(lineNumber, len(line)+1)

	token := value.NewToken(tokenType, line, startPos, endPos)
	
	// Add properties to token
	for key, val := range properties {
		token = token.WithProperty(key, val)
	}

	return &token, nil
}

// Front matter extraction
var frontMatterRe = regexp.MustCompile(`^---\s*\n(.*?\n)?---\s*\n`)

func (cmp *CommonMarkParser) extractFrontMatter(content string) (functional.Option[map[string]interface{}], string) {
	matches := frontMatterRe.FindStringSubmatch(content)
	if len(matches) > 0 {
		// For now, return empty front matter data
		// In a real implementation, this would parse YAML/TOML
		frontMatterData := make(map[string]interface{})
		frontMatterData["raw"] = matches[1]

		bodyContent := content[len(matches[0]):]
		return functional.Some(frontMatterData), bodyContent
	}

	return functional.None[map[string]interface{}](), content
}

// Helper methods for token classification
func (cmp *CommonMarkParser) isATXHeading(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "#")
}

func (cmp *CommonMarkParser) getATXHeadingLevel(line string) int {
	trimmed := strings.TrimSpace(line)
	level := 0
	for i, char := range trimmed {
		if char == '#' {
			level++
		} else {
			break
		}
		if i >= 6 { // Max 6 levels
			break
		}
	}
	return level
}

func (cmp *CommonMarkParser) getATXHeadingText(line string) string {
	trimmed := strings.TrimSpace(line)
	level := cmp.getATXHeadingLevel(line)
	if level == 0 {
		return trimmed
	}

	text := strings.TrimSpace(trimmed[level:])
	// Remove trailing hashes if present
	text = strings.TrimRight(text, "# ")
	return text
}

func (cmp *CommonMarkParser) isListItem(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	if len(trimmed) == 0 {
		return false
	}

	// Unordered list markers
	if strings.HasPrefix(trimmed, "- ") ||
		strings.HasPrefix(trimmed, "* ") ||
		strings.HasPrefix(trimmed, "+ ") {
		return true
	}

	// Ordered list markers (1. 2. etc.)
	re := regexp.MustCompile(`^\d+\. `)
	return re.MatchString(trimmed)
}

func (cmp *CommonMarkParser) getListMarker(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")

	// Unordered markers
	if strings.HasPrefix(trimmed, "- ") {
		return "-", false
	}
	if strings.HasPrefix(trimmed, "* ") {
		return "*", false
	}
	if strings.HasPrefix(trimmed, "+ ") {
		return "+", false
	}

	// Ordered markers
	re := regexp.MustCompile(`^(\d+\.) `)
	matches := re.FindStringSubmatch(trimmed)
	if len(matches) > 1 {
		return matches[1], true
	}

	return "", false
}

func (cmp *CommonMarkParser) getListItemText(line string) string {
	trimmed := strings.TrimLeft(line, " \t")

	// Remove marker for unordered lists
	for _, prefix := range []string{"- ", "* ", "+ "} {
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(trimmed[2:])
		}
	}

	// Remove marker for ordered lists
	re := regexp.MustCompile(`^\d+\. `)
	return strings.TrimSpace(re.ReplaceAllString(trimmed, ""))
}

func (cmp *CommonMarkParser) isFencedCodeBlock(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

func (cmp *CommonMarkParser) getCodeFenceInfo(line string) (language string, info string) {
	trimmed := strings.TrimSpace(line)
	if !cmp.isFencedCodeBlock(line) {
		return "", ""
	}

	// Remove fence characters
	var content string
	if strings.HasPrefix(trimmed, "```") {
		content = strings.TrimPrefix(trimmed, "```")
	} else if strings.HasPrefix(trimmed, "~~~") {
		content = strings.TrimPrefix(trimmed, "~~~")
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return "", ""
	}

	parts := strings.Fields(content)
	if len(parts) > 0 {
		language = parts[0]
		if len(parts) > 1 {
			info = strings.Join(parts[1:], " ")
		}
	}

	return language, info
}

func (cmp *CommonMarkParser) isBlockquote(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trimmed, ">")
}

func (cmp *CommonMarkParser) getBlockquoteText(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "> ") {
		return trimmed[2:]
	} else if strings.HasPrefix(trimmed, ">") {
		return trimmed[1:]
	}
	return trimmed
}

func (cmp *CommonMarkParser) isHorizontalRule(line string) bool {
	trimmed := strings.TrimSpace(line)
	
	// Check for various horizontal rule patterns
	patterns := []string{
		"---",
		"***",
		"___",
	}

	for _, pattern := range patterns {
		if strings.HasPrefix(trimmed, pattern) && len(strings.TrimLeft(trimmed, string(pattern[0]))) == 0 {
			return len(trimmed) >= 3
		}
	}

	return false
}
