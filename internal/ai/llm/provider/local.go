package provider

import (
	"context"
	"time"

	"github.com/danielss-dev/critica/internal/ai/llm"
)

// LocalProvider is a mock provider for testing
type LocalProvider struct {
	response string
}

// NewLocal creates a new local mock provider
func NewLocal() llm.Provider {
	return &LocalProvider{
		response: `Analysis Summary:
- Changes detected in configuration files
- No critical issues found
- Code quality is good
- Consider adding documentation for new features`,
	}
}

// Generate returns a canned response for testing
func (p *LocalProvider) Generate(ctx context.Context, prompt string) (string, error) {
	return p.response, nil
}

// Stream returns a streamed canned response for testing
func (p *LocalProvider) Stream(ctx context.Context, prompt string) (<-chan llm.Chunk, error) {
	chunks := make(chan llm.Chunk)

	go func() {
		defer close(chunks)

		words := []string{"Analysis", " ", "Summary", ":\n",
			"-", " ", "Changes", " ", "detected", "\n",
			"-", " ", "No", " ", "issues", " ", "found"}

		for _, word := range words {
			select {
			case <-ctx.Done():
				chunks <- llm.Chunk{Error: ctx.Err(), Done: true}
				return
			default:
				chunks <- llm.Chunk{Content: word, Done: false}
				time.Sleep(50 * time.Millisecond)
			}
		}

		chunks <- llm.Chunk{Done: true}
	}()

	return chunks, nil
}
