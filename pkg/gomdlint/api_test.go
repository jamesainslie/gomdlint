package gomdlint

import (
	"context"
	"testing"
)

func TestLintString(t *testing.T) {
	ctx := context.Background()

	// Test with valid markdown
	t.Run("ValidMarkdown", func(t *testing.T) {
		content := "# Title\n\nThis is a paragraph.\n"
		result, err := LintString(ctx, content)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if result.TotalFiles != 1 {
			t.Errorf("Expected 1 file, got %d", result.TotalFiles)
		}
	})

	// Test with markdown that has violations
	t.Run("MarkdownWithViolations", func(t *testing.T) {
		content := "#Title\n\nThis is a	tab character.\n"
		result, err := LintString(ctx, content)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result.TotalViolations == 0 {
			t.Error("Expected violations, got none")
		}
	})
}

func TestGetVersion(t *testing.T) {
	version := GetVersion()
	if version == "" {
		t.Error("Version should not be empty")
	}
	if version != Version {
		t.Errorf("Expected version %q, got %q", Version, version)
	}
}

func TestLintResult_String(t *testing.T) {
	result := &LintResult{
		Results:         make(map[string][]Violation),
		TotalViolations: 0,
		TotalFiles:      1,
	}

	// Test empty result
	str := result.String()
	if str != "" {
		t.Errorf("Expected empty string for no violations, got %q", str)
	}

	// Test with violations
	result.Results["test.md"] = []Violation{
		{
			LineNumber:      5,
			RuleNames:       []string{"MD001", "heading-increment"},
			RuleDescription: "Heading levels should only increment by one level at a time",
			ErrorDetail:     "Expected h2, found h3",
		},
	}
	result.TotalViolations = 1

	str = result.String()
	if str == "" {
		t.Error("Expected non-empty string for violations")
	}

	// Should contain file name, line number, and rule
	if !contains(str, "test.md") {
		t.Error("Result string should contain filename")
	}
	if !contains(str, "5") {
		t.Error("Result string should contain line number")
	}
	if !contains(str, "heading-increment") {
		t.Error("Result string should contain rule name")
	}
}

func TestLintOptions_Validation(t *testing.T) {
	ctx := context.Background()

	// Test with no files or strings
	t.Run("NoInput", func(t *testing.T) {
		options := LintOptions{}
		result, err := Lint(ctx, options)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result.TotalFiles != 0 {
			t.Errorf("Expected 0 files, got %d", result.TotalFiles)
		}
	})

	// Test with custom configuration
	t.Run("CustomConfig", func(t *testing.T) {
		options := LintOptions{
			Strings: map[string]string{"test": "# Title\n"},
			Config: map[string]interface{}{
				"MD041": false, // Disable first line heading rule
			},
		}

		_, err := Lint(ctx, options)
		if err != nil {
			t.Fatalf("Expected no error with custom config, got: %v", err)
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInner(s, substr)))
}

func containsInner(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
