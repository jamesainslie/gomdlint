package parser

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MultiParserService provides multi-parser functionality
type MultiParserService struct {
	registry      *ParserRegistry
	defaultParser Parser
	cache         map[string]ParseResult
	cacheMutex    sync.RWMutex
	config        MultiParserConfig
}

// MultiParserConfig configures the multi-parser service
type MultiParserConfig struct {
	EnableCaching     bool                    `json:"enable_caching"`
	MaxCacheSize      int                     `json:"max_cache_size"`
	DefaultParser     string                  `json:"default_parser"`
	ParserConfigs     map[string]ParserConfig `json:"parser_configs"`
	RuleParserMapping map[string]string       `json:"rule_parser_mapping"`
}

// NewMultiParserService creates a new multi-parser service
func NewMultiParserService() *MultiParserService {
	registry := NewParserRegistry()

	defaultParser, err := registry.GetDefaultParser()
	if err != nil {
		// Fallback to none parser if default fails
		defaultParser = NewNoneParser()
	}

	return &MultiParserService{
		registry:      registry,
		defaultParser: defaultParser,
		cache:         make(map[string]ParseResult),
		config: MultiParserConfig{
			EnableCaching:     true,
			MaxCacheSize:      1000,
			DefaultParser:     "gomdlint",
			ParserConfigs:     make(map[string]ParserConfig),
			RuleParserMapping: make(map[string]string),
		},
	}
}

// ParseDocument parses content using the default parser
func (mps *MultiParserService) ParseDocument(ctx context.Context, content string, filename string) functional.Result[[]value.Token] {
	result := mps.ParseDocumentWithParser(ctx, content, filename, "")
	if result.IsErr() {
		return functional.Err[[]value.Token](result.Error())
	}

	parseResult := result.Unwrap()
	return functional.Ok(parseResult.Tokens)
}

// ParseDocumentWithParser parses content using a specific parser
func (mps *MultiParserService) ParseDocumentWithParser(ctx context.Context, content string, filename string, parserName string) functional.Result[ParseResult] {
	// Use default parser if none specified
	if parserName == "" {
		parserName = mps.config.DefaultParser
	}

	// Check cache first
	if mps.config.EnableCaching {
		cacheKey := fmt.Sprintf("%s:%s:%s", parserName, filename, hashContent(content))
		if cached, exists := mps.getCachedResult(cacheKey); exists {
			return functional.Ok(cached)
		}
	}

	// Get parser
	parser, err := mps.registry.GetParser(parserName)
	if err != nil {
		// Fallback to default parser
		parser = mps.defaultParser
	}

	// Configure parser if needed
	if config, exists := mps.config.ParserConfigs[parser.Name()]; exists {
		if err := parser.Configure(config); err != nil {
			return functional.Err[ParseResult](fmt.Errorf("failed to configure parser %s: %w", parser.Name(), err))
		}
	}

	// Parse content
	result := parser.Parse(ctx, content, filename)
	if result.IsErr() {
		return result
	}

	// Cache result
	if mps.config.EnableCaching {
		cacheKey := fmt.Sprintf("%s:%s:%s", parser.Name(), filename, hashContent(content))
		mps.cacheResult(cacheKey, result.Unwrap())
	}

	return result
}

// ParseDocumentForRule parses content using the parser configured for a specific rule
func (mps *MultiParserService) ParseDocumentForRule(ctx context.Context, content string, filename string, ruleName string) functional.Result[ParseResult] {
	// Check if rule has a specific parser mapping
	parserName := mps.config.DefaultParser
	if mappedParser, exists := mps.config.RuleParserMapping[ruleName]; exists {
		parserName = mappedParser
	}

	return mps.ParseDocumentWithParser(ctx, content, filename, parserName)
}

// ParseReader parses content from a reader using the default parser
func (mps *MultiParserService) ParseReader(ctx context.Context, reader io.Reader, filename string) functional.Result[ParseResult] {
	return mps.ParseReaderWithParser(ctx, reader, filename, "")
}

// ParseReaderWithParser parses content from a reader using a specific parser
func (mps *MultiParserService) ParseReaderWithParser(ctx context.Context, reader io.Reader, filename string, parserName string) functional.Result[ParseResult] {
	// Use default parser if none specified
	if parserName == "" {
		parserName = mps.config.DefaultParser
	}

	// Get parser
	parser, err := mps.registry.GetParser(parserName)
	if err != nil {
		parser = mps.defaultParser
	}

	// Configure parser if needed
	if config, exists := mps.config.ParserConfigs[parser.Name()]; exists {
		if err := parser.Configure(config); err != nil {
			return functional.Err[ParseResult](fmt.Errorf("failed to configure parser %s: %w", parser.Name(), err))
		}
	}

	return parser.ParseReader(ctx, reader, filename)
}

// GetAvailableParsers returns information about all available parsers
func (mps *MultiParserService) GetAvailableParsers() []ParserInfo {
	return mps.registry.GetParserInfo()
}

// SetDefaultParser sets the default parser
func (mps *MultiParserService) SetDefaultParser(parserName string) error {
	parser, err := mps.registry.GetParser(parserName)
	if err != nil {
		return err
	}

	mps.defaultParser = parser
	mps.config.DefaultParser = parserName
	return nil
}

// ConfigureParser configures a specific parser
func (mps *MultiParserService) ConfigureParser(parserName string, config ParserConfig) error {
	parser, err := mps.registry.GetParser(parserName)
	if err != nil {
		return err
	}

	if err := parser.Configure(config); err != nil {
		return err
	}

	mps.config.ParserConfigs[parserName] = config
	return nil
}

// SetRuleParserMapping sets parser mapping for specific rules
func (mps *MultiParserService) SetRuleParserMapping(ruleName, parserName string) error {
	// Verify parser exists
	_, err := mps.registry.GetParser(parserName)
	if err != nil {
		return err
	}

	mps.config.RuleParserMapping[ruleName] = parserName
	return nil
}

// GetConfig returns the current configuration
func (mps *MultiParserService) GetConfig() MultiParserConfig {
	return mps.config
}

// UpdateConfig updates the service configuration
func (mps *MultiParserService) UpdateConfig(config MultiParserConfig) error {
	// Validate default parser
	if config.DefaultParser != "" {
		if err := mps.SetDefaultParser(config.DefaultParser); err != nil {
			return fmt.Errorf("invalid default parser: %w", err)
		}
	}

	// Validate rule parser mappings
	for rule, parser := range config.RuleParserMapping {
		if _, err := mps.registry.GetParser(parser); err != nil {
			return fmt.Errorf("invalid parser %s for rule %s: %w", parser, rule, err)
		}
	}

	mps.config = config
	return nil
}

// Cache management methods
func (mps *MultiParserService) getCachedResult(key string) (ParseResult, bool) {
	mps.cacheMutex.RLock()
	defer mps.cacheMutex.RUnlock()

	result, exists := mps.cache[key]
	return result, exists
}

func (mps *MultiParserService) cacheResult(key string, result ParseResult) {
	mps.cacheMutex.Lock()
	defer mps.cacheMutex.Unlock()

	// Simple cache size management
	if len(mps.cache) >= mps.config.MaxCacheSize {
		// Remove oldest entries (simple implementation)
		count := 0
		for k := range mps.cache {
			if count >= mps.config.MaxCacheSize/2 {
				break
			}
			delete(mps.cache, k)
			count++
		}
	}

	mps.cache[key] = result
}

// ClearCache clears the parsing cache
func (mps *MultiParserService) ClearCache() {
	mps.cacheMutex.Lock()
	defer mps.cacheMutex.Unlock()

	mps.cache = make(map[string]ParseResult)
}

// GetCacheStats returns cache statistics
func (mps *MultiParserService) GetCacheStats() map[string]interface{} {
	mps.cacheMutex.RLock()
	defer mps.cacheMutex.RUnlock()

	return map[string]interface{}{
		"enabled":   mps.config.EnableCaching,
		"size":      len(mps.cache),
		"max_size":  mps.config.MaxCacheSize,
		"hit_ratio": "not_implemented", // Would track hits/misses in real implementation
	}
}

// Simple hash function for content (for demo purposes)
func hashContent(content string) string {
	// In real implementation, would use crypto/sha256 or similar
	if len(content) < 32 {
		return content
	}
	return fmt.Sprintf("%d:%s...%s", len(content), content[:16], content[len(content)-16:])
}
