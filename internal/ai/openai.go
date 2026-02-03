package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/naotama2002/receipt-pdf-renamer/internal/config"
	"github.com/naotama2002/receipt-pdf-renamer/internal/pdf"
	"github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client    *openai.Client
	model     string
	converter *pdf.Converter
	baseURL   string
}

func NewOpenAIProvider(cfg *config.AIConfig) (*OpenAIProvider, error) {
	converter := pdf.NewConverter()
	if !converter.IsAvailable() {
		return nil, fmt.Errorf("OpenAI provider requires poppler for PDF conversion: install with 'brew install poppler'")
	}

	clientConfig := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientConfig.BaseURL = cfg.BaseURL
	}

	client := openai.NewClientWithConfig(clientConfig)

	return &OpenAIProvider{
		client:    client,
		model:     cfg.Model,
		converter: converter,
		baseURL:   cfg.BaseURL,
	}, nil
}

func (p *OpenAIProvider) Name() string {
	if p.baseURL != "" {
		return fmt.Sprintf("OpenAI-compatible (%s)", p.baseURL)
	}
	return "OpenAI"
}

func (p *OpenAIProvider) AnalyzeReceipt(ctx context.Context, pdfPath string) (*ReceiptInfo, error) {
	imageData, err := p.converter.ToImage(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to convert PDF to image: %w", err)
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:image/png;base64,%s", base64Image)

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: p.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL:    dataURL,
							Detail: openai.ImageURLDetailAuto,
						},
					},
					{
						Type: openai.ChatMessagePartTypeText,
						Text: analyzePrompt,
					},
				},
			},
		},
		MaxTokens: 1024,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	text := resp.Choices[0].Message.Content

	return parseOpenAIResponse(text)
}

func parseOpenAIResponse(text string) (*ReceiptInfo, error) {
	jsonStart := -1
	jsonEnd := -1
	for i, c := range text {
		if c == '{' && jsonStart == -1 {
			jsonStart = i
		}
		if c == '}' {
			jsonEnd = i + 1
		}
	}

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON found in response: %s", text)
	}

	jsonStr := text[jsonStart:jsonEnd]

	var info ReceiptInfo
	if err := json.Unmarshal([]byte(jsonStr), &info); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w, response: %s", err, text)
	}

	return &info, nil
}
