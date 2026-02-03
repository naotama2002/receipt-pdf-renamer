package tui

import (
	"context"
	"path/filepath"
	"regexp"
	"sort"
	"sync"

	"github.com/naotama2002/receipt-pdf-renamer/internal/ai"
	"github.com/naotama2002/receipt-pdf-renamer/internal/cache"
	"github.com/naotama2002/receipt-pdf-renamer/internal/renamer"
)

type ItemStatus int

const (
	StatusPending ItemStatus = iota
	StatusAnalyzing
	StatusReady
	StatusCached
	StatusRenamed
	StatusError
	StatusSkipped // 既にリネーム済みでスキップ
)

type FileItem struct {
	OriginalPath   string
	OriginalName   string
	NewName        string
	Info           *ai.ReceiptInfo
	Status         ItemStatus
	Error          error
	AlreadyRenamed bool // 既にリネーム済みのファイル
}

// ConfigInfo はTUIに表示する設定情報
type ConfigInfo struct {
	ProviderName   string
	Model          string
	MaxWorkers     int
	CacheEnabled   bool
	ServicePattern string // サービス名パターン（中間部分のみ）
}

type Model struct {
	files       []FileItem
	cursor      int
	selected    map[int]bool
	analyzing   bool
	renaming    bool
	done        bool
	err         error
	directory   string
	absPath     string // 絶対パス
	configInfo  ConfigInfo
	provider    ai.Provider
	cache       *cache.Cache
	renamer     *renamer.Renamer
	maxWorkers  int
	mu          sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
	renamedCnt  int
	failedCnt   int
	width       int
	height      int

	// テンプレート編集
	editingTemplate bool
	templateInput   string
	templateCursor  int
	templateSaved   bool   // 保存完了フラグ
	templateError   string // テンプレートエラーメッセージ
}

// YYYYMMDD-{サービス名}-xxx.pdf 形式にマッチする正規表現
var alreadyRenamedPattern = regexp.MustCompile(`^\d{8}-.+-.+\.pdf$`)

func NewModel(
	directory string,
	provider ai.Provider,
	cacheInstance *cache.Cache,
	renamerInstance *renamer.Renamer,
	maxWorkers int,
	configInfo ConfigInfo,
) *Model {
	ctx, cancel := context.WithCancel(context.Background())

	// 絶対パスを取得
	absPath, err := filepath.Abs(directory)
	if err != nil {
		absPath = directory
	}

	return &Model{
		files:      []FileItem{},
		selected:   make(map[int]bool),
		directory:  directory,
		absPath:    absPath,
		configInfo: configInfo,
		provider:   provider,
		cache:      cacheInstance,
		renamer:    renamerInstance,
		maxWorkers: maxWorkers,
		ctx:        ctx,
		cancel:     cancel,
		width:      80,
		height:     24,
	}
}

func (m *Model) ScanDirectory() error {
	pattern := filepath.Join(m.directory, "*.pdf")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	patternUpper := filepath.Join(m.directory, "*.PDF")
	matchesUpper, _ := filepath.Glob(patternUpper)
	matches = append(matches, matchesUpper...)

	seen := make(map[string]bool)
	var unique []string
	for _, path := range matches {
		if !seen[path] {
			seen[path] = true
			unique = append(unique, path)
		}
	}

	m.files = make([]FileItem, len(unique))
	for i, path := range unique {
		filename := filepath.Base(path)
		alreadyRenamed := isAlreadyRenamed(filename)

		status := StatusPending
		if alreadyRenamed {
			status = StatusSkipped
		}

		m.files[i] = FileItem{
			OriginalPath:   path,
			OriginalName:   filename,
			Status:         status,
			AlreadyRenamed: alreadyRenamed,
		}
	}

	// ソート: 未処理を先に、既にリネーム済みを後に
	sort.SliceStable(m.files, func(i, j int) bool {
		if m.files[i].AlreadyRenamed != m.files[j].AlreadyRenamed {
			return !m.files[i].AlreadyRenamed // 未処理が先
		}
		return m.files[i].OriginalName < m.files[j].OriginalName
	})

	// 選択状態を設定（既にリネーム済みは選択しない）
	for i, item := range m.files {
		m.selected[i] = !item.AlreadyRenamed
	}

	return nil
}

// isAlreadyRenamed はファイル名が既にリネーム済みの形式かチェック
func isAlreadyRenamed(filename string) bool {
	return alreadyRenamedPattern.MatchString(filename)
}

func (m *Model) ToggleSelection(index int) {
	if index >= 0 && index < len(m.files) {
		// スキップ状態のファイルは選択できない
		if m.files[index].AlreadyRenamed {
			return
		}
		m.selected[index] = !m.selected[index]
	}
}

func (m *Model) SelectAll() {
	for i := range m.files {
		// スキップ状態のファイルは選択しない
		if !m.files[i].AlreadyRenamed {
			m.selected[i] = true
		}
	}
}

func (m *Model) DeselectAll() {
	for i := range m.files {
		m.selected[i] = false
	}
}

func (m *Model) SelectedCount() int {
	count := 0
	for i, item := range m.files {
		if m.selected[i] && (item.Status == StatusReady || item.Status == StatusCached) {
			count++
		}
	}
	return count
}

func (m *Model) TotalReadyCount() int {
	count := 0
	for _, item := range m.files {
		if item.Status == StatusReady || item.Status == StatusCached {
			count++
		}
	}
	return count
}

func (m *Model) CachedCount() int {
	count := 0
	for _, item := range m.files {
		if item.Status == StatusCached {
			count++
		}
	}
	return count
}

func (m *Model) AnalyzingCount() int {
	count := 0
	for _, item := range m.files {
		if item.Status == StatusAnalyzing {
			count++
		}
	}
	return count
}

func (m *Model) ErrorCount() int {
	count := 0
	for _, item := range m.files {
		if item.Status == StatusError {
			count++
		}
	}
	return count
}

func (m *Model) SkippedCount() int {
	count := 0
	for _, item := range m.files {
		if item.Status == StatusSkipped {
			count++
		}
	}
	return count
}

func (m *Model) PendingCount() int {
	count := 0
	for _, item := range m.files {
		if item.Status == StatusPending {
			count++
		}
	}
	return count
}

// GetExampleFile は解析済みファイルから例を1つ取得する
func (m *Model) GetExampleFile() *FileItem {
	for i := range m.files {
		item := &m.files[i]
		if item.Info != nil && (item.Status == StatusReady || item.Status == StatusCached) {
			return item
		}
	}
	return nil
}
