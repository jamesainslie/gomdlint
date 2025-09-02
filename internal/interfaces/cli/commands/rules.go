package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gomdlint/gomdlint/internal/app/service"
	"github.com/spf13/cobra"
)

// NewRulesCommand creates the rules command for rule management and information.
func NewRulesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Rule management and information",
		Long:  `Display information about available linting rules, their configuration, and status.`,
	}

	cmd.AddCommand(
		newRulesListCommand(),
		newRulesInfoCommand(),
		newRulesTagsCommand(),
	)

	return cmd
}

func newRulesListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available rules",
		Long:  `Display a list of all available linting rules with their status and descriptions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listRules()
		},
	}

	cmd.Flags().Bool("enabled-only", false, "Show only enabled rules")
	cmd.Flags().Bool("disabled-only", false, "Show only disabled rules")
	cmd.Flags().String("tag", "", "Filter by tag")
	cmd.Flags().Bool("verbose", false, "Show detailed information")

	return cmd
}

func newRulesInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info <rule-name>",
		Short: "Show detailed information about a specific rule",
		Long:  `Display detailed information about a specific rule including its configuration options.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showRuleInfo(args[0])
		},
	}
}

func newRulesTagsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tags",
		Short: "List all rule tags",
		Long:  `Display all available rule tags with the number of rules in each tag.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listTags()
		},
	}
}

func listRules() error {
	// Create a rule engine to get rule information
	ruleEngine, err := service.NewRuleEngine()
	if err != nil {
		return fmt.Errorf("failed to create rule engine: %w", err)
	}

	rules := ruleEngine.GetAllRules()

	// Sort rules by primary name
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].PrimaryName() < rules[j].PrimaryName()
	})

	fmt.Printf("Available Rules (%d total):\n\n", len(rules))

	for _, rule := range rules {
		names := rule.Names()
		primaryName := names[0]
		aliases := ""
		if len(names) > 1 {
			aliases = " (" + strings.Join(names[1:], ", ") + ")"
		}

		enabled := ruleEngine.IsRuleEnabled(primaryName)
		status := ""
		if !enabled {
			status = ""
		}

		fmt.Printf("  %s %s%s\n", status, primaryName, aliases)
		fmt.Printf("    %s\n", rule.Description())

		tags := rule.Tags()
		if len(tags) > 0 {
			fmt.Printf("    Tags: %s\n", strings.Join(tags, ", "))
		}

		fmt.Println()
	}

	return nil
}

func showRuleInfo(ruleName string) error {
	ruleEngine, err := service.NewRuleEngine()
	if err != nil {
		return fmt.Errorf("failed to create rule engine: %w", err)
	}

	ruleOpt := ruleEngine.GetRuleByName(ruleName)
	if ruleOpt.IsNone() {
		return fmt.Errorf("rule '%s' not found", ruleName)
	}

	rule := ruleOpt.Unwrap()

	fmt.Printf("Rule Information: %s\n", rule.PrimaryName())
	fmt.Printf("=================%s\n", strings.Repeat("=", len(rule.PrimaryName())))
	fmt.Println()

	// Names and aliases
	names := rule.Names()
	fmt.Printf("Names: %s\n", strings.Join(names, ", "))

	// Description
	fmt.Printf("Description: %s\n", rule.Description())

	// Tags
	tags := rule.Tags()
	if len(tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(tags, ", "))
	}

	// Parser
	fmt.Printf("Parser: %s\n", rule.Parser())

	// Status
	enabled := ruleEngine.IsRuleEnabled(rule.PrimaryName())
	status := "Enabled"
	if !enabled {
		status = "Disabled"
	}
	fmt.Printf("Status: %s\n", status)

	// Configuration
	config := rule.Config()
	if len(config) > 0 {
		fmt.Println("\nDefault Configuration:")
		for key, value := range config {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	// Documentation URL
	if info := rule.Information(); info != nil {
		fmt.Printf("\nMore Information: %s\n", info.String())
	}

	return nil
}

func listTags() error {
	ruleEngine, err := service.NewRuleEngine()
	if err != nil {
		return fmt.Errorf("failed to create rule engine: %w", err)
	}

	rules := ruleEngine.GetAllRules()

	// Collect all tags and count rules per tag
	tagCounts := make(map[string]int)
	for _, rule := range rules {
		for _, tag := range rule.Tags() {
			tagCounts[tag]++
		}
	}

	// Sort tags alphabetically
	var tags []string
	for tag := range tagCounts {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	fmt.Printf("Available Tags (%d total):\n\n", len(tags))

	for _, tag := range tags {
		count := tagCounts[tag]
		fmt.Printf("  %-20s (%d rules)\n", tag, count)

		// Show sample rules for this tag
		tagRules := ruleEngine.GetRulesByTag(tag)
		if len(tagRules) > 0 {
			var ruleNames []string
			for i, rule := range tagRules {
				if i >= 5 { // Limit to first 5 rules
					ruleNames = append(ruleNames, "...")
					break
				}
				ruleNames = append(ruleNames, rule.PrimaryName())
			}
			fmt.Printf("    Rules: %s\n", strings.Join(ruleNames, ", "))
		}
		fmt.Println()
	}

	return nil
}
