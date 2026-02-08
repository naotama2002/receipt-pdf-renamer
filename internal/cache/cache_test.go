package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/naotama2002/receipt-pdf-renamer/internal/ai"
	"github.com/naotama2002/receipt-pdf-renamer/internal/config"
)

func setupTestCache(t *testing.T, enabled bool, ttl int) (*Cache, string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "cache_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cacheDir := filepath.Join(tmpDir, "analysis")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}

	cache := &Cache{
		dir:     cacheDir,
		enabled: enabled,
		ttl:     ttl,
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return cache, tmpDir, cleanup
}

func createTestPDF(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test PDF: %v", err)
	}
	return path
}

func TestCache_SetAndGet(t *testing.T) {
	cache, tmpDir, cleanup := setupTestCache(t, true, 0)
	defer cleanup()

	pdfPath := createTestPDF(t, tmpDir, "test.pdf", "test content")
	info := &ai.ReceiptInfo{
		Date:    "20250115",
		Service: "TestService",
	}

	// Set
	err := cache.Set(pdfPath, info)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get
	got, found := cache.Get(pdfPath)
	if !found {
		t.Error("Get() found = false, want true")
	}
	if got.Date != info.Date {
		t.Errorf("Date = %q, want %q", got.Date, info.Date)
	}
	if got.Service != info.Service {
		t.Errorf("Service = %q, want %q", got.Service, info.Service)
	}
}

func TestCache_GetNotFound(t *testing.T) {
	cache, tmpDir, cleanup := setupTestCache(t, true, 0)
	defer cleanup()

	pdfPath := createTestPDF(t, tmpDir, "test.pdf", "test content")

	// キャッシュに存在しないファイル
	got, found := cache.Get(pdfPath)
	if found {
		t.Error("Get() found = true for non-existent cache, want false")
	}
	if got != nil {
		t.Errorf("Get() = %v, want nil", got)
	}
}

func TestCache_Disabled(t *testing.T) {
	cache, tmpDir, cleanup := setupTestCache(t, false, 0)
	defer cleanup()

	pdfPath := createTestPDF(t, tmpDir, "test.pdf", "test content")
	info := &ai.ReceiptInfo{Date: "20250115", Service: "Test"}

	// Set は何もしない
	err := cache.Set(pdfPath, info)
	if err != nil {
		t.Errorf("Set() with disabled cache should not error, got %v", err)
	}

	// Get は見つからない
	_, found := cache.Get(pdfPath)
	if found {
		t.Error("Get() with disabled cache should return found=false")
	}
}

func TestCache_TTLExpiration(t *testing.T) {
	cache, tmpDir, cleanup := setupTestCache(t, true, 1) // TTL: 1日
	defer cleanup()

	pdfPath := createTestPDF(t, tmpDir, "test.pdf", "test content")

	// ハッシュを計算してキャッシュファイルを直接作成（期限切れの日時で）
	hash, _ := cache.hashFile(pdfPath)
	entry := CacheEntry{
		Hash:       hash,
		AnalyzedAt: time.Now().AddDate(0, 0, -2), // 2日前
		Result:     &ai.ReceiptInfo{Date: "20250115", Service: "Expired"},
	}
	data, _ := json.Marshal(entry)
	cachePath := filepath.Join(cache.dir, hash+".json")
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		t.Fatalf("Failed to write cache file: %v", err)
	}

	// 期限切れのキャッシュは取得できない
	_, found := cache.Get(pdfPath)
	if found {
		t.Error("Get() should return found=false for expired cache")
	}

	// キャッシュファイルが削除されていることを確認
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("Expired cache file should be deleted")
	}
}

func TestCache_TTLNotExpired(t *testing.T) {
	cache, tmpDir, cleanup := setupTestCache(t, true, 7) // TTL: 7日
	defer cleanup()

	pdfPath := createTestPDF(t, tmpDir, "test.pdf", "test content")
	info := &ai.ReceiptInfo{Date: "20250115", Service: "Recent"}

	// 通常通りセット
	if err := cache.Set(pdfPath, info); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// まだ有効なキャッシュは取得できる
	got, found := cache.Get(pdfPath)
	if !found {
		t.Error("Get() should return found=true for valid cache")
	}
	if got.Service != "Recent" {
		t.Errorf("Service = %q, want %q", got.Service, "Recent")
	}
}

func TestCache_TTLZeroMeansNoExpiration(t *testing.T) {
	cache, tmpDir, cleanup := setupTestCache(t, true, 0) // TTL: 0 (無期限)
	defer cleanup()

	pdfPath := createTestPDF(t, tmpDir, "test.pdf", "test content")

	// ハッシュを計算して古いキャッシュを作成
	hash, _ := cache.hashFile(pdfPath)
	entry := CacheEntry{
		Hash:       hash,
		AnalyzedAt: time.Now().AddDate(-1, 0, 0), // 1年前
		Result:     &ai.ReceiptInfo{Date: "20240115", Service: "Old"},
	}
	data, _ := json.Marshal(entry)
	cachePath := filepath.Join(cache.dir, hash+".json")
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		t.Fatalf("Failed to write cache file: %v", err)
	}

	// TTL=0 なら期限なしで取得できる
	got, found := cache.Get(pdfPath)
	if !found {
		t.Error("Get() should return found=true when TTL=0")
	}
	if got.Service != "Old" {
		t.Errorf("Service = %q, want %q", got.Service, "Old")
	}
}

func TestCache_SameContentDifferentPath(t *testing.T) {
	cache, tmpDir, cleanup := setupTestCache(t, true, 0)
	defer cleanup()

	content := "identical content"
	path1 := createTestPDF(t, tmpDir, "file1.pdf", content)
	path2 := createTestPDF(t, tmpDir, "file2.pdf", content)

	info := &ai.ReceiptInfo{Date: "20250115", Service: "Shared"}
	if err := cache.Set(path1, info); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// 同じ内容のファイルはキャッシュヒット
	got, found := cache.Get(path2)
	if !found {
		t.Error("Get() should find cache for file with same content")
	}
	if got.Service != "Shared" {
		t.Errorf("Service = %q, want %q", got.Service, "Shared")
	}
}

func TestCache_DifferentContentNoHit(t *testing.T) {
	cache, tmpDir, cleanup := setupTestCache(t, true, 0)
	defer cleanup()

	path1 := createTestPDF(t, tmpDir, "file1.pdf", "content A")
	path2 := createTestPDF(t, tmpDir, "file2.pdf", "content B")

	info := &ai.ReceiptInfo{Date: "20250115", Service: "A"}
	if err := cache.Set(path1, info); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// 異なる内容のファイルはキャッシュミス
	_, found := cache.Get(path2)
	if found {
		t.Error("Get() should not find cache for file with different content")
	}
}

func TestCache_Clear(t *testing.T) {
	cache, tmpDir, cleanup := setupTestCache(t, true, 0)
	defer cleanup()

	// 複数のキャッシュを作成
	for i := 0; i < 3; i++ {
		path := createTestPDF(t, tmpDir, "test"+string(rune('0'+i))+".pdf", "content"+string(rune('0'+i)))
		if err := cache.Set(path, &ai.ReceiptInfo{Date: "20250115", Service: "Test"}); err != nil {
			t.Fatalf("Set() error = %v", err)
		}
	}

	count, _ := cache.Count()
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}

	// クリア
	err := cache.Clear()
	if err != nil {
		t.Errorf("Clear() error = %v", err)
	}

	count, _ = cache.Count()
	if count != 0 {
		t.Errorf("Count() after Clear() = %d, want 0", count)
	}
}

func TestCache_Count(t *testing.T) {
	cache, tmpDir, cleanup := setupTestCache(t, true, 0)
	defer cleanup()

	// 初期状態
	count, _ := cache.Count()
	if count != 0 {
		t.Errorf("Initial Count() = %d, want 0", count)
	}

	// キャッシュを追加
	path := createTestPDF(t, tmpDir, "test.pdf", "content")
	if err := cache.Set(path, &ai.ReceiptInfo{Date: "20250115", Service: "Test"}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	count, _ = cache.Count()
	if count != 1 {
		t.Errorf("Count() = %d, want 1", count)
	}
}

func TestNew(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cache_new_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// config.DefaultCachePath() をオーバーライドできないので、
	// New関数が正常に動作することだけを確認
	cfg := &config.CacheConfig{
		Enabled: true,
		TTL:     7,
	}

	cache, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if cache == nil {
		t.Fatal("New() returned nil cache")
	}
	if !cache.enabled {
		t.Error("cache.enabled = false, want true")
	}
	if cache.ttl != 7 {
		t.Errorf("cache.ttl = %d, want 7", cache.ttl)
	}
}
