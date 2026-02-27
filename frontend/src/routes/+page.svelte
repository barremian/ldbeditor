<script lang="ts">
  import { Dialogs } from "@wailsio/runtime";
  import { LevelDBService, OpenDatabaseResult } from "../../bindings/ldbeditor";

  const RECENT_STORAGE_KEY = "recent-leveldb-paths";
  const MAX_RECENT = 10;

  // Startup view state
  let recentPaths: { path: string; label: string }[] = [];
  let loading = false;
  let errorMessage = "";

  // Editor view state
  let dbPath = "";
  let keys: string[] = [];
  let selectedKey: string | null = null;
  let selectedValue = "";
  let valueLoading = false;

  // Load recent paths from localStorage
  function loadRecentPaths() {
    try {
      const stored = localStorage.getItem(RECENT_STORAGE_KEY);
      if (stored) {
        recentPaths = JSON.parse(stored);
      } else {
        recentPaths = [];
      }
    } catch {
      recentPaths = [];
    }
  }

  // Save recent paths to localStorage
  function saveRecentPaths(paths: { path: string; label: string }[]) {
    recentPaths = paths.slice(0, MAX_RECENT);
    localStorage.setItem(RECENT_STORAGE_KEY, JSON.stringify(recentPaths));
  }

  // Add path to recent list
  function addToRecent(path: string) {
    const label = path.split(/[/\\]/).pop() || path;
    let paths = recentPaths.filter((p) => p.path !== path);
    paths.unshift({ path, label });
    saveRecentPaths(paths);
  }

  async function openDatabaseFromPath(path: string) {
    if (!path || path.trim() === "") return;

    loading = true;
    errorMessage = "";

    try {
      const result: OpenDatabaseResult =
        await LevelDBService.OpenDatabase(path);
      if (result.ok) {
        addToRecent(path);
        dbPath = path;
        await loadKeys();
      } else {
        await Dialogs.Error({
          Title: "Invalid Database",
          Message:
            result.error ||
            "The selected folder is not a valid LevelDB database.",
        });
      }
    } catch (err) {
      errorMessage = err instanceof Error ? err.message : String(err);
      await Dialogs.Error({
        Title: "Error",
        Message: errorMessage,
      });
    } finally {
      loading = false;
    }
  }

  async function openDatabaseFromDialog() {
    try {
      const path = await Dialogs.OpenFile({
        CanChooseDirectories: true,
        CanChooseFiles: false,
        Title: "Select LevelDB Database Folder",
      });

      // OpenFile returns string or string[] depending on options; for single dir it's a string
      const selectedPath = Array.isArray(path) ? path[0] : path;
      if (selectedPath) {
        await openDatabaseFromPath(selectedPath);
      }
    } catch (err) {
      console.error("Dialog error:", err);
    }
  }

  async function loadKeys() {
    loading = true;
    try {
      const keyList = await LevelDBService.GetKeys();
      keys = keyList ?? [];
      selectedKey = null;
      selectedValue = "";
    } catch (err) {
      errorMessage = err instanceof Error ? err.message : String(err);
    } finally {
      loading = false;
    }
  }

  async function selectKey(key: string) {
    selectedKey = key;
    valueLoading = true;
    selectedValue = "";

    try {
      const value = await LevelDBService.GetValue(key);
      selectedValue = value ?? "";
    } catch (err) {
      selectedValue = `Error: ${err instanceof Error ? err.message : String(err)}`;
    } finally {
      valueLoading = false;
    }
  }

  async function closeDatabase() {
    try {
      await LevelDBService.CloseDatabase();
    } catch {
      // ignore
    }
    dbPath = "";
    keys = [];
    selectedKey = null;
    selectedValue = "";
  }

  // Init
  loadRecentPaths();
</script>

{#if dbPath}
  <!-- Editor View -->
  <div class="editor-layout">
    <header class="editor-header">
      <button class="btn-back" on:click={closeDatabase}>← Close database</button
      >
      <span class="db-path" title={dbPath}
        >{dbPath.split(/[/\\]/).pop() || dbPath}</span
      >
    </header>

    <div class="editor-main">
      <aside class="keys-panel">
        <h3>Keys</h3>
        {#if loading}
          <div class="loading">Loading keys…</div>
        {:else if keys.length === 0}
          <div class="empty-state">No keys</div>
        {:else}
          <ul class="key-list">
            {#each keys as key}
              <li>
                <button
                  class="key-item"
                  class:selected={selectedKey === key}
                  on:click={() => selectKey(key)}
                >
                  {key.length > 80 ? key.slice(0, 80) + "…" : key}
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </aside>

      <section class="value-panel">
        <h3>
          Value {#if selectedKey}<span class="key-hint"
              >({selectedKey.length > 40
                ? selectedKey.slice(0, 40) + "…"
                : selectedKey})</span
            >{/if}
        </h3>
        {#if valueLoading}
          <div class="loading">Loading value…</div>
        {:else if selectedKey === null}
          <div class="empty-state">Select a key</div>
        {:else}
          <pre class="value-content">{selectedValue}</pre>
        {/if}
      </section>
    </div>
  </div>
{:else}
  <!-- Startup View -->
  <div class="startup-container">
    <h1>LevelDB Editor</h1>
    <button
      class="btn-open"
      on:click={openDatabaseFromDialog}
      disabled={loading}
    >
      {loading ? "Opening…" : "Open LevelDB database"}
    </button>

    {#if recentPaths.length > 0}
      <section class="recent-section">
        <h2>Recently opened</h2>
        <ul class="recent-list">
          {#each recentPaths as item}
            <li>
              <button
                class="recent-item"
                on:click={() => openDatabaseFromPath(item.path)}
                disabled={loading}
              >
                {item.label}
              </button>
            </li>
          {/each}
        </ul>
      </section>
    {/if}
  </div>
{/if}

<style>
  .startup-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    padding: 2rem;
  }

  .startup-container h1 {
    margin-bottom: 2rem;
    font-size: 2rem;
  }

  .btn-open {
    padding: 0.75rem 1.5rem;
    font-size: 1rem;
    border-radius: 6px;
    border: none;
    background: #646cff;
    color: white;
    cursor: pointer;
    margin-bottom: 2rem;
  }

  .btn-open:hover:not(:disabled) {
    background: #535bf2;
  }

  .btn-open:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .recent-section {
    width: 100%;
    max-width: 400px;
  }

  .recent-section h2 {
    font-size: 1rem;
    color: rgba(255, 255, 255, 0.7);
    margin-bottom: 0.75rem;
    font-weight: 500;
  }

  .recent-list {
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .recent-item {
    display: block;
    width: 100%;
    padding: 0.5rem 0.75rem;
    margin-bottom: 0.25rem;
    text-align: left;
    background: rgba(255, 255, 255, 0.08);
    border: none;
    border-radius: 4px;
    color: inherit;
    cursor: pointer;
    font-size: 0.9rem;
  }

  .recent-item:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.15);
  }

  .recent-item:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .editor-layout {
    display: flex;
    flex-direction: column;
    height: 100vh;
  }

  .editor-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 0.75rem 1rem;
    background: rgba(0, 0, 0, 0.2);
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .btn-back {
    padding: 0.4rem 0.75rem;
    font-size: 0.9rem;
    border-radius: 4px;
    border: none;
    background: rgba(255, 255, 255, 0.15);
    color: inherit;
    cursor: pointer;
  }

  .btn-back:hover {
    background: rgba(255, 255, 255, 0.25);
  }

  .db-path {
    font-size: 0.9rem;
    color: rgba(255, 255, 255, 0.8);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .editor-main {
    display: flex;
    flex: 1;
    min-height: 0;
  }

  .keys-panel,
  .value-panel {
    display: flex;
    flex-direction: column;
    padding: 1rem;
    overflow: hidden;
  }

  .keys-panel {
    width: 300px;
    min-width: 200px;
    border-right: 1px solid rgba(255, 255, 255, 0.1);
  }

  .value-panel {
    flex: 1;
    min-width: 0;
  }

  .keys-panel h3,
  .value-panel h3 {
    margin: 0 0 0.75rem 0;
    font-size: 0.9rem;
    font-weight: 600;
    color: rgba(255, 255, 255, 0.9);
  }

  .key-hint {
    font-weight: 400;
    color: rgba(255, 255, 255, 0.5);
    font-size: 0.85em;
  }

  .key-list {
    list-style: none;
    padding: 0;
    margin: 0;
    overflow-y: auto;
    flex: 1;
  }

  .key-item {
    display: block;
    width: 100%;
    padding: 0.5rem 0.75rem;
    margin-bottom: 0.25rem;
    text-align: left;
    background: transparent;
    border: none;
    border-radius: 4px;
    color: inherit;
    cursor: pointer;
    font-size: 0.85rem;
    word-break: break-all;
  }

  .key-item:hover {
    background: rgba(255, 255, 255, 0.08);
  }

  .key-item.selected {
    background: rgba(100, 108, 255, 0.4);
  }

  .value-content {
    flex: 1;
    margin: 0;
    padding: 1rem;
    overflow: auto;
    background: rgba(0, 0, 0, 0.2);
    border-radius: 4px;
    font-family:
      ui-monospace, "Cascadia Code", "Source Code Pro", Menlo, monospace;
    font-size: 0.85rem;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-all;
    user-select: text;
    -webkit-user-select: text;
  }

  .loading,
  .empty-state {
    color: rgba(255, 255, 255, 0.5);
    font-size: 0.9rem;
    padding: 1rem;
  }
</style>
