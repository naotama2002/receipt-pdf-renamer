package ai

import (
	"context"
	"fmt"

	"github.com/naotama2002/receipt-pdf-renamer/internal/config"
)

type ReceiptInfo struct {
	Date    string `json:"date"`
	Service string `json:"service"`
}

type Provider interface {
	AnalyzeReceipt(ctx context.Context, pdfPath string) (*ReceiptInfo, error)
	Name() string
}

func NewProvider(cfg *config.AIConfig) (Provider, error) {
	switch cfg.Provider {
	case "anthropic":
		return NewAnthropicProvider(cfg)
	case "openai":
		return NewOpenAIProvider(cfg)
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}
