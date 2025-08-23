package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gomdlint/gomdlint/pkg/gomdlint"
	"github.com/spf13/cobra"
)

// NewTUICommand creates the TUI command for interactive markdown linting.
func NewTUICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui [files...]",
		Short: "Interactive TUI for markdown linting",
		Long: `Launch an interactive Terminal User Interface (TUI) for markdown linting.
		
The TUI provides a modern, interactive way to:
- Browse and select files to lint
- View violations with detailed information  
- Navigate between files and violations
- Apply auto-fixes interactively
- Configure rules on the fly

Examples:
  gomdlint tui
  gomdlint tui docs/
  gomdlint tui --config .markdownlint.json *.md`,
		Args: cobra.ArbitraryArgs,
		RunE: runTUI,
	}

	return cmd
}

func runTUI(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Parse global flags
	configFile, _ := cmd.Flags().GetString("config")
	noConfig, _ := cmd.Flags().GetBool("no-config")

	// Initialize the TUI model
	model, err := newTUIModel(ctx, args, configFile, noConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize TUI: %w", err)
	}

	// Start the TUI program
	program := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// tuiModel represents the state of the TUI application.
type tuiModel struct {
	// Context and configuration
	ctx    context.Context
	files  []string
	config map[string]interface{}

	// Linting state
	lintResult *gomdlint.LintResult
	linting    bool

	// UI state
	currentView  viewType
	selectedFile int
	selectedViol int
	width        int
	height       int

	// Status and messages
	status     string
	lastUpdate time.Time

	// Styles
	styles *tuiStyles
}

type viewType int

const (
	viewFileList viewType = iota
	viewViolations
	viewDetails
	viewHelp
)

// tuiStyles contains all the styling for the TUI.
type tuiStyles struct {
	header     lipgloss.Style
	title      lipgloss.Style
	subtitle   lipgloss.Style
	status     lipgloss.Style
	selected   lipgloss.Style
	unselected lipgloss.Style
	violation  lipgloss.Style
	error      lipgloss.Style
	warning    lipgloss.Style
	success    lipgloss.Style
	help       lipgloss.Style
	border     lipgloss.Style
}

func newTUIStyles() *tuiStyles {
	return &tuiStyles{
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Background(lipgloss.Color("57")).
			Padding(0, 1),

		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")),

		subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),

		status: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Align(lipgloss.Right),

		selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Background(lipgloss.Color("57")).
			Padding(0, 1),

		unselected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Padding(0, 1),

		violation: lipgloss.NewStyle().
			Padding(0, 2),

		error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),

		warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")),

		success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")),

		help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")),

		border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1),
	}
}

func newTUIModel(ctx context.Context, args []string, configFile string, noConfig bool) (*tuiModel, error) {
	// Collect files
	files, err := collectFiles(args, []string{}, false)
	if err != nil {
		return nil, fmt.Errorf("failed to collect files: %w", err)
	}

	// Load configuration
	var config map[string]interface{}
	var configSource *ConfigurationSource
	if !noConfig {
		// Use shared XDG-aware configuration loading
		configSource, err = loadConfigurationSourceShared(configFile)
		if err != nil && configFile != "" {
			return nil, fmt.Errorf("failed to load configuration: %w", err)
		}
		if !configSource.IsDefault {
			config = configSource.Config
		}
	}

	model := &tuiModel{
		ctx:         ctx,
		files:       files,
		config:      config,
		currentView: viewFileList,
		status:      "Ready",
		lastUpdate:  time.Now(),
		styles:      newTUIStyles(),
	}

	// Store config source info for potential display
	if configSource != nil && !configSource.IsDefault {
		if configSource.IsHierarchy {
			model.status = fmt.Sprintf("Ready (hierarchical config: %d sources)", len(configSource.Sources))
		} else if len(configSource.Sources) > 0 {
			filename := filepath.Base(configSource.Sources[0].Path)
			model.status = fmt.Sprintf("Ready (config: %s)", filename)
		}
	}

	return model, nil
}

// Init implements bubbletea.Model.
func (m *tuiModel) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg { return tickMsg{} }),
		m.performLinting(),
	)
}

// Update implements bubbletea.Model.
func (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case lintCompleteMsg:
		m.linting = false
		m.lintResult = msg.result
		m.status = fmt.Sprintf("Linting complete: %d violations", msg.result.TotalViolations)
		return m, nil

	case lintErrorMsg:
		m.linting = false
		m.status = fmt.Sprintf("Linting error: %v", msg.err)
		return m, nil

	case tickMsg:
		m.lastUpdate = time.Now()
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg{} })
	}

	return m, nil
}

// View implements bubbletea.Model.
func (m *tuiModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	var content string

	switch m.currentView {
	case viewFileList:
		content = m.renderFileList()
	case viewViolations:
		content = m.renderViolations()
	case viewDetails:
		content = m.renderDetails()
	case viewHelp:
		content = m.renderHelp()
	}

	// Render header and status
	header := m.renderHeader()
	status := m.renderStatus()
	help := m.renderQuickHelp()

	// Calculate available space for content
	contentHeight := m.height - lipgloss.Height(header) - lipgloss.Height(status) - lipgloss.Height(help) - 2

	// Ensure content fits in available space
	if lipgloss.Height(content) > contentHeight {
		lines := strings.Split(content, "\n")
		if len(lines) > contentHeight {
			lines = lines[:contentHeight]
		}
		content = strings.Join(lines, "\n")
	}

	// Combine all sections
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		status,
		help,
	)
}

func (m *tuiModel) renderHeader() string {
	title := m.styles.header.Render("gomdlint TUI")
	subtitle := m.styles.subtitle.Render("Interactive Markdown Linter")

	// Add file count and violation summary if available
	info := ""
	if len(m.files) > 0 {
		info = fmt.Sprintf("%d files", len(m.files))
		if m.lintResult != nil {
			info += fmt.Sprintf(" • %d violations", m.lintResult.TotalViolations)
		}
	}

	if info != "" {
		subtitle += " • " + info
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle)
}

func (m *tuiModel) renderStatus() string {
	status := m.status
	if m.linting {
		// Use theme-aware status - will be updated to use theming system
		status = "Linting..."
	}

	timestamp := m.lastUpdate.Format("15:04:05")
	return m.styles.status.Width(m.width).Render(fmt.Sprintf("%s | %s", status, timestamp))
}

func (m *tuiModel) renderQuickHelp() string {
	var help string
	switch m.currentView {
	case viewFileList:
		help = "↑/↓: navigate • enter: view violations • r: re-lint • h: help • q: quit"
	case viewViolations:
		help = "↑/↓: navigate • enter: details • f: fix • b: back • h: help • q: quit"
	case viewDetails:
		help = "b: back • h: help • q: quit"
	case viewHelp:
		help = "b: back • q: quit"
	}

	return m.styles.help.Width(m.width).Render(help)
}

func (m *tuiModel) renderFileList() string {
	if len(m.files) == 0 {
		return m.styles.border.Width(m.width - 2).Render("No markdown files found")
	}

	var items []string
	for i, file := range m.files {
		style := m.styles.unselected
		if i == m.selectedFile {
			style = m.styles.selected
		}

		// Add violation count if available
		violationInfo := ""
		if m.lintResult != nil {
			if violations, exists := m.lintResult.Results[file]; exists {
				count := len(violations)
				if count > 0 {
					violationInfo = fmt.Sprintf(" (%d)", count)
					if count > 0 {
						violationInfo = m.styles.error.Render(violationInfo)
					}
				} else {
					violationInfo = m.styles.success.Render(" ✓")
				}
			}
		}

		items = append(items, style.Render(file+violationInfo))
	}

	content := strings.Join(items, "\n")
	return m.styles.border.Width(m.width - 2).Render(content)
}

func (m *tuiModel) renderViolations() string {
	if m.lintResult == nil || m.selectedFile >= len(m.files) {
		return m.styles.border.Width(m.width - 2).Render("No violations data available")
	}

	fileName := m.files[m.selectedFile]
	violations, exists := m.lintResult.Results[fileName]
	if !exists || len(violations) == 0 {
		return m.styles.border.Width(m.width - 2).Render("No violations found in this file ✓")
	}

	var items []string
	for i, violation := range violations {
		style := m.styles.unselected
		if i == m.selectedViol {
			style = m.styles.selected
		}

		// Format violation display
		ruleName := violation.RuleNames[0]
		if len(violation.RuleNames) > 1 {
			// Use alias if available
			for j := 1; j < len(violation.RuleNames); j++ {
				name := violation.RuleNames[j]
				if len(name) < 3 || name[:2] != "MD" {
					ruleName = name
					break
				}
			}
		}

		line := fmt.Sprintf("Line %d: %s - %s",
			violation.LineNumber, ruleName, violation.RuleDescription)

		if violation.ErrorDetail != "" {
			line += "\n  " + violation.ErrorDetail
		}

		items = append(items, style.Render(line))
	}

	title := m.styles.title.Render(fmt.Sprintf("Violations in %s", fileName))
	content := strings.Join(items, "\n")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		m.styles.border.Width(m.width-2).Render(content),
	)
}

func (m *tuiModel) renderDetails() string {
	if m.lintResult == nil || m.selectedFile >= len(m.files) {
		return m.styles.border.Width(m.width - 2).Render("No violation details available")
	}

	fileName := m.files[m.selectedFile]
	violations, exists := m.lintResult.Results[fileName]
	if !exists || m.selectedViol >= len(violations) {
		return m.styles.border.Width(m.width - 2).Render("No violation selected")
	}

	violation := violations[m.selectedViol]

	var details strings.Builder
	details.WriteString(m.styles.title.Render("Violation Details"))
	details.WriteString("\n\n")

	details.WriteString(fmt.Sprintf("File: %s\n", fileName))
	details.WriteString(fmt.Sprintf("Line: %d\n", violation.LineNumber))
	details.WriteString(fmt.Sprintf("Rule: %s\n", strings.Join(violation.RuleNames, ", ")))
	details.WriteString(fmt.Sprintf("Description: %s\n", violation.RuleDescription))

	if violation.ErrorDetail != "" {
		details.WriteString(fmt.Sprintf("Detail: %s\n", violation.ErrorDetail))
	}

	if violation.ErrorContext != "" {
		details.WriteString(fmt.Sprintf("Context: %s\n", violation.ErrorContext))
	}

	if violation.FixInfo != nil {
		details.WriteString("\nAuto-fix available: Yes")
	} else {
		details.WriteString("\nAuto-fix available: No")
	}

	if violation.RuleInformation != "" {
		details.WriteString(fmt.Sprintf("\nMore info: %s", violation.RuleInformation))
	}

	return m.styles.border.Width(m.width - 2).Render(details.String())
}

func (m *tuiModel) renderHelp() string {
	help := `gomdlint TUI Help

Navigation:
  ↑/↓ or j/k    Navigate up/down
  Enter         Select/view item
  b or Esc      Go back
  Tab           Switch between panes

Actions:
  r             Re-run linting
  f             Auto-fix violations
  s             Save current results

Views:
  1             File list view
  2             Violations view  
  3             Details view
  h or ?        This help screen

General:
  q or Ctrl+C   Quit application

Press 'b' or 'Esc' to go back to the main view.`

	return m.styles.border.Width(m.width - 2).Render(help)
}

func (m *tuiModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "h", "?":
		m.currentView = viewHelp
		return m, nil

	case "b", "esc":
		switch m.currentView {
		case viewViolations:
			m.currentView = viewFileList
		case viewDetails:
			m.currentView = viewViolations
		case viewHelp:
			m.currentView = viewFileList
		}
		return m, nil

	case "r":
		if m.currentView == viewFileList && !m.linting {
			return m, m.performLinting()
		}

	case "enter":
		switch m.currentView {
		case viewFileList:
			if len(m.files) > 0 {
				m.currentView = viewViolations
				m.selectedViol = 0
			}
		case viewViolations:
			m.currentView = viewDetails
		}
		return m, nil

	case "up", "k":
		switch m.currentView {
		case viewFileList:
			if m.selectedFile > 0 {
				m.selectedFile--
			}
		case viewViolations:
			if m.selectedViol > 0 {
				m.selectedViol--
			}
		}
		return m, nil

	case "down", "j":
		switch m.currentView {
		case viewFileList:
			if m.selectedFile < len(m.files)-1 {
				m.selectedFile++
			}
		case viewViolations:
			if m.lintResult != nil && m.selectedFile < len(m.files) {
				fileName := m.files[m.selectedFile]
				if violations, exists := m.lintResult.Results[fileName]; exists {
					if m.selectedViol < len(violations)-1 {
						m.selectedViol++
					}
				}
			}
		}
		return m, nil

	case "1":
		m.currentView = viewFileList
		return m, nil
	case "2":
		m.currentView = viewViolations
		return m, nil
	case "3":
		m.currentView = viewDetails
		return m, nil
	}

	return m, nil
}

// Custom messages for the TUI
type lintCompleteMsg struct {
	result *gomdlint.LintResult
}

type lintErrorMsg struct {
	err error
}

type tickMsg struct{}

func (m *tuiModel) performLinting() tea.Cmd {
	return func() tea.Msg {
		m.linting = true
		m.status = "Linting in progress..."

		options := gomdlint.LintOptions{
			Files:  m.files,
			Config: m.config,
		}

		result, err := gomdlint.Lint(m.ctx, options)
		if err != nil {
			return lintErrorMsg{err: err}
		}

		return lintCompleteMsg{result: result}
	}
}
