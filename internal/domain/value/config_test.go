package value

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test scenarios for configuration value objects following club/ standards
type configTestScenario struct {
	name        string
	config      map[string]interface{}
	expectValid bool
	expectError bool
}

func TestNewLintOptions(t *testing.T) {
	opts := NewLintOptions()

	assert.NotNil(t, opts, "LintOptions should not be nil")
	assert.NotNil(t, opts.Files, "Files should be initialized")
	assert.NotNil(t, opts.Strings, "Strings should be initialized")
	assert.NotNil(t, opts.Config, "Config should be initialized")
	assert.False(t, opts.NoInlineConfig, "NoInlineConfig should default to false")
	assert.Equal(t, 3, opts.ResultVersion, "ResultVersion should default to 3")
	assert.True(t, opts.HandleRuleFailures, "HandleRuleFailures should default to true")
}

func TestLintOptions_WithMethods(t *testing.T) {
	baseOpts := NewLintOptions()

	t.Run("WithFiles", func(t *testing.T) {
		files := []string{"file1.md", "file2.md"}
		opts := baseOpts.WithFiles(files)

		assert.Equal(t, files, opts.Files)
		assert.NotEqual(t, baseOpts, opts, "Should return new instance (immutable)")
		assert.Empty(t, baseOpts.Files, "Original should remain unchanged")
	})

	t.Run("WithStrings", func(t *testing.T) {
		strings := map[string]string{
			"test1": "# Title 1",
			"test2": "# Title 2",
		}
		opts := baseOpts.WithStrings(strings)

		assert.Equal(t, strings, opts.Strings)
	})

	t.Run("WithConfig", func(t *testing.T) {
		config := map[string]interface{}{
			"MD001": false,
			"MD013": map[string]interface{}{
				"line_length": 120,
			},
		}
		opts := baseOpts.WithConfig(config)

		assert.Equal(t, config, opts.Config)
	})

	t.Run("WithNoInlineConfig", func(t *testing.T) {
		opts := baseOpts.WithNoInlineConfig(true)

		assert.True(t, opts.NoInlineConfig)
	})

	t.Run("WithResultVersion", func(t *testing.T) {
		opts := baseOpts.WithResultVersion(2)

		assert.Equal(t, 2, opts.ResultVersion)
	})

	t.Run("WithHandleRuleFailures", func(t *testing.T) {
		opts := baseOpts.WithHandleRuleFailures(false)

		assert.False(t, opts.HandleRuleFailures)
	})

	t.Run("method chaining", func(t *testing.T) {
		opts := NewLintOptions().
			WithFiles([]string{"test.md"}).
			WithConfig(map[string]interface{}{"MD001": false}).
			WithNoInlineConfig(true).
			WithResultVersion(2)

		assert.Equal(t, []string{"test.md"}, opts.Files)
		assert.Equal(t, false, opts.Config["MD001"])
		assert.True(t, opts.NoInlineConfig)
		assert.Equal(t, 2, opts.ResultVersion)
	})
}

func TestLintOptions_Validation(t *testing.T) {
	scenarios := []configTestScenario{
		{
			name:        "valid basic config",
			config:      map[string]interface{}{"MD001": false},
			expectValid: true,
			expectError: false,
		},
		{
			name: "valid complex config",
			config: map[string]interface{}{
				"MD001": true,
				"MD013": map[string]interface{}{
					"line_length": 80,
					"tables":      false,
				},
				"MD041": false,
			},
			expectValid: true,
			expectError: false,
		},
		{
			name:        "empty config",
			config:      map[string]interface{}{},
			expectValid: true,
			expectError: false,
		},
		{
			name:        "nil config",
			config:      nil,
			expectValid: true,
			expectError: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			opts := NewLintOptions().WithConfig(scenario.config)

			// Basic validation - config should be set
			if scenario.config != nil {
				assert.Equal(t, scenario.config, opts.Config)
			}

			// Additional validation could be added here
			// For now, we just verify the config is accepted
			assert.NotNil(t, opts)
		})
	}
}

func TestLintOptions_EdgeCases(t *testing.T) {
	t.Run("nil files slice", func(t *testing.T) {
		opts := NewLintOptions().WithFiles(nil)
		assert.NotNil(t, opts.Files, "Should initialize empty slice for safety")
		assert.Empty(t, opts.Files, "Should be empty when given nil")
	})

	t.Run("nil strings map", func(t *testing.T) {
		opts := NewLintOptions().WithStrings(nil)
		assert.NotNil(t, opts.Strings, "Should initialize empty map for safety")
		assert.Empty(t, opts.Strings, "Should be empty when given nil")
	})

	t.Run("empty files slice", func(t *testing.T) {
		opts := NewLintOptions().WithFiles([]string{})
		assert.Equal(t, []string{}, opts.Files)
	})

	t.Run("empty strings map", func(t *testing.T) {
		opts := NewLintOptions().WithStrings(map[string]string{})
		assert.Equal(t, map[string]string{}, opts.Strings)
	})

	t.Run("invalid result version", func(t *testing.T) {
		opts := NewLintOptions().WithResultVersion(-1)
		assert.Equal(t, -1, opts.ResultVersion) // Should accept any value
	})

	t.Run("large files list", func(t *testing.T) {
		files := make([]string, 1000)
		for i := 0; i < 1000; i++ {
			files[i] = fmt.Sprintf("file%d.md", i)
		}

		opts := NewLintOptions().WithFiles(files)
		assert.Len(t, opts.Files, 1000)
	})

	t.Run("complex config values", func(t *testing.T) {
		config := map[string]interface{}{
			"string_value": "test",
			"int_value":    42,
			"float_value":  3.14,
			"bool_value":   true,
			"array_value":  []interface{}{"a", "b", "c"},
			"nested_map": map[string]interface{}{
				"inner_key": "inner_value",
			},
		}

		opts := NewLintOptions().WithConfig(config)
		assert.Equal(t, config, opts.Config)
	})
}

func TestLintOptions_Copy(t *testing.T) {
	original := NewLintOptions().
		WithFiles([]string{"file1.md", "file2.md"}).
		WithStrings(map[string]string{"test": "# Title"}).
		WithConfig(map[string]interface{}{"MD001": false}).
		WithNoInlineConfig(true)

	// Create a new options based on original
	copy := NewLintOptions().
		WithFiles(original.Files).
		WithStrings(original.Strings).
		WithConfig(original.Config).
		WithNoInlineConfig(original.NoInlineConfig)

	// Should have same values
	assert.Equal(t, original.Files, copy.Files)
	assert.Equal(t, original.Strings, copy.Strings)
	assert.Equal(t, original.Config, copy.Config)
	assert.Equal(t, original.NoInlineConfig, copy.NoInlineConfig)

	// Modifying copy should not affect original
	copy.Files[0] = "modified.md"
	assert.NotEqual(t, original.Files[0], copy.Files[0])
}

func TestLintOptions_ConfigMerging(t *testing.T) {
	base := NewLintOptions().WithConfig(map[string]interface{}{
		"MD001": true,
		"MD013": map[string]interface{}{
			"line_length": 80,
		},
	})

	// Add additional config
	additional := map[string]interface{}{
		"MD018": false,
		"MD013": map[string]interface{}{
			"line_length": 120,
			"tables":      false,
		},
	}

	// Merge configs (simplified - real implementation would handle deep merging)
	merged := make(map[string]interface{})
	for k, v := range base.Config {
		merged[k] = v
	}
	for k, v := range additional {
		merged[k] = v
	}

	result := base.WithConfig(merged)

	assert.Equal(t, false, result.Config["MD018"])
	assert.Equal(t, true, result.Config["MD001"])

	// MD013 should be overwritten
	md013, ok := result.Config["MD013"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 120, md013["line_length"])
	assert.Equal(t, false, md013["tables"])
}

// Benchmark tests
func BenchmarkNewLintOptions(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		opts := NewLintOptions()
		_ = opts
	}
}

func BenchmarkLintOptions_WithFiles(b *testing.B) {
	files := make([]string, 100)
	for i := 0; i < 100; i++ {
		files[i] = fmt.Sprintf("file%d.md", i)
	}

	opts := NewLintOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := opts.WithFiles(files)
		_ = result
	}
}

func BenchmarkLintOptions_WithConfig(b *testing.B) {
	config := map[string]interface{}{
		"MD001": true,
		"MD002": false,
		"MD013": map[string]interface{}{
			"line_length": 120,
			"tables":      false,
		},
	}

	opts := NewLintOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := opts.WithConfig(config)
		_ = result
	}
}
