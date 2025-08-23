package utils

import (
	"fmt"
	"reflect"
)

// DeepMergeConfig performs deep merging of configuration maps.
// Later configs override earlier configs. Arrays are replaced, not merged.
// This is designed specifically for configuration merging where later values
// should completely override earlier values.
func DeepMergeConfig(configs ...map[string]interface{}) map[string]interface{} {
	if len(configs) == 0 {
		return make(map[string]interface{})
	}

	result := make(map[string]interface{})

	for _, config := range configs {
		if config == nil {
			continue
		}
		result = mergeMap(result, config)
	}

	return result
}

// mergeMap recursively merges two maps, with values from 'override' taking precedence.
func mergeMap(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy all values from base
	for key, value := range base {
		result[key] = deepCopyValue(value)
	}

	// Merge/override with values from override map
	for key, overrideValue := range override {
		if baseValue, exists := result[key]; exists {
			// Both base and override have this key
			if isMap(baseValue) && isMap(overrideValue) {
				// Both are maps - merge recursively
				baseMap := baseValue.(map[string]interface{})
				overrideMap := overrideValue.(map[string]interface{})
				result[key] = mergeMap(baseMap, overrideMap)
			} else {
				// At least one is not a map - override completely
				result[key] = deepCopyValue(overrideValue)
			}
		} else {
			// Key only exists in override
			result[key] = deepCopyValue(overrideValue)
		}
	}

	return result
}

// isMap checks if a value is a map[string]interface{}
func isMap(value interface{}) bool {
	_, ok := value.(map[string]interface{})
	return ok
}

// deepCopyValue creates a deep copy of a configuration value
func deepCopyValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case map[string]interface{}:
		copy := make(map[string]interface{})
		for key, val := range v {
			copy[key] = deepCopyValue(val)
		}
		return copy
	case []interface{}:
		copy := make([]interface{}, len(v))
		for i, val := range v {
			copy[i] = deepCopyValue(val)
		}
		return copy
	case string, int, int64, float64, bool:
		// Primitive types are immutable in Go, safe to return directly
		return v
	default:
		// For other types, use reflection for a generic deep copy
		return deepCopyWithReflection(value)
	}
}

// deepCopyWithReflection provides a fallback deep copy using reflection
func deepCopyWithReflection(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	original := reflect.ValueOf(value)
	copy := reflect.New(original.Type()).Elem()

	deepCopyReflectValue(original, copy)
	return copy.Interface()
}

// deepCopyReflectValue recursively copies reflect values
func deepCopyReflectValue(original, copy reflect.Value) {
	switch original.Kind() {
	case reflect.Ptr:
		if !original.IsNil() {
			copy.Set(reflect.New(original.Elem().Type()))
			deepCopyReflectValue(original.Elem(), copy.Elem())
		}
	case reflect.Interface:
		if !original.IsNil() {
			originalValue := original.Elem()
			copyValue := reflect.New(originalValue.Type()).Elem()
			deepCopyReflectValue(originalValue, copyValue)
			copy.Set(copyValue)
		}
	case reflect.Struct:
		for i := 0; i < original.NumField(); i++ {
			if copy.Field(i).CanSet() {
				deepCopyReflectValue(original.Field(i), copy.Field(i))
			}
		}
	case reflect.Slice:
		if !original.IsNil() {
			copy.Set(reflect.MakeSlice(original.Type(), original.Len(), original.Cap()))
			for i := 0; i < original.Len(); i++ {
				deepCopyReflectValue(original.Index(i), copy.Index(i))
			}
		}
	case reflect.Map:
		if !original.IsNil() {
			copy.Set(reflect.MakeMap(original.Type()))
			for _, key := range original.MapKeys() {
				originalValue := original.MapIndex(key)
				copyValue := reflect.New(originalValue.Type()).Elem()
				deepCopyReflectValue(originalValue, copyValue)
				copy.SetMapIndex(key, copyValue)
			}
		}
	default:
		copy.Set(original)
	}
}

// ConfigurationMerger provides utilities for configuration merging with metadata
type ConfigurationMerger struct {
	sources []ConfigSource
}

// ConfigSource represents a single configuration source with metadata
type ConfigSource struct {
	Config   map[string]interface{}
	Path     string
	Type     ConfigSourceType
	Priority int // Higher priority overrides lower priority
}

// ConfigSourceType represents the type/origin of a configuration source
type ConfigSourceType string

const (
	ConfigSourceSystem  ConfigSourceType = "system"
	ConfigSourceUser    ConfigSourceType = "user"
	ConfigSourceProject ConfigSourceType = "project"
	ConfigSourceCLI     ConfigSourceType = "cli"
)

// NewConfigurationMerger creates a new configuration merger
func NewConfigurationMerger() *ConfigurationMerger {
	return &ConfigurationMerger{
		sources: make([]ConfigSource, 0),
	}
}

// AddSource adds a configuration source to the merger
func (cm *ConfigurationMerger) AddSource(config map[string]interface{}, path string, sourceType ConfigSourceType) {
	if config == nil {
		return
	}

	priority := getSourcePriority(sourceType)
	cm.sources = append(cm.sources, ConfigSource{
		Config:   config,
		Path:     path,
		Type:     sourceType,
		Priority: priority,
	})
}

// getSourcePriority returns the priority for different source types
func getSourcePriority(sourceType ConfigSourceType) int {
	switch sourceType {
	case ConfigSourceSystem:
		return 10
	case ConfigSourceUser:
		return 20
	case ConfigSourceProject:
		return 30
	case ConfigSourceCLI:
		return 40
	default:
		return 0
	}
}

// Merge performs the hierarchical merge and returns the final configuration
func (cm *ConfigurationMerger) Merge() map[string]interface{} {
	if len(cm.sources) == 0 {
		return make(map[string]interface{})
	}

	// Sort sources by priority (lowest to highest)
	// This ensures higher priority sources override lower priority ones
	sortedSources := make([]ConfigSource, len(cm.sources))
	copy(sortedSources, cm.sources)

	// Simple insertion sort by priority
	for i := 1; i < len(sortedSources); i++ {
		key := sortedSources[i]
		j := i - 1
		for j >= 0 && sortedSources[j].Priority > key.Priority {
			sortedSources[j+1] = sortedSources[j]
			j--
		}
		sortedSources[j+1] = key
	}

	// Extract configs in priority order
	configs := make([]map[string]interface{}, len(sortedSources))
	for i, source := range sortedSources {
		configs[i] = source.Config
	}

	return DeepMergeConfig(configs...)
}

// GetSources returns all configuration sources with their metadata
func (cm *ConfigurationMerger) GetSources() []ConfigSource {
	return cm.sources
}

// GetSourcePaths returns the paths of all configuration sources in priority order
func (cm *ConfigurationMerger) GetSourcePaths() []string {
	sources := cm.GetSources()
	paths := make([]string, 0, len(sources))

	// Sort by priority for consistent output
	sortedSources := make([]ConfigSource, len(sources))
	copy(sortedSources, sources)

	for i := 1; i < len(sortedSources); i++ {
		key := sortedSources[i]
		j := i - 1
		for j >= 0 && sortedSources[j].Priority > key.Priority {
			sortedSources[j+1] = sortedSources[j]
			j--
		}
		sortedSources[j+1] = key
	}

	for _, source := range sortedSources {
		if source.Path != "" {
			paths = append(paths, fmt.Sprintf("%s (%s)", source.Path, source.Type))
		}
	}

	return paths
}
