package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/naotama2002/receipt-pdf-renamer/internal/config"
)

type tickMsg time.Time

type scanCompleteMsg struct{}

type analyzeCompleteMsg struct{}

type renameCompleteMsg struct {
	renamed int
	failed  int
}

type templateSavedMsg struct {
	err error
}

type templateSavedClearMsg struct{}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.scanCmd(),
		m.tickCmd(),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if m.analyzing {
			return m, m.tickCmd()
		}
		return m, nil

	case scanCompleteMsg:
		m.analyzing = true
		return m, tea.Batch(m.analyzeCmd(), m.tickCmd())

	case analyzeCompleteMsg:
		m.analyzing = false
		return m, nil

	case renameCompleteMsg:
		m.renaming = false
		m.renamedCnt = msg.renamed
		m.failedCnt = msg.failed
		m.done = true
		return m, nil

	case templateSavedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.templateSaved = true
			// 2秒後に保存表示を消す
			return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return templateSavedClearMsg{}
			})
		}
		return m, nil

	case templateSavedClearMsg:
		m.templateSaved = false
		return m, nil
	}

	return m, nil
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.done {
		return m, tea.Quit
	}

	// テンプレート編集モード
	if m.editingTemplate {
		return m.handleTemplateEditKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.cancel()
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.files)-1 {
			m.cursor++
		}

	case " ":
		m.ToggleSelection(m.cursor)

	case "a":
		m.SelectAll()

	case "n":
		m.DeselectAll()

	case "t":
		// テンプレート編集モードに入る
		if !m.analyzing && !m.renaming {
			m.editingTemplate = true
			m.templateInput = m.configInfo.ServicePattern
			m.templateCursor = len(m.templateInput)
			m.templateError = ""
		}

	case "enter":
		if !m.analyzing && !m.renaming && m.SelectedCount() > 0 {
			m.renaming = true
			return m, m.renameCmd()
		}
	}

	return m, nil
}

func (m *Model) handleTemplateEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		// 編集をキャンセル
		m.editingTemplate = false
		m.templateInput = ""
		m.templateCursor = 0
		return m, nil

	case "enter":
		// パターンを保存
		newPattern := m.templateInput

		// パターンが変更された場合のみ処理
		if newPattern != m.configInfo.ServicePattern {
			// フルテンプレートを構築して検証
			fullTemplate := config.BuildFullTemplate(newPattern)
			if err := m.renamer.UpdateTemplate(fullTemplate); err != nil {
				// 無効なパターンの場合はエラー表示して編集を継続
				m.templateError = err.Error()
				return m, nil
			}

			// 成功したら編集モードを終了
			m.editingTemplate = false
			m.templateInput = ""
			m.templateCursor = 0
			m.templateError = ""
			m.configInfo.ServicePattern = newPattern

			// ファイル名を再生成
			m.regenerateNewNames()
			// ローカル設定ファイルに保存
			return m, m.saveTemplateCmd(newPattern)
		}

		// 変更がない場合は単に編集モードを終了
		m.editingTemplate = false
		m.templateInput = ""
		m.templateCursor = 0
		m.templateError = ""
		return m, nil

	case "left":
		if m.templateCursor > 0 {
			m.templateCursor--
		}
		return m, nil

	case "right":
		if m.templateCursor < len(m.templateInput) {
			m.templateCursor++
		}
		return m, nil

	case "home", "ctrl+a":
		m.templateCursor = 0
		return m, nil

	case "end", "ctrl+e":
		m.templateCursor = len(m.templateInput)
		return m, nil

	case "backspace":
		if m.templateCursor > 0 {
			m.templateInput = m.templateInput[:m.templateCursor-1] + m.templateInput[m.templateCursor:]
			m.templateCursor--
		}
		return m, nil

	case "delete":
		if m.templateCursor < len(m.templateInput) {
			m.templateInput = m.templateInput[:m.templateCursor] + m.templateInput[m.templateCursor+1:]
		}
		return m, nil

	default:
		// 通常の文字入力
		if len(key) == 1 || key == "." {
			m.templateInput = m.templateInput[:m.templateCursor] + key + m.templateInput[m.templateCursor:]
			m.templateCursor++
		} else if msg.Type == tea.KeyRunes {
			// 日本語などのマルチバイト文字
			runes := msg.Runes
			for _, r := range runes {
				m.templateInput = m.templateInput[:m.templateCursor] + string(r) + m.templateInput[m.templateCursor:]
				m.templateCursor++
			}
		}
		return m, nil
	}
}

func (m *Model) saveTemplateCmd(template string) tea.Cmd {
	return func() tea.Msg {
		err := config.SaveLocalConfig(m.directory, template)
		return templateSavedMsg{err: err}
	}
}

func (m *Model) regenerateNewNames() {
	for i := range m.files {
		item := &m.files[i]
		if item.Info != nil && (item.Status == StatusReady || item.Status == StatusCached) {
			newName, err := m.renamer.GenerateName(item.OriginalPath, item.Info)
			if err != nil {
				item.Status = StatusError
				item.Error = err
			} else {
				item.NewName = newName
			}
		}
	}
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) scanCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.ScanDirectory(); err != nil {
			m.err = err
			return nil
		}
		return scanCompleteMsg{}
	}
}

func (m *Model) analyzeCmd() tea.Cmd {
	return func() tea.Msg {
		m.analyzeFiles()
		return analyzeCompleteMsg{}
	}
}

func (m *Model) analyzeFiles() {
	sem := make(chan struct{}, m.maxWorkers)
	done := make(chan struct{})

	go func() {
		for i := range m.files {
			select {
			case <-m.ctx.Done():
				close(done)
				return
			default:
			}

			// 既にリネーム済みのファイルはスキップ
			m.mu.Lock()
			if m.files[i].AlreadyRenamed {
				m.mu.Unlock()
				continue
			}
			m.mu.Unlock()

			sem <- struct{}{}

			go func(index int) {
				defer func() { <-sem }()

				m.mu.Lock()
				item := m.files[index]
				m.mu.Unlock()

				if info, ok := m.cache.Get(item.OriginalPath); ok {
					newName, err := m.renamer.GenerateName(item.OriginalPath, info)
					if err != nil {
						item.Status = StatusError
						item.Error = err
					} else {
						item.Info = info
						item.NewName = newName
						item.Status = StatusCached
					}
				} else {
					m.mu.Lock()
					m.files[index].Status = StatusAnalyzing
					m.mu.Unlock()

					info, err := m.provider.AnalyzeReceipt(m.ctx, item.OriginalPath)
					if err != nil {
						item.Status = StatusError
						item.Error = err
					} else {
						m.cache.Set(item.OriginalPath, info)
						newName, err := m.renamer.GenerateName(item.OriginalPath, info)
						if err != nil {
							item.Status = StatusError
							item.Error = err
						} else {
							item.Info = info
							item.NewName = newName
							item.Status = StatusReady
						}
					}
				}

				m.mu.Lock()
				m.files[index] = item
				m.mu.Unlock()
			}(i)
		}

		for i := 0; i < m.maxWorkers; i++ {
			sem <- struct{}{}
		}

		close(done)
	}()

	<-done
}

func (m *Model) renameCmd() tea.Cmd {
	return func() tea.Msg {
		renamed := 0
		failed := 0

		for i, item := range m.files {
			if !m.selected[i] {
				continue
			}
			if item.Status != StatusReady && item.Status != StatusCached {
				continue
			}

			if err := m.renamer.Rename(item.OriginalPath, item.NewName); err != nil {
				m.files[i].Status = StatusError
				m.files[i].Error = err
				failed++
			} else {
				m.files[i].Status = StatusRenamed
				renamed++
			}
		}

		return renameCompleteMsg{renamed: renamed, failed: failed}
	}
}
