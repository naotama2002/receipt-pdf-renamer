package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/naotama2002/receipt-pdf-renamer/internal/config"
	"github.com/naotama2002/receipt-pdf-renamer/internal/pdf"
)

// OpenAICompatibleProvider はOpenAI互換APIを使うプロバイダー（Ollama, LM Studio等）
type OpenAICompatibleProvider struct {
	baseURL   string
	apiKey    string
	model     string
	extractor *pdf.Extractor
	client    *http.Client
}

// NewOpenAICompatibleProvider は OpenAICompatibleProvider を生成する
func NewOpenAICompatibleProvider(cfg *config.AIConfig) (*OpenAICompatibleProvider, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base_url is required for openai-compatible provider")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required for openai-compatible provider")
	}

	return &OpenAICompatibleProvider{
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:    cfg.APIKey,
		model:     cfg.Model,
		extractor: pdf.NewExtractor(cfg.PdfToTextPath),
		client:    &http.Client{},
	}, nil
}

func (p *OpenAICompatibleProvider) Name() string {
	return "OpenAI Compatible"
}

func (p *OpenAICompatibleProvider) AnalyzeReceipt(ctx context.Context, pdfPath string) (*ReceiptInfo, error) {
	// pdftotext でテキスト抽出
	text, err := p.extractor.ExtractText(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text from PDF: %w", err)
	}

	// テキスト + プロンプトをLLMに送信
	prompt := fmt.Sprintf("以下は領収書/請求書PDFから抽出されたテキストです:\n\n---\n%s\n---\n\n%s", text, analyzePrompt)

	responseText, err := p.callChatCompletion(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return parseTextResponse(responseText)
}

// openaiChatRequest はOpenAI chat/completions APIのリクエスト形式
type openaiChatRequest struct {
	Model    string              `json:"model"`
	Messages []openaiChatMessage `json:"messages"`
}

type openaiChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiChatResponse はOpenAI chat/completions APIのレスポンス形式
type openaiChatResponse struct {
	Choices []openaiChoice `json:"choices"`
	Error   *openaiError   `json:"error,omitempty"`
}

type openaiChoice struct {
	Message openaiChatMessage `json:"message"`
}

type openaiError struct {
	Message string `json:"message"`
}

func (p *OpenAICompatibleProvider) callChatCompletion(ctx context.Context, userMessage string) (string, error) {
	reqBody := openaiChatRequest{
		Model: p.model,
		Messages: []openaiChatMessage{
			{Role: "user", Content: userMessage},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI-compatible API: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI-compatible API returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var chatResp openaiChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("OpenAI-compatible API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenAI-compatible API")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// parseTextResponse はLLMのテキスト応答からJSONを抽出してReceiptInfoにパースする
func parseTextResponse(text string) (*ReceiptInfo, error) {
	if text == "" {
		return nil, fmt.Errorf("empty response from API")
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
