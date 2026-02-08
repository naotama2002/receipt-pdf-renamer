package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/naotama2002/receipt-pdf-renamer/internal/ai"
	"github.com/naotama2002/receipt-pdf-renamer/internal/cache"
	"github.com/naotama2002/receipt-pdf-renamer/internal/config"
	"github.com/naotama2002/receipt-pdf-renamer/internal/renamer"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/zalando/go-keyring"
)

// isAlreadyRenamed checks if the filename matches the renamed pattern (YYYYMMDD-xxx-xxx.pdf)
var renamedPattern = regexp.MustCompile(`^\d{8}-.+-.+\.pdf$`)

func isAlreadyRenamed(filename string) bool {
	return renamedPattern.MatchString(filename)
}

// ItemStatus はファイルの処理状態を表す
type ItemStatus string

const (
	StatusPending   ItemStatus = "pending"
	StatusAnalyzing ItemStatus = "analyzing"
	StatusReady     ItemStatus = "ready"
	StatusCached    ItemStatus = "cached"
	StatusRenamed   ItemStatus = "renamed"
	StatusError     ItemStatus = "error"
	StatusSkipped   ItemStatus = "skipped"
)

// FileItem はファイルの情報と状態を保持
type FileItem struct {
	ID             int        `json:"id"`
	OriginalPath   string     `json:"originalPath"`
	OriginalName   string     `json:"originalName"`
	NewName        string     `json:"newName"`
	Date           string     `json:"date"`
	Service        string     `json:"service"`
	Status         ItemStatus `json:"status"`
	Error          string     `json:"error"`
	Selected       bool       `json:"selected"`
	AlreadyRenamed bool       `json:"alreadyRenamed"`
}

// ConfigInfo は設定情報をフロントエンドに渡すためのDTO
type ConfigInfo struct {
	ProviderName          string `json:"providerName"`
	Model                 string `json:"model"`
	CacheEnabled          bool   `json:"cacheEnabled"`
	ServicePattern        string `json:"servicePattern"`
	ServicePatternIsEmpty bool   `json:"servicePatternIsEmpty"`
}

// RenameResult はリネーム結果
type RenameResult struct {
	TotalCount   int `json:"totalCount"`
	RenamedCount int `json:"renamedCount"`
	ErrorCount   int `json:"errorCount"`
	SkippedCount int `json:"skippedCount"`
}

// App はWailsアプリケーションの構造体
type App struct {
	ctx      context.Context
	config   *config.Config
	provider ai.Provider
	cache    *cache.Cache
	renamer  *renamer.Renamer

	files []FileItem
	mu    sync.RWMutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		files: make([]FileItem, 0),
	}
}

// Startup is called when the app starts
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	_ = a.initializeServices() // エラーは起動時なのでログ出力のみで続行
}

// DomReady is called when the DOM is ready
func (a *App) DomReady(ctx context.Context) {
	// Process command line arguments (for Windows context menu)
	args := os.Args[1:]
	if len(args) > 0 {
		var pdfFiles []string
		for _, arg := range args {
			if strings.HasSuffix(strings.ToLower(arg), ".pdf") {
				pdfFiles = append(pdfFiles, arg)
			}
		}
		if len(pdfFiles) > 0 {
			a.AddFiles(pdfFiles)
			runtime.EventsEmit(a.ctx, "files-updated", a.GetFiles())
		}
	}
}

// Shutdown is called when the app is shutting down
func (a *App) Shutdown(ctx context.Context) {
}

// OnFileOpen is called when a file is opened via "Open With" on macOS
func (a *App) OnFileOpen(filePath string) {
	// Add the file to the list
	a.AddFiles([]string{filePath})

	// Emit event to update the frontend
	runtime.EventsEmit(a.ctx, "files-updated", a.GetFiles())
}

func (a *App) initializeServices() error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	a.config = cfg

	provider, err := ai.NewProvider(&cfg.AI)
	if err != nil {
		return fmt.Errorf("failed to create AI provider: %w", err)
	}
	a.provider = provider

	cacheInstance, err := cache.New(&cfg.Cache)
	if err != nil {
		return fmt.Errorf("failed to create cache: %w", err)
	}
	a.cache = cacheInstance

	renamerInstance, err := renamer.New(&cfg.Format)
	if err != nil {
		return fmt.Errorf("failed to create renamer: %w", err)
	}
	a.renamer = renamerInstance

	return nil
}

// GetConfig returns the current configuration
func (a *App) GetConfig() ConfigInfo {
	if a.config == nil {
		return ConfigInfo{ServicePatternIsEmpty: true}
	}

	return ConfigInfo{
		ProviderName:          a.config.ProviderDisplayName(),
		Model:                 a.config.AI.Model,
		CacheEnabled:          a.config.Cache.Enabled,
		ServicePattern:        a.config.Format.ServicePattern,
		ServicePatternIsEmpty: a.config.Format.ServicePattern == "",
	}
}

// HasAPIKey checks if an API key is configured
func (a *App) HasAPIKey() bool {
	return a.config != nil && a.config.AI.APIKey != ""
}

// AddFiles adds PDF files to the list
func (a *App) AddFiles(paths []string) []FileItem {
	a.mu.Lock()
	defer a.mu.Unlock()

	startID := len(a.files)
	for i, path := range paths {
		if !strings.HasSuffix(strings.ToLower(path), ".pdf") {
			continue
		}

		// 重複チェック
		duplicate := false
		for _, f := range a.files {
			if f.OriginalPath == path {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}

		filename := filepath.Base(path)
		alreadyRenamed := isAlreadyRenamed(filename)

		item := FileItem{
			ID:             startID + i,
			OriginalPath:   path,
			OriginalName:   filename,
			Status:         StatusPending,
			Selected:       !alreadyRenamed, // 既にリネーム済みならデフォルト非選択
			AlreadyRenamed: alreadyRenamed,
		}

		// 既にリネーム済みならスキップ状態にする
		if alreadyRenamed {
			item.Status = StatusSkipped
			item.Error = "既にリネーム済みの形式です"
		}

		a.files = append(a.files, item)
	}

	return a.files
}

// GetFiles returns all files
func (a *App) GetFiles() []FileItem {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.files
}

// ClearFiles clears all files
func (a *App) ClearFiles() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.files = make([]FileItem, 0)
}

// ToggleFileSelection toggles the selection of a file
func (a *App) ToggleFileSelection(id int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.files {
		if a.files[i].ID == id {
			a.files[i].Selected = !a.files[i].Selected
			break
		}
	}
}

// SelectAll selects all files
func (a *App) SelectAll() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.files {
		if a.files[i].Status == StatusReady || a.files[i].Status == StatusCached {
			a.files[i].Selected = true
		}
	}
}

// DeselectAll deselects all files
func (a *App) DeselectAll() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.files {
		a.files[i].Selected = false
	}
}

// AnalyzeFiles analyzes the selected files
func (a *App) AnalyzeFiles() {
	go a.analyzeFilesAsync()
}

func (a *App) analyzeFilesAsync() {
	a.mu.Lock()
	filesToAnalyze := make([]int, 0)
	for i, f := range a.files {
		if f.Status == StatusPending {
			a.files[i].Status = StatusAnalyzing
			filesToAnalyze = append(filesToAnalyze, i)
		}
	}
	a.mu.Unlock()

	// Emit event to update UI
	runtime.EventsEmit(a.ctx, "files-updated", a.GetFiles())

	// Worker pool
	maxWorkers := a.config.AI.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 3
	}

	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for _, idx := range filesToAnalyze {
		wg.Add(1)
		go func(fileIdx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			a.analyzeFile(fileIdx)
			runtime.EventsEmit(a.ctx, "files-updated", a.GetFiles())
		}(idx)
	}

	wg.Wait()
	runtime.EventsEmit(a.ctx, "analysis-complete", a.GetFiles())
}

func (a *App) analyzeFile(idx int) {
	a.mu.RLock()
	file := a.files[idx]
	a.mu.RUnlock()

	// Check cache first
	if a.cache != nil {
		if info, found := a.cache.Get(file.OriginalPath); found {
			newName, err := a.renamer.GenerateName(file.OriginalPath, info)
			if err == nil {
				a.mu.Lock()
				a.files[idx].Date = info.Date
				a.files[idx].Service = info.Service
				a.files[idx].NewName = newName
				a.files[idx].Status = StatusCached
				a.mu.Unlock()
				return
			}
		}
	}

	// Analyze with AI
	info, err := a.provider.AnalyzeReceipt(a.ctx, file.OriginalPath)
	if err != nil {
		a.mu.Lock()
		a.files[idx].Status = StatusError
		a.files[idx].Error = err.Error()
		a.mu.Unlock()
		return
	}

	// Save to cache
	if a.cache != nil {
		_ = a.cache.Set(file.OriginalPath, info) // キャッシュ保存エラーは無視
	}

	// Generate new name
	newName, err := a.renamer.GenerateName(file.OriginalPath, info)
	if err != nil {
		a.mu.Lock()
		a.files[idx].Status = StatusError
		a.files[idx].Error = err.Error()
		a.mu.Unlock()
		return
	}

	a.mu.Lock()
	a.files[idx].Date = info.Date
	a.files[idx].Service = info.Service
	a.files[idx].NewName = newName
	a.files[idx].Status = StatusReady
	a.mu.Unlock()
}

// RenameFiles renames selected files
func (a *App) RenameFiles() RenameResult {
	a.mu.Lock()
	defer a.mu.Unlock()

	result := RenameResult{}

	for i := range a.files {
		if !a.files[i].Selected {
			continue
		}
		if a.files[i].Status != StatusReady && a.files[i].Status != StatusCached {
			continue
		}

		result.TotalCount++

		// Skip if already renamed
		if a.files[i].OriginalName == a.files[i].NewName {
			a.files[i].Status = StatusSkipped
			result.SkippedCount++
			continue
		}

		err := a.renamer.Rename(a.files[i].OriginalPath, a.files[i].NewName)
		if err != nil {
			a.files[i].Status = StatusError
			a.files[i].Error = err.Error()
			result.ErrorCount++
			continue
		}

		a.files[i].Status = StatusRenamed
		result.RenamedCount++
	}

	runtime.EventsEmit(a.ctx, "files-updated", a.files)
	return result
}

// UpdateServicePattern updates the service pattern template
func (a *App) UpdateServicePattern(pattern string) error {
	fullTemplate := config.BuildFullTemplate(pattern)
	if err := config.ValidateTemplate(fullTemplate); err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}

	if err := a.renamer.UpdateTemplate(fullTemplate); err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	a.config.Format.ServicePattern = pattern
	a.config.Format.Template = fullTemplate

	// Regenerate names for all ready files
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.files {
		if a.files[i].Status == StatusReady || a.files[i].Status == StatusCached {
			info := &ai.ReceiptInfo{
				Date:    a.files[i].Date,
				Service: a.files[i].Service,
			}
			newName, err := a.renamer.GenerateName(a.files[i].OriginalPath, info)
			if err == nil {
				a.files[i].NewName = newName
			}
		}
	}

	return nil
}

// OpenFileDialog opens a file dialog to select PDF files
func (a *App) OpenFileDialog() ([]string, error) {
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "PDFファイルを選択",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "PDF Files",
				Pattern:     "*.pdf",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// OpenFolderDialog opens a folder dialog
func (a *App) OpenFolderDialog() (string, error) {
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "フォルダを選択",
	})
	if err != nil {
		return "", err
	}
	return folder, nil
}

// ScanFolder scans a folder for PDF files
func (a *App) ScanFolder(folderPath string) ([]string, error) {
	var pdfFiles []string

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".pdf") {
			pdfFiles = append(pdfFiles, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return pdfFiles, nil
}

// ClearCache clears the analysis cache
func (a *App) ClearCache() error {
	if a.cache == nil {
		return nil
	}
	return a.cache.Clear()
}

// GetCacheCount returns the number of cached items
func (a *App) GetCacheCount() int {
	if a.cache == nil {
		return 0
	}
	count, _ := a.cache.Count()
	return count
}

const keyringService = "receipt-pdf-renamer"

// SaveAPIKey saves the API key to the system keyring
func (a *App) SaveAPIKey(provider, apiKey string) error {
	keyName := provider + "-api-key"
	if err := keyring.Set(keyringService, keyName, apiKey); err != nil {
		return fmt.Errorf("failed to save API key: %w", err)
	}

	// Update config and reinitialize provider
	a.config.AI.Provider = provider
	a.config.AI.APIKey = apiKey
	if a.config.AI.Model == "" {
		switch provider {
		case "anthropic":
			a.config.AI.Model = "claude-sonnet-4-20250514"
		case "openai":
			a.config.AI.Model = "gpt-4o"
		}
	}

	newProvider, err := ai.NewProvider(&a.config.AI)
	if err != nil {
		return fmt.Errorf("failed to create AI provider: %w", err)
	}
	a.provider = newProvider

	return nil
}

// GetAPIKey retrieves the API key from the system keyring
func (a *App) GetAPIKey(provider string) (string, error) {
	keyName := provider + "-api-key"
	secret, err := keyring.Get(keyringService, keyName)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", nil
		}
		return "", fmt.Errorf("failed to get API key: %w", err)
	}
	return secret, nil
}

// DeleteAPIKey removes the API key from the system keyring
func (a *App) DeleteAPIKey(provider string) error {
	keyName := provider + "-api-key"
	if err := keyring.Delete(keyringService, keyName); err != nil {
		if err == keyring.ErrNotFound {
			return nil
		}
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	return nil
}

// SettingsInfo contains settings for the settings dialog
type SettingsInfo struct {
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	HasAPIKey      bool   `json:"hasApiKey"`
	CacheEnabled   bool   `json:"cacheEnabled"`
	CacheCount     int    `json:"cacheCount"`
	ServicePattern string `json:"servicePattern"`
}

// GetSettings returns current settings
func (a *App) GetSettings() SettingsInfo {
	if a.config == nil {
		return SettingsInfo{}
	}

	hasKey := a.config.AI.APIKey != ""

	return SettingsInfo{
		Provider:       a.config.AI.Provider,
		Model:          a.config.AI.Model,
		HasAPIKey:      hasKey,
		CacheEnabled:   a.config.Cache.Enabled,
		CacheCount:     a.GetCacheCount(),
		ServicePattern: a.config.Format.ServicePattern,
	}
}

// SaveSettings saves settings
func (a *App) SaveSettings(provider, model, servicePattern string) error {
	// Update provider if changed
	if provider != a.config.AI.Provider || model != a.config.AI.Model {
		a.config.AI.Provider = provider
		a.config.AI.Model = model

		// Try to get API key from keyring
		apiKey, _ := a.GetAPIKey(provider)
		if apiKey != "" {
			a.config.AI.APIKey = apiKey
			newProvider, err := ai.NewProvider(&a.config.AI)
			if err != nil {
				return fmt.Errorf("failed to create AI provider: %w", err)
			}
			a.provider = newProvider
		}
	}

	// Update service pattern if changed
	if servicePattern != a.config.Format.ServicePattern {
		if err := a.UpdateServicePattern(servicePattern); err != nil {
			return err
		}
	}

	return nil
}

// GetAvailableModels returns available models for a provider
func (a *App) GetAvailableModels(provider string) []string {
	switch provider {
	case "anthropic":
		return []string{
			"claude-sonnet-4-20250514",
		}
	case "openai":
		// OpenAI is for local LLM, so no preset models
		return []string{}
	default:
		return []string{}
	}
}

// GetBaseURL returns the current base URL for OpenAI-compatible API
func (a *App) GetBaseURL() string {
	if a.config == nil {
		return ""
	}
	return a.config.AI.BaseURL
}

// SaveSettingsWithEndpoint saves settings including endpoint
func (a *App) SaveSettingsWithEndpoint(provider, model, baseURL, servicePattern string) error {
	// Update provider settings
	a.config.AI.Provider = provider
	a.config.AI.Model = model
	a.config.AI.BaseURL = baseURL

	// Try to get API key from keyring
	apiKey, _ := a.GetAPIKey(provider)
	if apiKey != "" {
		a.config.AI.APIKey = apiKey
	}

	// Reinitialize provider if we have an API key
	if a.config.AI.APIKey != "" {
		newProvider, err := ai.NewProvider(&a.config.AI)
		if err != nil {
			return fmt.Errorf("failed to create AI provider: %w", err)
		}
		a.provider = newProvider
	}

	// Update service pattern if changed
	if servicePattern != a.config.Format.ServicePattern {
		if err := a.UpdateServicePattern(servicePattern); err != nil {
			return err
		}
	}

	// Save settings to config file
	if err := a.config.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}
