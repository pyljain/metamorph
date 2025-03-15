package llm

import (
	"context"

	"google.golang.org/genai"
)

type VertexLLM struct {
	client *genai.Client
}

func NewVertex(ctx context.Context, project string) (*VertexLLM, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: "us-central1",
		Backend:  genai.BackendVertexAI,
	})

	if err != nil {
		return nil, err
	}

	return &VertexLLM{client: client}, nil
}

func (v *VertexLLM) Generate(ctx context.Context, prompt string) (string, error) {
	resp, err := v.client.Models.GenerateContent(ctx, "gemini-2.0-flash", genai.Text(prompt), nil)
	if err != nil {
		return "", err
	}

	return resp.Candidates[0].Content.Parts[0].Text, nil
}
