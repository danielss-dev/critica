package provider

import (
	"context"
	"fmt"

	"github.com/danielss-dev/critica/internal/ai/llm"
	"github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements the LLM provider for OpenAI
type OpenAIProvider struct {
	client *openai.Client
	model  string
}

// NewOpenAI creates a new OpenAI provider
func NewOpenAI(apiKey, model string) llm.Provider {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &OpenAIProvider{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

// Generate sends a prompt to OpenAI and returns the response
func (p *OpenAIProvider) Generate(ctx context.Context, prompt string) (string, error) {
	resp, err := p.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: p.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("openai api error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from openai")
	}

	return resp.Choices[0].Message.Content, nil
}

// Stream sends a prompt to OpenAI and streams the response
func (p *OpenAIProvider) Stream(ctx context.Context, prompt string) (<-chan llm.Chunk, error) {
	stream, err := p.client.CreateChatCompletionStream(
		ctx,
		openai.ChatCompletionRequest{
			Model: p.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Stream: true,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("openai stream error: %w", err)
	}

	chunks := make(chan llm.Chunk)

	go func() {
		defer close(chunks)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err != nil {
				chunks <- llm.Chunk{Error: err, Done: true}
				return
			}

			if len(response.Choices) > 0 {
				content := response.Choices[0].Delta.Content
				chunks <- llm.Chunk{
					Content: content,
					Done:    false,
				}
			}
		}
	}()

	return chunks, nil
}
