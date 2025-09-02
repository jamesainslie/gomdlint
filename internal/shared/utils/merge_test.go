package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeepMergeConfig_EmptyConfigs(t *testing.T) {
	result := DeepMergeConfig()
	assert.Empty(t, result)
	assert.NotNil(t, result)
}

func TestDeepMergeConfig_SingleConfig(t *testing.T) {
	config := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	result := DeepMergeConfig(config)
	assert.Equal(t, config, result)

	// Verify it's a deep copy, not the same reference
	result["key1"] = "modified"
	assert.Equal(t, "value1", config["key1"])
}

func TestDeepMergeConfig_NilConfigs(t *testing.T) {
	config := map[string]interface{}{
		"key1": "value1",
	}

	result := DeepMergeConfig(nil, config, nil)
	assert.Equal(t, config, result)
}

func TestDeepMergeConfig_SimpleOverride(t *testing.T) {
	config1 := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	config2 := map[string]interface{}{
		"key1": "overridden",
		"key3": true,
	}

	result := DeepMergeConfig(config1, config2)

	expected := map[string]interface{}{
		"key1": "overridden",
		"key2": 42,
		"key3": true,
	}

	assert.Equal(t, expected, result)
}

func TestDeepMergeConfig_NestedMaps(t *testing.T) {
	config1 := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 5432,
			"ssl":  false,
		},
		"logging": map[string]interface{}{
			"level": "info",
		},
	}

	config2 := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "remote.example.com",
			"ssl":  true,
		},
		"logging": map[string]interface{}{
			"format": "json",
		},
	}

	result := DeepMergeConfig(config1, config2)

	expected := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "remote.example.com",
			"port": 5432,
			"ssl":  true,
		},
		"logging": map[string]interface{}{
			"level":  "info",
			"format": "json",
		},
	}

	assert.Equal(t, expected, result)
}

func TestDeepMergeConfig_ArrayReplacement(t *testing.T) {
	config1 := map[string]interface{}{
		"tags":  []string{"tag1", "tag2"},
		"ports": []int{80, 443},
	}

	config2 := map[string]interface{}{
		"tags":  []string{"tag3", "tag4", "tag5"},
		"ports": []int{8080},
	}

	result := DeepMergeConfig(config1, config2)

	expected := map[string]interface{}{
		"tags":  []string{"tag3", "tag4", "tag5"},
		"ports": []int{8080},
	}

	assert.Equal(t, expected, result)
}

func TestDeepMergeConfig_MixedTypes(t *testing.T) {
	config1 := map[string]interface{}{
		"value": "string",
	}

	config2 := map[string]interface{}{
		"value": 42,
	}

	config3 := map[string]interface{}{
		"value": map[string]interface{}{
			"nested": true,
		},
	}

	result := DeepMergeConfig(config1, config2, config3)

	expected := map[string]interface{}{
		"value": map[string]interface{}{
			"nested": true,
		},
	}

	assert.Equal(t, expected, result)
}

func TestDeepMergeConfig_DeepNesting(t *testing.T) {
	config1 := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"value1": "original",
					"value2": "keep",
				},
			},
		},
	}

	config2 := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"value1": "overridden",
					"value3": "new",
				},
			},
		},
	}

	result := DeepMergeConfig(config1, config2)

	expected := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"value1": "overridden",
					"value2": "keep",
					"value3": "new",
				},
			},
		},
	}

	assert.Equal(t, expected, result)
}

func TestDeepMergeConfig_MultipleConfigs(t *testing.T) {
	config1 := map[string]interface{}{
		"a": 1,
		"b": 2,
	}

	config2 := map[string]interface{}{
		"b": 20,
		"c": 3,
	}

	config3 := map[string]interface{}{
		"c": 30,
		"d": 4,
	}

	config4 := map[string]interface{}{
		"d": 40,
		"e": 5,
	}

	result := DeepMergeConfig(config1, config2, config3, config4)

	expected := map[string]interface{}{
		"a": 1,
		"b": 20,
		"c": 30,
		"d": 40,
		"e": 5,
	}

	assert.Equal(t, expected, result)
}

func TestDeepMergeConfig_ComplexStructures(t *testing.T) {
	config1 := map[string]interface{}{
		"server": map[string]interface{}{
			"host": "localhost",
			"port": 8080,
			"tls": map[string]interface{}{
				"enabled": false,
				"cert":    "/path/to/cert",
			},
		},
		"features": []string{"feature1", "feature2"},
		"metadata": map[string]interface{}{
			"version": "1.0.0",
			"build":   "12345",
		},
	}

	config2 := map[string]interface{}{
		"server": map[string]interface{}{
			"port": 9090,
			"tls": map[string]interface{}{
				"enabled": true,
				"key":     "/path/to/key",
			},
			"timeout": 30,
		},
		"features": []string{"feature3", "feature4"},
		"metadata": map[string]interface{}{
			"version": "1.1.0",
			"author":  "test",
		},
	}

	result := DeepMergeConfig(config1, config2)

	expected := map[string]interface{}{
		"server": map[string]interface{}{
			"host": "localhost",
			"port": 9090,
			"tls": map[string]interface{}{
				"enabled": true,
				"cert":    "/path/to/cert",
				"key":     "/path/to/key",
			},
			"timeout": 30,
		},
		"features": []string{"feature3", "feature4"},
		"metadata": map[string]interface{}{
			"version": "1.1.0",
			"build":   "12345",
			"author":  "test",
		},
	}

	assert.Equal(t, expected, result)
}

func TestDeepMergeConfig_PreservesOriginal(t *testing.T) {
	config1 := map[string]interface{}{
		"key1": "value1",
		"nested": map[string]interface{}{
			"subkey": "subvalue",
		},
	}

	config2 := map[string]interface{}{
		"key2": "value2",
		"nested": map[string]interface{}{
			"subkey2": "subvalue2",
		},
	}

	originalConfig1 := map[string]interface{}{
		"key1": "value1",
		"nested": map[string]interface{}{
			"subkey": "subvalue",
		},
	}

	result := DeepMergeConfig(config1, config2)

	// Verify original configs are not modified
	assert.Equal(t, originalConfig1, config1)

	// Modify result and verify originals are still unchanged
	result["key1"] = "modified"
	result["nested"].(map[string]interface{})["subkey"] = "modified"

	assert.Equal(t, "value1", config1["key1"])
	assert.Equal(t, "subvalue", config1["nested"].(map[string]interface{})["subkey"])
}

// Test helper functions
func TestIsMap(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{"string_map", map[string]interface{}{"key": "value"}, true},
		{"string", "not a map", false},
		{"int", 42, false},
		{"slice", []string{"a", "b"}, false},
		{"nil", nil, false},
		{"empty_map", map[string]interface{}{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isMap(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeepCopyValue(t *testing.T) {
	t.Run("primitive_types", func(t *testing.T) {
		tests := []struct {
			name  string
			input interface{}
		}{
			{"string", "hello"},
			{"int", 42},
			{"float", 3.14},
			{"bool", true},
			{"nil", nil},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := deepCopyValue(tt.input)
				assert.Equal(t, tt.input, result)
			})
		}
	})

	t.Run("string_slice", func(t *testing.T) {
		original := []string{"a", "b", "c"}
		copied := deepCopyValue(original).([]string)

		assert.Equal(t, original, copied)

		// Modify copy and verify original is unchanged
		copied[0] = "modified"
		assert.Equal(t, "a", original[0])
	})

	t.Run("int_slice", func(t *testing.T) {
		original := []int{1, 2, 3}
		copied := deepCopyValue(original).([]int)

		assert.Equal(t, original, copied)

		// Modify copy and verify original is unchanged
		copied[0] = 999
		assert.Equal(t, 1, original[0])
	})

	t.Run("interface_slice", func(t *testing.T) {
		original := []interface{}{"string", 42, true}
		copied := deepCopyValue(original).([]interface{})

		assert.Equal(t, original, copied)

		// Modify copy and verify original is unchanged
		copied[0] = "modified"
		assert.Equal(t, "string", original[0])
	})

	t.Run("nested_map", func(t *testing.T) {
		original := map[string]interface{}{
			"key1": "value1",
			"nested": map[string]interface{}{
				"subkey": "subvalue",
			},
		}

		copied := deepCopyValue(original).(map[string]interface{})

		assert.Equal(t, original, copied)

		// Modify copy and verify original is unchanged
		copied["key1"] = "modified"
		copied["nested"].(map[string]interface{})["subkey"] = "modified"

		assert.Equal(t, "value1", original["key1"])
		assert.Equal(t, "subvalue", original["nested"].(map[string]interface{})["subkey"])
	})

	t.Run("unsupported_type", func(t *testing.T) {
		type CustomType struct {
			Field string
		}

		original := CustomType{Field: "test"}
		copied := deepCopyValue(original)

		// Should return the same value for unsupported types
		assert.Equal(t, original, copied)
	})
}

// Benchmark tests
func BenchmarkDeepMergeConfig_Simple(b *testing.B) {
	config1 := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	config2 := map[string]interface{}{
		"key1": "overridden",
		"key4": false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DeepMergeConfig(config1, config2)
	}
}

func BenchmarkDeepMergeConfig_Complex(b *testing.B) {
	config1 := map[string]interface{}{
		"server": map[string]interface{}{
			"host": "localhost",
			"port": 8080,
			"tls": map[string]interface{}{
				"enabled": false,
				"cert":    "/path/to/cert",
			},
		},
		"features": []string{"feature1", "feature2"},
		"metadata": map[string]interface{}{
			"version": "1.0.0",
			"build":   "12345",
		},
	}

	config2 := map[string]interface{}{
		"server": map[string]interface{}{
			"port": 9090,
			"tls": map[string]interface{}{
				"enabled": true,
				"key":     "/path/to/key",
			},
		},
		"features": []string{"feature3"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DeepMergeConfig(config1, config2)
	}
}

func BenchmarkDeepMergeConfig_Multiple(b *testing.B) {
	configs := make([]map[string]interface{}, 5)
	for i := 0; i < 5; i++ {
		configs[i] = map[string]interface{}{
			fmt.Sprintf("key%d", i): fmt.Sprintf("value%d", i),
			"shared":                fmt.Sprintf("shared%d", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DeepMergeConfig(configs...)
	}
}
