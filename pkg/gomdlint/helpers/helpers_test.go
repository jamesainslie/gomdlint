package helpers

import (
	"strings"
	"testing"

	"github.com/gomdlint/gomdlint/internal/domain/value"
)

func TestFilterTokensByType(t *testing.T) {
	// Create test tokens
	tokens := []value.Token{
		value.NewToken(value.TokenTypeATXHeading, "# Heading", value.NewPosition(1, 1), value.NewPosition(1, 10)),
		value.NewToken(value.TokenTypeParagraph, "Paragraph text", value.NewPosition(2, 1), value.NewPosition(2, 15)),
		value.NewToken(value.TokenTypeATXHeading, "## Subheading", value.NewPosition(3, 1), value.NewPosition(3, 14)),
	}

	headings := FilterTokensByType(tokens, value.TokenTypeATXHeading)

	if len(headings) != 2 {
		t.Errorf("expected 2 headings, got %d", len(headings))
	}

	for _, heading := range headings {
		if !heading.IsType(value.TokenTypeATXHeading) {
			t.Error("filtered token should be ATX heading")
		}
	}
}

func TestGetTokensOfTypes(t *testing.T) {
	tokens := []value.Token{
		value.NewToken(value.TokenTypeATXHeading, "# Heading", value.NewPosition(1, 1), value.NewPosition(1, 10)),
		value.NewToken(value.TokenTypeParagraph, "Paragraph", value.NewPosition(2, 1), value.NewPosition(2, 10)),
		value.NewToken(value.TokenTypeSetextHeading, "Heading", value.NewPosition(3, 1), value.NewPosition(3, 8)),
		value.NewToken(value.TokenTypeCodeFenced, "```code```", value.NewPosition(4, 1), value.NewPosition(4, 11)),
	}

	headings := GetTokensOfTypes(tokens, value.TokenTypeATXHeading, value.TokenTypeSetextHeading)

	if len(headings) != 2 {
		t.Errorf("expected 2 headings, got %d", len(headings))
	}
}

func TestFindTokensInRange(t *testing.T) {
	tokens := []value.Token{
		value.NewToken(value.TokenTypeParagraph, "Line 1", value.NewPosition(1, 1), value.NewPosition(1, 6)),
		value.NewToken(value.TokenTypeParagraph, "Line 2", value.NewPosition(2, 1), value.NewPosition(2, 6)),
		value.NewToken(value.TokenTypeParagraph, "Line 3", value.NewPosition(3, 1), value.NewPosition(3, 6)),
		value.NewToken(value.TokenTypeParagraph, "Line 4", value.NewPosition(4, 1), value.NewPosition(4, 6)),
	}

	inRange := FindTokensInRange(tokens, 2, 3)

	if len(inRange) != 2 {
		t.Errorf("expected 2 tokens in range, got %d", len(inRange))
	}

	if inRange[0].StartLine() != 2 || inRange[1].StartLine() != 3 {
		t.Error("tokens in range should be from lines 2 and 3")
	}
}

func TestIsBlankLine(t *testing.T) {
	testCases := []struct {
		line     string
		expected bool
	}{
		{"", true},
		{"   ", true},
		{"\t\t", true},
		{"  \n  ", true},
		{"text", false},
		{"  text  ", false},
	}

	for _, tc := range testCases {
		result := IsBlankLine(tc.line)
		if result != tc.expected {
			t.Errorf("IsBlankLine(%q) = %v, expected %v", tc.line, result, tc.expected)
		}
	}
}

func TestCountLeadingSpaces(t *testing.T) {
	testCases := []struct {
		line     string
		expected int
	}{
		{"", 0},
		{"no spaces", 0},
		{" one space", 1},
		{"    four spaces", 4},
		{"\ttab", 0}, // Only counts spaces, not tabs
		{"  \t mixed", 2},
	}

	for _, tc := range testCases {
		result := CountLeadingSpaces(tc.line)
		if result != tc.expected {
			t.Errorf("CountLeadingSpaces(%q) = %d, expected %d", tc.line, result, tc.expected)
		}
	}
}

func TestCountLeadingTabs(t *testing.T) {
	testCases := []struct {
		line     string
		expected int
	}{
		{"", 0},
		{"no tabs", 0},
		{"\tone tab", 1},
		{"\t\t\ttree tabs", 3},
		{"  spaces", 0}, // Only counts tabs, not spaces
		{"\t  mixed", 1},
	}

	for _, tc := range testCases {
		result := CountLeadingTabs(tc.line)
		if result != tc.expected {
			t.Errorf("CountLeadingTabs(%q) = %d, expected %d", tc.line, result, tc.expected)
		}
	}
}

func TestCountTrailingSpaces(t *testing.T) {
	testCases := []struct {
		line     string
		expected int
	}{
		{"", 0},
		{"no spaces", 0},
		{"one space ", 1},
		{"four spaces    ", 4},
		{"text\t", 0}, // Only counts spaces, not tabs
		{"text  \t", 2},
	}

	for _, tc := range testCases {
		result := CountTrailingSpaces(tc.line)
		if result != tc.expected {
			t.Errorf("CountTrailingSpaces(%q) = %d, expected %d", tc.line, result, tc.expected)
		}
	}
}

func TestHasTrailingWhitespace(t *testing.T) {
	testCases := []struct {
		line     string
		expected bool
	}{
		{"", false},
		{"no whitespace", false},
		{"has space ", true},
		{"has tab\t", true},
		{"has newline\n", true},
	}

	for _, tc := range testCases {
		result := HasTrailingWhitespace(tc.line)
		if result != tc.expected {
			t.Errorf("HasTrailingWhitespace(%q) = %v, expected %v", tc.line, result, tc.expected)
		}
	}
}

func TestHasHardTabs(t *testing.T) {
	testCases := []struct {
		line     string
		expected bool
	}{
		{"", false},
		{"no tabs", false},
		{"has\ttab", true},
		{"\tstarts with tab", true},
		{"ends with tab\t", true},
	}

	for _, tc := range testCases {
		result := HasHardTabs(tc.line)
		if result != tc.expected {
			t.Errorf("HasHardTabs(%q) = %v, expected %v", tc.line, result, tc.expected)
		}
	}
}

func TestGetIndentationType(t *testing.T) {
	spaceLines := []string{
		"    space indented",
		"  two spaces",
		"normal line",
	}

	tabLines := []string{
		"\ttab indented",
		"\t\ttwo tabs",
		"normal line",
	}

	mixedLines := []string{
		"    space indented",
		"\ttab indented",
		"normal line",
	}

	if GetIndentationType(spaceLines[0]) != "spaces" {
		t.Error("expected spaces")
	}

	if GetIndentationType(tabLines[0]) != "tabs" {
		t.Error("expected tabs")
	}

	if GetIndentationType(mixedLines[0]) != "spaces" {
		t.Error("expected spaces for mixed (first char)")
	}
}

func TestExtractFrontMatter(t *testing.T) {
	contentWithFM := `---
title: Test
author: Someone
---

# Main Content`

	contentWithoutFM := `# Just Content
No front matter here`

	// Test with front matter
	fm, body, hasFM := ExtractFrontMatter(contentWithFM)
	if !hasFM {
		t.Error("should detect front matter")
	}

	if !strings.Contains(fm, "title: Test") {
		t.Error("front matter should contain title")
	}

	if !strings.HasPrefix(body, "# Main Content") {
		t.Error("body should start with main content")
	}

	// Test without front matter
	_, body2, hasFM2 := ExtractFrontMatter(contentWithoutFM)
	if hasFM2 {
		t.Error("should not detect front matter")
	}

	if body2 != contentWithoutFM {
		t.Error("body should be unchanged when no front matter")
	}
}

func TestIsATXHeading(t *testing.T) {
	testCases := []struct {
		line     string
		expected bool
	}{
		{"# Heading", true},
		{"## Heading 2", true},
		{"### Heading 3", true},
		{"#### Heading 4", true},
		{"##### Heading 5", true},
		{"###### Heading 6", true},
		{"####### Too many", false}, // Only 1-6 allowed
		{"#No space", false},        // Needs space after #
		{"Not a heading", false},
		{"#", true}, // Just # is valid
	}

	for _, tc := range testCases {
		result := IsATXHeading(tc.line)
		if result != tc.expected {
			t.Errorf("IsATXHeading(%q) = %v, expected %v", tc.line, result, tc.expected)
		}
	}
}

func TestGetATXHeadingLevel(t *testing.T) {
	testCases := []struct {
		line     string
		expected int
	}{
		{"# Heading", 1},
		{"## Heading 2", 2},
		{"### Heading 3", 3},
		{"#### Heading 4", 4},
		{"##### Heading 5", 5},
		{"###### Heading 6", 6},
		{"Not heading", 0},
		{"#", 1},
	}

	for _, tc := range testCases {
		result := GetATXHeadingLevel(tc.line)
		if result != tc.expected {
			t.Errorf("GetATXHeadingLevel(%q) = %d, expected %d", tc.line, result, tc.expected)
		}
	}
}

func TestGetATXHeadingText(t *testing.T) {
	testCases := []struct {
		line     string
		expected string
	}{
		{"# Heading", "Heading"},
		{"## Heading with more text", "Heading with more text"},
		{"### ", ""},
		{"#", ""},
		{"Not heading", ""},
	}

	for _, tc := range testCases {
		result := GetATXHeadingText(tc.line)
		if result != tc.expected {
			t.Errorf("GetATXHeadingText(%q) = %q, expected %q", tc.line, result, tc.expected)
		}
	}
}

func TestIsSetextHeading(t *testing.T) {
	testCases := []struct {
		line     string
		nextLine string
		expected bool
	}{
		{"Heading", "=======", true},
		{"Heading", "-------", true},
		{"Heading", "====", true},
		{"Heading", "----", true},
		{"", "====", false},           // Empty line
		{"Heading", "===x===", false}, // Mixed characters
		{"Heading", "", false},        // Empty underline
		{"Heading", "normal text", false},
	}

	for _, tc := range testCases {
		result := IsSetextHeading(tc.line, tc.nextLine)
		if result != tc.expected {
			t.Errorf("IsSetextHeading(%q, %q) = %v, expected %v", tc.line, tc.nextLine, result, tc.expected)
		}
	}
}

func TestIsListItem(t *testing.T) {
	testCases := []struct {
		line     string
		expected bool
	}{
		{"- Item", true},
		{"* Item", true},
		{"+ Item", true},
		{"1. Numbered", true},
		{"10. Double digit", true},
		{"  - Indented", true},
		{"-No space", false},
		{"1.No space", false},
		{"Not a list", false},
	}

	for _, tc := range testCases {
		result := IsListItem(tc.line)
		if result != tc.expected {
			t.Errorf("IsListItem(%q) = %v, expected %v", tc.line, result, tc.expected)
		}
	}
}

func TestGetListMarker(t *testing.T) {
	testCases := []struct {
		line     string
		expected string
	}{
		{"- Item", "-"},
		{"* Item", "*"},
		{"+ Item", "+"},
		{"1. Numbered", "1."},
		{"10. Double digit", "10."},
		{"  - Indented", "-"},
		{"Not a list", ""},
	}

	for _, tc := range testCases {
		result := GetListMarker(tc.line)
		if result != tc.expected {
			t.Errorf("GetListMarker(%q) = %q, expected %q", tc.line, result, tc.expected)
		}
	}
}

func TestIsOrderedList(t *testing.T) {
	testCases := []struct {
		marker   string
		expected bool
	}{
		{"1.", true},
		{"10.", true},
		{"1)", true},
		{"-", false},
		{"*", false},
		{"+", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := IsOrderedList(tc.marker)
		if result != tc.expected {
			t.Errorf("IsOrderedList(%q) = %v, expected %v", tc.marker, result, tc.expected)
		}
	}
}

func TestIsFencedCodeBlock(t *testing.T) {
	testCases := []struct {
		line     string
		expected bool
	}{
		{"```", true},
		{"```javascript", true},
		{"~~~", true},
		{"~~~python", true},
		{"  ```", true}, // Indented
		{"````", true},  // More than 3
		{"``", false},   // Less than 3
		{"Not code", false},
	}

	for _, tc := range testCases {
		result := IsFencedCodeBlock(tc.line)
		if result != tc.expected {
			t.Errorf("IsFencedCodeBlock(%q) = %v, expected %v", tc.line, result, tc.expected)
		}
	}
}

func TestGetCodeFenceInfo(t *testing.T) {
	testCases := []struct {
		line    string
		expLang string
		expInfo string
	}{
		{"```", "", ""},
		{"```javascript", "javascript", ""},
		{"```python startline=1", "python", "startline=1"},
		{"~~~bash", "bash", ""},
		{"  ```go", "go", ""},
		{"Not code", "", ""},
	}

	for _, tc := range testCases {
		lang, info := GetCodeFenceInfo(tc.line)
		if lang != tc.expLang || info != tc.expInfo {
			t.Errorf("GetCodeFenceInfo(%q) = (%q, %q), expected (%q, %q)",
				tc.line, lang, info, tc.expLang, tc.expInfo)
		}
	}
}

func TestIsTableRow(t *testing.T) {
	testCases := []struct {
		line     string
		expected bool
	}{
		{"| Col 1 | Col 2 |", true},
		{"|Col1|Col2|", true},
		{"Col1 | Col2", true},
		{"| Single column", true},
		{"No pipes here", false},
		{"|", false}, // Too short
		{"", false},
	}

	for _, tc := range testCases {
		result := IsTableRow(tc.line)
		if result != tc.expected {
			t.Errorf("IsTableRow(%q) = %v, expected %v", tc.line, result, tc.expected)
		}
	}
}

func TestIsTableSeparator(t *testing.T) {
	testCases := []struct {
		line     string
		expected bool
	}{
		{"|---|---|", true},
		{"| :--- | ---: |", true},
		{"| :---: | --- |", true},
		{"|--|", true},
		{"| Normal | Row |", false},
		{"No pipes", false},
		{"|abc|", false}, // Non-separator chars
	}

	for _, tc := range testCases {
		result := IsTableSeparator(tc.line)
		if result != tc.expected {
			t.Errorf("IsTableSeparator(%q) = %v, expected %v", tc.line, result, tc.expected)
		}
	}
}

func TestGetTableCells(t *testing.T) {
	testCases := []struct {
		line     string
		expected []string
	}{
		{"| Col 1 | Col 2 |", []string{"Col 1", "Col 2"}},
		{"|A|B|C|", []string{"A", "B", "C"}},
		{"A | B", []string{"A", "B"}},
		{"| Single |", []string{"Single"}},
	}

	for _, tc := range testCases {
		result := GetTableCells(tc.line)
		if len(result) != len(tc.expected) {
			t.Errorf("GetTableCells(%q) returned %d cells, expected %d", tc.line, len(result), len(tc.expected))
			continue
		}

		for i, cell := range result {
			if cell != tc.expected[i] {
				t.Errorf("GetTableCells(%q)[%d] = %q, expected %q", tc.line, i, cell, tc.expected[i])
			}
		}
	}
}

func TestValidateLineLength(t *testing.T) {
	testCases := []struct {
		line        string
		maxLength   int
		excludeCode bool
		expected    bool
	}{
		{"Short line", 20, false, true},
		{"This is a very long line that exceeds the limit", 20, false, false},
		{"    code block that is long", 20, true, true},   // Excluded
		{"    code block that is long", 20, false, false}, // Not excluded
		{"```long code fence```", 15, true, true},         // Excluded
		{"```long code fence```", 15, false, false},       // Not excluded
	}

	for _, tc := range testCases {
		result := ValidateLineLength(tc.line, tc.maxLength, tc.excludeCode)
		if result != tc.expected {
			t.Errorf("ValidateLineLength(%q, %d, %v) = %v, expected %v",
				tc.line, tc.maxLength, tc.excludeCode, result, tc.expected)
		}
	}
}

func TestIsURL(t *testing.T) {
	testCases := []struct {
		text     string
		expected bool
	}{
		{"https://example.com", true},
		{"http://test.org", true},
		{"ftp://files.com", false}, // Only http/https
		{"not a url", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := IsURL(tc.text)
		if result != tc.expected {
			t.Errorf("IsURL(%q) = %v, expected %v", tc.text, result, tc.expected)
		}
	}
}

func TestIsEmail(t *testing.T) {
	testCases := []struct {
		text     string
		expected bool
	}{
		{"test@example.com", true},
		{"user.name+tag@domain.co.uk", true},
		{"invalid.email", false},
		{"@domain.com", false},
		{"user@", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := IsEmail(tc.text)
		if result != tc.expected {
			t.Errorf("IsEmail(%q) = %v, expected %v", tc.text, result, tc.expected)
		}
	}
}

func TestCreateFixHelpers(t *testing.T) {
	// Test line replacement
	lineReplacement := CreateLineReplacement(5, "New content")
	if lineReplacement.LineNumber.Unwrap() != 5 {
		t.Error("line replacement should have correct line number")
	}

	// Test text insertion
	insertion := CreateTextInsertion(10, 5, "inserted text")
	if insertion.LineNumber.Unwrap() != 10 || insertion.EditColumn.Unwrap() != 5 {
		t.Error("text insertion should have correct position")
	}

	// Test text deletion
	deletion := CreateTextDeletion(3, 7, 10)
	if deletion.LineNumber.Unwrap() != 3 || deletion.EditColumn.Unwrap() != 7 || deletion.DeleteLength.Unwrap() != 10 {
		t.Error("text deletion should have correct parameters")
	}

	// Test text replacement
	replacement := CreateTextReplacement(1, 1, 5, "replacement")
	if replacement.LineNumber.Unwrap() != 1 || replacement.DeleteLength.Unwrap() != 5 {
		t.Error("text replacement should have correct parameters")
	}
}

func TestMathUtilities(t *testing.T) {
	// Test Min
	if Min(5, 3) != 3 {
		t.Error("Min(5, 3) should be 3")
	}

	if Min(1, 1) != 1 {
		t.Error("Min(1, 1) should be 1")
	}

	// Test Max
	if Max(5, 3) != 5 {
		t.Error("Max(5, 3) should be 5")
	}

	if Max(1, 1) != 1 {
		t.Error("Max(1, 1) should be 1")
	}

	// Test Clamp
	if Clamp(5, 1, 10) != 5 {
		t.Error("Clamp(5, 1, 10) should be 5")
	}

	if Clamp(-1, 1, 10) != 1 {
		t.Error("Clamp(-1, 1, 10) should be 1")
	}

	if Clamp(15, 1, 10) != 10 {
		t.Error("Clamp(15, 1, 10) should be 10")
	}
}

func TestStringUtilities(t *testing.T) {
	// Test NormalizeWhitespace
	result := NormalizeWhitespace("  multiple   spaces  ")
	expected := "multiple spaces"
	if result != expected {
		t.Errorf("NormalizeWhitespace result %q, expected %q", result, expected)
	}

	// Test SplitLines with different line endings
	text := "line1\nline2\r\nline3\rline4"
	lines := SplitLines(text)
	if len(lines) != 4 {
		t.Errorf("SplitLines should return 4 lines, got %d", len(lines))
	}

	// Test JoinLines
	joined := JoinLines([]string{"a", "b", "c"}, "\n")
	if joined != "a\nb\nc" {
		t.Errorf("JoinLines result %q, expected %q", joined, "a\nb\nc")
	}
}
