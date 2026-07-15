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

const analyzePrompt = `この領収書/請求書/明細書PDFを読み取り、以下の3つの情報を抽出してください。

1. **支払日**: YYYYMMDD形式で抽出してください。
   - 優先順位: Payment Date / Paid Date > Invoice Date / 請求日 > Issue Date / 発行日
   - 月名の変換 (必ず参照): January=01, February=02, March=03, April=04, May=05, June=06,
     July=07, August=08, September=09, October=10, November=11, December=12
   - 変換例: "May 3, 2026" → "20260503" / "January 15, 2025" → "20250115"
   - JSON出力前に「文書内の日付表記 → YYYYMMDD」の変換を確認すること

2. **サービス名/会社名**: 請求元・発行元の会社名またはサービス名。

3. **金額**: この文書自体を代表する金額を通貨記号付きで抽出してください。まずタイトルや冒頭の文言から文書の種類（返金書類か、通常の領収書・請求書か）を判定すること。

   **返金書類の場合**（タイトルが "Refund" / "返金"、または冒頭に "$X refunded on ..." という記載がある）:
   - 優先順位: "Total refunded without credit note" > "Refunded on <日付>" の行の金額 > 冒頭の "$X refunded on ..." の金額
   - この文書内にある "Amount Paid" や "Total" は返金対象となった過去の請求（元の支払い）の金額であり、この返金書類自体の金額ではないため使用しないこと

   **通常の領収書・請求書の場合**:
   - 優先順位: "Amount Paid" / "Total Paid" / "支払い済み金額" / "ご請求額（決済額）" > "Total" / "Amount Due" / "合計" / "請求金額" > "Subtotal" / "小計"
   - "Applied balance", "Credit applied", "適用クレジット" など、クレジット残高や割引が Total から差し引かれている場合、"Total" ではなく差し引き後の実際の支払い金額（Amount Paid）を優先する
   - 文書冒頭に "$676.08 paid on July 3, 2026" のように支払い金額が明記されている場合、それが実際の支払い金額である

   共通:
   - 通貨記号を必ず含める（$, ¥, €, £ など）。記号がない場合は文脈から判断する
   - 桁区切り(カンマ)や小数点はそのまま保持する（例: $1,234.56, ¥10,000）
   - 金額が見つからない場合は空文字列にする

まず文書の種類（返金書類 or 通常の領収書・請求書）を書き出し、次に見つけた日付・金額関連の表記をそのまま書き出す（例: "Document type: Refund" / "Total refunded without credit note: $34.46" / "Invoice Date: May 3, 2026" / "Total: $1,000.19" / "Applied balance: -$324.11" / "Amount paid: $676.08"）。その上でYYYYMMDD形式への変換と、この文書自体の金額の判断を行ってから、以下のJSON形式のみで回答してください：
{"date": "YYYYMMDD", "service": "サービス名", "amount": "$100.00"}`
