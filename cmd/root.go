package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/danielss-dev/critica/internal/ai/agent"
	"github.com/danielss-dev/critica/internal/ai/llm"
	"github.com/danielss-dev/critica/internal/ai/llm/provider"
	"github.com/danielss-dev/critica/internal/ai/permission"
	"github.com/danielss-dev/critica/internal/ai/session"
	"github.com/danielss-dev/critica/internal/ai/tools"
	"github.com/danielss-dev/critica/internal/config"
	"github.com/danielss-dev/critica/internal/git"
	"github.com/danielss-dev/critica/internal/parser"
	"github.com/danielss-dev/critica/internal/ui"
	"github.com/spf13/cobra"
)

var (
	staged      bool
	cached      bool
	noColor     bool
	unified     bool
	interactive bool

	// AI flags
	aiEnabled  bool
	aiAnalyze  bool
	aiSuggest  bool
	aiExplain  bool
	aiApply    bool
	aiFormat   string
	aiOutput   string
	aiSeverity string
	aiCI       bool
	aiProvider string
	aiModel    string

	appConfig *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "critica [path]",
	Short: "A beautiful git diff viewer for the terminal",
	Long: `Critica displays git diffs in a beautiful split-screen format.

You can view diffs for:
  - The entire repository (no arguments)
  - A specific file or directory (provide path as argument)

Examples:
  critica                    # Show diff for entire repo
  critica src/main.go        # Show diff for specific file
  critica src/               # Show diff for directory
  critica --staged           # Show staged changes
  critica --cached           # Show cached changes (alias for --staged)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDiff,
}

func init() {
	rootCmd.Flags().BoolVarP(&staged, "staged", "s", false, "Show only staged changes")
	rootCmd.Flags().BoolVarP(&cached, "cached", "c", false, "Show only cached changes (same as --staged)")
	rootCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable color output")
	rootCmd.Flags().BoolVarP(&unified, "unified", "u", false, "Show unified diff view (non-split)")
	rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode with fuzzy finder and collapsible files")

	// AI flags
	rootCmd.Flags().BoolVar(&aiEnabled, "ai", false, "Enable AI analysis")
	rootCmd.Flags().BoolVar(&aiAnalyze, "analyze", false, "Analyze changes with AI")
	rootCmd.Flags().BoolVar(&aiSuggest, "suggest", false, "Get AI suggestions")
	rootCmd.Flags().BoolVar(&aiExplain, "explain", false, "Explain changes with AI")
	rootCmd.Flags().BoolVar(&aiApply, "apply", false, "Apply AI suggestions (requires permission)")
	rootCmd.Flags().StringVar(&aiFormat, "format", "text", "Output format: text|md|json|sarif")
	rootCmd.Flags().StringVar(&aiOutput, "output", "", "Output file (default: stdout)")
	rootCmd.Flags().StringVar(&aiSeverity, "severity-threshold", "info", "Severity threshold: info|warning|error")
	rootCmd.Flags().BoolVar(&aiCI, "ci", false, "CI mode (non-interactive, no writes unless --apply)")
	rootCmd.Flags().StringVar(&aiProvider, "provider", "", "AI provider override: openai|anthropic|local")
	rootCmd.Flags().StringVar(&aiModel, "model", "", "AI model override")

	rootCmd.PersistentPreRunE = applyConfig
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runDiff(cmd *cobra.Command, args []string) error {
	// Get the path (default to current directory)
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Determine if we should show staged changes
	showStaged := staged || cached

	// Check if we're in a git repository
	if !git.IsGitRepository(path) {
		return fmt.Errorf("not a git repository: %s", path)
	}

	// Select diff mode
	diffMode := git.DiffModeAll
	if showStaged {
		diffMode = git.DiffModeStaged
	} else if appConfig != nil {
		switch appConfig.DiffMode {
		case config.DiffModeAll:
			diffMode = git.DiffModeAll
		case config.DiffModeUnstaged:
			diffMode = git.DiffModeUnstaged
		case config.DiffModeStaged:
			diffMode = git.DiffModeStaged
		}
	}

	// Get the git diff
	diffOutput, err := git.GetDiffForMode(path, diffMode)
	if err != nil {
		return fmt.Errorf("failed to get git diff: %w", err)
	}

	// Check if there are any changes
	if diffOutput == "" {
		fmt.Println("No changes to display")
		return nil
	}

	// Parse the diff output
	files, err := parser.ParseDiff(diffOutput)
	if err != nil {
		return fmt.Errorf("failed to parse diff: %w", err)
	}

	rendererOpts := ui.RendererOptions{
		UseColor: !noColor,
		Unified:  unified,
	}

	if appConfig != nil {
		rendererOpts.DiffStyle = appConfig.DiffStyle
		rendererOpts.AddedTextColor = appConfig.AddedTextColor
		rendererOpts.DeletedTextColor = appConfig.DeletedTextColor
	}

	// Run in interactive mode or static mode
	if interactive {
		var stagedFiles []parser.FileDiff
		var unstagedFiles []parser.FileDiff

		if showStaged {
			stagedFiles = files
		} else {
			stagedOutput, err := git.GetDiffForMode(path, git.DiffModeStaged)
			if err != nil {
				return fmt.Errorf("failed to get staged diff: %w", err)
			}
			stagedFiles, err = parser.ParseDiff(stagedOutput)
			if err != nil {
				return fmt.Errorf("failed to parse staged diff: %w", err)
			}

			unstagedOutput, err := git.GetDiffForMode(path, git.DiffModeUnstaged)
			if err != nil {
				return fmt.Errorf("failed to get unstaged diff: %w", err)
			}
			unstagedFiles, err = parser.ParseDiff(unstagedOutput)
			if err != nil {
				return fmt.Errorf("failed to parse unstaged diff: %w", err)
			}
		}

		// Initialize AI agent if AI is enabled
		var aiAgent agent.Service
		var aiEnabledInTUI bool
		if aiEnabled || aiAnalyze || aiSuggest || aiExplain || (appConfig != nil && appConfig.AIEnabled != nil && *appConfig.AIEnabled) {
			agentSvc, err := initializeAI()
			if err != nil {
				// If AI initialization fails, continue without AI
				fmt.Fprintf(os.Stderr, "Warning: AI initialization failed: %v\n", err)
			} else {
				aiAgent = agentSvc
				aiEnabledInTUI = true
			}
		}

		return ui.RunInteractive(files, stagedFiles, unstagedFiles, rendererOpts, aiAgent, aiEnabledInTUI)
	}

	// Handle AI operations for non-interactive mode
	if aiEnabled || aiAnalyze || aiSuggest || aiExplain {
		return runAIMode(diffOutput, files)
	}

	// Render the diff statically
	renderer := ui.NewRenderer(rendererOpts)
	renderer.Render(files)

	return nil
}

func applyConfig(cmd *cobra.Command, _ []string) error {
	if appConfig != nil {
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	appConfig = cfg

	applyBool := func(flagName string, target *bool, value *bool) {
		if value == nil {
			return
		}
		if cmd.Flags().Changed(flagName) {
			return
		}
		*target = *value
	}

	applyBool("interactive", &interactive, cfg.Interactive)
	applyBool("unified", &unified, cfg.Unified)
	applyBool("no-color", &noColor, cfg.NoColor)

	if cfg.DiffMode == config.DiffModeStaged {
		if !cmd.Flags().Changed("staged") && !cmd.Flags().Changed("cached") {
			staged = true
		}
	}

	return nil
}

// runAIMode executes AI analysis/suggestion/explanation
func runAIMode(diffContent string, files []parser.FileDiff) error {
	// Initialize AI service
	agentSvc, err := initializeAI()
	if err != nil {
		return fmt.Errorf("failed to initialize AI: %w", err)
	}

	ctx := context.Background()
	var events <-chan agent.AgentEvent

	// Determine which AI operation to run
	if aiAnalyze {
		events, err = agentSvc.AnalyzeDiff(ctx, diffContent)
	} else if aiSuggest {
		events, err = agentSvc.SuggestImprovements(ctx, "", diffContent)
	} else if aiExplain {
		events, err = agentSvc.ExplainChanges(ctx, diffContent)
	} else {
		// Default to analysis
		events, err = agentSvc.AnalyzeDiff(ctx, diffContent)
	}

	if err != nil {
		return fmt.Errorf("AI operation failed: %w", err)
	}

	// Collect and format results
	return processAIEvents(events, files)
}

// initializeAI creates and configures the AI service
func initializeAI() (agent.Service, error) {
	if appConfig == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	// Override provider if specified
	providerName := appConfig.AIProvider
	if aiProvider != "" {
		providerName = aiProvider
	}

	// Override model if specified
	modelName := appConfig.AIModel
	if aiModel != "" {
		modelName = aiModel
	}

	// Get API key with fallback to provider-specific environment variables
	apiKey := appConfig.AIAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("CRITICA_AI_API_KEY")
	}

	// If still no API key, try provider-specific environment variables
	if apiKey == "" {
		switch providerName {
		case config.AIProviderOpenAI:
			apiKey = os.Getenv("OPENAI_API_KEY")
		case config.AIProviderAnthropic:
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		}
	}

	// Create provider
	var llmProvider llm.Provider
	switch providerName {
	case config.AIProviderOpenAI:
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API key required (set CRITICA_AI_API_KEY or OPENAI_API_KEY)")
		}
		llmProvider = provider.NewOpenAI(apiKey, modelName)
	case config.AIProviderAnthropic:
		if apiKey == "" {
			return nil, fmt.Errorf("anthropic API key required (set CRITICA_AI_API_KEY or ANTHROPIC_API_KEY)")
		}
		llmProvider = provider.NewAnthropic(apiKey, modelName)
	case config.AIProviderLocal:
		llmProvider = provider.NewLocal()
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", providerName)
	}

	// Create services
	permService := permission.NewService()
	if aiCI {
		permService.SetCIMode(true)
	}
	if aiApply {
		// Allow write operations
		permService.SetCIMode(false)
	}

	sessService := session.NewService()

	return agent.NewAgent(llmProvider, permService, sessService), nil
}

// processAIEvents collects events and formats output
func processAIEvents(events <-chan agent.AgentEvent, files []parser.FileDiff) error {
	var findings []tools.Finding
	var analysisContent strings.Builder

	for event := range events {
		if event.Error != nil {
			return fmt.Errorf("AI error: %w", event.Error)
		}

		analysisContent.WriteString(event.Content)
		analysisContent.WriteString("\n")

		// Convert to findings for report
		if event.Type == agent.AgentEventTypeAnalysis || event.Type == agent.AgentEventTypeSuggestion {
			findings = append(findings, tools.Finding{
				Severity:    tools.SeverityInfo,
				Title:       string(event.Type),
				Description: event.Content,
				Category:    "ai-review",
			})
		}
	}

	// Create report
	report := tools.ReviewReport{
		Timestamp: time.Now(),
		Findings:  findings,
		Summary:   analysisContent.String(),
		FileCount: len(files),
	}

	// Format and output
	formatter := tools.GetFormatter(aiFormat)
	output, err := formatter.Format(report)
	if err != nil {
		return fmt.Errorf("failed to format report: %w", err)
	}

	// Write to file or stdout
	if aiOutput != "" {
		if err := os.WriteFile(aiOutput, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		fmt.Printf("Report written to: %s\n", aiOutput)
	} else {
		fmt.Println(output)
	}

	// Exit code based on severity threshold
	if aiCI {
		threshold := parseSeverity(aiSeverity)
		for _, finding := range findings {
			if severityLevel(finding.Severity) >= severityLevel(threshold) {
				os.Exit(1)
			}
		}
	}

	return nil
}

// parseSeverity converts string to Severity
func parseSeverity(s string) tools.Severity {
	switch strings.ToLower(s) {
	case "error":
		return tools.SeverityError
	case "warning":
		return tools.SeverityWarning
	default:
		return tools.SeverityInfo
	}
}

// severityLevel returns numeric level for comparison
func severityLevel(s tools.Severity) int {
	switch s {
	case tools.SeverityError:
		return 3
	case tools.SeverityWarning:
		return 2
	case tools.SeverityInfo:
		return 1
	default:
		return 0
	}
}
