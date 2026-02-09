<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    GetConfig,
    HasAPIKey,
    AddFiles,
    GetFiles,
    ClearFiles,
    AnalyzeFiles,
    RenameFiles,
    ToggleFileSelection,
    SelectAll,
    DeselectAll,
    OpenFileDialog,
    OpenFolderDialog,
    ScanFolder,
    UpdateServicePattern,
    GetServicePatternHistory
  } from '../wailsjs/go/main/App.js';
  import { EventsOn, EventsOff, OnFileDrop, OnFileDropOff } from '../wailsjs/runtime/runtime.js';
  import Settings from './lib/Settings.svelte';

  interface FileItem {
    id: number;
    originalPath: string;
    originalName: string;
    newName: string;
    date: string;
    service: string;
    status: string;
    error: string;
    selected: boolean;
    alreadyRenamed: boolean;
  }

  interface ConfigInfo {
    providerName: string;
    model: string;
    cacheEnabled: boolean;
    servicePattern: string;
    servicePatternIsEmpty: boolean;
  }

  interface RenameResult {
    totalCount: number;
    renamedCount: number;
    errorCount: number;
    skippedCount: number;
  }

  let files: FileItem[] = [];
  let config: ConfigInfo | null = null;
  let hasApiKey = false;
  let isDragging = false;
  let isAnalyzing = false;
  let isRenaming = false;
  let resultMessage = '';
  let servicePattern = '';
  let editingPattern = false;
  let showSettings = false;
  let settingsComponent: Settings;
  let patternHistory: string[] = [];
  let patternInputEl: HTMLInputElement;
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;

  onMount(async () => {
    config = await GetConfig();
    hasApiKey = await HasAPIKey();
    servicePattern = config?.servicePattern || '';

    EventsOn('files-updated', (updatedFiles: FileItem[]) => {
      files = updatedFiles;
      isAnalyzing = files.some(f => f.status === 'analyzing');
    });

    EventsOn('analysis-complete', (updatedFiles: FileItem[]) => {
      files = updatedFiles;
      isAnalyzing = false;
    });

    EventsOn('keyring-error', (error: string) => {
      resultMessage = error;
    });

    // Wails native file drop handler (useDropTarget: false = entire window)
    OnFileDrop(async (x: number, y: number, paths: string[]) => {
      console.log('OnFileDrop called:', x, y, paths);
      if (paths && paths.length > 0) {
        files = await AddFiles(paths);
      }
    }, false);

    // Fetch any files that were added before event listener was registered
    // (e.g., files passed via Finder "Open With")
    files = await GetFiles();
  });

  onDestroy(() => {
    EventsOff('files-updated');
    EventsOff('analysis-complete');
    EventsOff('keyring-error');
    OnFileDropOff();
  });

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
    isDragging = true;
  }

  function handleDragLeave(e: DragEvent) {
    e.preventDefault();
    isDragging = false;
  }

  async function handleDrop(e: DragEvent) {
    e.preventDefault();
    isDragging = false;

    const droppedFiles = e.dataTransfer?.files;
    if (!droppedFiles) return;

    const paths: string[] = [];
    for (let i = 0; i < droppedFiles.length; i++) {
      const file = droppedFiles[i];
      // @ts-ignore - Wails provides the path property
      if (file.path) {
        // @ts-ignore
        paths.push(file.path);
      }
    }

    if (paths.length > 0) {
      files = await AddFiles(paths);
    }
  }

  async function openFileDialog() {
    const selectedFiles = await OpenFileDialog();
    if (selectedFiles && selectedFiles.length > 0) {
      files = await AddFiles(selectedFiles);
    }
  }

  async function openFolderDialog() {
    const folder = await OpenFolderDialog();
    if (folder) {
      const pdfFiles = await ScanFolder(folder);
      if (pdfFiles && pdfFiles.length > 0) {
        files = await AddFiles(pdfFiles);
      }
    }
  }

  async function startAnalysis() {
    if (!hasApiKey) {
      resultMessage = 'APIキーが設定されていません。環境変数 ANTHROPIC_API_KEY を設定するか、設定画面でAPIキーを入力してください。';
      return;
    }
    isAnalyzing = true;
    resultMessage = '';
    await AnalyzeFiles();
  }

  async function startRename() {
    isRenaming = true;
    resultMessage = '';
    const result: RenameResult = await RenameFiles();
    isRenaming = false;

    if (result.renamedCount > 0) {
      resultMessage = `${result.renamedCount}件のファイルをリネームしました`;
    }
    if (result.errorCount > 0) {
      resultMessage += ` (${result.errorCount}件のエラー)`;
    }
    if (result.skippedCount > 0) {
      resultMessage += ` (${result.skippedCount}件スキップ)`;
    }
  }

  async function clearAllFiles() {
    await ClearFiles();
    files = [];
    resultMessage = '';
  }

  async function toggleSelection(id: number) {
    await ToggleFileSelection(id);
    files = await GetFiles();
  }

  async function selectAllFiles() {
    await SelectAll();
    files = await GetFiles();
  }

  async function deselectAllFiles() {
    await DeselectAll();
    files = await GetFiles();
  }

  async function savePattern() {
    try {
      await UpdateServicePattern(servicePattern);
      editingPattern = false;
      files = await GetFiles();
      // 履歴を更新
      patternHistory = await GetServicePatternHistory();
    } catch (e: any) {
      resultMessage = `テンプレートエラー: ${e}`;
    }
  }

  async function startEditingPattern() {
    editingPattern = true;
    // 履歴を読み込む
    patternHistory = await GetServicePatternHistory();
    // 入力欄にフォーカス
    setTimeout(() => patternInputEl?.focus(), 0);
  }

  function selectFromHistory(pattern: string) {
    servicePattern = pattern;
    // 入力欄にフォーカスを移す
    setTimeout(() => patternInputEl?.focus(), 0);
  }

  function cancelEditing() {
    editingPattern = false;
  }

  // フィルタリングされた履歴（リアクティブ）
  $: filteredHistory = patternHistory.filter(p => {
    if (!servicePattern || servicePattern.trim() === '') {
      return true; // 空欄なら全件表示
    }
    // 部分一致（大文字小文字区別なし）
    return p.toLowerCase().includes(servicePattern.toLowerCase());
  });

  // 編集中かつ履歴があればドロップダウン表示
  $: showSuggestions = editingPattern && filteredHistory.length > 0;

  function getStatusLabel(status: string): string {
    switch (status) {
      case 'pending': return '待機中';
      case 'analyzing': return '解析中...';
      case 'ready': return '解析完了';
      case 'cached': return 'キャッシュ';
      case 'renamed': return 'リネーム完了';
      case 'error': return 'エラー';
      case 'skipped': return 'スキップ';
      default: return status;
    }
  }

  function getStatusClass(status: string): string {
    switch (status) {
      case 'pending': return 'status-pending';
      case 'analyzing': return 'status-analyzing';
      case 'ready': return 'status-ready';
      case 'cached': return 'status-cached';
      case 'renamed': return 'status-renamed';
      case 'error': return 'status-error';
      case 'skipped': return 'status-skipped';
      default: return '';
    }
  }

  async function openSettings() {
    showSettings = true;
    // Wait for component to mount, then call open()
    setTimeout(() => {
      settingsComponent?.open();
    }, 0);
  }

  async function onSettingsSaved() {
    // Reload config
    config = await GetConfig();
    hasApiKey = await HasAPIKey();
    servicePattern = config?.servicePattern || '';
  }

  function closeSettings() {
    showSettings = false;
  }

  $: pendingCount = files.filter(f => f.status === 'pending').length;
  $: readyCount = files.filter(f => f.status === 'ready' || f.status === 'cached').length;
  $: selectedCount = files.filter(f => f.selected && (f.status === 'ready' || f.status === 'cached')).length;
  $: canAnalyze = pendingCount > 0 && hasApiKey && !isAnalyzing;
  $: servicePatternIsEmpty = !servicePattern || servicePattern.trim() === '';
  $: canRename = selectedCount > 0 && !isRenaming && !isAnalyzing && !servicePatternIsEmpty;

  // Sort files: not already renamed first, then already renamed
  $: sortedFiles = [...files].sort((a, b) => {
    if (a.alreadyRenamed === b.alreadyRenamed) return 0;
    return a.alreadyRenamed ? 1 : -1;
  });

  // Get sample service name from analyzed files for preview
  $: sampleFile = files.find(f => f.service && (f.status === 'ready' || f.status === 'cached'));
  $: sampleServiceName = sampleFile?.service || 'サービス名';
  $: sampleDate = sampleFile?.date || '20250207';
  $: sampleOriginalName = sampleFile ? sampleFile.originalName.replace('.pdf', '') : 'invoice';

  // Generate preview with actual values
  function getPatternPreview(pattern: string): string {
    if (!pattern || pattern.trim() === '') return '(未設定)';
    return pattern.replace(/\{\{\.Service\}\}/g, sampleServiceName);
  }
</script>

<main>
  <header>
    <h1>Receipt PDF Renamer</h1>
    <div class="header-right">
      {#if config}
        <div class="config-info">
          <span class="provider">{config.providerName}</span>
          <span class="model">{config.model}</span>
        </div>
      {/if}
      <button class="btn-icon" on:click={openSettings} title="設定">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="3"></circle>
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
        </svg>
      </button>
    </div>
  </header>

  <div
    class="drop-zone"
    class:dragging={isDragging}
    on:dragover={handleDragOver}
    on:dragleave={handleDragLeave}
    on:drop={handleDrop}
    role="button"
    tabindex="0"
  >
    <div class="drop-content">
      <p class="drop-icon">📄</p>
      <p>PDFファイルをドラッグ&ドロップ</p>
      <p class="drop-hint">または</p>
      <div class="button-group">
        <button class="btn btn-secondary" on:click={openFileDialog}>ファイルを選択</button>
        <button class="btn btn-secondary" on:click={openFolderDialog}>フォルダを選択</button>
      </div>
    </div>
  </div>

  {#if files.length > 0}
    <div class="toolbar">
      <div class="toolbar-left">
        <span class="file-count">{files.length}件のファイル</span>
        {#if readyCount > 0}
          <button class="btn-link" on:click={selectAllFiles}>全選択</button>
          <button class="btn-link" on:click={deselectAllFiles}>全解除</button>
        {/if}
      </div>
      <div class="toolbar-right">
        {#if pendingCount > 0}
          <button
            class="btn btn-primary"
            on:click={startAnalysis}
            disabled={!canAnalyze}
          >
            {isAnalyzing ? '解析中...' : `解析開始 (${pendingCount}件)`}
          </button>
        {/if}
        {#if readyCount > 0}
          <button
            class="btn btn-success"
            on:click={startRename}
            disabled={!canRename}
          >
            {isRenaming ? 'リネーム中...' : `リネーム実行 (${selectedCount}件)`}
          </button>
        {/if}
        <button class="btn btn-danger" on:click={clearAllFiles}>クリア</button>
      </div>
    </div>

    <div class="pattern-editor" class:pattern-empty={servicePatternIsEmpty}>
      <span class="pattern-label">サービス名:</span>
      {#if editingPattern}
        <div class="pattern-input-wrapper">
          <input
            type="text"
            bind:this={patternInputEl}
            bind:value={servicePattern}
            class="pattern-input"
            class:has-suggestions={showSuggestions}
            placeholder={`{{.Service}}`}
            autocomplete="off"
            on:keydown={(e) => {
              if (e.key === 'Enter') savePattern();
              if (e.key === 'Escape') cancelEditing();
            }}
          />
          {#if showSuggestions}
            <div class="history-dropdown">
              <div class="history-header">
                {#if servicePattern && servicePattern.trim() !== ''}
                  候補
                {:else}
                  履歴
                {/if}
              </div>
              {#each filteredHistory as historyItem}
                <button
                  class="history-item"
                  on:click={() => selectFromHistory(historyItem)}
                >
                  <code>{historyItem}</code>
                </button>
              {/each}
            </div>
          {/if}
        </div>
        <button class="btn btn-small" on:click={savePattern}>保存</button>
        <button class="btn btn-small btn-secondary" on:click={cancelEditing}>キャンセル</button>
        <span class="pattern-hint">※ {'{{.Service}}'} = 解析されたサービス名</span>
      {:else}
        <code class="pattern-display" class:empty={servicePatternIsEmpty}>{getPatternPreview(servicePattern)}</code>
        <button class="btn btn-small btn-primary" on:click={startEditingPattern}>
          {servicePatternIsEmpty ? '設定する' : '編集'}
        </button>
      {/if}
      {#if !servicePatternIsEmpty}
        <span class="pattern-preview">例: {sampleDate}-{getPatternPreview(servicePattern)}-{sampleOriginalName}.pdf</span>
      {:else}
        <span class="pattern-warning">リネームするにはサービス名を設定してください</span>
      {/if}
    </div>

    <div class="file-list">
      {#each sortedFiles as file (file.id)}
        <div class="file-item" class:selected={file.selected} class:already-renamed={file.alreadyRenamed}>
          <div class="file-checkbox">
            {#if file.status === 'ready' || file.status === 'cached'}
              <input
                type="checkbox"
                checked={file.selected}
                on:change={() => toggleSelection(file.id)}
              />
            {/if}
          </div>
          <div class="file-info">
            <div class="file-name">{file.originalName}</div>
            {#if file.newName && file.status !== 'pending' && file.status !== 'skipped'}
              <div class="file-new-name">→ {file.newName}</div>
            {/if}
            {#if file.error && !file.alreadyRenamed}
              <div class="file-error">{file.error}</div>
            {/if}
            {#if file.alreadyRenamed}
              <div class="file-already-renamed">既にリネーム済みの形式です</div>
            {/if}
          </div>
          <div class="file-status {getStatusClass(file.status)}">
            {getStatusLabel(file.status)}
          </div>
        </div>
      {/each}
    </div>
  {/if}

  {#if resultMessage}
    <div class="result-message">{resultMessage}</div>
  {/if}

  {#if !hasApiKey}
    <div class="warning">
      APIキーが設定されていません。<button class="btn-link" on:click={openSettings}>設定画面</button>からAPIキーを設定してください。
    </div>
  {/if}
</main>

{#if showSettings}
  <Settings
    bind:this={settingsComponent}
    on:saved={onSettingsSaved}
    on:close={closeSettings}
  />
{/if}

<style>
  :global(html, body) {
    margin: 0;
    padding: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
    background: #f5f5f5;
    min-height: 100vh;
    overflow-x: hidden;
  }

  main {
    max-width: 100%;
    margin: 0;
    padding: 20px;
    padding-top: 10px;
    background: #f5f5f5;
    min-height: calc(100vh - 30px);
  }

  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
    padding: 10px 15px;
    padding-left: 80px; /* macOS window buttons */
    background: white;
    border-radius: 10px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    --wails-draggable: drag;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 15px;
    --wails-draggable: no-drag;
  }

  .btn-icon {
    background: none;
    border: none;
    cursor: pointer;
    padding: 8px;
    border-radius: 6px;
    color: #666;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s ease;
  }

  .btn-icon:hover {
    background: #f0f0f0;
    color: #333;
  }

  h1 {
    margin: 0;
    font-size: 1.5rem;
    color: #333;
  }

  .config-info {
    display: flex;
    gap: 10px;
    font-size: 0.8rem;
  }

  .provider {
    background: #e3f2fd;
    padding: 4px 8px;
    border-radius: 4px;
    color: #1976d2;
  }

  .model {
    background: #f3e5f5;
    padding: 4px 8px;
    border-radius: 4px;
    color: #7b1fa2;
  }

  .drop-zone {
    border: 2px dashed #ccc;
    border-radius: 12px;
    padding: 40px;
    text-align: center;
    transition: all 0.3s ease;
    cursor: pointer;
    background: white;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .drop-zone.dragging {
    border-color: #667eea;
    background: #f0f4ff;
  }

  .drop-zone:hover {
    border-color: #667eea;
    background: #fafafa;
  }

  .drop-content p {
    margin: 8px 0;
    color: #666;
  }

  .drop-icon {
    font-size: 3rem;
    margin-bottom: 10px !important;
  }

  .drop-hint {
    font-size: 0.9rem;
    color: #999 !important;
  }

  .button-group {
    display: flex;
    gap: 10px;
    justify-content: center;
    margin-top: 15px;
  }

  .btn {
    padding: 10px 20px;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.95rem;
    transition: all 0.2s ease;
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary {
    background: #667eea;
    color: white;
  }

  .btn-primary:hover:not(:disabled) {
    background: #5a6fd6;
  }

  .btn-secondary {
    background: #e0e0e0;
    color: #333;
  }

  .btn-secondary:hover:not(:disabled) {
    background: #d0d0d0;
  }

  .btn-success {
    background: #4caf50;
    color: white;
  }

  .btn-success:hover:not(:disabled) {
    background: #43a047;
  }

  .btn-danger {
    background: #f44336;
    color: white;
  }

  .btn-danger:hover:not(:disabled) {
    background: #e53935;
  }

  .btn-small {
    padding: 5px 10px;
    font-size: 0.85rem;
  }

  .btn-link {
    background: none;
    border: none;
    color: #667eea;
    cursor: pointer;
    font-size: 0.9rem;
    padding: 5px;
  }

  .btn-link:hover {
    text-decoration: underline;
  }

  .toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin: 20px 0;
    padding: 15px;
    background: white;
    border-radius: 10px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .toolbar-left {
    display: flex;
    align-items: center;
    gap: 15px;
  }

  .toolbar-right {
    display: flex;
    gap: 10px;
  }

  .file-count {
    font-weight: 500;
    color: #333;
  }

  .pattern-editor {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 15px;
    padding: 12px 15px;
    background: white;
    border-left: 4px solid #667eea;
    border-radius: 10px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    flex-wrap: wrap;
  }

  .pattern-label {
    font-weight: 500;
    color: #666;
  }

  .pattern-display {
    background: #f5f5f5;
    padding: 5px 10px;
    border-radius: 4px;
    font-family: monospace;
    color: #333;
  }

  .pattern-input-wrapper {
    position: relative;
    display: flex;
    align-items: center;
  }

  .pattern-input {
    padding: 5px 10px;
    border: 1px solid #ccc;
    border-radius: 4px;
    font-family: monospace;
    width: 200px;
  }

  .pattern-input.has-suggestions {
    border-bottom-left-radius: 0;
    border-bottom-right-radius: 0;
    border-bottom-color: transparent;
  }

  .history-dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    background: white;
    border: 1px solid #ccc;
    border-top: none;
    border-radius: 0 0 6px 6px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    z-index: 100;
    max-height: 200px;
    overflow-y: auto;
  }

  .history-header {
    padding: 6px 10px;
    font-size: 0.7rem;
    font-weight: 600;
    color: #999;
    background: #fafafa;
    border-bottom: 1px solid #eee;
  }

  .history-item {
    display: block;
    width: 100%;
    padding: 8px 12px;
    text-align: left;
    background: none;
    border: none;
    border-bottom: 1px solid #eee;
    cursor: pointer;
    transition: background 0.2s ease;
  }

  .history-item:last-child {
    border-bottom: none;
  }

  .history-item:hover {
    background: #f0f4ff;
  }

  .history-item code {
    font-size: 0.85rem;
    color: #333;
  }

  .pattern-preview {
    font-size: 0.85rem;
    color: #666;
    margin-left: auto;
  }

  .pattern-hint {
    font-size: 0.8rem;
    color: #999;
    width: 100%;
    margin-top: 5px;
  }

  .pattern-editor.pattern-empty {
    border-left-color: #ff9800;
  }

  .pattern-display.empty {
    color: #999;
    font-style: italic;
  }

  .pattern-warning {
    font-size: 0.85rem;
    color: #e65100;
    margin-left: auto;
  }

  .file-list {
    background: white;
    border-radius: 10px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    overflow: hidden;
  }

  .file-item {
    display: flex;
    align-items: center;
    padding: 12px 15px;
    border-bottom: 1px solid #eee;
    transition: background 0.2s ease;
  }

  .file-item:last-child {
    border-bottom: none;
  }

  .file-item:hover {
    background: #fafafa;
  }

  .file-item.selected {
    background: #e8f5e9;
  }

  .file-checkbox {
    width: 30px;
  }

  .file-info {
    flex: 1;
    min-width: 0;
  }

  .file-name {
    font-weight: 500;
    color: #333;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .file-new-name {
    font-size: 0.9rem;
    color: #4caf50;
    margin-top: 4px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .file-error {
    font-size: 0.85rem;
    color: #f44336;
    margin-top: 4px;
  }

  .file-already-renamed {
    font-size: 0.85rem;
    color: #666;
    margin-top: 4px;
    font-style: italic;
  }

  .file-item.already-renamed {
    opacity: 0.6;
  }

  .file-item.already-renamed .file-name {
    color: #666;
  }

  .file-status {
    padding: 4px 12px;
    border-radius: 20px;
    font-size: 0.85rem;
    font-weight: 500;
    white-space: nowrap;
  }

  .status-pending {
    background: #e0e0e0;
    color: #666;
  }

  .status-analyzing {
    background: #fff3e0;
    color: #ef6c00;
    animation: pulse 1.5s infinite;
  }

  .status-ready {
    background: #e3f2fd;
    color: #1976d2;
  }

  .status-cached {
    background: #f3e5f5;
    color: #7b1fa2;
  }

  .status-renamed {
    background: #e8f5e9;
    color: #388e3c;
  }

  .status-error {
    background: #ffebee;
    color: #c62828;
  }

  .status-skipped {
    background: #eceff1;
    color: #546e7a;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.6; }
  }

  .result-message {
    margin-top: 20px;
    padding: 15px;
    background: white;
    border-left: 4px solid #4caf50;
    border-radius: 10px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    color: #2e7d32;
    text-align: center;
  }

  .warning {
    margin-top: 20px;
    padding: 15px;
    background: white;
    border-left: 4px solid #ff9800;
    border-radius: 10px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    color: #e65100;
    text-align: center;
  }

  .warning code {
    background: #fff;
    padding: 2px 6px;
    border-radius: 4px;
    font-family: monospace;
  }
</style>
