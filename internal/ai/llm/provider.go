package llm

import "context"

// Chunk represents a streaming response chunk
type Chunk struct {
	Content string
	Done    bool
	Error   error
}

// Provider defines the interface for LLM providers
type Provider interface {
	Generate(ctx context.Context, prompt string) (string, error)
	Stream(ctx context.Context, prompt string) (<-chan Chunk, error)
}
