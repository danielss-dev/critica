package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/danielss-dev/critica/internal/ai"
	"github.com/danielss-dev/critica/internal/git"
	"github.com/danielss-dev/critica/internal/parser"
	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI-powered analysis and suggestions for git diffs",
	Long: `AI commands provide intelligent analysis of your git diffs including:
- Code quality analysis
- Improvement suggestions
- Bug detection
- Commit message generation
- PR description generation
- Change explanations`,
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze [path]",
	Short: "Perform comprehensive AI analysis of git diff",
	Long: `Analyze the git diff with AI to get insights about:
- Code quality assessment
- Potential issues and bugs
- Security concerns
- Performance implications
- Improvement suggestions
- Explanations of changes`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAIAnalysis,
}

var commitCmd = &cobra.Command{
	Use:   "commit [path]",
	Short: "Generate conventional commit message",
	Long: `Generate a conventional commit message based on the git diff.
Uses AI to analyze changes and create appropriate commit messages following conventional commit format.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAIGenerateCommit,
}

var prCmd = &cobra.Command{
	Use:   "pr [path]",
	Short: "Generate PR description",
	Long: `Generate a comprehensive PR description based on the git diff.
Creates detailed PR descriptions including summary, changes, and testing considerations.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAIGeneratePR,
}

var improveCmd = &cobra.Command{
	Use:   "improve [path]",
	Short: "Get code improvement suggestions",
	Long: `Get specific, actionable improvement suggestions for the git diff.
Focuses on code quality, performance, security, and best practices.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAIImprovements,
}

var explainCmd = &cobra.Command{
	Use:   "explain [path]",
	Short: "Explain code changes",
	Long: `Get detailed explanations of what changed in the git diff.
Provides clear explanations of what each change does and why it might have been made.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAIExplain,
}

func init() {
	rootCmd.AddCommand(aiCmd)
	aiCmd.AddCommand(analyzeCmd)
	aiCmd.AddCommand(commitCmd)
	aiCmd.AddCommand(prCmd)
	aiCmd.AddCommand(improveCmd)
	aiCmd.AddCommand(explainCmd)
}

func runAIAnalysis(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Check if we're in a git repository
	if !git.IsGitRepository(path) {
		return fmt.Errorf("not a git repository: %s", path)
	}

	// Get the diff
	diffOutput, err := git.GetDiff(path, staged)
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	if diffOutput == "" {
		fmt.Println("No changes to analyze")
		return nil
	}

	// Parse the diff
	files, err := parser.ParseDiff(diffOutput)
	if err != nil {
		return fmt.Errorf("failed to parse diff: %w", err)
	}

	// Load AI configuration
	aiConfig := ai.LoadConfig()
	if aiConfig.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	// Create AI service
	aiService := ai.NewService(aiConfig)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("🤖 Analyzing changes with AI...")
	fmt.Println()

	// Perform analysis
	result, err := aiService.AnalyzeDiff(ctx, files)
	if err != nil {
		return fmt.Errorf("AI analysis failed: %w", err)
	}

	// Display results
	displayAnalysisResult(result)
	return nil
}

func runAIGenerateCommit(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Check if we're in a git repository
	if !git.IsGitRepository(path) {
		return fmt.Errorf("not a git repository: %s", path)
	}

	// Get the diff
	diffOutput, err := git.GetDiff(path, staged)
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	if diffOutput == "" {
		fmt.Println("No changes to commit")
		return nil
	}

	// Parse the diff
	files, err := parser.ParseDiff(diffOutput)
	if err != nil {
		return fmt.Errorf("failed to parse diff: %w", err)
	}

	// Load AI configuration
	aiConfig := ai.LoadConfig()
	if aiConfig.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	// Create AI service
	aiService := ai.NewService(aiConfig)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("🤖 Generating commit message...")
	fmt.Println()

	// Generate commit message
	commitMsg, err := aiService.GenerateCommitMessage(ctx, files)
	if err != nil {
		return fmt.Errorf("commit message generation failed: %w", err)
	}

	fmt.Println("Generated commit message:")
	fmt.Println("─" + strings.Repeat("─", len(commitMsg)))
	fmt.Println(commitMsg)
	fmt.Println("─" + strings.Repeat("─", len(commitMsg)))
	return nil
}

func runAIGeneratePR(cmd *cobra.Command, args []string) error {
	path := "."
	targetBranch := ""

	// Parse arguments
	if len(args) > 0 {
		// Check if first argument is a path or branch name
		if git.IsGitRepository(args[0]) {
			path = args[0]
			if len(args) > 1 {
				targetBranch = args[1]
			}
		} else {
			// First argument is a branch name
			targetBranch = args[0]
		}
	}

	// Check if we're in a git repository
	if !git.IsGitRepository(path) {
		return fmt.Errorf("not a git repository: %s", path)
	}

	// Get current branch
	currentBranch, err := git.GetCurrentBranch(path)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get all branches (local and remote)
	localBranches, err := git.GetAllBranches(path)
	if err != nil {
		return fmt.Errorf("failed to get local branches: %w", err)
	}

	remoteBranches, err := git.GetRemoteBranches(path)
	var branches []string
	if err != nil {
		// If remote branches fail, just use local branches
		branches = localBranches
	} else {
		// Combine local and remote branches
		branches = append(localBranches, remoteBranches...)
	}

	if len(branches) < 2 {
		fmt.Println("Not enough branches for comparison")
		return nil
	}

	// Show branch selection
	fmt.Println("Available branches:")
	for i, branch := range branches {
		marker := "  "
		if branch == currentBranch {
			marker = "→ "
		}
		fmt.Printf("%s%d. %s%s\n", marker, i+1, branch, func() string {
			if branch == currentBranch {
				return " (current)"
			}
			return ""
		}())
	}
	fmt.Println()
	fmt.Printf("Current branch: %s\n", currentBranch)
	fmt.Println("Selecting target branch to compare against (where you want to merge into)")
	fmt.Println()

	// If target branch is provided as parameter, use it
	if targetBranch != "" {
		// Validate that the target branch exists
		found := false
		for _, branch := range branches {
			if branch == targetBranch {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("target branch '%s' not found in available branches", targetBranch)
		}

		// Don't allow comparing branch to itself
		if targetBranch == currentBranch {
			return fmt.Errorf("cannot compare branch to itself")
		}

		fmt.Printf("Using target branch: %s\n", targetBranch)
	} else {
		// Prompt user to select target branch
		fmt.Print("Enter target branch number (1-", len(branches), "): ")
		var input string
		fmt.Scanln(&input)

		// Parse selection
		var selection int
		if _, err := fmt.Sscanf(input, "%d", &selection); err != nil {
			return fmt.Errorf("invalid selection: %s", input)
		}

		if selection < 1 || selection > len(branches) {
			return fmt.Errorf("selection out of range: %d", selection)
		}

		targetBranch = branches[selection-1]

		// Don't allow comparing branch to itself
		if targetBranch == currentBranch {
			return fmt.Errorf("cannot compare branch to itself")
		}
	}

	fmt.Printf("Comparing %s → %s\n", currentBranch, targetBranch)
	fmt.Println()

	// Get diff between branches
	diffOutput, err := git.GetBranchDiff(path, currentBranch, targetBranch)
	if err != nil {
		return fmt.Errorf("failed to get branch diff: %w", err)
	}

	if diffOutput == "" {
		fmt.Println("No changes between branches")
		return nil
	}

	// Load AI configuration
	aiConfig := ai.LoadConfig()
	if aiConfig.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	// Create AI service
	aiService := ai.NewService(aiConfig)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("🤖 Generating PR description...")
	fmt.Println()

	// Generate PR description with branch context
	prDesc, err := aiService.GeneratePRDescriptionWithBranches(ctx, diffOutput, currentBranch, targetBranch)
	if err != nil {
		return fmt.Errorf("PR description generation failed: %w", err)
	}

	fmt.Println("Generated PR description:")
	fmt.Println("─" + strings.Repeat("─", 50))
	fmt.Println(prDesc)
	fmt.Println("─" + strings.Repeat("─", 50))
	return nil
}

func runAIImprovements(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Check if we're in a git repository
	if !git.IsGitRepository(path) {
		return fmt.Errorf("not a git repository: %s", path)
	}

	// Get the diff
	diffOutput, err := git.GetDiff(path, staged)
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	if diffOutput == "" {
		fmt.Println("No changes to improve")
		return nil
	}

	// Parse the diff
	files, err := parser.ParseDiff(diffOutput)
	if err != nil {
		return fmt.Errorf("failed to parse diff: %w", err)
	}

	// Load AI configuration
	aiConfig := ai.LoadConfig()
	if aiConfig.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	// Create AI service
	aiService := ai.NewService(aiConfig)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("🤖 Analyzing for improvements...")
	fmt.Println()

	// Get improvements
	improvements, err := aiService.SuggestImprovements(ctx, files)
	if err != nil {
		return fmt.Errorf("improvement suggestions failed: %w", err)
	}

	if len(improvements) == 0 {
		fmt.Println("No specific improvements suggested.")
		return nil
	}

	fmt.Println("Improvement suggestions:")
	fmt.Println("─" + strings.Repeat("─", 30))
	for i, improvement := range improvements {
		fmt.Printf("%d. %s\n", i+1, improvement)
	}
	fmt.Println("─" + strings.Repeat("─", 30))
	return nil
}

func runAIExplain(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Check if we're in a git repository
	if !git.IsGitRepository(path) {
		return fmt.Errorf("not a git repository: %s", path)
	}

	// Get the diff
	diffOutput, err := git.GetDiff(path, staged)
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	if diffOutput == "" {
		fmt.Println("No changes to explain")
		return nil
	}

	// Parse the diff
	files, err := parser.ParseDiff(diffOutput)
	if err != nil {
		return fmt.Errorf("failed to parse diff: %w", err)
	}

	// Load AI configuration
	aiConfig := ai.LoadConfig()
	if aiConfig.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	// Create AI service
	aiService := ai.NewService(aiConfig)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("🤖 Explaining changes...")
	fmt.Println()

	// Get explanation
	explanation, err := aiService.ExplainChanges(ctx, files)
	if err != nil {
		return fmt.Errorf("change explanation failed: %w", err)
	}

	fmt.Println("Change explanation:")
	fmt.Println("─" + strings.Repeat("─", 30))
	fmt.Println(explanation)
	fmt.Println("─" + strings.Repeat("─", 30))
	return nil
}

func displayAnalysisResult(result *ai.AnalysisResult) {
	fmt.Println("📊 Analysis Results")
	fmt.Println("─" + strings.Repeat("─", 50))

	if result.Summary != "" {
		fmt.Println("📝 Summary:")
		fmt.Println(result.Summary)
		fmt.Println()
	}

	if result.CodeQuality != "" {
		fmt.Println("🏆 Code Quality:")
		fmt.Println(result.CodeQuality)
		fmt.Println()
	}

	if len(result.Issues) > 0 {
		fmt.Println("⚠️  Issues Found:")
		for i, issue := range result.Issues {
			fmt.Printf("  %d. %s\n", i+1, issue)
		}
		fmt.Println()
	}

	if len(result.Improvements) > 0 {
		fmt.Println("💡 Improvement Suggestions:")
		for i, improvement := range result.Improvements {
			fmt.Printf("  %d. %s\n", i+1, improvement)
		}
		fmt.Println()
	}

	if len(result.SecurityNotes) > 0 {
		fmt.Println("🔒 Security Notes:")
		for i, note := range result.SecurityNotes {
			fmt.Printf("  %d. %s\n", i+1, note)
		}
		fmt.Println()
	}

	if len(result.PerformanceNotes) > 0 {
		fmt.Println("⚡ Performance Notes:")
		for i, note := range result.PerformanceNotes {
			fmt.Printf("  %d. %s\n", i+1, note)
		}
		fmt.Println()
	}

	if result.CommitMessage != "" {
		fmt.Println("📝 Suggested Commit Message:")
		fmt.Println("─" + strings.Repeat("─", 30))
		fmt.Println(result.CommitMessage)
		fmt.Println("─" + strings.Repeat("─", 30))
		fmt.Println()
	}

	if result.PRDescription != "" {
		fmt.Println("📋 PR Description:")
		fmt.Println("─" + strings.Repeat("─", 30))
		fmt.Println(result.PRDescription)
		fmt.Println("─" + strings.Repeat("─", 30))
	}
}
