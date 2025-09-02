package helpers

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/gomdlint/gomdlint/internal/domain/value"
)

// Token manipulation helpers

// FilterTokensByType filters tokens by their type
func FilterTokensByType(tokens []value.Token, tokenType value.TokenType) []value.Token {
	var filtered []value.Token
	for _, token := range tokens {
		if token.IsType(tokenType) {
			filtered = append(filtered, token)
		}
	}
	return filtered
}

// FindTokensInRange finds tokens within a specific line range
func FindTokensInRange(tokens []value.Token, startLine, endLine int) []value.Token {
	var inRange []value.Token
	for _, token := range tokens {
		if token.StartLine() >= startLine && token.EndLine() <= endLine {
			inRange = append(inRange, token)
		}
	}
	return inRange
}

// GetTokensOfTypes returns tokens matching any of the specified types
func GetTokensOfTypes(tokens []value.Token, types ...value.TokenType) []value.Token {
	var matched []value.Token
	for _, token := range tokens {
		if token.IsOneOfTypes(types...) {
			matched = append(matched, token)
		}
	}
	return matched
}

// Text analysis helpers

// IsBlankLine checks if a line contains only whitespace
func IsBlankLine(line string) bool {
	return strings.TrimSpace(line) == ""
}

// CountLeadingSpaces counts leading spaces in a line
func CountLeadingSpaces(line string) int {
	count := 0
	for _, char := range line {
		if char == ' ' {
			count++
		} else {
			break
		}
	}
	return count
}

// CountLeadingTabs counts leading tabs in a line
func CountLeadingTabs(line string) int {
	count := 0
	for _, char := range line {
		if char == '\t' {
			count++
		} else {
			break
		}
	}
	return count
}

// GetIndentationType determines if line uses spaces or tabs for indentation
func GetIndentationType(line string) string {
	if len(line) == 0 {
		return "none"
	}

	firstChar := line[0]
	switch firstChar {
	case ' ':
		return "spaces"
	case '\t':
		return "tabs"
	default:
		return "none"
	}
}

// Regular expressions for common patterns
var (
	FrontMatterRe = regexp.MustCompile(`^---\s*\n(.*?\n)?---\s*\n`)
	HTMLEntityRe  = regexp.MustCompile(`&(?:#\d+|#[xX][\da-fA-F]+|[a-zA-Z]{2,31});`)
	EmojiRe       = regexp.MustCompile(`:[\w+-]+:`)
	URLRe         = regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`)
	EmailRe       = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
)

// Front matter helpers

// ExtractFrontMatter extracts YAML front matter from content
func ExtractFrontMatter(content string) (frontMatter string, body string, hasFrontMatter bool) {
	matches := FrontMatterRe.FindStringSubmatch(content)
	if len(matches) > 0 {
		return matches[1], content[len(matches[0]):], true
	}
	return "", content, false
}

// RemoveFrontMatter removes front matter and returns the body content
func RemoveFrontMatter(content string) string {
	_, body, _ := ExtractFrontMatter(content)
	return body
}

// HasFrontMatter checks if content has YAML front matter
func HasFrontMatter(content string) bool {
	return FrontMatterRe.MatchString(content)
}

// Heading helpers

// IsATXHeading checks if a line is an ATX-style heading
func IsATXHeading(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "#") &&
		(len(trimmed) == 1 || trimmed[1] == ' ' || trimmed[1] == '#')
}

// GetATXHeadingLevel returns the heading level (1-6) for ATX headings
func GetATXHeadingLevel(line string) int {
	trimmed := strings.TrimSpace(line)
	level := 0
	for i, char := range trimmed {
		if char == '#' && level < 6 {
			level++
		} else {
			break
		}
	}
	return level
}

// GetATXHeadingText extracts the text from an ATX heading
func GetATXHeadingText(line string) string {
	trimmed := strings.TrimSpace(line)
	level := GetATXHeadingLevel(line)
	if level == 0 {
		return trimmed
	}

	text := strings.TrimSpace(trimmed[level:])
	// Remove trailing hashes if present
	text = strings.TrimRight(text, "# ")
	return text
}

// IsSetextHeading checks if a line pair forms a Setext heading
func IsSetextHeading(line, nextLine string) bool {
	if strings.TrimSpace(line) == "" {
		return false
	}

	nextTrimmed := strings.TrimSpace(nextLine)
	if len(nextTrimmed) == 0 {
		return false
	}

	// Check if next line is all = or -
	char := nextTrimmed[0]
	if char != '=' && char != '-' {
		return false
	}

	for _, c := range nextTrimmed {
		if c != rune(char) {
			return false
		}
	}

	return len(nextTrimmed) >= 3
}

// GetSetextHeadingLevel returns 1 for = underline, 2 for - underline
func GetSetextHeadingLevel(underline string) int {
	trimmed := strings.TrimSpace(underline)
	if len(trimmed) == 0 {
		return 0
	}

	switch trimmed[0] {
	case '=':
		return 1
	case '-':
		return 2
	default:
		return 0
	}
}

// List helpers

// IsListItem checks if a line is a list item
func IsListItem(line string) bool {
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

	// Ordered list markers
	for i, char := range trimmed {
		if unicode.IsDigit(char) {
			continue
		} else if char == '.' && i > 0 && i < len(trimmed)-1 && trimmed[i+1] == ' ' {
			return true
		} else {
			break
		}
	}

	return false
}

// GetListMarker extracts the marker from a list item
func GetListMarker(line string) string {
	trimmed := strings.TrimLeft(line, " \t")

	// Unordered markers
	if strings.HasPrefix(trimmed, "- ") {
		return "-"
	}
	if strings.HasPrefix(trimmed, "* ") {
		return "*"
	}
	if strings.HasPrefix(trimmed, "+ ") {
		return "+"
	}

	// Ordered markers
	for i, char := range trimmed {
		if unicode.IsDigit(char) {
			continue
		} else if char == '.' {
			return trimmed[:i+1]
		} else {
			break
		}
	}

	return ""
}

// IsOrderedListItem checks if a line is an ordered list item
func IsOrderedListItem(line string) bool {
	marker := GetListMarker(line)
	return strings.HasSuffix(marker, ".")
}

// IsUnorderedListItem checks if a line is an unordered list item
func IsUnorderedListItem(line string) bool {
	marker := GetListMarker(line)
	return marker == "-" || marker == "*" || marker == "+"
}

// Code block helpers

// IsFencedCodeBlock checks if a line starts a fenced code block
func IsFencedCodeBlock(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

// GetCodeFenceInfo extracts language and info from a fenced code block line
func GetCodeFenceInfo(line string) (language string, info string) {
	trimmed := strings.TrimSpace(line)
	if !IsFencedCodeBlock(line) {
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

// IsIndentedCodeBlock checks if a line is part of an indented code block
func IsIndentedCodeBlock(line string) bool {
	if IsBlankLine(line) {
		return false
	}
	return CountLeadingSpaces(line) >= 4 || CountLeadingTabs(line) >= 1
}

// Table helpers

// IsTableRow checks if a line appears to be a table row
func IsTableRow(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.Contains(trimmed, "|") && len(trimmed) > 1
}

// IsTableSeparator checks if a line is a table separator row
func IsTableSeparator(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.Contains(trimmed, "|") {
		return false
	}

	// Remove pipes and check if remaining chars are only - : and spaces
	cleaned := strings.ReplaceAll(trimmed, "|", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")

	if len(cleaned) == 0 {
		return false
	}

	for _, char := range cleaned {
		if char != '-' && char != ':' {
			return false
		}
	}

	return true
}

// CountTableColumns counts the number of columns in a table row
func CountTableColumns(line string) int {
	trimmed := strings.TrimSpace(line)
	if !strings.Contains(trimmed, "|") {
		return 0
	}

	// Count pipes, but ignore escaped pipes
	count := 0
	escaped := false
	for _, char := range trimmed {
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		if char == '|' {
			count++
		}
	}

	// Table format: |col1|col2|col3| has count+1 columns
	// But also handle: col1|col2|col3 which has count+1 columns
	if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
		return count - 1
	}
	return count + 1
}

// Emphasis helpers

// HasTrailingPunctuation checks if text ends with punctuation
func HasTrailingPunctuation(text string) bool {
	if len(text) == 0 {
		return false
	}

	lastChar := rune(text[len(text)-1])
	return strings.ContainsRune(".,;:!?", lastChar)
}

// IsEmphasisMarker checks if a character is an emphasis marker
func IsEmphasisMarker(char rune) bool {
	return char == '*' || char == '_'
}

// CountEmphasisMarkers counts consecutive emphasis markers
func CountEmphasisMarkers(text string, startPos int) int {
	if startPos >= len(text) {
		return 0
	}

	marker := rune(text[startPos])
	if !IsEmphasisMarker(marker) {
		return 0
	}

	count := 0
	for i := startPos; i < len(text); i++ {
		if rune(text[i]) == marker {
			count++
		} else {
			break
		}
	}

	return count
}

// URL and link helpers

// IsURL checks if text appears to be a URL
func IsURL(text string) bool {
	return URLRe.MatchString(text)
}

// IsEmail checks if text appears to be an email address
func IsEmail(text string) bool {
	return EmailRe.MatchString(text)
}

// ExtractURLs finds all URLs in text
func ExtractURLs(text string) []string {
	return URLRe.FindAllString(text, -1)
}

// ExtractEmails finds all email addresses in text
func ExtractEmails(text string) []string {
	return EmailRe.FindAllString(text, -1)
}

// Whitespace helpers

// HasTrailingWhitespace checks if a line has trailing whitespace
func HasTrailingWhitespace(line string) bool {
	return len(line) > 0 && unicode.IsSpace(rune(line[len(line)-1]))
}

// CountTrailingSpaces counts trailing spaces in a line
func CountTrailingSpaces(line string) int {
	count := 0
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] == ' ' {
			count++
		} else {
			break
		}
	}
	return count
}

// CountTrailingTabs counts trailing tabs in a line
func CountTrailingTabs(line string) int {
	count := 0
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] == '\t' {
			count++
		} else {
			break
		}
	}
	return count
}

// HasHardTabs checks if a line contains tab characters
func HasHardTabs(line string) bool {
	return strings.Contains(line, "\t")
}

// ReplaceHardTabs replaces tabs with spaces
func ReplaceHardTabs(line string, spacesPerTab int) string {
	spaces := strings.Repeat(" ", spacesPerTab)
	return strings.ReplaceAll(line, "\t", spaces)
}

// Blockquote helpers

// IsBlockquote checks if a line is a blockquote
func IsBlockquote(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trimmed, ">")
}

// GetBlockquoteLevel returns the nesting level of a blockquote
func GetBlockquoteLevel(line string) int {
	trimmed := strings.TrimLeft(line, " \t")
	level := 0
	for i, char := range trimmed {
		if char == '>' {
			level++
			// Skip optional space after >
			if i+1 < len(trimmed) && trimmed[i+1] == ' ' {
				i++
			}
		} else {
			break
		}
	}
	return level
}

// GetBlockquoteText extracts text from a blockquote line
func GetBlockquoteText(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	
	// Remove > markers and optional spaces
	for strings.HasPrefix(trimmed, ">") {
		trimmed = trimmed[1:]
		if strings.HasPrefix(trimmed, " ") {
			trimmed = trimmed[1:]
		}
	}
	
	return trimmed
}

// Horizontal rule helpers

// IsHorizontalRule checks if a line is a horizontal rule
func IsHorizontalRule(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Must be at least 3 characters
	if len(trimmed) < 3 {
		return false
	}

	// Check for various horizontal rule patterns
	patterns := []rune{'-', '*', '_'}

	for _, pattern := range patterns {
		if isHorizontalRulePattern(trimmed, pattern) {
			return true
		}
	}

	return false
}

// isHorizontalRulePattern checks if line matches a specific HR pattern
func isHorizontalRulePattern(line string, marker rune) bool {
	charCount := 0
	spaceCount := 0

	for _, char := range line {
		if char == marker {
			charCount++
		} else if char == ' ' || char == '\t' {
			spaceCount++
		} else {
			return false // Invalid character
		}
	}

	return charCount >= 3
}

// Link helpers

// IsInlineLink checks if text contains inline links
func IsInlineLink(text string) bool {
	return strings.Contains(text, "](") && strings.Contains(text, "[")
}

// IsReferenceLink checks if text contains reference links
func IsReferenceLink(text string) bool {
	// Pattern: [text][ref] or [text][]
	return regexp.MustCompile(`\[[^\]]*\]\s*\[[^\]]*\]`).MatchString(text)
}

// HTML helpers

// HasInlineHTML checks if text contains HTML tags
func HasInlineHTML(text string) bool {
	return regexp.MustCompile(`<[^>]+>`).MatchString(text)
}

// IsHTMLBlock checks if a line starts an HTML block
func IsHTMLBlock(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">")
}

// Fix helpers

// CreateLineReplacement creates fix info for replacing an entire line
func CreateLineReplacement(lineNumber int, newContent string) *value.FixInfo {
	return value.NewFixInfo().
		WithLineNumber(lineNumber).
		WithEditColumn(1).
		WithDeleteLength(-1). // Delete entire line
		WithReplaceText(newContent)
}

// CreateTextInsertion creates fix info for inserting text
func CreateTextInsertion(lineNumber, column int, text string) *value.FixInfo {
	return value.NewFixInfo().
		WithLineNumber(lineNumber).
		WithEditColumn(column).
		WithInsertText(text)
}

// CreateTextDeletion creates fix info for deleting text
func CreateTextDeletion(lineNumber, column, length int) *value.FixInfo {
	return value.NewFixInfo().
		WithLineNumber(lineNumber).
		WithEditColumn(column).
		WithDeleteLength(length)
}

// CreateTextReplacement creates fix info for replacing text
func CreateTextReplacement(lineNumber, column, length int, replacement string) *value.FixInfo {
	return value.NewFixInfo().
		WithLineNumber(lineNumber).
		WithEditColumn(column).
		WithDeleteLength(length).
		WithReplaceText(replacement)
}

// Validation helpers

// ValidateLineLength checks if a line exceeds the specified length
func ValidateLineLength(line string, maxLength int) bool {
	return len(line) <= maxLength
}

// ValidateNoTrailingSpaces checks if a line has no trailing spaces
func ValidateNoTrailingSpaces(line string) bool {
	if len(line) == 0 {
		return true
	}
	return line[len(line)-1] != ' '
}

// ValidateNoHardTabs checks if a line contains no tab characters
func ValidateNoHardTabs(line string) bool {
	return !strings.Contains(line, "\t")
}

// Content analysis helpers

// CountWords counts words in text (simple whitespace splitting)
func CountWords(text string) int {
	fields := strings.Fields(text)
	return len(fields)
}

// CountSentences counts sentences in text (simple period counting)
func CountSentences(text string) int {
	return strings.Count(text, ".") + strings.Count(text, "!") + strings.Count(text, "?")
}

// GetTextComplexity returns a simple complexity score based on word and sentence count
func GetTextComplexity(text string) float64 {
	words := CountWords(text)
	sentences := CountSentences(text)
	
	if sentences == 0 {
		return 0
	}
	
	return float64(words) / float64(sentences)
}

// Document structure helpers

// FindHeadings returns all heading tokens from a token list
func FindHeadings(tokens []value.Token) []value.Token {
	return GetTokensOfTypes(tokens, value.TokenTypeATXHeading, value.TokenTypeSetextHeading)
}

// FindLists returns all list-related tokens
func FindLists(tokens []value.Token) []value.Token {
	return GetTokensOfTypes(tokens, value.TokenTypeList, value.TokenTypeListItem)
}

// FindCodeBlocks returns all code block tokens
func FindCodeBlocks(tokens []value.Token) []value.Token {
	return GetTokensOfTypes(tokens, value.TokenTypeCodeFenced, value.TokenTypeCodeIndented)
}

// FindBlockquotes returns all blockquote tokens
func FindBlockquotes(tokens []value.Token) []value.Token {
	return FilterTokensByType(tokens, value.TokenTypeBlockQuote)
}