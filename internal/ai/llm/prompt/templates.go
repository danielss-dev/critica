package prompt

import "fmt"

// DiffAnalysisPrompt generates a prompt for analyzing git diffs
func DiffAnalysisPrompt(diffContent string) string {
	return fmt.Sprintf(`Analyze this git diff and provide insights:

%s

Please provide:
1. Summary of changes
2. Potential issues or risks
3. Code quality assessment
4. Suggestions for improvement

Be concise and actionable.`, diffContent)
}

// SuggestImprovementsPrompt generates a prompt for suggesting code improvements
func SuggestImprovementsPrompt(filePath string, changes []string) string {
	changesText := ""
	for _, change := range changes {
		changesText += change + "\n"
	}

	return fmt.Sprintf(`Review these code changes in file %s and suggest improvements:

%s

For each suggestion, provide:
- Title: Brief description
- Type: "improvement", "bug_fix", or "optimization"
- Confidence: 0.0 to 1.0
- Description: Detailed explanation
- Code: Optional code snippet if applicable

Return suggestions as a JSON array.`, filePath, changesText)
}

// ExplainChangesPrompt generates a prompt for explaining code changes
func ExplainChangesPrompt(diffContent string) string {
	return fmt.Sprintf(`Explain what these code changes do in clear, simple language:

%s

Focus on:
- What functionality is being added/modified/removed
- Why these changes might have been made
- The impact on the codebase

Keep it concise and understandable for developers of all levels.`, diffContent)
}
