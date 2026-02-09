<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import {
    GetSettings,
    SaveSettingsWithEndpoint,
    SaveAPIKey,
    GetAPIKey,
    DeleteAPIKey,
    GetAvailableModels,
    GetBaseURL,
    ClearCache,
    GetCacheCount
  } from '../../wailsjs/go/main/App.js';

  const dispatch = createEventDispatcher();

  interface SettingsInfo {
    provider: string;
    model: string;
    hasApiKey: boolean;
    apiKeySource: string; // "none", "config_file", "env_var", "keyring"
    cacheEnabled: boolean;
    cacheCount: number;
    servicePattern: string;
  }

  function getApiKeySourceLabel(source: string): string {
    switch (source) {
      case 'config_file': return '設定ファイル';
      case 'env_var': return '環境変数';
      case 'keyring': return 'Keychain';
      default: return '未設定';
    }
  }

  let settings: SettingsInfo | null = null;
  let provider = 'anthropic';
  let model = '';
  let customModel = '';
  let useCustomModel = false;
  let baseURL = '';
  let apiKey = '';
  let servicePattern = '';
  let availableModels: string[] = [];
  let cacheCount = 0;
  let saving = false;
  let message = '';
  let messageType: 'success' | 'error' = 'success';

  export async function open() {
    settings = await GetSettings();
    provider = settings.provider || 'anthropic';
    model = settings.model || '';
    servicePattern = settings.servicePattern || '';
    cacheCount = settings.cacheCount || 0;
    apiKey = '';
    baseURL = await GetBaseURL() || '';

    // Load available models
    availableModels = await GetAvailableModels(provider);

    // Check if current model is in preset list
    if (model && !availableModels.includes(model)) {
      useCustomModel = true;
      customModel = model;
    } else {
      useCustomModel = false;
      customModel = '';
      if (!model && availableModels.length > 0) {
        model = availableModels[0];
      }
    }

    // OpenAI always uses custom model (for local LLM)
    if (provider === 'openai') {
      useCustomModel = true;
      customModel = model;
    }

    // Check if API key exists (don't show it, just indicate)
    const existingKey = await GetAPIKey(provider);
    if (existingKey) {
      apiKey = ''; // Don't show actual key, just placeholder
    }
  }

  async function onProviderChange() {
    availableModels = await GetAvailableModels(provider);

    if (provider === 'openai') {
      // OpenAI: always custom model
      useCustomModel = true;
      customModel = '';
      model = '';
    } else {
      // Anthropic: preset model by default
      useCustomModel = false;
      customModel = '';
      if (availableModels.length > 0) {
        model = availableModels[0];
      }
      // Clear baseURL when switching to Anthropic
      baseURL = '';
    }

    // Check if API key exists for new provider
    const existingKey = await GetAPIKey(provider);
    apiKey = '';
  }

  async function saveChanges() {
    saving = true;
    message = '';

    try {
      // Save API key if provided
      if (apiKey.trim()) {
        await SaveAPIKey(provider, apiKey.trim());
      }

      // Determine which model to save
      const modelToSave = useCustomModel ? customModel : model;

      // Save other settings with endpoint
      await SaveSettingsWithEndpoint(provider, modelToSave, baseURL, servicePattern);

      message = '設定を保存しました';
      messageType = 'success';

      // Notify parent to refresh
      dispatch('saved');

      setTimeout(() => {
        dispatch('close');
      }, 1000);
    } catch (e: any) {
      message = `エラー: ${e}`;
      messageType = 'error';
    } finally {
      saving = false;
    }
  }

  async function clearCacheData() {
    try {
      await ClearCache();
      cacheCount = await GetCacheCount();
      message = 'キャッシュをクリアしました';
      messageType = 'success';
    } catch (e: any) {
      message = `エラー: ${e}`;
      messageType = 'error';
    }
  }

  async function removeAPIKey() {
    try {
      await DeleteAPIKey(provider);
      message = 'APIキーを削除しました';
      messageType = 'success';
    } catch (e: any) {
      message = `エラー: ${e}`;
      messageType = 'error';
    }
  }

  function close() {
    dispatch('close');
  }
</script>

<div class="overlay" on:click={close} on:keydown={(e) => e.key === 'Escape' && close()} role="button" tabindex="0">
  <div class="modal" on:click|stopPropagation role="dialog" aria-modal="true">
    <div class="modal-header">
      <h2>設定</h2>
      <button class="close-btn" on:click={close}>&times;</button>
    </div>

    <div class="modal-body">
      <section class="setting-section">
        <h3>AI プロバイダー</h3>
        <div class="form-group">
          <label for="provider">プロバイダー</label>
          <select id="provider" bind:value={provider} on:change={onProviderChange}>
            <option value="anthropic">Anthropic (Claude)</option>
            <option value="openai">OpenAI互換 (ローカルLLM/GPT)</option>
          </select>
        </div>

        {#if provider === 'anthropic'}
          <div class="form-group">
            <label for="model">モデル</label>
            <select id="model" bind:value={model} disabled={useCustomModel}>
              {#each availableModels as m}
                <option value={m}>{m}</option>
              {/each}
            </select>
          </div>

          <div class="form-group checkbox-group">
            <label>
              <input type="checkbox" bind:checked={useCustomModel} />
              カスタムモデルを使用
            </label>
          </div>

          {#if useCustomModel}
            <div class="form-group">
              <label for="customModel">カスタムモデル名</label>
              <input
                type="text"
                id="customModel"
                bind:value={customModel}
                placeholder="例: claude-3-5-sonnet-20241022"
              />
            </div>
          {/if}
        {:else}
          <!-- OpenAI: ローカルLLM向け、カスタムモデルのみ -->
          <div class="form-group">
            <label for="customModel">モデル名</label>
            <input
              type="text"
              id="customModel"
              bind:value={customModel}
              placeholder="例: gpt-4o, llama3, etc."
            />
            <span class="hint">ローカルLLMの場合、使用するモデル名を入力</span>
          </div>

          <div class="form-group">
            <label for="baseURL">エンドポイントURL</label>
            <input
              type="text"
              id="baseURL"
              bind:value={baseURL}
              placeholder="例: http://localhost:11434/v1"
            />
            <span class="hint">OpenAI互換APIのエンドポイント（ローカルLLM用）</span>
          </div>
        {/if}

        <div class="form-group">
          <label for="apiKey">APIキー</label>
          <input
            type="password"
            id="apiKey"
            bind:value={apiKey}
            placeholder={settings?.hasApiKey ? '(設定済み - 変更する場合は入力)' : 'APIキーを入力'}
          />
          <div class="api-key-info">
            {#if settings?.hasApiKey}
              <span class="api-key-source">取得元: <strong>{getApiKeySourceLabel(settings.apiKeySource)}</strong></span>
              <button class="btn btn-small btn-danger" on:click={removeAPIKey}>キーを削除</button>
            {:else}
              <span class="api-key-source no-key">APIキーが設定されていません</span>
            {/if}
          </div>
        </div>
      </section>

      <section class="setting-section">
        <h3>リネーム設定</h3>
        <div class="form-group">
          <label for="servicePattern">サービス名パターン</label>
          <input
            type="text"
            id="servicePattern"
            bind:value={servicePattern}
            placeholder={`{{.Service}}`}
          />
          <span class="hint">{'{{.Service}}'} = 解析されたサービス名</span>
        </div>
      </section>

      <section class="setting-section">
        <h3>キャッシュ</h3>
        <div class="cache-info">
          <span>キャッシュ件数: {cacheCount}件</span>
          <button class="btn btn-small btn-secondary" on:click={clearCacheData}>クリア</button>
        </div>
      </section>

      {#if message}
        <div class="message" class:success={messageType === 'success'} class:error={messageType === 'error'}>
          {message}
        </div>
      {/if}
    </div>

    <div class="modal-footer">
      <button class="btn btn-secondary" on:click={close}>キャンセル</button>
      <button class="btn btn-primary" on:click={saveChanges} disabled={saving}>
        {saving ? '保存中...' : '保存'}
      </button>
    </div>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    background: white;
    border-radius: 12px;
    width: 500px;
    max-width: 90vw;
    max-height: 90vh;
    overflow: auto;
    box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 20px;
    border-bottom: 1px solid #eee;
  }

  .modal-header h2 {
    margin: 0;
    font-size: 1.3rem;
    color: #333;
  }

  .close-btn {
    background: none;
    border: none;
    font-size: 1.5rem;
    cursor: pointer;
    color: #666;
    padding: 0;
    line-height: 1;
  }

  .close-btn:hover {
    color: #333;
  }

  .modal-body {
    padding: 20px;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    padding: 20px;
    border-top: 1px solid #eee;
  }

  .setting-section {
    margin-bottom: 25px;
  }

  .setting-section h3 {
    font-size: 1rem;
    color: #333;
    margin: 0 0 15px 0;
    padding-bottom: 8px;
    border-bottom: 1px solid #eee;
  }

  .form-group {
    margin-bottom: 15px;
  }

  .form-group label {
    display: block;
    font-size: 0.9rem;
    color: #666;
    margin-bottom: 5px;
  }

  .form-group input,
  .form-group select {
    width: 100%;
    padding: 10px;
    border: 1px solid #ddd;
    border-radius: 6px;
    font-size: 0.95rem;
    box-sizing: border-box;
  }

  .form-group input:focus,
  .form-group select:focus {
    outline: none;
    border-color: #667eea;
  }

  .hint {
    display: block;
    font-size: 0.8rem;
    color: #999;
    margin-top: 5px;
  }

  .checkbox-group {
    display: flex;
    align-items: center;
  }

  .checkbox-group label {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    font-size: 0.9rem;
    color: #666;
  }

  .checkbox-group input[type="checkbox"] {
    width: auto;
    margin: 0;
  }

  .cache-info {
    display: flex;
    align-items: center;
    gap: 15px;
  }

  .message {
    padding: 12px;
    border-radius: 6px;
    margin-top: 15px;
    text-align: center;
  }

  .message.success {
    background: #e8f5e9;
    color: #2e7d32;
  }

  .message.error {
    background: #ffebee;
    color: #c62828;
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

  .api-key-info {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-top: 8px;
  }

  .api-key-source {
    font-size: 0.85rem;
    color: #666;
  }

  .api-key-source strong {
    color: #1976d2;
  }

  .api-key-source.no-key {
    color: #e65100;
  }
</style>
