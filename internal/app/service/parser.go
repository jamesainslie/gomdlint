package service

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// ParserService provides high-performance CommonMark/GFM parsing functionality.
// It converts markdown text into a structured token tree for rule processing.
type ParserService struct {
	// Configuration options
	enableGFM        bool
	enableMath       bool
	enableDirectives bool
	enableFootnotes  bool

	// Regex patterns for various constructs (compiled once for performance)
	atxHeadingRe    *regexp.Regexp
	setextHeadingRe *regexp.Regexp
	codeBlockRe     *regexp.Regexp
	fencedCodeRe    *regexp.Regexp
	blockquoteRe    *regexp.Regexp
	listItemRe      *regexp.Regexp
	linkRe          *regexp.Regexp
	imageRe         *regexp.Regexp
	emphasisRe      *regexp.Regexp
	strongRe        *regexp.Regexp
	inlineCodeRe    *regexp.Regexp
	htmlTagRe       *regexp.Regexp
	thematicBreakRe *regexp.Regexp

	// GFM extensions
	tableCellRe     *regexp.Regexp
	autolinkRe      *regexp.Regexp
	strikethroughRe *regexp.Regexp

	// Performance optimizations
	lineCache  map[string][]string
	tokenCache map[string][]value.Token

	// Thread safety
	cacheMutex sync.RWMutex
}

// NewParserService creates a new parser service with optimized regex compilation.
func NewParserService() *ParserService {
	ps := &ParserService{
		enableGFM:        true,
		enableMath:       true,
		enableDirectives: true,
		enableFootnotes:  true,
		lineCache:        make(map[string][]string),
		tokenCache:       make(map[string][]value.Token),
	}

	// Pre-compile all regex patterns for maximum performance
	ps.compilePatterns()

	return ps
}

// compilePatterns pre-compiles all regex patterns used by the parser.
func (ps *ParserService) compilePatterns() {
	// ATX headings: # Title, ## Title, etc.
	ps.atxHeadingRe = regexp.MustCompile(`^(#{1,6})(?:\s+(.*))?$`)

	// Setext headings: Title followed by === or ---
	ps.setextHeadingRe = regexp.MustCompile(`^(=+|-+)\s*$`)

	// Indented code blocks: 4+ spaces or 1+ tabs
	ps.codeBlockRe = regexp.MustCompile(`^(?:    |\t)(.*)$`)

	// Fenced code blocks: ``` or ~~~
	ps.fencedCodeRe = regexp.MustCompile(`^(\s{0,3})([` + "`" + `~]{3,})\s*(.*)$`)

	// Blockquotes: > text
	ps.blockquoteRe = regexp.MustCompile(`^(\s{0,3}>\s?)(.*)$`)

	// List items: - text, * text, + text, 1. text
	ps.listItemRe = regexp.MustCompile(`^(\s*)([-*+]|\d{1,9}[.)])(\s+)(.*)$`)

	// Links: [text](url) or [text](url "title")
	ps.linkRe = regexp.MustCompile(`\[([^\]]*)\]\(([^)]*?)(?:\s+"([^"]*)")?\)`)

	// Images: ![alt](url) or ![alt](url "title")
	ps.imageRe = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]*?)(?:\s+"([^"]*)")?\)`)

	// Emphasis: *text* or _text_
	ps.emphasisRe = regexp.MustCompile(`(?:^|[^*_])([*_])([^*_\s][^*_]*?)([*_])`)

	// Strong emphasis: **text** or __text__
	ps.strongRe = regexp.MustCompile(`(?:^|[^*_])([*_]{2})([^*_\s][^*_]*?)([*_]{2})`)

	// Inline code: `code`
	ps.inlineCodeRe = regexp.MustCompile("(" + "`" + "+)([^" + "`" + "].*?)(" + "`" + "+)")

	// HTML tags
	ps.htmlTagRe = regexp.MustCompile(`</?[a-zA-Z][a-zA-Z0-9]*(?:\s+[^>]*)?/?>`)

	// Thematic breaks: ---, ***, ___
	ps.thematicBreakRe = regexp.MustCompile(`^(?:\s{0,3})((?:-\s*){3,}|(?:\*\s*){3,}|(?:_\s*){3,})\s*$`)

	if ps.enableGFM {
		// GFM table cells: | cell |
		ps.tableCellRe = regexp.MustCompile(`\|([^|]*)\|`)

		// GFM autolinks: https://example.com
		ps.autolinkRe = regexp.MustCompile(`https?://[^\s<>]+`)

		// GFM strikethrough: ~~text~~
		ps.strikethroughRe = regexp.MustCompile(`~~([^~]+)~~`)
	}
}

// ParseDocument parses markdown content into a structured token tree.
func (ps *ParserService) ParseDocument(ctx context.Context, content string, filename string) functional.Result[[]value.Token] {
	// Check cache first for performance
	ps.cacheMutex.RLock()
	cached, exists := ps.tokenCache[content]
	ps.cacheMutex.RUnlock()

	if exists {
		return functional.Ok(cached)
	}

	// Split content into lines for processing
	lines := ps.splitLines(content)

	// Parse the document structure
	result := ps.parseLines(ctx, lines, filename)
	if result.IsErr() {
		return result
	}

	tokens := result.Unwrap()

	// Cache the result for future use
	ps.cacheMutex.Lock()
	ps.tokenCache[content] = tokens
	ps.cacheMutex.Unlock()

	return functional.Ok(tokens)
}

// splitLines efficiently splits content into lines while preserving line ending information.
func (ps *ParserService) splitLines(content string) []string {
	// Check cache first
	ps.cacheMutex.RLock()
	cached, exists := ps.lineCache[content]
	ps.cacheMutex.RUnlock()

	if exists {
		return cached
	}

	// Split on various line endings
	lines := strings.Split(content, "\n")

	// Handle Windows line endings
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(line, "\r")
	}

	// Cache the result
	ps.cacheMutex.Lock()
	ps.lineCache[content] = lines
	ps.cacheMutex.Unlock()

	return lines
}

// parseLines processes lines into tokens using a state machine approach.
func (ps *ParserService) parseLines(ctx context.Context, lines []string, filename string) functional.Result[[]value.Token] {
	var tokens []value.Token

	// Parsing state
	state := &parseState{
		lines:        lines,
		lineNum:      0,
		inCodeBlock:  false,
		inBlockquote: false,
		listStack:    make([]listState, 0),
		tableState:   &tableState{},
	}

	// Process each line
	for state.lineNum < len(lines) {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return functional.Err[[]value.Token](ctx.Err())
		default:
		}

		line := lines[state.lineNum]
		lineTokens := ps.parseLine(state, line)
		tokens = append(tokens, lineTokens...)

		state.lineNum++
	}

	// Post-process tokens to create proper hierarchy
	processedTokens := ps.postProcessTokens(tokens)

	return functional.Ok(processedTokens)
}

// parseState maintains parsing state across lines.
type parseState struct {
	lines          []string
	lineNum        int
	inCodeBlock    bool
	codeBlockFence string
	inBlockquote   bool
	listStack      []listState
	tableState     *tableState
}

// listState tracks nested list information.
type listState struct {
	marker   string
	indent   int
	ordered  bool
	startNum int
}

// tableState tracks table parsing state.
type tableState struct {
	inTable      bool
	headerParsed bool
	columnCount  int
}

// parseLine parses a single line into tokens based on current state.
func (ps *ParserService) parseLine(state *parseState, line string) []value.Token {
	var tokens []value.Token

	// Skip empty lines but track them
	if strings.TrimSpace(line) == "" {
		tokens = append(tokens, ps.createLineToken(line, state.lineNum, value.TokenTypeLineEnding))
		return tokens
	}

	// Check for fenced code block boundaries
	if matches := ps.fencedCodeRe.FindStringSubmatch(line); matches != nil {
		fence := matches[2]

		if !state.inCodeBlock {
			// Starting a code block
			state.inCodeBlock = true
			state.codeBlockFence = fence

			token := ps.createLineToken(line, state.lineNum, value.TokenTypeCodeFenced)
			if len(matches) > 3 && matches[3] != "" {
				token = token.WithProperty("language", strings.TrimSpace(matches[3]))
			}
			tokens = append(tokens, token)
			return tokens
		} else if strings.HasPrefix(fence, string(state.codeBlockFence[0])) && len(fence) >= len(state.codeBlockFence) {
			// Ending a code block
			state.inCodeBlock = false
			state.codeBlockFence = ""

			tokens = append(tokens, ps.createLineToken(line, state.lineNum, value.TokenTypeCodeFenced))
			return tokens
		}
	}

	// If we're in a code block, treat everything as code content
	if state.inCodeBlock {
		tokens = append(tokens, ps.createLineToken(line, state.lineNum, value.TokenTypeText))
		return tokens
	}

	// Parse various block-level constructs
	tokens = append(tokens, ps.parseBlockLevel(state, line)...)

	return tokens
}

// parseBlockLevel parses block-level markdown constructs.
func (ps *ParserService) parseBlockLevel(state *parseState, line string) []value.Token {
	var tokens []value.Token

	// ATX Headings (# ## ###)
	if matches := ps.atxHeadingRe.FindStringSubmatch(line); matches != nil {
		level := len(matches[1])
		text := ""
		if len(matches) > 2 {
			text = strings.TrimSpace(matches[2])
		}

		token := ps.createLineToken(line, state.lineNum, value.TokenTypeATXHeading)
		token = token.WithProperty("level", level)
		token = token.WithProperty("text", text)

		// Parse inline content within heading
		if text != "" {
			inlineTokens := ps.parseInlineContent(text, state.lineNum)
			token = token.WithChildren(inlineTokens)
		}

		tokens = append(tokens, token)
		return tokens
	}

	// Setext Headings (underlined with = or -)
	if state.lineNum > 0 && ps.setextHeadingRe.MatchString(line) {
		prevLine := state.lines[state.lineNum-1]
		if strings.TrimSpace(prevLine) != "" {
			underline := strings.TrimSpace(line)
			level := 1
			if underline[0] == '-' {
				level = 2
			}

			token := ps.createLineToken(prevLine, state.lineNum-1, value.TokenTypeSetextHeading)
			token = token.WithProperty("level", level)
			token = token.WithProperty("text", strings.TrimSpace(prevLine))

			// Parse inline content
			inlineTokens := ps.parseInlineContent(strings.TrimSpace(prevLine), state.lineNum-1)
			token = token.WithChildren(inlineTokens)

			tokens = append(tokens, token)

			// Add underline token
			tokens = append(tokens, ps.createLineToken(line, state.lineNum, value.TokenTypeSetextHeadingUnderline))
			return tokens
		}
	}

	// Blockquotes
	if matches := ps.blockquoteRe.FindStringSubmatch(line); matches != nil {
		content := matches[2]
		token := ps.createLineToken(line, state.lineNum, value.TokenTypeBlockQuote)

		if content != "" {
			// Recursively parse blockquote content
			contentTokens := ps.parseInlineContent(content, state.lineNum)
			token = token.WithChildren(contentTokens)
		}

		tokens = append(tokens, token)
		return tokens
	}

	// List items
	if matches := ps.listItemRe.FindStringSubmatch(line); matches != nil {
		indent := len(matches[1])
		marker := matches[2]
		content := matches[4]

		token := ps.createLineToken(line, state.lineNum, value.TokenTypeListItem)
		token = token.WithProperty("marker", marker)
		token = token.WithProperty("indent", indent)

		// Determine if ordered or unordered
		ordered := unicode.IsDigit(rune(marker[0]))
		token = token.WithProperty("ordered", ordered)

		if content != "" {
			contentTokens := ps.parseInlineContent(content, state.lineNum)
			token = token.WithChildren(contentTokens)
		}

		tokens = append(tokens, token)
		return tokens
	}

	// Thematic breaks
	if ps.thematicBreakRe.MatchString(line) {
		tokens = append(tokens, ps.createLineToken(line, state.lineNum, value.TokenTypeThematicBreak))
		return tokens
	}

	// Indented code blocks
	if matches := ps.codeBlockRe.FindStringSubmatch(line); matches != nil {
		token := ps.createLineToken(line, state.lineNum, value.TokenTypeCodeIndented)
		token = token.WithProperty("content", matches[1])
		tokens = append(tokens, token)
		return tokens
	}

	// HTML blocks
	if ps.htmlTagRe.MatchString(line) {
		tokens = append(tokens, ps.createLineToken(line, state.lineNum, value.TokenTypeHTMLFlow))
		return tokens
	}

	// GFM Tables
	if ps.enableGFM && ps.isTableLine(line) {
		tableToken := ps.parseTableLine(line, state)
		if tableToken != nil {
			tokens = append(tokens, *tableToken)
			return tokens
		}
	}

	// Default: paragraph content
	token := ps.createLineToken(line, state.lineNum, value.TokenTypeParagraph)
	inlineTokens := ps.parseInlineContent(line, state.lineNum)
	token = token.WithChildren(inlineTokens)
	tokens = append(tokens, token)

	return tokens
}

// parseInlineContent parses inline markdown constructs within text.
func (ps *ParserService) parseInlineContent(text string, lineNum int) []value.Token {
	var tokens []value.Token
	pos := 0

	// Track position as we process the text
	for pos < len(text) {
		// Find the next inline construct
		nextPos, token := ps.findNextInlineToken(text, pos, lineNum)

		// Add any plain text before the construct
		if nextPos > pos {
			plainText := text[pos:nextPos]
			if strings.TrimSpace(plainText) != "" {
				textToken := ps.createInlineToken(plainText, lineNum, pos, value.TokenTypeText)
				tokens = append(tokens, textToken)
			}
		}

		// Add the construct token if found
		if token != nil {
			tokens = append(tokens, *token)
			pos = token.EndColumn()
		} else {
			// No more constructs, add remaining text
			if pos < len(text) {
				remainingText := text[pos:]
				if strings.TrimSpace(remainingText) != "" {
					textToken := ps.createInlineToken(remainingText, lineNum, pos, value.TokenTypeText)
					tokens = append(tokens, textToken)
				}
			}
			break
		}
	}

	return tokens
}

// findNextInlineToken finds the next inline markdown construct in text.
func (ps *ParserService) findNextInlineToken(text string, startPos int, lineNum int) (int, *value.Token) {
	searchText := text[startPos:]

	// Try to find various inline constructs in order of precedence
	constructs := []struct {
		regex     *regexp.Regexp
		tokenType value.TokenType
		handler   func([]string, int, int) *value.Token
	}{
		{ps.inlineCodeRe, value.TokenTypeCodeText, ps.handleInlineCode},
		{ps.strongRe, value.TokenTypeStrong, ps.handleStrong},
		{ps.emphasisRe, value.TokenTypeEmphasis, ps.handleEmphasis},
		{ps.linkRe, value.TokenTypeLink, ps.handleLink},
		{ps.imageRe, value.TokenTypeImage, ps.handleImage},
		{ps.htmlTagRe, value.TokenTypeHTMLText, ps.handleHTMLText},
	}

	if ps.enableGFM {
		constructs = append(constructs, []struct {
			regex     *regexp.Regexp
			tokenType value.TokenType
			handler   func([]string, int, int) *value.Token
		}{
			{ps.autolinkRe, value.TokenTypeAutolink, ps.handleAutolink},
			{ps.strikethroughRe, value.TokenTypeText, ps.handleStrikethrough}, // GFM strikethrough
		}...)
	}

	// Find the earliest match
	earliestPos := len(searchText)
	var earliestMatch []string
	var earliestHandler func([]string, int, int) *value.Token

	for _, construct := range constructs {
		if matches := construct.regex.FindStringSubmatch(searchText); matches != nil {
			matchPos := strings.Index(searchText, matches[0])
			if matchPos < earliestPos {
				earliestPos = matchPos
				earliestMatch = matches
				earliestHandler = construct.handler
			}
		}
	}

	if earliestMatch != nil {
		token := earliestHandler(earliestMatch, lineNum, startPos+earliestPos)
		return startPos + earliestPos, token
	}

	return len(text), nil
}

// Token handler functions
func (ps *ParserService) handleInlineCode(matches []string, lineNum int, pos int) *value.Token {
	token := ps.createInlineToken(matches[0], lineNum, pos, value.TokenTypeCodeText)
	token = token.WithProperty("content", matches[2])
	return &token
}

func (ps *ParserService) handleStrong(matches []string, lineNum int, pos int) *value.Token {
	token := ps.createInlineToken(matches[0], lineNum, pos, value.TokenTypeStrong)
	token = token.WithProperty("content", matches[2])
	return &token
}

func (ps *ParserService) handleEmphasis(matches []string, lineNum int, pos int) *value.Token {
	token := ps.createInlineToken(matches[0], lineNum, pos, value.TokenTypeEmphasis)
	token = token.WithProperty("content", matches[2])
	return &token
}

func (ps *ParserService) handleLink(matches []string, lineNum int, pos int) *value.Token {
	token := ps.createInlineToken(matches[0], lineNum, pos, value.TokenTypeLink)
	token = token.WithProperty("text", matches[1])
	token = token.WithProperty("url", matches[2])
	if len(matches) > 3 && matches[3] != "" {
		token = token.WithProperty("title", matches[3])
	}
	return &token
}

func (ps *ParserService) handleImage(matches []string, lineNum int, pos int) *value.Token {
	token := ps.createInlineToken(matches[0], lineNum, pos, value.TokenTypeImage)
	token = token.WithProperty("alt", matches[1])
	token = token.WithProperty("url", matches[2])
	if len(matches) > 3 && matches[3] != "" {
		token = token.WithProperty("title", matches[3])
	}
	return &token
}

func (ps *ParserService) handleHTMLText(matches []string, lineNum int, pos int) *value.Token {
	token := ps.createInlineToken(matches[0], lineNum, pos, value.TokenTypeHTMLText)
	return &token
}

func (ps *ParserService) handleAutolink(matches []string, lineNum int, pos int) *value.Token {
	token := ps.createInlineToken(matches[0], lineNum, pos, value.TokenTypeAutolink)
	token = token.WithProperty("url", matches[0])
	return &token
}

func (ps *ParserService) handleStrikethrough(matches []string, lineNum int, pos int) *value.Token {
	token := ps.createInlineToken(matches[0], lineNum, pos, value.TokenTypeText)
	token = token.WithProperty("strikethrough", true)
	token = token.WithProperty("content", matches[1])
	return &token
}

// Table parsing helpers
func (ps *ParserService) isTableLine(line string) bool {
	return strings.Contains(line, "|") && strings.TrimSpace(line) != ""
}

func (ps *ParserService) parseTableLine(line string, state *parseState) *value.Token {
	if ps.tableCellRe.MatchString(line) {
		token := ps.createLineToken(line, state.lineNum, value.TokenTypeTableRow)

		// Extract cells
		cells := ps.tableCellRe.FindAllStringSubmatch(line, -1)
		var cellTokens []value.Token

		for _, cell := range cells {
			cellContent := strings.TrimSpace(cell[1])
			cellToken := ps.createInlineToken(cellContent, state.lineNum, 0, value.TokenTypeTableCell)

			// Parse inline content within cell
			if cellContent != "" {
				inlineTokens := ps.parseInlineContent(cellContent, state.lineNum)
				cellToken = cellToken.WithChildren(inlineTokens)
			}

			cellTokens = append(cellTokens, cellToken)
		}

		token = token.WithChildren(cellTokens)
		return &token
	}

	return nil
}

// Helper functions to create tokens
func (ps *ParserService) createLineToken(text string, lineNum int, tokenType value.TokenType) value.Token {
	start := value.Position{Line: lineNum + 1, Column: 1, Offset: 0}
	end := value.Position{Line: lineNum + 1, Column: len(text) + 1, Offset: len(text)}

	return value.NewToken(tokenType, text, start, end)
}

func (ps *ParserService) createInlineToken(text string, lineNum int, columnStart int, tokenType value.TokenType) value.Token {
	start := value.Position{Line: lineNum + 1, Column: columnStart + 1, Offset: columnStart}
	end := value.Position{Line: lineNum + 1, Column: columnStart + len(text) + 1, Offset: columnStart + len(text)}

	return value.NewToken(tokenType, text, start, end)
}

// postProcessTokens creates proper token hierarchy and relationships.
func (ps *ParserService) postProcessTokens(tokens []value.Token) []value.Token {
	// Group tokens by document structure
	var processedTokens []value.Token

	// This is a simplified version - a full implementation would properly
	// nest list items, blockquotes, and other hierarchical structures
	processedTokens = tokens

	return processedTokens
}

// Performance optimization: Clear caches when they get too large
func (ps *ParserService) ClearCaches() {
	if len(ps.lineCache) > 1000 {
		ps.lineCache = make(map[string][]string)
	}
	if len(ps.tokenCache) > 100 {
		ps.tokenCache = make(map[string][]value.Token)
	}
}
