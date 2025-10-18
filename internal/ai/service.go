package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/danielss-dev/critica/internal/parser"
	"github.com/sashabaranov/go-openai"
)

// Service handles AI operations for code analysis
type Service struct {
	client *openai.Client
	config *Config
}

// Config holds AI service configuration
type Config struct {
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float32
	BaseURL     string
}

// AnalysisResult contains the AI analysis results
type AnalysisResult struct {
	Summary          string   `json:"summary"`
	Improvements     []string `json:"improvements"`
	Issues           []string `json:"issues"`
	Explanations     []string `json:"explanations"`
	CommitMessage    string   `json:"commit_message"`
	PRDescription    string   `json:"pr_description"`
	CodeQuality      string   `json:"code_quality"`
	SecurityNotes    []string `json:"security_notes"`
	PerformanceNotes []string `json:"performance_notes"`
}

// NewService creates a new AI service instance
func NewService(config *Config) *Service {
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	client := openai.NewClientWithConfig(clientConfig)

	return &Service{
		client: client,
		config: config,
	}
}

// LoadConfig loads AI configuration from environment variables and config file
func LoadConfig() *Config {
	config := &Config{
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		Model:       getEnvOrDefault("OPENAI_MODEL", "gpt-4o-mini"),
		MaxTokens:   4000,
		Temperature: 0.3,
		BaseURL:     os.Getenv("OPENAI_BASE_URL"),
	}

	return config
}

// AnalyzeDiff performs comprehensive analysis of git diff changes
func (s *Service) AnalyzeDiff(ctx context.Context, files []parser.FileDiff) (*AnalysisResult, error) {
	if len(files) == 0 {
		return &AnalysisResult{}, nil
	}

	// Prepare the diff content for analysis
	diffContent := s.prepareDiffContent(files)

	// Create the analysis prompt
	prompt := s.buildAnalysisPrompt(diffContent)

	// Call the AI service
	response, err := s.callAI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	// Parse the response
	result, err := s.parseAnalysisResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return result, nil
}

// GenerateCommitMessage generates a commit message based on the changes
func (s *Service) GenerateCommitMessage(ctx context.Context, files []parser.FileDiff) (string, error) {
	if len(files) == 0 {
		return "No changes to commit", nil
	}

	diffContent := s.prepareDiffContent(files)
	prompt := s.buildCommitMessagePrompt(diffContent)

	response, err := s.callAI(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("commit message generation failed: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// GeneratePRDescription generates a PR description based on the changes
func (s *Service) GeneratePRDescription(ctx context.Context, files []parser.FileDiff) (string, error) {
	if len(files) == 0 {
		return "No changes to describe", nil
	}

	diffContent := s.prepareDiffContent(files)
	prompt := s.buildPRDescriptionPrompt(diffContent)

	response, err := s.callAI(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("PR description generation failed: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// SuggestImprovements provides code improvement suggestions
func (s *Service) SuggestImprovements(ctx context.Context, files []parser.FileDiff) ([]string, error) {
	if len(files) == 0 {
		return []string{}, nil
	}

	diffContent := s.prepareDiffContent(files)
	prompt := s.buildImprovementsPrompt(diffContent)

	response, err := s.callAI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("improvement suggestions failed: %w", err)
	}

	// Parse the response as a list of improvements
	improvements := strings.Split(response, "\n")
	var result []string
	for _, imp := range improvements {
		imp = strings.TrimSpace(imp)
		if imp != "" && !strings.HasPrefix(imp, "-") {
			result = append(result, imp)
		}
	}

	return result, nil
}

// ExplainChanges provides explanations for the code changes
func (s *Service) ExplainChanges(ctx context.Context, files []parser.FileDiff) (string, error) {
	if len(files) == 0 {
		return "No changes to explain", nil
	}

	diffContent := s.prepareDiffContent(files)
	prompt := s.buildExplanationPrompt(diffContent)

	response, err := s.callAI(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("change explanation failed: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// prepareDiffContent converts FileDiff objects to a readable diff format
func (s *Service) prepareDiffContent(files []parser.FileDiff) string {
	var content strings.Builder

	for _, file := range files {
		content.WriteString(fmt.Sprintf("File: %s\n", file.NewPath))
		if file.IsNew {
			content.WriteString("Status: New file\n")
		} else if file.IsDeleted {
			content.WriteString("Status: Deleted file\n")
		} else if file.IsRenamed {
			content.WriteString(fmt.Sprintf("Status: Renamed from %s\n", file.OldPath))
		} else {
			content.WriteString("Status: Modified\n")
		}
		content.WriteString("\n")

		for _, hunk := range file.Hunks {
			content.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
				hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines))

			for _, line := range hunk.Lines {
				var prefix string
				switch line.Type {
				case parser.LineAdded:
					prefix = "+"
				case parser.LineDeleted:
					prefix = "-"
				case parser.LineUnchanged:
					prefix = " "
				}
				content.WriteString(fmt.Sprintf("%s%s\n", prefix, line.Content))
			}
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	return content.String()
}

// buildAnalysisPrompt creates a comprehensive analysis prompt
func (s *Service) buildAnalysisPrompt(diffContent string) string {
	return fmt.Sprintf(`Analyze the following git diff and provide a comprehensive analysis in JSON format. Focus on:

1. Code quality and best practices
2. Potential bugs or issues
3. Security concerns
4. Performance implications
5. Code improvements and suggestions
6. Clear explanations of what changed and why

Please respond with a JSON object containing:
- summary: A brief overview of the changes
- improvements: Array of specific improvement suggestions
- issues: Array of potential problems or bugs
- explanations: Array of explanations for complex changes
- commit_message: A conventional commit message
- pr_description: A detailed PR description
- code_quality: Assessment of overall code quality
- security_notes: Array of security-related observations
- performance_notes: Array of performance-related observations

Git diff:
%s

Respond only with valid JSON, no additional text.`, diffContent)
}

// buildCommitMessagePrompt creates a prompt for commit message generation
func (s *Service) buildCommitMessagePrompt(diffContent string) string {
	return fmt.Sprintf(`Generate a conventional commit message for the following git diff. Use the format:
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]

Types: feat, fix, docs, style, refactor, perf, test, chore, build, ci, revert

Git diff:
%s

Respond only with the commit message, no additional text.`, diffContent)
}

// buildPRDescriptionPrompt creates a prompt for PR description generation
func (s *Service) buildPRDescriptionPrompt(diffContent string) string {
	return fmt.Sprintf(`Generate a comprehensive PR description for the following git diff. Include:

1. Summary of changes
2. What was changed and why
3. Testing considerations
4. Breaking changes (if any)
5. Screenshots or examples (if applicable)

Git diff:
%s

Respond with a well-formatted PR description.`, diffContent)
}

// buildImprovementsPrompt creates a prompt for improvement suggestions
func (s *Service) buildImprovementsPrompt(diffContent string) string {
	return fmt.Sprintf(`Analyze the following git diff and provide specific, actionable improvement suggestions. Focus on:

1. Code quality and readability
2. Performance optimizations
3. Security improvements
4. Best practices adherence
5. Error handling
6. Documentation needs

Git diff:
%s

Provide each suggestion as a separate line, starting with a brief description.`, diffContent)
}

// buildExplanationPrompt creates a prompt for change explanations
func (s *Service) buildExplanationPrompt(diffContent string) string {
	return fmt.Sprintf(`Explain the following git diff changes in detail. Provide:

1. What each change does
2. Why the change was made (if apparent)
3. Impact of the changes
4. Any potential side effects
5. How the changes relate to each other

Git diff:
%s

Provide a clear, comprehensive explanation.`, diffContent)
}

// callAI makes a request to the AI service
func (s *Service) callAI(ctx context.Context, prompt string) (string, error) {
	req := openai.ChatCompletionRequest{
		Model: s.config.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		MaxTokens:   s.config.MaxTokens,
		Temperature: s.config.Temperature,
	}

	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI service")
	}

	return resp.Choices[0].Message.Content, nil
}

// parseAnalysisResponse parses the AI response into AnalysisResult
func (s *Service) parseAnalysisResponse(response string) (*AnalysisResult, error) {
	var result AnalysisResult

	// Try to parse as JSON first
	if err := json.Unmarshal([]byte(response), &result); err == nil {
		return &result, nil
	}

	// If JSON parsing fails, create a basic result
	result.Summary = response
	result.Improvements = []string{}
	result.Issues = []string{}
	result.Explanations = []string{response}
	result.CommitMessage = "Update code"
	result.PRDescription = response
	result.CodeQuality = "Unknown"
	result.SecurityNotes = []string{}
	result.PerformanceNotes = []string{}

	return &result, nil
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
