package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/naotama2002/receipt-pdf-renamer/internal/config"
)

type AnthropicProvider struct {
	client *anthropic.Client
	model  string
}

func NewAnthropicProvider(cfg *config.AIConfig) (*AnthropicProvider, error) {
	client := anthropic.NewClient(option.WithAPIKey(cfg.APIKey))

	return &AnthropicProvider{
		client: &client,
		model:  cfg.Model,
	}, nil
}

func (p *AnthropicProvider) Name() string {
	return "Anthropic Claude"
}

func (p *AnthropicProvider) AnalyzeReceipt(ctx context.Context, pdfPath string) (*ReceiptInfo, error) {
	pdfData, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF file: %w", err)
	}

	base64PDF := base64.StdEncoding.EncodeToString(pdfData)

	message, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewDocumentBlock(anthropic.Base64PDFSourceParam{
					Data: base64PDF,
				}),
				anthropic.NewTextBlock(analyzePrompt),
			),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call Anthropic API: %w", err)
	}

	return parseResponse(message)
}

func parseResponse(message *anthropic.Message) (*ReceiptInfo, error) {
	if len(message.Content) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	text := ""
	for _, block := range message.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}

	if text == "" {
		return nil, fmt.Errorf("no text response from API")
	}

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

const analyzePrompt = `この領収書/請求書から以下の情報を抽出してください：
1. 支払日（Paid date / Invoice date / Date）をYYYYMMDD形式で
2. サービス名/会社名

必ず以下のJSON形式のみで回答してください。説明文は不要です：
{"date": "YYYYMMDD", "service": "サービス名"}`
