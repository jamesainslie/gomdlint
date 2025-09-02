package parser

import (
	"fmt"
	"sync"
)

// DefaultParserRegistry implements ParserRegistry interface
type DefaultParserRegistry struct {
	parsers map[string]Parser
	mutex   sync.RWMutex
}

// NewParserRegistry creates a new parser registry with built-in parsers
func NewParserRegistry() *DefaultParserRegistry {
	registry := &DefaultParserRegistry{
		parsers: make(map[string]Parser),
	}

	// Register built-in parsers
	registry.registerBuiltInParsers()

	return registry
}

// registerBuiltInParsers registers all built-in parsers
func (pr *DefaultParserRegistry) registerBuiltInParsers() {
	// Register CommonMark parser
	if commonMarkParser := NewCommonMarkParser(); commonMarkParser != nil {
		pr.parsers[commonMarkParser.Name()] = commonMarkParser
	}

	// Register Goldmark parser
	if goldmarkParser := NewGoldmarkParser(); goldmarkParser != nil {
		pr.parsers[goldmarkParser.Name()] = goldmarkParser
	}

	// Register Blackfriday parser
	if blackfridayParser := NewBlackfridayParser(); blackfridayParser != nil {
		pr.parsers[blackfridayParser.Name()] = blackfridayParser
	}

	// Register None parser (passthrough)
	if noneParser := NewNoneParser(); noneParser != nil {
		pr.parsers[noneParser.Name()] = noneParser
	}
}

// RegisterParser registers a new parser
func (pr *DefaultParserRegistry) RegisterParser(parser Parser) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	name := parser.Name()
	if name == "" {
		return fmt.Errorf("parser name cannot be empty")
	}

	if _, exists := pr.parsers[name]; exists {
		return fmt.Errorf("parser %s already registered", name)
	}

	pr.parsers[name] = parser
	return nil
}

// UnregisterParser removes a parser from the registry
func (pr *DefaultParserRegistry) UnregisterParser(name string) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	if _, exists := pr.parsers[name]; !exists {
		return fmt.Errorf("parser %s not found", name)
	}

	delete(pr.parsers, name)
	return nil
}

// GetParser retrieves a parser by name
func (pr *DefaultParserRegistry) GetParser(name string) (Parser, error) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	parser, exists := pr.parsers[name]
	if !exists {
		return nil, fmt.Errorf("parser %s not found", name)
	}

	return parser, nil
}

// ListParsers returns all registered parser names
func (pr *DefaultParserRegistry) ListParsers() []string {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	names := make([]string, 0, len(pr.parsers))
	for name := range pr.parsers {
		names = append(names, name)
	}

	return names
}

// GetParserInfo returns detailed information about a parser
func (pr *DefaultParserRegistry) GetParserInfo(name string) (*ParserInfo, error) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	parser, exists := pr.parsers[name]
	if !exists {
		return nil, fmt.Errorf("parser %s not found", name)
	}

	info := &ParserInfo{
		Name:                parser.Name(),
		Version:             parser.Version(),
		SupportedExtensions: parser.SupportedExtensions(),
		SupportsAsync:       parser.SupportsAsync(),
		SupportsStreaming:   parser.SupportsStreaming(),
		Description:         fmt.Sprintf("%s parser v%s", parser.Name(), parser.Version()),
	}

	return info, nil
}

// GetAllParserInfo returns information about all registered parsers
func (pr *DefaultParserRegistry) GetAllParserInfo() []*ParserInfo {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	infos := make([]*ParserInfo, 0, len(pr.parsers))
	for _, parser := range pr.parsers {
		info := &ParserInfo{
			Name:                parser.Name(),
			Version:             parser.Version(),
			SupportedExtensions: parser.SupportedExtensions(),
			SupportsAsync:       parser.SupportsAsync(),
			SupportsStreaming:   parser.SupportsStreaming(),
			Description:         fmt.Sprintf("%s parser v%s", parser.Name(), parser.Version()),
		}
		infos = append(infos, info)
	}

	return infos
}

// GetDefaultParser returns the default parser (CommonMark)
func (pr *DefaultParserRegistry) GetDefaultParser() Parser {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	// Try CommonMark first
	if parser, exists := pr.parsers["commonmark"]; exists {
		return parser
	}

	// Fallback to first available parser
	for _, parser := range pr.parsers {
		return parser
	}

	return nil
}

// ValidateParser checks if a parser is properly configured
func (pr *DefaultParserRegistry) ValidateParser(name string) error {
	parser, err := pr.GetParser(name)
	if err != nil {
		return err
	}

	// Check if parser supports required extensions
	extensions := parser.SupportedExtensions()
	if len(extensions) == 0 {
		return fmt.Errorf("parser %s does not specify supported extensions", name)
	}

	return nil
}

// GetParserForExtension returns the best parser for a given file extension
func (pr *DefaultParserRegistry) GetParserForExtension(extension string) (Parser, error) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	// Normalize extension
	if extension != "" && extension[0] != '.' {
		extension = "." + extension
	}

	// Find parsers that support this extension
	for _, parser := range pr.parsers {
		for _, supportedExt := range parser.SupportedExtensions() {
			if supportedExt == extension {
				return parser, nil
			}
		}
	}

	return nil, fmt.Errorf("no parser found for extension %s", extension)
}

// globalParserRegistry is the singleton instance
var (
	globalParserRegistry *DefaultParserRegistry
	registryOnce         sync.Once
)

// GetGlobalParserRegistry returns the global parser registry
func GetGlobalParserRegistry() *DefaultParserRegistry {
	registryOnce.Do(func() {
		globalParserRegistry = NewParserRegistry()
	})
	return globalParserRegistry
}