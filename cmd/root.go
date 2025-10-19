package cmd

import (
	"fmt"
	"os"

	"github.com/danielss-dev/critica/internal/ai"
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
	aiEnabled   bool

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
	rootCmd.Flags().BoolVar(&aiEnabled, "ai", false, "Enable AI analysis and suggestions")
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

		// Initialize AI service for interactive mode
		var aiService *ai.Service
		aiConfig := ai.LoadConfig()
		if aiConfig.APIKey != "" {
			aiService = ai.NewService(aiConfig)
		}

		return ui.RunInteractive(files, stagedFiles, unstagedFiles, rendererOpts, aiService)
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
