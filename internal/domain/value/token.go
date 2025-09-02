package value

import (
	"fmt"
)

// TokenType represents the type of a markdown token.
// This follows the micromark token type system for compatibility.
type TokenType string

// String returns the string representation of TokenType.
func (t TokenType) String() string {
	return string(t)
}

// Common token types from micromark parser
const (
	// Document structure
	TokenTypeDocument   TokenType = "document"
	TokenTypeParagraph  TokenType = "paragraph"
	TokenTypeLineEnding TokenType = "lineEnding"
	TokenTypeContent    TokenType = "content"

	// Headings
	TokenTypeATXHeading             TokenType = "atxHeading"
	TokenTypeATXHeadingText         TokenType = "atxHeadingText"
	TokenTypeSetextHeading          TokenType = "setextHeading"
	TokenTypeSetextHeadingText      TokenType = "setextHeadingText"
	TokenTypeSetextHeadingUnderline TokenType = "setextHeadingUnderline"

	// Lists
	TokenTypeList           TokenType = "list"
	TokenTypeListItem       TokenType = "listItem"
	TokenTypeListItemValue  TokenType = "listItemValue"
	TokenTypeListItemMarker TokenType = "listItemMarker"
	TokenTypeListItemPrefix TokenType = "listItemPrefix"

	// Code
	TokenTypeCodeFenced          TokenType = "codeFenced"
	TokenTypeCodeFencedFence     TokenType = "codeFencedFence"
	TokenTypeCodeFencedFenceInfo TokenType = "codeFencedFenceInfo"
	TokenTypeCodeIndented        TokenType = "codeIndented"
	TokenTypeCodeText            TokenType = "codeText"

	// Blockquotes
	TokenTypeBlockQuote       TokenType = "blockQuote"
	TokenTypeBlockQuotePrefix TokenType = "blockQuotePrefix"
	TokenTypeBlockQuoteMarker TokenType = "blockQuoteMarker"

	// Links and images
	TokenTypeLink           TokenType = "link"
	TokenTypeLinkReference  TokenType = "linkReference"
	TokenTypeImage          TokenType = "image"
	TokenTypeImageReference TokenType = "imageReference"
	TokenTypeAutolink       TokenType = "autolink"

	// Emphasis
	TokenTypeEmphasis TokenType = "emphasis"
	TokenTypeStrong   TokenType = "strong"

	// Thematic breaks
	TokenTypeThematicBreak TokenType = "thematicBreak"

	// HTML
	TokenTypeHTMLFlow TokenType = "htmlFlow"
	TokenTypeHTMLText TokenType = "htmlText"

	// Tables (GFM)
	TokenTypeTable          TokenType = "table"
	TokenTypeTableRow       TokenType = "tableRow"
	TokenTypeTableCell      TokenType = "tableCell"
	TokenTypeTableDelimiter TokenType = "tableDelimiter"

	// Math
	TokenTypeMath     TokenType = "math"
	TokenTypeMathFlow TokenType = "mathFlow"

	// Text content
	TokenTypeText       TokenType = "text"
	TokenTypeWhitespace TokenType = "whitespace"

	// Legacy aliases for backward compatibility
	TokenTypeHeading        TokenType = "atxHeading"
	TokenTypeBlockquote     TokenType = "blockQuote"
	TokenTypeCodeBlock      TokenType = "codeFenced"
	TokenTypeHorizontalRule TokenType = "thematicBreak"
)

// Position represents a position in the source document.
type Position struct {
	Line   int // 1-based line number
	Column int // 1-based column number
	Offset int // 0-based byte offset
}

// NewPosition creates a new Position with the specified line and column.
func NewPosition(line, column int) Position {
	return Position{
		Line:   line,
		Column: column,
		Offset: 0, // Will be calculated when needed
	}
}

// Range represents a range in the source document.
type Range struct {
	Start Position
	End   Position
}

// NewRange creates a new Range with the specified start and end positions.
func NewRange(start, end Position) *Range {
	return &Range{
		Start: start,
		End:   end,
	}
}

// Token represents a parsed markdown token with position information.
// Tokens are immutable value objects that represent elements in the parsed document.
type Token struct {
	Type     TokenType
	Text     string
	Range    Range
	Children []Token

	// Additional metadata
	Properties map[string]interface{}
}

// NewToken creates a new Token with the specified properties.
func NewToken(tokenType TokenType, text string, start, end Position) Token {
	return Token{
		Type:       tokenType,
		Text:       text,
		Range:      Range{Start: start, End: end},
		Children:   make([]Token, 0),
		Properties: make(map[string]interface{}),
	}
}

// WithChildren returns a new Token with the specified children.
func (t Token) WithChildren(children []Token) Token {
	newToken := t
	newToken.Children = make([]Token, len(children))
	copy(newToken.Children, children)
	return newToken
}

// WithProperty returns a new Token with an additional property.
func (t Token) WithProperty(key string, value interface{}) Token {
	newToken := t
	newToken.Properties = make(map[string]interface{})
	for k, v := range t.Properties {
		newToken.Properties[k] = v
	}
	newToken.Properties[key] = value
	return newToken
}

// StartLine returns the starting line number (1-based).
func (t Token) StartLine() int {
	return t.Range.Start.Line
}

// EndLine returns the ending line number (1-based).
func (t Token) EndLine() int {
	return t.Range.End.Line
}

// StartColumn returns the starting column number (1-based).
func (t Token) StartColumn() int {
	return t.Range.Start.Column
}

// EndColumn returns the ending column number (1-based).
func (t Token) EndColumn() int {
	return t.Range.End.Column
}

// Length returns the length of the token text.
func (t Token) Length() int {
	return len(t.Text)
}

// IsType checks if the token matches the given type.
func (t Token) IsType(tokenType TokenType) bool {
	return t.Type == tokenType
}

// IsOneOfTypes checks if the token matches any of the given types.
func (t Token) IsOneOfTypes(types ...TokenType) bool {
	for _, tokenType := range types {
		if t.Type == tokenType {
			return true
		}
	}
	return false
}

// HasChildren returns true if the token has child tokens.
func (t Token) HasChildren() bool {
	return len(t.Children) > 0
}

// FindChildren returns all direct children matching the given predicate.
func (t Token) FindChildren(predicate func(Token) bool) []Token {
	var matches []Token
	for _, child := range t.Children {
		if predicate(child) {
			matches = append(matches, child)
		}
	}
	return matches
}

// FindChildrenByType returns all direct children of the specified type.
func (t Token) FindChildrenByType(tokenType TokenType) []Token {
	return t.FindChildren(func(token Token) bool {
		return token.Type == tokenType
	})
}

// FindDescendants returns all descendants (recursive) matching the given predicate.
func (t Token) FindDescendants(predicate func(Token) bool) []Token {
	var matches []Token

	for _, child := range t.Children {
		if predicate(child) {
			matches = append(matches, child)
		}
		// Recursively search children
		matches = append(matches, child.FindDescendants(predicate)...)
	}

	return matches
}

// FindDescendantsByType returns all descendants of the specified type.
func (t Token) FindDescendantsByType(tokenType TokenType) []Token {
	return t.FindDescendants(func(token Token) bool {
		return token.Type == tokenType
	})
}

// GetProperty returns a property value if it exists.
func (t Token) GetProperty(key string) (interface{}, bool) {
	value, exists := t.Properties[key]
	return value, exists
}

// GetStringProperty returns a string property value.
func (t Token) GetStringProperty(key string) (string, bool) {
	if value, exists := t.Properties[key]; exists {
		if str, ok := value.(string); ok {
			return str, true
		}
	}
	return "", false
}

// GetIntProperty returns an integer property value.
func (t Token) GetIntProperty(key string) (int, bool) {
	if value, exists := t.Properties[key]; exists {
		if i, ok := value.(int); ok {
			return i, true
		}
	}
	return 0, false
}

// String implements the Stringer interface for debugging.
func (t Token) String() string {
	return fmt.Sprintf("Token{Type: %s, Text: %q, Range: %d:%d-%d:%d}",
		t.Type,
		truncateText(t.Text, 30),
		t.Range.Start.Line, t.Range.Start.Column,
		t.Range.End.Line, t.Range.End.Column)
}

// IsHeading returns true if the token represents a heading.
func (t Token) IsHeading() bool {
	return t.IsOneOfTypes(TokenTypeATXHeading, TokenTypeSetextHeading)
}

// IsCodeBlock returns true if the token represents a code block.
func (t Token) IsCodeBlock() bool {
	return t.IsOneOfTypes(TokenTypeCodeFenced, TokenTypeCodeIndented)
}

// IsList returns true if the token represents a list or list item.
func (t Token) IsList() bool {
	return t.IsOneOfTypes(TokenTypeList, TokenTypeListItem)
}

// IsText returns true if the token contains text content.
func (t Token) IsText() bool {
	return t.IsOneOfTypes(TokenTypeText, TokenTypeATXHeadingText, TokenTypeSetextHeadingText)
}

// truncateText truncates text for display purposes.
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}
