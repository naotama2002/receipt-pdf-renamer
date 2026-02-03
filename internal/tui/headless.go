package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"sync"

	"github.com/naotama2002/receipt-pdf-renamer/internal/ai"
	"github.com/naotama2002/receipt-pdf-renamer/internal/cache"
	"github.com/naotama2002/receipt-pdf-renamer/internal/renamer"
)

type HeadlessRunner struct {
	directory  string
	provider   ai.Provider
	cache      *cache.Cache
	renamer    *renamer.Renamer
	maxWorkers int
	dryRun     bool
}

type HeadlessResult struct {
	Renamed int
	Failed  int
	Skipped int
	Errors  []error
}

func NewHeadlessRunner(
	directory string,
	provider ai.Provider,
	cacheInstance *cache.Cache,
	renamerInstance *renamer.Renamer,
	maxWorkers int,
	dryRun bool,
) *HeadlessRunner {
	return &HeadlessRunner{
		directory:  directory,
		provider:   provider,
		cache:      cacheInstance,
		renamer:    renamerInstance,
		maxWorkers: maxWorkers,
		dryRun:     dryRun,
	}
}

func (r *HeadlessRunner) Run(ctx context.Context) (*HeadlessResult, error) {
	pattern := filepath.Join(r.directory, "*.pdf")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	patternUpper := filepath.Join(r.directory, "*.PDF")
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

	if len(unique) == 0 {
		fmt.Println("No PDF files found in", r.directory)
		return &HeadlessResult{}, nil
	}

	// 未処理ファイルと既にリネーム済みファイルを分類
	var toProcess []string
	var skipped []string
	for _, path := range unique {
		filename := filepath.Base(path)
		if isAlreadyRenamed(filename) {
			skipped = append(skipped, path)
		} else {
			toProcess = append(toProcess, path)
		}
	}

	// ソート
	sort.Strings(toProcess)
	sort.Strings(skipped)

	fmt.Printf("Scanning %s...\n", r.directory)
	fmt.Printf("Found %d PDF files (%d to process, %d already renamed)\n\n", len(unique), len(toProcess), len(skipped))

	if len(toProcess) == 0 {
		fmt.Println("No files to process.")
		return &HeadlessResult{Skipped: len(skipped)}, nil
	}

	result := &HeadlessResult{Skipped: len(skipped)}
	var mu sync.Mutex

	sem := make(chan struct{}, r.maxWorkers)
	var wg sync.WaitGroup

	for i, path := range toProcess {
		wg.Add(1)

		go func(index int, pdfPath string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			sem <- struct{}{}
			defer func() { <-sem }()

			originalName := filepath.Base(pdfPath)
			fmt.Printf("[%d/%d] %s\n", index+1, len(toProcess), originalName)

			var info *ai.ReceiptInfo
			var cached bool

			if cachedInfo, ok := r.cache.Get(pdfPath); ok {
				info = cachedInfo
				cached = true
			} else {
				var err error
				info, err = r.provider.AnalyzeReceipt(ctx, pdfPath)
				if err != nil {
					mu.Lock()
					result.Failed++
					result.Errors = append(result.Errors, fmt.Errorf("%s: %w", originalName, err))
					mu.Unlock()
					fmt.Printf("      ✗ Error: %s\n\n", err.Error())
					return
				}
				r.cache.Set(pdfPath, info)
			}

			newName, err := r.renamer.GenerateName(pdfPath, info)
			if err != nil {
				mu.Lock()
				result.Failed++
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", originalName, err))
				mu.Unlock()
				fmt.Printf("      ✗ Error: %s\n\n", err.Error())
				return
			}

			if r.dryRun {
				status := ""
				if cached {
					status = " (cached)"
				}
				fmt.Printf("      → %s%s [dry-run]\n\n", newName, status)
				mu.Lock()
				result.Renamed++
				mu.Unlock()
				return
			}

			if err := r.renamer.Rename(pdfPath, newName); err != nil {
				mu.Lock()
				result.Failed++
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", originalName, err))
				mu.Unlock()
				fmt.Printf("      ✗ Error: %s\n\n", err.Error())
				return
			}

			status := ""
			if cached {
				status = " (cached)"
			}
			fmt.Printf("      → %s ✓%s\n\n", newName, status)
			mu.Lock()
			result.Renamed++
			mu.Unlock()
		}(i, path)
	}

	wg.Wait()

	fmt.Println("---")
	if r.dryRun {
		fmt.Printf("Completed (dry-run): %d would be renamed, %d failed, %d skipped\n", result.Renamed, result.Failed, result.Skipped)
	} else {
		fmt.Printf("Completed: %d renamed, %d failed, %d skipped\n", result.Renamed, result.Failed, result.Skipped)
	}

	return result, nil
}
