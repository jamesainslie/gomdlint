package parser

import (
	"context"
	"strings"
	"testing"
)

func TestParserRegistry_NewParserRegistry(t *testing.T) {
	registry := NewParserRegistry()

	if registry == nil {
		t.Fatal("expected non-nil parser registry")
	}

	parsers := registry.ListParsers()
	if len(parsers) == 0 {
		t.Error("expected some built-in parsers to be registered")
	}

	// Should have at least gomdlint and none parsers
	foundGomdlint := false
	foundNone := false

	for _, name := range parsers {
		switch name {
		case "gomdlint":
			foundGomdlint = true
		case "none":
			foundNone = true
		}
	}

	if !foundGomdlint {
		t.Error("expected gomdlint parser to be registered")
	}

	if !foundNone {
		t.Error("expected none parser to be registered")
	}
}

func TestParserRegistry_GetParser(t *testing.T) {
	registry := NewParserRegistry()

	// Test direct name lookup
	parser, err := registry.GetParser("gomdlint")
	if err != nil {
		t.Errorf("failed to get gomdlint parser: %v", err)
	}

	if parser == nil {
		t.Error("expected non-nil parser")
	}

	if parser.Name() != "gomdlint" {
		t.Errorf("expected parser name 'gomdlint', got %s", parser.Name())
	}

	// Test alias lookup
	parser, err = registry.GetParser("commonmark")
	if err != nil {
		t.Errorf("failed to get parser by alias 'commonmark': %v", err)
	}

	if parser.Name() != "gomdlint" {
		t.Errorf("expected parser name 'gomdlint' for alias 'commonmark', got %s", parser.Name())
	}

	// Test case insensitive lookup
	parser, err = registry.GetParser("GOMDLINT")
	if err != nil {
		t.Errorf("failed to get parser with uppercase name: %v", err)
	}

	// Test non-existent parser
	_, err = registry.GetParser("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent parser")
	}
}

func TestParserRegistry_SetDefaultParser(t *testing.T) {
	registry := NewParserRegistry()

	// Set default to none parser
	err := registry.SetDefaultParser("none")
	if err != nil {
		t.Errorf("failed to set default parser: %v", err)
	}

	defaultParser, err := registry.GetDefaultParser()
	if err != nil {
		t.Errorf("failed to get default parser: %v", err)
	}

	if defaultParser.Name() != "none" {
		t.Errorf("expected default parser 'none', got %s", defaultParser.Name())
	}

	// Test setting invalid default
	err = registry.SetDefaultParser("nonexistent")
	if err == nil {
		t.Error("expected error when setting non-existent default parser")
	}
}

func TestGomdlintParserAdapter_Parse(t *testing.T) {
	parser := NewGomdlintParserAdapter()

	content := `# Test Heading

This is a paragraph with some **bold** text.

- List item 1
- List item 2

> This is a blockquote`

	ctx := context.Background()
	result := parser.Parse(ctx, content, "test.md")

	if result.IsErr() {
		t.Fatalf("parse failed: %v", result.Error())
	}

	parseResult := result.Unwrap()

	if len(parseResult.Tokens) == 0 {
		t.Error("expected some tokens to be generated")
	}

	if len(parseResult.Lines) != 8 {
		t.Errorf("expected 8 lines, got %d", len(parseResult.Lines))
	}

	if parseResult.Metadata["parser"] != "gomdlint" {
		t.Errorf("expected parser metadata 'gomdlint', got %v", parseResult.Metadata["parser"])
	}

	if parseResult.Metadata["filename"] != "test.md" {
		t.Errorf("expected filename metadata 'test.md', got %v", parseResult.Metadata["filename"])
	}
}

func TestGomdlintParserAdapter_FrontMatter(t *testing.T) {
	parser := NewGomdlintParserAdapter()

	content := `---
title: Test Document
author: Test Author
tags: [test, markdown]
---

# Main Content

This is the main content.`

	ctx := context.Background()
	result := parser.Parse(ctx, content, "test.md")

	if result.IsErr() {
		t.Fatalf("parse failed: %v", result.Error())
	}

	parseResult := result.Unwrap()

	if parseResult.FrontMatter.IsNone() {
		t.Error("expected front matter to be detected")
	}

	frontMatter := parseResult.FrontMatter.Unwrap()
	if frontMatter["title"] != "Test Document" {
		t.Errorf("expected title 'Test Document', got %v", frontMatter["title"])
	}

	if frontMatter["author"] != "Test Author" {
		t.Errorf("expected author 'Test Author', got %v", frontMatter["author"])
	}
}

func TestNoneParser_Parse(t *testing.T) {
	parser := NewNoneParser()

	content := `# Test Heading
Some content
Another line`

	ctx := context.Background()
	result := parser.Parse(ctx, content, "test.md")

	if result.IsErr() {
		t.Fatalf("parse failed: %v", result.Error())
	}

	parseResult := result.Unwrap()

	// None parser should create text tokens for each non-empty line
	if len(parseResult.Tokens) != 3 {
		t.Errorf("expected 3 tokens, got %d", len(parseResult.Tokens))
	}

	// All tokens should be text tokens
	for i, token := range parseResult.Tokens {
		if !token.IsText() {
			t.Errorf("token %d should be text token, got %s", i, token.Type)
		}

		lineNum, exists := token.GetIntProperty("line_number")
		if !exists {
			t.Errorf("token %d should have line_number property", i)
		}

		if lineNum != i+1 {
			t.Errorf("token %d should have line_number %d, got %d", i, i+1, lineNum)
		}
	}

	if parseResult.Metadata["parser"] != "none" {
		t.Errorf("expected parser metadata 'none', got %v", parseResult.Metadata["parser"])
	}
}

func TestMultiParserService_NewMultiParserService(t *testing.T) {
	service := NewMultiParserService()

	if service == nil {
		t.Fatal("expected non-nil multi-parser service")
	}

	if service.registry == nil {
		t.Error("expected registry to be initialized")
	}

	if service.defaultParser == nil {
		t.Error("expected default parser to be set")
	}

	parsers := service.GetAvailableParsers()
	if len(parsers) == 0 {
		t.Error("expected some parsers to be available")
	}
}

func TestMultiParserService_ParseDocument(t *testing.T) {
	service := NewMultiParserService()

	content := `# Test Heading

This is a test document.`

	ctx := context.Background()
	result := service.ParseDocument(ctx, content, "test.md")

	if result.IsErr() {
		t.Fatalf("parse failed: %v", result.Error())
	}

	tokens := result.Unwrap()

	if len(tokens) == 0 {
		t.Error("expected some tokens to be generated")
	}
}

func TestMultiParserService_ParseDocumentWithParser(t *testing.T) {
	service := NewMultiParserService()

	content := `# Test Heading
Some content`

	ctx := context.Background()

	// Test with none parser
	result := service.ParseDocumentWithParser(ctx, content, "test.md", "none")
	if result.IsErr() {
		t.Fatalf("parse with none parser failed: %v", result.Error())
	}

	parseResult := result.Unwrap()
	if parseResult.Metadata["parser"] != "none" {
		t.Errorf("expected parser metadata 'none', got %v", parseResult.Metadata["parser"])
	}

	// Test with gomdlint parser
	result = service.ParseDocumentWithParser(ctx, content, "test.md", "gomdlint")
	if result.IsErr() {
		t.Fatalf("parse with gomdlint parser failed: %v", result.Error())
	}

	parseResult = result.Unwrap()
	if parseResult.Metadata["parser"] != "gomdlint" {
		t.Errorf("expected parser metadata 'gomdlint', got %v", parseResult.Metadata["parser"])
	}

	// Test with non-existent parser (should fallback to default)
	result = service.ParseDocumentWithParser(ctx, content, "test.md", "nonexistent")
	if result.IsErr() {
		t.Fatalf("parse with fallback failed: %v", result.Error())
	}
}

func TestMultiParserService_SetRuleParserMapping(t *testing.T) {
	service := NewMultiParserService()

	// Set rule to use none parser
	err := service.SetRuleParserMapping("MD001", "none")
	if err != nil {
		t.Errorf("failed to set rule parser mapping: %v", err)
	}

	// Verify mapping was set
	config := service.GetConfig()
	if config.RuleParserMapping["MD001"] != "none" {
		t.Errorf("expected rule mapping 'none', got %s", config.RuleParserMapping["MD001"])
	}

	// Test parse document for rule
	content := `# Test`
	ctx := context.Background()

	result := service.ParseDocumentForRule(ctx, content, "test.md", "MD001")
	if result.IsErr() {
		t.Fatalf("parse for rule failed: %v", result.Error())
	}

	parseResult := result.Unwrap()
	if parseResult.Metadata["parser"] != "none" {
		t.Errorf("expected parser 'none' for MD001 rule, got %v", parseResult.Metadata["parser"])
	}

	// Test invalid parser mapping
	err = service.SetRuleParserMapping("MD002", "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent parser")
	}
}

func TestMultiParserService_Caching(t *testing.T) {
	service := NewMultiParserService()

	content := `# Test Heading
Some content`

	ctx := context.Background()

	// Parse document twice
	result1 := service.ParseDocumentWithParser(ctx, content, "test.md", "none")
	if result1.IsErr() {
		t.Fatalf("first parse failed: %v", result1.Error())
	}

	result2 := service.ParseDocumentWithParser(ctx, content, "test.md", "none")
	if result2.IsErr() {
		t.Fatalf("second parse failed: %v", result2.Error())
	}

	// Results should be equivalent (from cache)
	parseResult1 := result1.Unwrap()
	parseResult2 := result2.Unwrap()

	if len(parseResult1.Tokens) != len(parseResult2.Tokens) {
		t.Error("cached result should have same number of tokens")
	}

	// Check cache stats
	stats := service.GetCacheStats()
	if !stats["enabled"].(bool) {
		t.Error("cache should be enabled")
	}

	if stats["size"].(int) == 0 {
		t.Error("cache should contain entries")
	}
}

func TestMultiParserService_ConfigureParser(t *testing.T) {
	service := NewMultiParserService()

	config := DefaultParserConfig()
	config.EnableTables = false
	config.EnableFootnotes = true

	err := service.ConfigureParser("none", config)
	if err != nil {
		t.Errorf("failed to configure parser: %v", err)
	}

	// Verify configuration was stored
	serviceConfig := service.GetConfig()
	storedConfig, exists := serviceConfig.ParserConfigs["none"]
	if !exists {
		t.Error("parser configuration should be stored")
	}

	if storedConfig.EnableTables != false {
		t.Error("EnableTables should be false")
	}

	if storedConfig.EnableFootnotes != true {
		t.Error("EnableFootnotes should be true")
	}

	// Test configuring non-existent parser
	err = service.ConfigureParser("nonexistent", config)
	if err == nil {
		t.Error("expected error when configuring non-existent parser")
	}
}

func TestParserCapabilities(t *testing.T) {
	parsers := []Parser{
		NewGomdlintParserAdapter(),
		NewGoldmarkParser(),
		NewBlackfridayParser(),
		NewNoneParser(),
	}

	for _, parser := range parsers {
		t.Run(parser.Name(), func(t *testing.T) {
			// Test basic metadata
			if parser.Name() == "" {
				t.Error("parser should have a name")
			}

			if parser.Version() == "" {
				t.Error("parser should have a version")
			}

			extensions := parser.SupportedExtensions()
			if len(extensions) == 0 {
				t.Error("parser should support some extensions")
			}

			// Verify .md extension is supported
			foundMd := false
			for _, ext := range extensions {
				if strings.ToLower(ext) == ".md" {
					foundMd = true
					break
				}
			}
			if !foundMd {
				t.Error("parser should support .md extension")
			}

			// Test configuration
			config := parser.GetConfig()
			err := parser.Configure(config)
			if err != nil {
				t.Errorf("parser should accept its own configuration: %v", err)
			}

			// Test parsing simple content
			content := "# Test\nSimple content"
			ctx := context.Background()

			result := parser.Parse(ctx, content, "test.md")
			if result.IsErr() {
				t.Errorf("parser should handle simple content: %v", result.Error())
			}

			parseResult := result.Unwrap()
			if len(parseResult.Tokens) == 0 {
				t.Error("parser should generate tokens")
			}

			if len(parseResult.Lines) == 0 {
				t.Error("parser should return lines")
			}

			if parseResult.Metadata == nil {
				t.Error("parser should provide metadata")
			}
		})
	}
}
