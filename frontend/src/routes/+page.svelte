<script lang="ts">
  import { onDestroy, onMount, tick } from "svelte";
  import { Dialogs, Window } from "@wailsio/runtime";
  import * as LevelDBService from "../../bindings/ldbeditor/leveldbservice";
  import { OpenDatabaseResult } from "../../bindings/ldbeditor/models";
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
  import { Tabs, TabsList, TabsTrigger } from "$lib/components/ui/tabs";
  import {
    Check,
    ChevronDown,
    Database,
    FileText,
    FolderOpen,
    History,
    KeyRound,
    MoreHorizontal,
    Pencil,
    Plus,
    RefreshCcw,
    Save,
    Trash2,
    X,
  } from "lucide-svelte";

  const RECENT_STORAGE_KEY = "recent-leveldb-paths";
  const MAX_RECENT = 10;
  const KEY_PANE_WIDTH_STORAGE_KEY = "editor-key-pane-width";
  const VALUE_FORMAT_PREFS_STORAGE_KEY = "value-format-preferences";
  const KEY_SEARCH_DEBOUNCE_MS = 300;
  const DEFAULT_KEY_PANE_WIDTH = 320;
  const MIN_KEY_PANE_WIDTH = 300;
  const MIN_VALUE_PANE_WIDTH = 320;
  const GRID_GAP_PX = 16;
  const SPLIT_CONTAINER_PADDING_PX = 16;
  const SPLIT_CONTAINER_HORIZONTAL_INSET_PX = SPLIT_CONTAINER_PADDING_PX * 2;
  const AUTO_REFRESH_OPTIONS = [
    { label: "Off", intervalMs: 0 },
    { label: "5s", intervalMs: 5000 },
    { label: "15s", intervalMs: 15000 },
    { label: "30s", intervalMs: 30000 },
    { label: "60s", intervalMs: 60000 },
  ] as const;
  const REFRESH_RING_RADIUS = 6;
  const REFRESH_RING_CIRCUMFERENCE = 2 * Math.PI * REFRESH_RING_RADIUS;
  const REFRESH_COUNTDOWN_TICK_MS = 200;
  const MIN_REFRESH_FEEDBACK_MS = 150;

  // Startup view state
  let recentPaths: { path: string; label: string }[] = [];
  let loading = false;
  let errorMessage = "";
  type WorkspaceTab =
    | { id: string; type: "dashboard"; title: string }
    | { id: string; type: "database"; title: string; path: string };
  let nextTabId = 2;
  let tabs: WorkspaceTab[] = [
    { id: "tab-1", type: "dashboard", title: "Dashboard" },
  ];
  let activeTabId = tabs[0].id;

  // Editor view state
  let dbPath = "";
  let keys: string[] = [];
  let filteredKeys: string[] = [];
  let selectedKey: string | null = null;
  let originalValueRaw = "";
  let editorValueRaw = "";
  let editorValue = "";
  let isSaving = false;
  let isDeleting = false;
  let showCreateForm = false;
  let isCreating = false;
  let isRenaming = false;
  let newKeyInput = "";
  let newValueInput = "";
  let editingKey: string | null = null;
  let renameInput = "";
  let keyPendingDelete: string | null = null;
  let isValueEditing = false;
  let valueLoading = false;
  let isDesktopLayout = false;
  let keyPaneWidth = DEFAULT_KEY_PANE_WIDTH;
  let hasLoadedPaneWidth = false;
  let isResizingPane = false;
  let editorSplitContainer: HTMLDivElement | null = null;
  let detachPointerListeners: (() => void) | null = null;
  let isDirty = false;
  let valueValidationError = "";
  let createKeyValidationError = "";
  let createValueValidationError = "";
  let renameValidationError = "";
  let keySearchInput = "";
  let debouncedKeySearch = "";
  let keySearchDebounceTimeout: ReturnType<typeof setTimeout> | null = null;
  let renameInputElement: HTMLInputElement | null = null;
  let isRefreshing = false;
  let autoRefreshIntervalMs = 0;
  let nextRefreshAt: number | null = null;
  let countdownNow = Date.now();
  let isRefreshMenuOpen = false;
  let refreshMenuContainer: HTMLDivElement | null = null;
  let isViewMenuOpen = false;
  let viewMenuContainer: HTMLDivElement | null = null;
  let autoRefreshTimeout: ReturnType<typeof setTimeout> | null = null;
  let refreshCountdownInterval: ReturnType<typeof setInterval> | null = null;
  let refreshRequestId = 0;
  let valueRequestId = 0;
  let prettyPrintJson = false;
  let dbLocked = false;
  let valueFormatPrefsByDatabase: Record<
    string,
    { prettyPrintJson: boolean; dbLocked: boolean }
  > = {};

  type TabViewState = {
    keys: string[];
    selectedKey: string | null;
    keySearchInput: string;
    debouncedKeySearch: string;
    keyListScrollTop: number | null;
    originalValueRaw: string;
    editorValueRaw: string;
    isValueEditing: boolean;
  };
  let tabStateMap: Record<string, TabViewState> = {};
  let keyListViewportEl: HTMLDivElement | null = null;
  let pendingKeyListScrollTop: number | null | undefined = undefined;
  $: showEditorLoadingOverlay = loading || isRefreshing;
  $: showValueLoadingOverlay = valueLoading;
  $: editorLoadingOverlayText = isRefreshing
    ? "Refreshing database…"
    : "Loading keys…";

  function getSelectedKeyRowElement(viewportEl: HTMLDivElement) {
    return viewportEl.querySelector<HTMLElement>(
      '[data-key-item-selected="true"]'
    );
  }

  function restoreKeyListScrollForSelectedKey() {
    const viewportEl = keyListViewportEl;
    if (!viewportEl || !selectedKey) return;

    const selectedRowEl = getSelectedKeyRowElement(viewportEl);
    if (!selectedRowEl) return;

    selectedRowEl.scrollIntoView({ behavior: "smooth", block: "nearest" });
  }

  function restoreKeyListScroll(scrollTop: number | null) {
    const viewportEl = keyListViewportEl;
    if (!viewportEl) return;
    if (scrollTop !== null) {
      viewportEl.scrollTop = scrollTop;
      return;
    }
    restoreKeyListScrollForSelectedKey();
  }

  function queueKeyListScrollRestore(scrollTop: number | null) {
    pendingKeyListScrollTop = scrollTop;
  }

  function handleKeyListScroll() {
    const activeTab = getActiveTab();
    if (
      !activeTab ||
      activeTab.type !== "database" ||
      !dbPath ||
      activeTab.path !== dbPath ||
      !keyListViewportEl
    ) {
      return;
    }
    const existing = tabStateMap[activeTab.id];
    if (!existing) return;
    existing.keyListScrollTop = keyListViewportEl.scrollTop;
    tabStateMap = tabStateMap;
  }

  function getHexValidationError(display: string, label: string): string {
    if (!display.startsWith("0x")) return "";
    const raw = display.slice(2);
    if (raw.length % 2 !== 0) {
      return `${label} hex input must have an even number of characters.`;
    }
    if (!/^[0-9a-fA-F]*$/.test(raw)) {
      return `${label} hex input may only contain 0-9 and a-f.`;
    }
    return "";
  }

  $: isDirty = selectedKey !== null && editorValueRaw !== originalValueRaw;
  $: valueValidationError = getHexValidationError(editorValueRaw, "Value");
  $: createKeyValidationError = getHexValidationError(newKeyInput, "New key");
  $: createValueValidationError = getHexValidationError(
    newValueInput,
    "New value"
  );
  $: renameValidationError = getHexValidationError(renameInput, "New key");
  $: filteredKeys = debouncedKeySearch
    ? keys.filter((key) => key.toLowerCase().includes(debouncedKeySearch))
    : keys;
  $: scheduleDebouncedKeySearch(keySearchInput);
  $: autoRefreshLabel = getAutoRefreshLabel(autoRefreshIntervalMs);
  $: remainingAutoRefreshMs =
    autoRefreshIntervalMs > 0 && nextRefreshAt
      ? Math.max(0, nextRefreshAt - countdownNow)
      : 0;
  $: autoRefreshProgress =
    autoRefreshIntervalMs > 0
      ? Math.max(0, Math.min(1, remainingAutoRefreshMs / autoRefreshIntervalMs))
      : 1;

  function scheduleDebouncedKeySearch(value: string) {
    if (keySearchDebounceTimeout) {
      clearTimeout(keySearchDebounceTimeout);
    }
    keySearchDebounceTimeout = setTimeout(() => {
      debouncedKeySearch = value.trim().toLowerCase();
    }, KEY_SEARCH_DEBOUNCE_MS);
  }

  function clearKeySearch() {
    keySearchInput = "";
    debouncedKeySearch = "";
    if (keySearchDebounceTimeout) {
      clearTimeout(keySearchDebounceTimeout);
      keySearchDebounceTimeout = null;
    }
  }

  function clearSelection() {
    valueRequestId += 1;
    selectedKey = null;
    originalValueRaw = "";
    editorValueRaw = "";
    editorValue = "";
    isValueEditing = false;
  }

  function tryFormatJson(value: string): {
    formatted: string;
    isJson: boolean;
  } {
    try {
      const parsed = JSON.parse(value);
      return { formatted: JSON.stringify(parsed, null, 2), isJson: true };
    } catch {
      return { formatted: value, isJson: false };
    }
  }

  function formatValueForDisplay(value: string): string {
    if (!prettyPrintJson) return value;
    return tryFormatJson(value).formatted;
  }

  function syncEditorDisplayWithRawValue() {
    editorValue = formatValueForDisplay(editorValueRaw);
  }

  function loadValueFormatPrefs() {
    try {
      const stored = localStorage.getItem(VALUE_FORMAT_PREFS_STORAGE_KEY);
      if (!stored) {
        valueFormatPrefsByDatabase = {};
        return;
      }
      const parsed = JSON.parse(stored) as Record<
        string,
        { prettyPrintJson?: unknown; dbLocked?: unknown }
      >;
      valueFormatPrefsByDatabase = Object.entries(parsed ?? {}).reduce(
        (acc, [path, preference]) => {
          if (!path || typeof preference !== "object" || preference === null) {
            return acc;
          }
          acc[path] = {
            prettyPrintJson: Boolean(preference.prettyPrintJson),
            dbLocked: Boolean(preference.dbLocked),
          };
          return acc;
        },
        {} as Record<string, { prettyPrintJson: boolean; dbLocked: boolean }>
      );
    } catch {
      valueFormatPrefsByDatabase = {};
    }
  }

  function saveValueFormatPrefs() {
    try {
      localStorage.setItem(
        VALUE_FORMAT_PREFS_STORAGE_KEY,
        JSON.stringify(valueFormatPrefsByDatabase)
      );
    } catch {
      // ignore localStorage failures
    }
  }

  function getPrettyPrintEnabled(path: string): boolean {
    return Boolean(valueFormatPrefsByDatabase[path]?.prettyPrintJson);
  }

  function getDatabaseLocked(path: string): boolean {
    return Boolean(valueFormatPrefsByDatabase[path]?.dbLocked);
  }

  function setPrettyPrintEnabled(path: string, enabled: boolean) {
    if (!path) return;
    const existing = valueFormatPrefsByDatabase[path];
    valueFormatPrefsByDatabase = {
      ...valueFormatPrefsByDatabase,
      [path]: {
        prettyPrintJson: enabled,
        dbLocked: Boolean(existing?.dbLocked),
      },
    };
    saveValueFormatPrefs();
  }

  function setDatabaseLocked(path: string, locked: boolean) {
    if (!path) return;
    const existing = valueFormatPrefsByDatabase[path];
    valueFormatPrefsByDatabase = {
      ...valueFormatPrefsByDatabase,
      [path]: {
        prettyPrintJson: Boolean(existing?.prettyPrintJson),
        dbLocked: locked,
      },
    };
    saveValueFormatPrefs();
  }

  async function toggleDatabaseLock() {
    if (!dbPath) return;
    const nextLocked = !dbLocked;

    try {
      await LevelDBService.SetDatabaseLocked(dbPath, nextLocked);
      dbLocked = nextLocked;
      setDatabaseLocked(dbPath, nextLocked);

      if (nextLocked) {
        showCreateForm = false;
        resetRenameForm();
        keyPendingDelete = null;
        if (isValueEditing) {
          revertValueChanges();
        }
      }
    } catch (err) {
      await Dialogs.Error({
        Title: "Database lock update failed",
        Message: err instanceof Error ? err.message : String(err),
      });
    }
  }

  function togglePrettyPrintJson() {
    if (!dbPath) return;
    prettyPrintJson = !prettyPrintJson;
    setPrettyPrintEnabled(dbPath, prettyPrintJson);
    syncEditorDisplayWithRawValue();
  }

  function handleValueInput(event: Event) {
    const target = event.currentTarget;
    if (!(target instanceof HTMLTextAreaElement)) return;
    editorValue = target.value;
    editorValueRaw = target.value;
  }

  function getAutoRefreshLabel(intervalMs: number) {
    return (
      AUTO_REFRESH_OPTIONS.find((option) => option.intervalMs === intervalMs)
        ?.label ?? "Custom"
    );
  }

  function clearAutoRefreshTimer() {
    if (!autoRefreshTimeout) return;
    clearTimeout(autoRefreshTimeout);
    autoRefreshTimeout = null;
  }

  function clearRefreshCountdownTicker() {
    if (!refreshCountdownInterval) return;
    clearInterval(refreshCountdownInterval);
    refreshCountdownInterval = null;
  }

  function startRefreshCountdownTicker() {
    clearRefreshCountdownTicker();
    if (!dbPath || autoRefreshIntervalMs <= 0) return;
    countdownNow = Date.now();
    refreshCountdownInterval = setInterval(() => {
      countdownNow = Date.now();
    }, REFRESH_COUNTDOWN_TICK_MS);
  }

  function scheduleNextAutoRefresh() {
    clearAutoRefreshTimer();
    if (!dbPath || autoRefreshIntervalMs <= 0) {
      nextRefreshAt = null;
      return;
    }

    const scheduledAt = Date.now();
    nextRefreshAt = scheduledAt + autoRefreshIntervalMs;
    countdownNow = scheduledAt;
    autoRefreshTimeout = setTimeout(() => {
      void runAutoRefreshTick();
    }, autoRefreshIntervalMs);
  }

  function stopAutoRefresh() {
    autoRefreshIntervalMs = 0;
    nextRefreshAt = null;
    isRefreshMenuOpen = false;
    clearAutoRefreshTimer();
    clearRefreshCountdownTicker();
  }

  function setAutoRefreshInterval(intervalMs: number) {
    autoRefreshIntervalMs = intervalMs;
    isRefreshMenuOpen = false;
    if (intervalMs <= 0 || !dbPath) {
      nextRefreshAt = null;
      clearAutoRefreshTimer();
      clearRefreshCountdownTicker();
      return;
    }
    startRefreshCountdownTicker();
    scheduleNextAutoRefresh();
  }

  function getRefreshButtonTitle() {
    if (autoRefreshIntervalMs <= 0) return "Refresh now";
    const seconds = Math.ceil(remainingAutoRefreshMs / 1000);
    return `Refresh now (auto ${autoRefreshLabel}, ${seconds}s remaining)`;
  }

  async function waitForMinimumRefreshFeedback(startedAtMs: number) {
    const elapsedMs = Date.now() - startedAtMs;
    const remainingMs = MIN_REFRESH_FEEDBACK_MS - elapsedMs;
    if (remainingMs <= 0) return;
    await new Promise((resolve) => setTimeout(resolve, remainingMs));
  }

  async function confirmDiscardUnsavedChanges() {
    if (!isDirty) return true;
    return window.confirm(
      "You have unsaved value edits. Discard these changes?"
    );
  }

  function resetCreateForm() {
    newKeyInput = "";
    newValueInput = "";
    showCreateForm = false;
  }

  function resetRenameForm() {
    renameInput = "";
    editingKey = null;
  }

  function getContainerWidth() {
    return editorSplitContainer?.clientWidth ?? window.innerWidth;
  }

  function getSplitContentWidth(containerWidth: number) {
    return Math.max(0, containerWidth - SPLIT_CONTAINER_HORIZONTAL_INSET_PX);
  }

  function clampKeyPaneWidth(
    width: number,
    containerWidth: number = getContainerWidth()
  ) {
    const contentWidth = getSplitContentWidth(containerWidth);
    const maxKeyWidth = Math.max(
      MIN_KEY_PANE_WIDTH,
      contentWidth - MIN_VALUE_PANE_WIDTH - GRID_GAP_PX
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
        String(Math.round(keyPaneWidth))
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

  function removeFromRecent(path: string) {
    const paths = recentPaths.filter((p) => p.path !== path);
    saveRecentPaths(paths);
  }

  function getTabLabel(path: string) {
    return path.split(/[/\\]/).pop() || path;
  }

  function getActiveTab() {
    return tabs.find((tab) => tab.id === activeTabId) ?? null;
  }

  function findDatabaseTabByPath(path: string) {
    return (
      tabs.find((tab) => tab.type === "database" && tab.path === path) ?? null
    );
  }

  function captureCurrentTabState() {
    const activeTab = getActiveTab();
    if (
      !activeTab ||
      activeTab.type !== "database" ||
      !dbPath ||
      activeTab.path !== dbPath
    ) {
      return;
    }
    tabStateMap[activeTab.id] = {
      keys: [...keys],
      selectedKey,
      keySearchInput,
      debouncedKeySearch,
      keyListScrollTop:
        keyListViewportEl?.scrollTop ??
        tabStateMap[activeTab.id]?.keyListScrollTop ??
        null,
      originalValueRaw,
      editorValueRaw,
      isValueEditing,
    };
  }

  function restoreTabState(state: TabViewState): number | null {
    keys = [...state.keys];
    keySearchInput = state.keySearchInput;
    debouncedKeySearch = state.debouncedKeySearch;
    if (state.selectedKey && !state.keys.includes(state.selectedKey)) {
      clearSelection();
    } else {
      selectedKey = state.selectedKey;
      originalValueRaw = state.originalValueRaw;
      editorValueRaw = state.editorValueRaw;
      syncEditorDisplayWithRawValue();
      isValueEditing = state.isValueEditing && state.selectedKey !== null;
    }
    valueLoading = false;
    return state.keyListScrollTop;
  }

  function resetEditorViewState() {
    valueRequestId += 1;
    stopAutoRefresh();
    dbPath = "";
    prettyPrintJson = false;
    dbLocked = false;
    keys = [];
    filteredKeys = [];
    keySearchInput = "";
    debouncedKeySearch = "";
    if (keySearchDebounceTimeout) {
      clearTimeout(keySearchDebounceTimeout);
      keySearchDebounceTimeout = null;
    }
    clearSelection();
    resetCreateForm();
    resetRenameForm();
    keyPendingDelete = null;
    isValueEditing = false;
    isRefreshMenuOpen = false;
    isViewMenuOpen = false;
  }

  async function activateTab(
    tabId: string,
    options: { skipDirtyCheck?: boolean; force?: boolean } = {}
  ) {
    if (tabId === activeTabId && !options.force) return;
    if (
      !options.skipDirtyCheck &&
      isDirty &&
      !(await confirmDiscardUnsavedChanges())
    )
      return;

    const nextTab = tabs.find((tab) => tab.id === tabId);
    if (!nextTab) return;

    captureCurrentTabState();

    activeTabId = tabId;

    if (nextTab.type === "dashboard") {
      resetEditorViewState();
      return;
    }

    dbPath = nextTab.path;
    prettyPrintJson = getPrettyPrintEnabled(dbPath);
    dbLocked = getDatabaseLocked(dbPath);
    await LevelDBService.SetDatabaseLocked(dbPath, dbLocked);
    const saved = tabStateMap[tabId];
    if (saved) {
      const savedScrollTop = restoreTabState(saved);
      await tick();
      queueKeyListScrollRestore(savedScrollTop);
      return;
    }

    await loadKeys();
  }

  $: if (pendingKeyListScrollTop !== undefined && keyListViewportEl) {
    restoreKeyListScroll(pendingKeyListScrollTop);
    pendingKeyListScrollTop = undefined;
  }

  async function addDashboardTab() {
    const tab: WorkspaceTab = {
      id: `tab-${nextTabId++}`,
      type: "dashboard",
      title: "Dashboard",
    };
    tabs = [...tabs, tab];
    await activateTab(tab.id);
  }

  function handleTabValueChange(nextTabIdValue: string | undefined) {
    if (!nextTabIdValue) return;
    void activateTab(nextTabIdValue);
  }

  function canCloseTab(tab: WorkspaceTab): boolean {
    return !(tabs.length === 1 && tab.type === "dashboard");
  }

  async function closeTab(tabId: string) {
    const tabIndex = tabs.findIndex((tab) => tab.id === tabId);
    if (tabIndex === -1) return;
    const tab = tabs[tabIndex];
    if (!canCloseTab(tab)) return;

    const closingActiveTab = tabId === activeTabId;
    if (closingActiveTab && isDirty && !(await confirmDiscardUnsavedChanges()))
      return;

    if (tab.type === "database") {
      try {
        await LevelDBService.CloseDatabase(tab.path);
      } catch {
        // ignore close errors while closing tab
      }
    }

    delete tabStateMap[tabId];
    tabStateMap = tabStateMap;

    tabs = tabs.filter((item) => item.id !== tabId);
    if (tabs.length === 0) {
      const dashboardTab: WorkspaceTab = {
        id: tab.id,
        type: "dashboard",
        title: "Dashboard",
      };
      tabs = [dashboardTab];
      await activateTab(dashboardTab.id, { skipDirtyCheck: true, force: true });
      return;
    }

    if (!closingActiveTab) return;

    const nextIndex = Math.max(0, Math.min(tabIndex, tabs.length - 1));
    await activateTab(tabs[nextIndex].id, { skipDirtyCheck: true });
  }

  async function openDatabaseFromPath(path: string) {
    if (!path || path.trim() === "") return;

    loading = true;
    errorMessage = "";

    try {
      const result: OpenDatabaseResult =
        await LevelDBService.OpenDatabase(path);
      if (result.ok) {
        const canonicalPath = result.canonicalPath || path;
        addToRecent(canonicalPath);
        const activeTab = getActiveTab();
        const activeDashboardTabId =
          activeTab && activeTab.type === "dashboard" ? activeTab.id : null;
        const existingTab = findDatabaseTabByPath(canonicalPath);
        if (existingTab) {
          if (result.alreadyOpen) {
            try {
              await LevelDBService.CloseDatabase(canonicalPath);
            } catch {
              // ignore lease rebalance failures while reusing an existing tab
            }
          }
          if (activeDashboardTabId && activeDashboardTabId !== existingTab.id) {
            tabs = tabs.filter((tab) => tab.id !== activeDashboardTabId);
          }
          await activateTab(existingTab.id, { skipDirtyCheck: true });
          return;
        }

        if (activeDashboardTabId) {
          delete tabStateMap[activeDashboardTabId];
          tabStateMap = tabStateMap;
          const replacementTab: WorkspaceTab = {
            id: activeDashboardTabId,
            type: "database",
            title: getTabLabel(canonicalPath),
            path: canonicalPath,
          };
          tabs = tabs.map((tab) =>
            tab.id === activeDashboardTabId ? replacementTab : tab
          );
          await activateTab(activeDashboardTabId, {
            skipDirtyCheck: true,
            force: true,
          });
          return;
        }

        const newTab: WorkspaceTab = {
          id: `tab-${nextTabId++}`,
          type: "database",
          title: getTabLabel(canonicalPath),
          path: canonicalPath,
        };
        tabs = [...tabs, newTab];
        await activateTab(newTab.id);
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

  async function reloadDatabase({
    preserveSelection = false,
  }: {
    preserveSelection?: boolean;
  } = {}) {
    const requestId = ++refreshRequestId;
    const selectedBeforeReload = preserveSelection ? selectedKey : null;
    loading = true;
    try {
      const keyList = await LevelDBService.GetKeys(dbPath);
      if (requestId !== refreshRequestId) return false;
      keys = keyList ?? [];

      if (!selectedBeforeReload) {
        clearSelection();
        return true;
      }

      if (!keys.includes(selectedBeforeReload)) {
        clearSelection();
        return true;
      }

      await fetchValueForSelectedKey(selectedBeforeReload);
      if (requestId !== refreshRequestId) return false;
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      errorMessage = message;
      throw new Error(message);
    } finally {
      loading = false;
    }
  }

  async function loadKeys() {
    await reloadDatabase();
  }

  async function refreshDatabase(source: "manual" | "auto") {
    if (!dbPath || isRefreshing || loading) {
      if (source === "auto" && dbPath && autoRefreshIntervalMs > 0) {
        scheduleNextAutoRefresh();
      }
      return;
    }

    if (isDirty && !(await confirmDiscardUnsavedChanges())) {
      if (source === "auto" && autoRefreshIntervalMs > 0) {
        scheduleNextAutoRefresh();
      }
      return;
    }

    isRefreshMenuOpen = false;
    isRefreshing = true;
    const refreshStartedAtMs = Date.now();
    try {
      await reloadDatabase({ preserveSelection: true });
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      await Dialogs.Error({
        Title: source === "auto" ? "Auto-refresh failed" : "Refresh failed",
        Message: `Could not refresh the open database. ${message}`.trim(),
      });
    } finally {
      await waitForMinimumRefreshFeedback(refreshStartedAtMs);
      isRefreshing = false;
      if (autoRefreshIntervalMs > 0) {
        scheduleNextAutoRefresh();
      }
    }
  }

  async function runAutoRefreshTick() {
    if (!dbPath || autoRefreshIntervalMs <= 0) return;
    await refreshDatabase("auto");
  }

  function triggerManualRefresh() {
    void refreshDatabase("manual");
  }

  async function fetchValueForSelectedKey(key: string) {
    const requestId = ++valueRequestId;
    selectedKey = key;
    isValueEditing = false;
    valueLoading = true;
    originalValueRaw = "";
    editorValueRaw = "";
    editorValue = "";

    try {
      const value = await LevelDBService.GetValue(dbPath, key);
      if (requestId !== valueRequestId || selectedKey !== key) return;
      originalValueRaw = value ?? "";
      editorValueRaw = originalValueRaw;
      editorValue = formatValueForDisplay(editorValueRaw);
    } catch (err) {
      if (requestId !== valueRequestId || selectedKey !== key) return;
      const message = `Error: ${err instanceof Error ? err.message : String(err)}`;
      originalValueRaw = message;
      editorValueRaw = message;
      editorValue = formatValueForDisplay(editorValueRaw);
    } finally {
      if (requestId === valueRequestId) {
        valueLoading = false;
      }
    }
  }

  async function selectKey(key: string) {
    if (selectedKey === key) return;
    if (!(await confirmDiscardUnsavedChanges())) return;
    resetRenameForm();
    await fetchValueForSelectedKey(key);
  }

  function startValueEdit() {
    if (!selectedKey || valueLoading || isSaving || dbLocked) return;
    isValueEditing = true;
  }

  async function acceptRemoteConflictValue(
    key: string,
    currentValueExists: boolean,
    currentValue: string
  ) {
    if (currentValueExists) {
      originalValueRaw = currentValue;
      editorValueRaw = currentValue;
      syncEditorDisplayWithRawValue();
      isValueEditing = false;
      return;
    }

    await loadKeys();
    if (!keys.includes(key)) {
      selectedKey = null;
      originalValueRaw = "";
      editorValueRaw = "";
      editorValue = "";
      isValueEditing = false;
      return;
    }

    await fetchValueForSelectedKey(key);
  }

  async function saveValue() {
    if (
      !selectedKey ||
      !isValueEditing ||
      isSaving ||
      dbLocked ||
      !isDirty ||
      valueValidationError
    ) {
      return;
    }

    const keyToSave = selectedKey;
    const expectedValueRaw = originalValueRaw;
    const localValueRaw = editorValueRaw;

    isSaving = true;
    try {
      const saveResult = await LevelDBService.PutValueIfUnchanged(
        dbPath,
        keyToSave,
        expectedValueRaw,
        localValueRaw,
        false
      );

      if (saveResult.conflict) {
        const overwriteLocalChanges = window.confirm(
          "This value was changed by another application after you started editing.\n\n" +
            "Press OK to overwrite with your local changes, or Cancel to accept the remote value and discard your local edits."
        );

        if (overwriteLocalChanges) {
          const overwriteResult = await LevelDBService.PutValueIfUnchanged(
            dbPath,
            keyToSave,
            expectedValueRaw,
            localValueRaw,
            true
          );
          if (!overwriteResult.ok) {
            throw new Error("Failed to overwrite value.");
          }
          originalValueRaw = localValueRaw;
          syncEditorDisplayWithRawValue();
          isValueEditing = false;
          return;
        }

        await acceptRemoteConflictValue(
          keyToSave,
          saveResult.currentValueExists,
          saveResult.currentValue
        );
        return;
      }

      if (!saveResult.ok) {
        throw new Error("Save failed.");
      }

      originalValueRaw = localValueRaw;
      syncEditorDisplayWithRawValue();
      isValueEditing = false;
    } catch (err) {
      await Dialogs.Error({
        Title: "Save failed",
        Message: err instanceof Error ? err.message : String(err),
      });
    } finally {
      isSaving = false;
    }
  }

  function revertValueChanges() {
    editorValueRaw = originalValueRaw;
    syncEditorDisplayWithRawValue();
    isValueEditing = false;
  }

  function startRename(key: string) {
    if (isRenaming || dbLocked) return;
    editingKey = key;
    renameInput = key;
    void tick().then(() => {
      if (!renameInputElement) return;
      renameInputElement.focus();
      const end = renameInputElement.value.length;
      renameInputElement.setSelectionRange(end, end);
    });
  }

  async function createKey() {
    if (
      isCreating ||
      dbLocked ||
      createKeyValidationError ||
      createValueValidationError
    ) {
      return;
    }
    if (!newKeyInput) {
      await Dialogs.Error({
        Title: "Invalid key",
        Message: "Key cannot be empty.",
      });
      return;
    }
    if (keys.includes(newKeyInput)) {
      await Dialogs.Error({
        Title: "Key already exists",
        Message: "Choose a different key name.",
      });
      return;
    }
    if (!(await confirmDiscardUnsavedChanges())) return;

    isCreating = true;
    try {
      await LevelDBService.PutValue(dbPath, newKeyInput, newValueInput);
      const createdKey = newKeyInput;
      await loadKeys();
      resetCreateForm();
      await fetchValueForSelectedKey(createdKey);
    } catch (err) {
      await Dialogs.Error({
        Title: "Create key failed",
        Message: err instanceof Error ? err.message : String(err),
      });
    } finally {
      isCreating = false;
    }
  }

  async function renameEditingKey() {
    if (!editingKey || isRenaming || dbLocked || renameValidationError) return;
    if (!renameInput) {
      await Dialogs.Error({
        Title: "Invalid key",
        Message: "New key cannot be empty.",
      });
      return;
    }
    if (renameInput === editingKey) {
      resetRenameForm();
      return;
    }
    if (keys.includes(renameInput)) {
      await Dialogs.Error({
        Title: "Key already exists",
        Message: "Choose a different key name.",
      });
      return;
    }
    if (isDirty && !(await confirmDiscardUnsavedChanges())) return;

    isRenaming = true;
    try {
      await LevelDBService.RenameKey(dbPath, editingKey, renameInput);
      const renamedKey = renameInput;
      await loadKeys();
      resetRenameForm();
      await fetchValueForSelectedKey(renamedKey);
    } catch (err) {
      await Dialogs.Error({
        Title: "Rename key failed",
        Message: err instanceof Error ? err.message : String(err),
      });
    } finally {
      isRenaming = false;
    }
  }

  function requestDeleteKey(key: string) {
    if (isDeleting || dbLocked) return;
    keyPendingDelete = key;
  }

  async function confirmDeleteKey() {
    if (!keyPendingDelete || isDeleting || dbLocked) return;
    const targetKey = keyPendingDelete;
    if (
      isDirty &&
      selectedKey === targetKey &&
      !(await confirmDiscardUnsavedChanges())
    ) {
      return;
    }

    isDeleting = true;
    try {
      await LevelDBService.DeleteKey(dbPath, targetKey);
      await loadKeys();
      if (editingKey === targetKey) {
        resetRenameForm();
      }
      keyPendingDelete = null;
    } catch (err) {
      await Dialogs.Error({
        Title: "Delete key failed",
        Message: err instanceof Error ? err.message : String(err),
      });
    } finally {
      isDeleting = false;
    }
  }

  async function handleTitlebarDoubleClick() {
    const platformHint =
      `${navigator.platform} ${navigator.userAgent}`.toLowerCase();
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

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && (isRefreshMenuOpen || isViewMenuOpen)) {
        isRefreshMenuOpen = false;
        isViewMenuOpen = false;
        return;
      }

      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "s") {
        if (
          selectedKey &&
          isValueEditing &&
          isDirty &&
          !dbLocked &&
          !isSaving &&
          !valueValidationError
        ) {
          event.preventDefault();
          void saveValue();
        }
        return;
      }

      if (event.key === "Escape") {
        if (editingKey) {
          resetRenameForm();
          return;
        }
        if (keyPendingDelete && !isDeleting) {
          keyPendingDelete = null;
          return;
        }
        if (isValueEditing) {
          revertValueChanges();
          return;
        }
        if (showCreateForm) {
          resetCreateForm();
          return;
        }
      }
    };

    const onPointerDown = (event: PointerEvent) => {
      const target = event.target;
      if (!(target instanceof Node)) return;

      if (
        isRefreshMenuOpen &&
        refreshMenuContainer &&
        !refreshMenuContainer.contains(target)
      ) {
        isRefreshMenuOpen = false;
      }

      if (
        isViewMenuOpen &&
        viewMenuContainer &&
        !viewMenuContainer.contains(target)
      ) {
        isViewMenuOpen = false;
      }
    };

    window.addEventListener("keydown", onKeyDown);
    window.addEventListener("pointerdown", onPointerDown);

    return () => {
      mediaQuery.removeEventListener("change", updateLayoutMode);
      window.removeEventListener("resize", onWindowResize);
      window.removeEventListener("keydown", onKeyDown);
      window.removeEventListener("pointerdown", onPointerDown);
      stopPaneResize();
    };
  });

  onDestroy(() => {
    stopPaneResize();
    stopAutoRefresh();
    if (keySearchDebounceTimeout) {
      clearTimeout(keySearchDebounceTimeout);
    }
  });

  // Init
  loadRecentPaths();
  loadValueFormatPrefs();
</script>

{#if dbPath}
  <div class="editor-layout flex h-screen flex-col">
    <div
      class="titlebar-drag-region shrink-0 border-b border-border/50 bg-background/70 backdrop-blur-sm"
      role="none"
      on:dblclick={handleTitlebarDoubleClick}
    ></div>
    <div class="bg-background/80 pt-2 backdrop-blur-sm">
      <ScrollArea orientation="horizontal" class="w-full">
        <Tabs
          value={activeTabId}
          onValueChange={handleTabValueChange}
          class="w-full"
        >
          <TabsList
            class="relative h-auto w-max min-w-full items-end border-b border-border !border-x-0 !border-t-0 bg-transparent p-0 rounded-none"
          >
            {#each tabs as tab, index (tab.id)}
              <div
                class={`-mb-px flex max-w-[264px] items-center gap-1 rounded-t-md border border-transparent pl-1 pr-1 ${
                  activeTabId === tab.id
                    ? "relative z-10 border-t-border border-l-border border-r-border border-b-transparent bg-background text-foreground"
                    : `text-muted-foreground hover:bg-muted/40 ${
                        (index < tabs.length - 1 &&
                          activeTabId !== tabs[index + 1].id) ||
                        index === tabs.length - 1
                          ? "border-r-border/60"
                          : ""
                      }`
                }`}
              >
                <TabsTrigger
                  value={tab.id}
                  class="min-w-0 max-w-[220px] flex-1 rounded-none bg-transparent px-2 py-1.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
                  title={tab.type === "database" ? tab.path : tab.title}
                >
                  <span class="block truncate">
                    {tab.type === "database" ? tab.title : "Dashboard"}
                  </span>
                </TabsTrigger>
                {#if canCloseTab(tab)}
                  <Button
                    variant="ghost"
                    size="icon"
                    class="ml-0.5 h-6 w-6 shrink-0 text-muted-foreground hover:text-foreground"
                    title={`Close ${tab.title}`}
                    on:click={(event) => {
                      event.stopPropagation();
                      void closeTab(tab.id);
                    }}
                  >
                    <X class="h-3.5 w-3.5" />
                  </Button>
                {/if}
              </div>
            {/each}
            <Button
              variant="ghost"
              size="icon"
              class="h-7 w-7 rounded-t-md text-muted-foreground hover:bg-muted/40 hover:text-foreground"
              title="New dashboard tab"
              on:click={() => {
                void addDashboardTab();
              }}
            >
              <Plus class="h-4 w-4" />
            </Button>
          </TabsList>
        </Tabs>
      </ScrollArea>
    </div>
    <header
      class="relative z-40 overflow-visible bg-background/80 px-4 pt-3 pb-2 backdrop-blur-sm"
    >
      <div
        class="flex flex-col gap-2 md:flex-row md:items-center md:justify-between"
      >
        <div class="flex min-w-0 items-center justify-between gap-2 md:flex-1">
          <div class="flex min-w-0 items-center gap-2">
            <Badge
              variant="secondary"
              class="max-w-[70vw] truncate md:max-w-[40vw]"
              title={dbPath}
            >
              <Database class="mr-1.5 h-3.5 w-3.5" />
              {dbPath.split(/[/\\]/).pop() || dbPath}
            </Badge>
            {#if dbLocked}
              <Badge variant="outline">Read-only</Badge>
            {/if}
            {#if autoRefreshIntervalMs > 0}
              <Badge variant="outline">Auto {autoRefreshLabel}</Badge>
            {/if}
          </div>
        </div>

        <div
          class="flex items-center justify-between gap-2 md:shrink-0 md:justify-end"
        >
          <div
            bind:this={refreshMenuContainer}
            class="relative flex items-stretch"
          >
            <Button
              variant="outline"
              size="sm"
              class="gap-1.5 rounded-r-none border-r-0 pr-2"
              title={getRefreshButtonTitle()}
              disabled={isRefreshing}
              on:click={triggerManualRefresh}
            >
              {#if isRefreshing}
                <RefreshCcw class="h-3.5 w-3.5 animate-spin" />
              {:else if autoRefreshIntervalMs > 0}
                <svg
                  class="h-3.5 w-3.5 -rotate-90"
                  viewBox="0 0 16 16"
                  aria-hidden="true"
                >
                  <circle
                    cx="8"
                    cy="8"
                    r={REFRESH_RING_RADIUS}
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    class="opacity-20"
                  />
                  <circle
                    cx="8"
                    cy="8"
                    r={REFRESH_RING_RADIUS}
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-dasharray={`${REFRESH_RING_CIRCUMFERENCE} ${REFRESH_RING_CIRCUMFERENCE}`}
                    stroke-dashoffset={`${REFRESH_RING_CIRCUMFERENCE * (1 - autoRefreshProgress)}`}
                  />
                </svg>
              {:else}
                <RefreshCcw class="h-3.5 w-3.5" />
              {/if}
              {isRefreshing ? "Refreshing…" : "Refresh"}
            </Button>
            <Button
              variant="outline"
              size="sm"
              class="w-8 rounded-l-none px-0"
              aria-label="Auto-refresh options"
              aria-haspopup="menu"
              aria-expanded={isRefreshMenuOpen}
              disabled={isRefreshing}
              on:click={() => {
                if (isRefreshing) return;
                isViewMenuOpen = false;
                isRefreshMenuOpen = !isRefreshMenuOpen;
              }}
            >
              <ChevronDown class="h-3.5 w-3.5" />
            </Button>

            {#if isRefreshMenuOpen}
              <div
                class={`absolute right-0 top-10 z-50 min-w-40 rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-md ${
                  isRefreshing ? "pointer-events-none opacity-70" : ""
                }`}
                role="menu"
                aria-label="Auto-refresh interval options"
              >
                {#each AUTO_REFRESH_OPTIONS as option}
                  <button
                    type="button"
                    class={`flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-left text-sm transition-colors hover:bg-muted ${
                      option.intervalMs === autoRefreshIntervalMs
                        ? "bg-muted/80"
                        : ""
                    }`}
                    role="menuitemradio"
                    aria-checked={option.intervalMs === autoRefreshIntervalMs}
                    disabled={isRefreshing}
                    on:click={() => setAutoRefreshInterval(option.intervalMs)}
                  >
                    <span>{option.label}</span>
                    {#if option.intervalMs === autoRefreshIntervalMs}
                      <Check class="h-3.5 w-3.5" />
                    {/if}
                  </button>
                {/each}
              </div>
            {/if}
          </div>

          <div bind:this={viewMenuContainer} class="relative">
            <Button
              variant="outline"
              size="icon"
              class="h-8 w-8"
              title="View and safety settings"
              aria-label="View and safety settings"
              aria-haspopup="menu"
              aria-expanded={isViewMenuOpen}
              on:click={() => {
                isRefreshMenuOpen = false;
                isViewMenuOpen = !isViewMenuOpen;
              }}
            >
              <MoreHorizontal class="h-3.5 w-3.5" />
            </Button>

            {#if isViewMenuOpen}
              <div
                class="absolute right-0 top-10 z-50 min-w-48 rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-md"
                role="menu"
                aria-label="View and safety settings"
              >
                <button
                  type="button"
                  class={`flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-left text-sm transition-colors hover:bg-muted ${
                    dbLocked ? "bg-muted/80" : ""
                  }`}
                  role="menuitemcheckbox"
                  aria-checked={dbLocked}
                  on:click={() => {
                    void toggleDatabaseLock();
                  }}
                >
                  <span>Read-only</span>
                  <span
                    class={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border transition-colors ${
                      dbLocked
                        ? "border-primary/40 bg-primary"
                        : "border-border bg-muted"
                    }`}
                    aria-hidden="true"
                  >
                    <span
                      class={`inline-block h-4 w-4 rounded-full bg-background shadow-sm transition-transform ${
                        dbLocked ? "translate-x-4" : "translate-x-0.5"
                      }`}
                    ></span>
                  </span>
                </button>
                <button
                  type="button"
                  class={`flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-left text-sm transition-colors hover:bg-muted ${
                    prettyPrintJson ? "bg-muted/80" : ""
                  }`}
                  role="menuitemcheckbox"
                  aria-checked={prettyPrintJson}
                  on:click={() => {
                    togglePrettyPrintJson();
                  }}
                >
                  <span>Pretty JSON</span>
                  <span
                    class={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border transition-colors ${
                      prettyPrintJson
                        ? "border-primary/40 bg-primary"
                        : "border-border bg-muted"
                    }`}
                    aria-hidden="true"
                  >
                    <span
                      class={`inline-block h-4 w-4 rounded-full bg-background shadow-sm transition-transform ${
                        prettyPrintJson ? "translate-x-4" : "translate-x-0.5"
                      }`}
                    ></span>
                  </span>
                </button>
              </div>
            {/if}
          </div>
        </div>
      </div>
    </header>

    <div
      bind:this={editorSplitContainer}
      class="relative grid min-h-0 flex-1 gap-4 px-4 pb-4 pt-2 md:grid-cols-1"
      aria-busy={showEditorLoadingOverlay}
      style={isDesktopLayout
        ? `grid-template-columns: ${keyPaneWidth}px minmax(${MIN_VALUE_PANE_WIDTH}px, 1fr);`
        : undefined}
    >
      {#if isDesktopLayout}
        <button
          type="button"
          class="group absolute bottom-4 top-4 z-10 w-2 -translate-x-1/2 cursor-col-resize rounded-full bg-transparent"
          style={`left: ${SPLIT_CONTAINER_PADDING_PX + keyPaneWidth + GRID_GAP_PX / 2}px;`}
          aria-label="Resize key and value panes"
          on:pointerdown={startPaneResize}
        >
          <span
            class={`mx-auto block h-full rounded-full transition-all ${
              isResizingPane
                ? "w-0.5 bg-primary/55"
                : "w-px bg-border/55 group-hover:bg-border/75"
            }`}
          ></span>
        </button>
      {/if}

      <Card
        class="flex min-h-0 min-w-0 flex-col"
        style={isDesktopLayout
          ? `min-width: ${MIN_KEY_PANE_WIDTH}px;`
          : undefined}
      >
        <CardHeader class="space-y-3 pb-3">
          <div class="flex items-center justify-between gap-2">
            <CardTitle class="flex items-center gap-2 text-base">
              <KeyRound class="h-4 w-4" />
              Keys
            </CardTitle>
            <div class="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                class="gap-1.5"
                disabled={dbLocked}
                on:click={() => {
                  showCreateForm = !showCreateForm;
                  if (!showCreateForm) {
                    newKeyInput = "";
                    newValueInput = "";
                  }
                }}
              >
                <Plus class="h-3.5 w-3.5" />
                New
              </Button>
            </div>
          </div>
          <CardDescription>
            {#if keySearchInput.trim()}
              {filteredKeys.length} of {keys.length} entries
            {:else}
              {keys.length} entries
            {/if}
          </CardDescription>
        </CardHeader>
        <CardContent class="flex min-h-0 flex-1 flex-col gap-3 px-3 pb-3">
          <div class="relative w-full">
            <input
              class="w-full rounded-md border border-input bg-background px-2 py-1.5 pr-8 text-sm transition-colors focus-visible:border-primary focus-visible:outline-none focus-visible:ring-0"
              placeholder="Search keys..."
              bind:value={keySearchInput}
            />
            {#if keySearchInput.length > 0}
              <button
                type="button"
                class="absolute right-1.5 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                aria-label="Clear search"
                on:click={clearKeySearch}
              >
                <X class="h-3.5 w-3.5" />
              </button>
            {/if}
          </div>

          {#if showCreateForm}
            <div
              class="space-y-2 rounded-md border border-border/70 bg-muted/20 p-2"
            >
              <input
                class="w-full rounded-md border border-input bg-background px-2 py-1.5 text-sm"
                placeholder="New key (text or 0x...)"
                bind:value={newKeyInput}
                disabled={isCreating || dbLocked}
              />
              {#if createKeyValidationError}
                <p class="text-xs text-destructive">
                  {createKeyValidationError}
                </p>
              {/if}
              <textarea
                class="min-h-20 w-full rounded-md border border-input bg-background p-2 font-mono text-sm"
                placeholder="Initial value (text or 0x...)"
                bind:value={newValueInput}
                disabled={isCreating || dbLocked}
              ></textarea>
              {#if createValueValidationError}
                <p class="text-xs text-destructive">
                  {createValueValidationError}
                </p>
              {/if}
              <div class="flex items-center gap-2">
                <Button
                  size="sm"
                  class="gap-1.5"
                  disabled={isCreating ||
                    dbLocked ||
                    !!createKeyValidationError ||
                    !!createValueValidationError}
                  on:click={createKey}
                >
                  <Save class="h-3.5 w-3.5" />
                  {isCreating ? "Creating…" : "Create key"}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  on:click={resetCreateForm}
                  disabled={isCreating}
                >
                  Cancel
                </Button>
              </div>
            </div>
          {/if}

          <div class="relative min-h-0 flex-1">
            {#if keys.length === 0}
              <div class="px-2 py-3 text-sm text-muted-foreground">
                No keys yet. Use <strong>New</strong> to create your first key.
              </div>
            {:else if filteredKeys.length === 0}
              <div class="px-2 py-3 text-sm text-muted-foreground">
                No keys match "{keySearchInput.trim()}".
              </div>
            {:else}
              <div
                class="h-full min-h-0 overflow-auto rounded-md border border-border/70"
                bind:this={keyListViewportEl}
                on:scroll={handleKeyListScroll}
              >
                <ul class="w-full space-y-1 p-2">
                  {#each filteredKeys as key}
                    <li class="group w-full">
                      <div
                        data-key-item-selected={selectedKey === key
                          ? "true"
                          : undefined}
                        class={`flex items-center gap-1 rounded-md px-2 py-1.5 text-sm transition-colors ${
                          selectedKey === key
                            ? "bg-primary/15 text-primary"
                            : "hover:bg-muted"
                        }`}
                      >
                        {#if editingKey === key}
                          <div class="min-w-0 flex-1">
                            <input
                              class="h-7 w-full rounded-md border border-input bg-background px-2 text-xs text-foreground transition-colors focus-visible:border-primary focus-visible:outline-none focus-visible:ring-0"
                              bind:this={renameInputElement}
                              bind:value={renameInput}
                              disabled={isRenaming || dbLocked}
                              on:click|stopPropagation
                              on:keydown={(event) => {
                                if (event.key === "Enter") {
                                  event.preventDefault();
                                  void renameEditingKey();
                                }
                              }}
                            />
                            {#if renameValidationError}
                              <p class="mt-1 text-xs text-destructive">
                                {renameValidationError}
                              </p>
                            {/if}
                          </div>
                          <div class="ml-1 flex shrink-0 items-center gap-0.5">
                            <Button
                              variant="ghost"
                              size="icon"
                              class="h-7 w-7"
                              disabled={isRenaming ||
                                dbLocked ||
                                !renameInput ||
                                !!renameValidationError}
                              on:click={(event) => {
                                event.stopPropagation();
                                void renameEditingKey();
                              }}
                            >
                              <Check class="h-3.5 w-3.5" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon"
                              class="h-7 w-7"
                              disabled={isRenaming || dbLocked}
                              on:click={(event) => {
                                event.stopPropagation();
                                resetRenameForm();
                              }}
                            >
                              <X class="h-3.5 w-3.5" />
                            </Button>
                          </div>
                        {:else}
                          <button
                            class="min-w-0 flex-1 text-left"
                            title={key}
                            on:click={() => selectKey(key)}
                          >
                            <span class="block w-full truncate">{key}</span>
                          </button>
                          <div
                            class="ml-1 flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100"
                          >
                            <Button
                              variant="ghost"
                              size="icon"
                              class="h-7 w-7"
                              disabled={dbLocked}
                              on:click={(event) => {
                                event.stopPropagation();
                                startRename(key);
                              }}
                            >
                              <Pencil class="h-3.5 w-3.5" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon"
                              class="h-7 w-7 text-destructive hover:text-destructive"
                              disabled={isDeleting || dbLocked}
                              on:click={(event) => {
                                event.stopPropagation();
                                requestDeleteKey(key);
                              }}
                            >
                              <Trash2 class="h-3.5 w-3.5" />
                            </Button>
                          </div>
                        {/if}
                      </div>
                    </li>
                  {/each}
                </ul>
              </div>
            {/if}
          </div>
        </CardContent>
      </Card>

      <Card
        class="flex min-h-0 min-w-0 flex-col"
        style={isDesktopLayout
          ? `min-width: ${MIN_VALUE_PANE_WIDTH}px;`
          : undefined}
        aria-busy={showValueLoadingOverlay}
      >
        <CardHeader class="space-y-3 pb-3">
          <div class="flex items-center justify-between gap-2">
            <CardTitle class="flex items-center gap-2 text-base">
              <FileText class="h-4 w-4" />
              Value
            </CardTitle>
            <div class="flex items-center gap-2">
              {#if isDirty}
                <Badge variant="secondary">Unsaved</Badge>
              {/if}
              {#if isValueEditing}
                <Button
                  size="sm"
                  class="gap-1.5"
                  disabled={!selectedKey ||
                    !isDirty ||
                    !!valueValidationError ||
                    isSaving ||
                    dbLocked}
                  on:click={saveValue}
                >
                  <Save class="h-3.5 w-3.5" />
                  {isSaving ? "Saving…" : "Save"}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  class="gap-1.5"
                  disabled={isSaving}
                  on:click={revertValueChanges}
                >
                  <X class="h-3.5 w-3.5" />
                  Cancel
                </Button>
              {:else}
                <Button
                  size="sm"
                  class="gap-1.5"
                  disabled={!selectedKey ||
                    valueLoading ||
                    isSaving ||
                    dbLocked}
                  on:click={startValueEdit}
                >
                  <Pencil class="h-3.5 w-3.5" />
                  Edit
                </Button>
              {/if}
            </div>
          </div>
          <CardDescription>
            {#if selectedKey}
              <span class="block w-full truncate" title={selectedKey}
                >{selectedKey}</span
              >
            {:else}
              Select a key to inspect its value
            {/if}
          </CardDescription>
        </CardHeader>
        <CardContent class="flex min-h-0 flex-1 flex-col gap-2 px-3 pb-3">
          <div class="relative flex min-h-0 flex-1 flex-col">
            {#if selectedKey === null}
              <div class="px-1 py-3 text-sm text-muted-foreground">Select a key</div>
            {:else}
              <textarea
                class={`h-full min-h-0 flex-1 resize-none cursor-text select-text rounded-md border border-border/70 p-2 font-mono text-sm leading-relaxed transition-colors focus-visible:outline-none focus-visible:ring-0 ${
                  isValueEditing
                    ? "bg-background text-foreground focus-visible:border-primary"
                    : "bg-muted/50 text-foreground"
                }`}
                bind:value={editorValue}
                on:input={handleValueInput}
                readonly={!isValueEditing || isSaving || dbLocked}
                spellcheck="false"
              ></textarea>
              {#if valueValidationError}
                <p class="px-1 text-xs text-destructive">
                  {valueValidationError}
                </p>
              {/if}
            {/if}

            {#if showValueLoadingOverlay}
              <div
                class="absolute inset-0 z-10 flex items-center justify-center rounded-md bg-background/45 backdrop-blur-md supports-[backdrop-filter]:bg-background/35"
              >
                <div
                  role="status"
                  aria-live="polite"
                  class="flex items-center gap-2 rounded-full border border-border/80 bg-background/95 px-4 py-2 text-sm font-medium text-foreground shadow-lg ring-1 ring-border/40"
                >
                  <RefreshCcw class="h-4 w-4 animate-spin" />
                  <span>Loading value…</span>
                </div>
              </div>
            {/if}
          </div>
        </CardContent>
      </Card>

      {#if showEditorLoadingOverlay}
        <div
          class="absolute inset-0 z-20 flex items-center justify-center rounded-lg bg-background/45 backdrop-blur-md supports-[backdrop-filter]:bg-background/35"
        >
          <div
            role="status"
            aria-live="polite"
            class="flex items-center gap-2 rounded-full border border-border/80 bg-background/95 px-4 py-2 text-sm font-medium text-foreground shadow-lg ring-1 ring-border/40"
          >
            <RefreshCcw class="h-4 w-4 animate-spin" />
            <span>{editorLoadingOverlayText}</span>
          </div>
        </div>
      {/if}
    </div>
  </div>
{:else}
  <div class="flex h-screen flex-col">
    <div
      class="titlebar-drag-region shrink-0 border-b border-border/50 bg-background/70 backdrop-blur-sm"
      role="none"
      on:dblclick={handleTitlebarDoubleClick}
    ></div>
    <div class="bg-background/80 pt-2 backdrop-blur-sm">
      <ScrollArea orientation="horizontal" class="w-full">
        <Tabs
          value={activeTabId}
          onValueChange={handleTabValueChange}
          class="w-full"
        >
          <TabsList
            class="relative h-auto w-max min-w-full items-end border-b border-border !border-x-0 !border-t-0 bg-transparent p-0 rounded-none"
          >
            {#each tabs as tab, index (tab.id)}
              <div
                class={`-mb-px flex max-w-[264px] items-center gap-1 rounded-t-md border border-transparent pl-1 pr-1 ${
                  activeTabId === tab.id
                    ? "relative z-10 border-t-border border-l-border border-r-border border-b-transparent bg-background text-foreground"
                    : `text-muted-foreground hover:bg-muted/40 ${
                        (index < tabs.length - 1 &&
                          activeTabId !== tabs[index + 1].id) ||
                        index === tabs.length - 1
                          ? "border-r-border/60"
                          : ""
                      }`
                }`}
              >
                <TabsTrigger
                  value={tab.id}
                  class="min-w-0 max-w-[220px] flex-1 rounded-none bg-transparent px-2 py-1.5 data-[state=active]:bg-transparent data-[state=active]:shadow-none"
                  title={tab.type === "database" ? tab.path : tab.title}
                >
                  <span class="block truncate">
                    {tab.type === "database" ? tab.title : "Dashboard"}
                  </span>
                </TabsTrigger>
                {#if canCloseTab(tab)}
                  <Button
                    variant="ghost"
                    size="icon"
                    class="ml-0.5 h-6 w-6 shrink-0 text-muted-foreground hover:text-foreground"
                    title={`Close ${tab.title}`}
                    on:click={(event) => {
                      event.stopPropagation();
                      void closeTab(tab.id);
                    }}
                  >
                    <X class="h-3.5 w-3.5" />
                  </Button>
                {/if}
              </div>
            {/each}
            <Button
              variant="ghost"
              size="icon"
              class="h-7 w-7 rounded-t-md text-muted-foreground hover:bg-muted/40 hover:text-foreground"
              title="New dashboard tab"
              on:click={() => {
                void addDashboardTab();
              }}
            >
              <Plus class="h-4 w-4" />
            </Button>
          </TabsList>
        </Tabs>
      </ScrollArea>
    </div>
    <div class="flex min-h-0 flex-1 items-center justify-center p-6">
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
                  <div class="group relative">
                    <Button
                      variant="ghost"
                      class="w-full justify-start pr-10 font-normal"
                      on:click={() => openDatabaseFromPath(item.path)}
                      disabled={loading}
                    >
                      {item.label}
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      class="absolute right-1 top-1 h-7 w-7 opacity-0 transition-opacity pointer-events-none text-muted-foreground hover:text-foreground group-hover:opacity-100 group-hover:pointer-events-auto group-focus-within:opacity-100 group-focus-within:pointer-events-auto"
                      aria-label={`Remove ${item.label} from recently opened`}
                      title="Remove from recently opened"
                      on:click={(event) => {
                        event.stopPropagation();
                        removeFromRecent(item.path);
                      }}
                      disabled={loading}
                    >
                      <X class="h-3.5 w-3.5" />
                    </Button>
                  </div>
                {/each}
              </div>
            </section>
          {/if}
        </CardContent>
      </Card>
    </div>
  </div>
{/if}

{#if keyPendingDelete}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-background/70 p-4 backdrop-blur-sm"
    role="presentation"
    on:click={() => {
      if (!isDeleting) {
        keyPendingDelete = null;
      }
    }}
  >
    <Card
      class="w-full max-w-md border-destructive/40"
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-key-title"
      aria-describedby="delete-key-description"
      on:click={(event) => {
        event.stopPropagation();
      }}
    >
      <CardHeader class="space-y-2">
        <CardTitle id="delete-key-title">Delete key?</CardTitle>
        <CardDescription id="delete-key-description">
          Delete "{keyPendingDelete}"? This action cannot be undone.
        </CardDescription>
      </CardHeader>
      <CardContent class="flex justify-end gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={isDeleting || dbLocked}
          on:click={() => {
            keyPendingDelete = null;
          }}
        >
          Cancel
        </Button>
        <Button
          variant="destructive"
          size="sm"
          disabled={isDeleting || dbLocked}
          on:click={confirmDeleteKey}
        >
          <Trash2 class="mr-1.5 h-3.5 w-3.5" />
          {isDeleting ? "Deleting…" : "Delete"}
        </Button>
      </CardContent>
    </Card>
  </div>
{/if}
