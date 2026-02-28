<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { Dialogs, Window } from "@wailsio/runtime";
  import { LevelDBService, OpenDatabaseResult } from "../../bindings/ldbeditor";
  import { Badge } from "$lib/components/ui/badge";
  import { Button } from "$lib/components/ui/button";
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from "$lib/components/ui/card";
  import { ScrollArea } from "$lib/components/ui/scroll-area";
  import {
    Database,
    FileText,
    FolderOpen,
    History,
    KeyRound,
    X,
  } from "lucide-svelte";

  const RECENT_STORAGE_KEY = "recent-leveldb-paths";
  const MAX_RECENT = 10;
  const KEY_PANE_WIDTH_STORAGE_KEY = "editor-key-pane-width";
  const DEFAULT_KEY_PANE_WIDTH = 320;
  const MIN_KEY_PANE_WIDTH = 240;
  const MIN_VALUE_PANE_WIDTH = 320;
  const GRID_GAP_PX = 16;
  const SPLIT_CONTAINER_PADDING_PX = 16;
  const SPLIT_CONTAINER_HORIZONTAL_INSET_PX = SPLIT_CONTAINER_PADDING_PX * 2;

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
  let isDesktopLayout = false;
  let keyPaneWidth = DEFAULT_KEY_PANE_WIDTH;
  let hasLoadedPaneWidth = false;
  let isResizingPane = false;
  let editorSplitContainer: HTMLDivElement | null = null;
  let detachPointerListeners: (() => void) | null = null;

  function getContainerWidth() {
    return editorSplitContainer?.clientWidth ?? window.innerWidth;
  }

  function getSplitContentWidth(containerWidth: number) {
    return Math.max(0, containerWidth - SPLIT_CONTAINER_HORIZONTAL_INSET_PX);
  }

  function clampKeyPaneWidth(
    width: number,
    containerWidth: number = getContainerWidth(),
  ) {
    const contentWidth = getSplitContentWidth(containerWidth);
    const maxKeyWidth = Math.max(
      MIN_KEY_PANE_WIDTH,
      contentWidth - MIN_VALUE_PANE_WIDTH - GRID_GAP_PX,
    );

    return Math.min(Math.max(width, MIN_KEY_PANE_WIDTH), maxKeyWidth);
  }

  function loadPaneWidthPreference() {
    try {
      const rawWidth = localStorage.getItem(KEY_PANE_WIDTH_STORAGE_KEY);
      if (!rawWidth) {
        keyPaneWidth = clampKeyPaneWidth(DEFAULT_KEY_PANE_WIDTH);
        return;
      }

      const parsedWidth = Number(rawWidth);
      if (!Number.isFinite(parsedWidth)) {
        keyPaneWidth = clampKeyPaneWidth(DEFAULT_KEY_PANE_WIDTH);
        return;
      }

      keyPaneWidth = clampKeyPaneWidth(parsedWidth);
    } catch {
      keyPaneWidth = clampKeyPaneWidth(DEFAULT_KEY_PANE_WIDTH);
    }
  }

  function savePaneWidthPreference() {
    try {
      localStorage.setItem(
        KEY_PANE_WIDTH_STORAGE_KEY,
        String(Math.round(keyPaneWidth)),
      );
    } catch {
      // ignore localStorage failures
    }
  }

  function syncKeyPaneWidthToViewport() {
    if (!isDesktopLayout) return;
    keyPaneWidth = clampKeyPaneWidth(keyPaneWidth);
  }

  function stopPaneResize() {
    if (detachPointerListeners) {
      detachPointerListeners();
      detachPointerListeners = null;
    }
    isResizingPane = false;
    document.body.style.cursor = "";
    savePaneWidthPreference();
  }

  function startPaneResize(event: PointerEvent) {
    if (!isDesktopLayout || !editorSplitContainer) return;

    const handle = event.currentTarget as HTMLElement | null;
    if (!handle) return;

    event.preventDefault();
    handle.setPointerCapture(event.pointerId);
    isResizingPane = true;
    document.body.style.cursor = "col-resize";

    const onPointerMove = (moveEvent: PointerEvent) => {
      if (!editorSplitContainer) return;
      const containerRect = editorSplitContainer.getBoundingClientRect();
      const nextWidth =
        moveEvent.clientX -
        containerRect.left -
        SPLIT_CONTAINER_PADDING_PX -
        GRID_GAP_PX / 2;
      keyPaneWidth = clampKeyPaneWidth(nextWidth, containerRect.width);
    };

    const onPointerUp = (upEvent: PointerEvent) => {
      if (handle.hasPointerCapture(upEvent.pointerId)) {
        handle.releasePointerCapture(upEvent.pointerId);
      }
      stopPaneResize();
    };

    window.addEventListener("pointermove", onPointerMove);
    window.addEventListener("pointerup", onPointerUp);
    window.addEventListener("pointercancel", onPointerUp);

    detachPointerListeners = () => {
      window.removeEventListener("pointermove", onPointerMove);
      window.removeEventListener("pointerup", onPointerUp);
      window.removeEventListener("pointercancel", onPointerUp);
    };
  }

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

  async function handleTitlebarDoubleClick() {
    const platformHint = `${navigator.platform} ${navigator.userAgent}`.toLowerCase();
    if (!platformHint.includes("mac")) return;

    try {
      await Window.Zoom();
    } catch {
      await Window.ToggleMaximise();
    }
  }

  onMount(() => {
    const mediaQuery = window.matchMedia("(min-width: 768px)");
    const updateLayoutMode = () => {
      isDesktopLayout = mediaQuery.matches;
      if (!isDesktopLayout) return;

      if (!hasLoadedPaneWidth) {
        loadPaneWidthPreference();
        hasLoadedPaneWidth = true;
        return;
      }

      syncKeyPaneWidthToViewport();
    };

    updateLayoutMode();

    const onWindowResize = () => {
      syncKeyPaneWidthToViewport();
    };

    mediaQuery.addEventListener("change", updateLayoutMode);
    window.addEventListener("resize", onWindowResize);

    return () => {
      mediaQuery.removeEventListener("change", updateLayoutMode);
      window.removeEventListener("resize", onWindowResize);
      stopPaneResize();
    };
  });

  onDestroy(() => {
    stopPaneResize();
  });

  // Init
  loadRecentPaths();
</script>

{#if dbPath}
  <div class="editor-layout flex h-screen flex-col">
    <div
      class="titlebar-drag-region shrink-0 border-b border-border/50 bg-background/70 backdrop-blur-sm"
      role="none"
      on:dblclick={handleTitlebarDoubleClick}
    ></div>
    <header
      class="flex items-center justify-between border-b border-border bg-background/80 px-4 py-3 backdrop-blur-sm"
    >
      <Button
        variant="outline"
        size="sm"
        class="gap-2"
        on:click={closeDatabase}
      >
        <X class="h-4 w-4" />
        Close database
      </Button>
      <Badge variant="secondary" class="max-w-[60vw] truncate" title={dbPath}>
        <Database class="mr-1.5 h-3.5 w-3.5" />
        {dbPath.split(/[/\\]/).pop() || dbPath}
      </Badge>
    </header>

    <div
      bind:this={editorSplitContainer}
      class="relative grid min-h-0 flex-1 gap-4 p-4 md:grid-cols-1"
      style={
        isDesktopLayout
          ? `grid-template-columns: ${keyPaneWidth}px minmax(${MIN_VALUE_PANE_WIDTH}px, 1fr);`
          : undefined
      }
    >
      {#if isDesktopLayout}
        <button
          type="button"
          class={`absolute bottom-4 top-4 z-10 w-3 -translate-x-1/2 cursor-col-resize rounded-full transition-colors ${
            isResizingPane ? "bg-primary/20" : "hover:bg-muted"
          }`}
          style={`left: ${SPLIT_CONTAINER_PADDING_PX + keyPaneWidth + GRID_GAP_PX / 2}px;`}
          aria-label="Resize key and value panes"
          on:pointerdown={startPaneResize}
        >
          <span
            class={`mx-auto block h-full w-px ${
              isResizingPane ? "bg-primary/70" : "bg-border/70"
            }`}
          ></span>
        </button>
      {/if}

      <Card
        class="flex min-h-0 min-w-0 flex-col"
        style={isDesktopLayout ? `min-width: ${MIN_KEY_PANE_WIDTH}px;` : undefined}
      >
        <CardHeader class="pb-3">
          <CardTitle class="flex items-center gap-2 text-base">
            <KeyRound class="h-4 w-4" />
            Keys
          </CardTitle>
          <CardDescription>{keys.length} entries</CardDescription>
        </CardHeader>
        <CardContent class="min-h-0 flex-1 px-3 pb-3">
          {#if loading}
            <div class="px-2 py-3 text-sm text-muted-foreground">
              Loading keys…
            </div>
          {:else if keys.length === 0}
            <div class="px-2 py-3 text-sm text-muted-foreground">No keys</div>
          {:else}
            <ScrollArea class="h-full rounded-md border border-border/70">
              <ul class="space-y-1 p-2">
                {#each keys as key}
                  <li>
                    <button
                      class={`w-full rounded-md px-2 py-1.5 text-left text-sm transition-colors ${
                        selectedKey === key
                          ? "bg-primary/15 text-primary"
                          : "hover:bg-muted"
                      }`}
                      on:click={() => selectKey(key)}
                    >
                      {key.length > 80 ? key.slice(0, 80) + "…" : key}
                    </button>
                  </li>
                {/each}
              </ul>
            </ScrollArea>
          {/if}
        </CardContent>
      </Card>

      <Card
        class="flex min-h-0 min-w-0 flex-col"
        style={
          isDesktopLayout ? `min-width: ${MIN_VALUE_PANE_WIDTH}px;` : undefined
        }
      >
        <CardHeader class="pb-3">
          <CardTitle class="flex items-center gap-2 text-base">
            <FileText class="h-4 w-4" />
            Value
          </CardTitle>
          <CardDescription>
            {#if selectedKey}
              {selectedKey.length > 60
                ? selectedKey.slice(0, 60) + "…"
                : selectedKey}
            {:else}
              Select a key to inspect its value
            {/if}
          </CardDescription>
        </CardHeader>
        <CardContent class="min-h-0 flex-1 px-3 pb-3">
          {#if valueLoading}
            <div class="px-1 py-3 text-sm text-muted-foreground">
              Loading value…
            </div>
          {:else if selectedKey === null}
            <div class="px-1 py-3 text-sm text-muted-foreground">
              Select a key
            </div>
          {:else}
            <ScrollArea
              class="h-full rounded-md border border-border/70 bg-muted/20"
            >
              <pre
                class="p-2 font-mono text-xs leading-relaxed text-foreground/90 whitespace-pre-wrap break-all select-text">
{selectedValue}</pre>
            </ScrollArea>
          {/if}
        </CardContent>
      </Card>
    </div>
  </div>
{:else}
  <div class="flex min-h-screen items-center justify-center p-6">
    <Card class="w-full max-w-2xl">
      <CardHeader class="space-y-4">
        <div class="flex items-center justify-between">
          <CardTitle class="flex items-center gap-2 text-2xl">
            <Database class="h-6 w-6 text-primary" />
            LevelDB Editor
          </CardTitle>
          <Badge variant="outline">Desktop</Badge>
        </div>
        <CardDescription>
          Open a LevelDB folder to browse keys and inspect values instantly.
        </CardDescription>
      </CardHeader>
      <CardContent class="space-y-6">
        <Button
          class="gap-2"
          on:click={openDatabaseFromDialog}
          disabled={loading}
        >
          <FolderOpen class="h-4 w-4" />
          {loading ? "Opening…" : "Open LevelDB database"}
        </Button>

        {#if errorMessage}
          <p class="text-sm text-destructive">{errorMessage}</p>
        {/if}

        {#if recentPaths.length > 0}
          <section class="space-y-3">
            <h2
              class="flex items-center gap-2 text-sm font-medium text-muted-foreground"
            >
              <History class="h-4 w-4" />
              Recently opened
            </h2>
            <div class="space-y-2">
              {#each recentPaths as item}
                <Button
                  variant="ghost"
                  class="w-full justify-start font-normal"
                  on:click={() => openDatabaseFromPath(item.path)}
                  disabled={loading}
                >
                  {item.label}
                </Button>
              {/each}
            </div>
          </section>
        {/if}
      </CardContent>
    </Card>
  </div>
{/if}
