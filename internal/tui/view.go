package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	analyzingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	cachedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33"))

	skippedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Padding(0, 1)
)

var (
	editStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true)

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("238"))
)

func (m *Model) View() string {
	if m.done {
		return m.viewDone()
	}

	if m.err != nil {
		return m.viewError()
	}

	// テンプレート編集モード
	if m.editingTemplate {
		return m.viewTemplateEdit()
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("receipt-pdf-renamer"))
	b.WriteString("\n\n")

	// 設定情報を表示
	b.WriteString(m.renderConfig())
	b.WriteString("\n")

	// ディレクトリ（絶対パス）
	b.WriteString(fmt.Sprintf("  %s %s\n", dimStyle.Render("Directory:"), m.absPath))

	statusParts := []string{
		fmt.Sprintf("%d files", len(m.files)),
	}
	if pending := m.PendingCount(); pending > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d pending", pending))
	}
	if cached := m.CachedCount(); cached > 0 {
		statusParts = append(statusParts, cachedStyle.Render(fmt.Sprintf("%d cached", cached)))
	}
	if analyzing := m.AnalyzingCount(); analyzing > 0 {
		statusParts = append(statusParts, analyzingStyle.Render(fmt.Sprintf("%d analyzing", analyzing)))
	}
	if errCnt := m.ErrorCount(); errCnt > 0 {
		statusParts = append(statusParts, errorStyle.Render(fmt.Sprintf("%d errors", errCnt)))
	}
	if skipped := m.SkippedCount(); skipped > 0 {
		statusParts = append(statusParts, skippedStyle.Render(fmt.Sprintf("%d skipped", skipped)))
	}
	b.WriteString(fmt.Sprintf("  %s %s\n\n", dimStyle.Render("Status:"), strings.Join(statusParts, ", ")))

	visibleStart := 0
	visibleEnd := len(m.files)
	maxVisible := m.height - 14 // 設定情報+テンプレート表示分
	if maxVisible < 5 {
		maxVisible = 5
	}

	if len(m.files) > maxVisible {
		if m.cursor >= maxVisible/2 {
			visibleStart = m.cursor - maxVisible/2
		}
		visibleEnd = visibleStart + maxVisible
		if visibleEnd > len(m.files) {
			visibleEnd = len(m.files)
			visibleStart = visibleEnd - maxVisible
			if visibleStart < 0 {
				visibleStart = 0
			}
		}
	}

	for i := visibleStart; i < visibleEnd; i++ {
		b.WriteString(m.renderFileItem(i))
		b.WriteString("\n")
	}

	if len(m.files) == 0 {
		b.WriteString(dimStyle.Render("  No PDF files found in this directory.\n"))
	}

	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

func (m *Model) renderFileItem(index int) string {
	item := m.files[index]
	isCursor := index == m.cursor
	isSelected := m.selected[index]

	var checkbox string
	if item.AlreadyRenamed {
		// スキップ状態のファイルは選択不可を示す
		checkbox = skippedStyle.Render("[-]")
	} else if isSelected {
		checkbox = selectedStyle.Render("[x]")
	} else {
		checkbox = dimStyle.Render("[ ]")
	}

	cursor := "  "
	if isCursor {
		cursor = cursorStyle.Render("> ")
	}

	var status string
	var nameStyle lipgloss.Style
	switch item.Status {
	case StatusPending:
		status = dimStyle.Render("pending")
		nameStyle = lipgloss.NewStyle()
	case StatusAnalyzing:
		status = analyzingStyle.Render("analyzing...")
		nameStyle = lipgloss.NewStyle()
	case StatusReady:
		status = successStyle.Render("ready")
		nameStyle = lipgloss.NewStyle()
	case StatusCached:
		status = cachedStyle.Render("cached")
		nameStyle = lipgloss.NewStyle()
	case StatusRenamed:
		status = successStyle.Render("renamed")
		nameStyle = lipgloss.NewStyle()
	case StatusError:
		status = errorStyle.Render("error")
		nameStyle = lipgloss.NewStyle()
	case StatusSkipped:
		status = skippedStyle.Render("already renamed")
		nameStyle = skippedStyle
	}

	filename := item.OriginalName
	if item.Status == StatusSkipped {
		filename = nameStyle.Render(filename)
	}

	line := fmt.Sprintf("%s%s %s  %s", cursor, checkbox, filename, status)

	if item.NewName != "" && (item.Status == StatusReady || item.Status == StatusCached) {
		line += "\n" + dimStyle.Render(fmt.Sprintf("       → %s", item.NewName))
	}

	if item.Status == StatusError && item.Error != nil {
		line += "\n" + errorStyle.Render(fmt.Sprintf("       ✗ %s", item.Error.Error()))
	}

	return line
}

func (m *Model) renderConfig() string {
	var b strings.Builder

	// 1行目: Provider, Model, Workers, Cache
	parts := []string{
		fmt.Sprintf("%s %s", dimStyle.Render("Provider:"), m.configInfo.ProviderName),
		fmt.Sprintf("%s %s", dimStyle.Render("Model:"), m.configInfo.Model),
		fmt.Sprintf("%s %d", dimStyle.Render("Workers:"), m.configInfo.MaxWorkers),
	}

	cacheStatus := successStyle.Render("on")
	if !m.configInfo.CacheEnabled {
		cacheStatus = dimStyle.Render("off")
	}
	parts = append(parts, fmt.Sprintf("%s %s", dimStyle.Render("Cache:"), cacheStatus))

	b.WriteString("  " + strings.Join(parts, "  "))
	b.WriteString("\n")

	// 2行目: Format - フォーマットパターンを表示
	b.WriteString(fmt.Sprintf("  %s YYYYMMDD-%s-{{.OriginalName}}.pdf",
		dimStyle.Render("Format:"),
		m.configInfo.ServicePattern))

	if m.templateSaved {
		b.WriteString(successStyle.Render(" ✓"))
	}
	b.WriteString(" ")
	b.WriteString(dimStyle.Render("(t to edit)"))

	return b.String()
}

func (m *Model) renderHelp() string {
	if m.analyzing {
		return helpStyle.Render("  Analyzing... Press q to cancel")
	}

	if m.renaming {
		return helpStyle.Render("  Renaming files...")
	}

	parts := []string{
		"↑/↓ navigate",
		"space select",
		"a all",
		"n none",
		"t template",
	}

	if m.SelectedCount() > 0 {
		parts = append(parts, fmt.Sprintf("enter rename (%d)", m.SelectedCount()))
	}

	parts = append(parts, "q quit")

	return helpStyle.Render("  " + strings.Join(parts, " | "))
}

func (m *Model) viewTemplateEdit() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("receipt-pdf-renamer"))
	b.WriteString(" - ")
	b.WriteString(editStyle.Render("Edit Format"))
	b.WriteString("\n\n")

	// 解析済みファイルから例を取得
	example := m.GetExampleFile()

	// 説明
	b.WriteString(dimStyle.Render("  Format: YYYYMMDD-{pattern}-original.pdf"))
	b.WriteString("\n")
	if example != nil && example.Info != nil {
		b.WriteString(fmt.Sprintf("  %s {{.Service}} → %s",
			dimStyle.Render("Variable:"),
			successStyle.Render(example.Info.Service)))
	} else {
		b.WriteString(dimStyle.Render("  Variable: {{.Service}} = service name from receipt"))
	}
	b.WriteString("\n\n")

	// パターン入力欄（カーソル付き）
	b.WriteString(fmt.Sprintf("  %s\n", dimStyle.Render("Pattern:")))

	input := m.templateInput
	cursorPos := m.templateCursor
	if cursorPos > len(input) {
		cursorPos = len(input)
	}

	before := input[:cursorPos]
	after := input[cursorPos:]
	b.WriteString(fmt.Sprintf("  %s%s%s%s%s\n",
		dimStyle.Render("YYYYMMDD-"),
		inputStyle.Render(before),
		cursorStyle.Render("|"),
		inputStyle.Render(after),
		dimStyle.Render("-original.pdf")))

	// エラーがあれば表示
	if m.templateError != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %s", m.templateError)))
		b.WriteString("\n")
	}

	// プレビュー - 実際の値で表示
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Preview: "))
	if example != nil && example.Info != nil {
		// 実際のサービス名を使ってプレビュー
		previewPattern := m.templateInput
		previewPattern = strings.ReplaceAll(previewPattern, "{{.Service}}", example.Info.Service)
		previewName := fmt.Sprintf("%s-%s-%s.pdf",
			example.Info.Date,
			previewPattern,
			strings.TrimSuffix(example.OriginalName, ".pdf"))
		b.WriteString(previewName)
	} else {
		b.WriteString("20250101-" + m.templateInput + "-receipt.pdf")
	}
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("  enter save | esc cancel | ←/→ move cursor"))

	return b.String()
}

func (m *Model) viewDone() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("receipt-pdf-renamer"))
	b.WriteString("\n\n")

	if m.renamedCnt > 0 {
		b.WriteString(successStyle.Render(fmt.Sprintf("  ✓ %d file(s) renamed successfully\n", m.renamedCnt)))
	}

	if m.failedCnt > 0 {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  ✗ %d file(s) failed\n", m.failedCnt)))
	}

	if m.renamedCnt == 0 && m.failedCnt == 0 {
		b.WriteString(dimStyle.Render("  No files were renamed.\n"))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  Press any key to exit"))

	return b.String()
}

func (m *Model) viewError() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("receipt-pdf-renamer"))
	b.WriteString("\n\n")

	b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %s\n", m.err.Error())))

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  Press any key to exit"))

	return b.String()
}
