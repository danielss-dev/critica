package provider

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/danielss-dev/critica/internal/ai/llm"
)

// AnthropicProvider implements the LLM provider for Anthropic Claude
type AnthropicProvider struct {
	client *anthropic.Client
	model  string
}

// NewAnthropic creates a new Anthropic provider
func NewAnthropic(apiKey, model string) llm.Provider {
	if model == "" {
		model = "claude-3-5-haiku-20241022"
	}
	return &AnthropicProvider{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

// Generate sends a prompt to Anthropic and returns the response
func (p *AnthropicProvider) Generate(ctx context.Context, prompt string) (string, error) {
	message, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F(p.model),
		MaxTokens: anthropic.F(int64(4096)),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		}),
	})

	if err != nil {
		return "", fmt.Errorf("anthropic api error: %w", err)
	}

	if len(message.Content) == 0 {
		return "", fmt.Errorf("no response from anthropic")
	}

	// Extract text content from the response
	var result string
	for _, block := range message.Content {
		if block.Type == anthropic.ContentBlockTypeText {
			result += block.Text
		}
	}

	return result, nil
}

// Stream sends a prompt to Anthropic and streams the response
func (p *AnthropicProvider) Stream(ctx context.Context, prompt string) (<-chan llm.Chunk, error) {
	stream := p.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F(p.model),
		MaxTokens: anthropic.F(int64(4096)),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		}),
	})

	chunks := make(chan llm.Chunk)

	go func() {
		defer close(chunks)

		for stream.Next() {
			event := stream.Current()

			switch delta := event.Delta.(type) {
			case anthropic.ContentBlockDeltaEventDelta:
				if delta.Type == anthropic.ContentBlockDeltaEventDeltaTypeTextDelta {
					chunks <- llm.Chunk{
						Content: delta.Text,
						Done:    false,
					}
				}
			}
		}

		if err := stream.Err(); err != nil {
			chunks <- llm.Chunk{Error: err, Done: true}
			return
		}

		chunks <- llm.Chunk{Done: true}
	}()

	return chunks, nil
}
