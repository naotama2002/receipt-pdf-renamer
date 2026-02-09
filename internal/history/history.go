package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// MaxItems is the maximum number of history items to keep
const MaxItems = 20

// History manages service pattern history
type History struct {
	filePath string
}

// New creates a new History with the default file path
func New() *History {
	return &History{
		filePath: defaultFilePath(),
	}
}

// NewWithPath creates a new History with a custom file path (for testing)
func NewWithPath(filePath string) *History {
	return &History{
		filePath: filePath,
	}
}

func defaultFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "receipt-pdf-renamer", "service_pattern_history.json")
}

// Get returns the service pattern history (most recent first)
func (h *History) Get() []string {
	data, err := os.ReadFile(h.filePath)
	if err != nil {
		return []string{}
	}

	var history []string
	if err := json.Unmarshal(data, &history); err != nil {
		return []string{}
	}

	return history
}

// Add adds a pattern to the history (most recent first, no duplicates)
func (h *History) Add(pattern string) error {
	if pattern == "" {
		return nil
	}

	// Load existing history
	history := h.Get()

	// Remove duplicate if exists and prepend the new pattern
	newHistory := make([]string, 0, len(history)+1)
	newHistory = append(newHistory, pattern)
	for _, p := range history {
		if p != pattern {
			newHistory = append(newHistory, p)
		}
	}

	// Limit history size
	if len(newHistory) > MaxItems {
		newHistory = newHistory[:MaxItems]
	}

	// Save to file
	data, err := json.MarshalIndent(newHistory, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	dir := filepath.Dir(h.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	if err := os.WriteFile(h.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}
