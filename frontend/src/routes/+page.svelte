<script lang="ts">
  import { onDestroy, onMount, tick } from "svelte";
  import { Dialogs, Events, Window } from "@wailsio/runtime";
  import * as LevelDBService from "../../bindings/ldbeditor/leveldbservice";
  import { OpenDatabaseResult } from "../../bindings/ldbeditor/models";
  import * as WindowService from "../../bindings/ldbeditor/windowservice";
  import { shortcutConfig } from "$lib/shortcuts/config";
  import { ShortcutCommand } from "$lib/shortcuts/commands";
  import { createShortcutManager } from "$lib/shortcuts/manager";
  import {
    confirmCloseLastTabPreference,
    keyPaneWidthPreference,
    recentPathsPreference,
    valueFormatPreference,
  } from "$lib/preferences";
  import { Badge } from "$lib/components/ui/badge";
  import { Button } from "$lib/components/ui/button";
  import { DropdownMenuItem } from "$lib/components/ui/dropdown-menu";
  import { SplitButton } from "$lib/components/ui/split-button";
  import { ThreeDotMenu } from "$lib/components/ui/three-dot-menu";
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from "$lib/components/ui/card";
  import SettingsDialog from "$lib/components/settings-dialog.svelte";
  import TabStrip from "$lib/components/tab-strip.svelte";
  import ValueCodeMirror from "$lib/components/value-codemirror.svelte";
  import {
    Check,
    Copy,
    Database,
    DatabaseZap,
    FileText,
    FolderOpen,
    History,
    KeyRound,
    Pencil,
    Plus,
    RefreshCcw,
    Save,
    Trash2,
    X,
  } from "lucide-svelte";

  const KEY_SEARCH_DEBOUNCE_MS = 300;
  const DEFAULT_KEY_PANE_WIDTH = 320;
  const MIN_KEY_PANE_WIDTH = 300;
  const MIN_VALUE_PANE_WIDTH = 320;
  const GRID_GAP_PX = 16;
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
  const shortcutManager = createShortcutManager(shortcutConfig);

  // Startup view state
  let isOpeningDatabase = false;
  let isCreatingDatabase = false;
  let isLoadingKeys = false;
  let errorMessage = "";
  let pendingCreatePath: string | null = null;
  type WorkspaceTab =
    | { id: string; type: "dashboard"; title: string }
    | { id: string; type: "database"; title: string; path: string };
  type PendingUnsavedClose = { tabId: string; closeWindowAfter: boolean };
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
  let pendingUnsavedClose: PendingUnsavedClose | null = null;
  let pendingCloseLastTab = false;
  let showSettingsDialog = false;
  let isValueEditing = false;
  let valueLoading = false;
  let isDesktopLayout = false;
  let keyPaneWidth = DEFAULT_KEY_PANE_WIDTH;
  let isResizingPane = false;
  let editorSplitContainer: HTMLDivElement | null = null;
  let detachPointerListeners: (() => void) | null = null;
  let isDirty = false;
  let dirtyTabIds = new Set<string>();
  let valueValidationError = "";
  let createKeyValidationError = "";
  let createValueValidationError = "";
  let renameValidationError = "";
  let keySearchInput = "";
  let debouncedKeySearch = "";
  let keySearchDebounceTimeout: ReturnType<typeof setTimeout> | null = null;
  let hasCopiedValue = false;
  let copyValueFeedbackTimeout: ReturnType<typeof setTimeout> | null = null;
  let renameInputElement: HTMLInputElement | null = null;
  let isRefreshing = false;
  let autoRefreshIntervalMs = 0;
  let nextRefreshAt: number | null = null;
  let countdownNow = Date.now();
  let autoRefreshTimeout: ReturnType<typeof setTimeout> | null = null;
  let refreshCountdownInterval: ReturnType<typeof setInterval> | null = null;
  let refreshRequestId = 0;
  let valueRequestId = 0;
  let isWindowsRuntime = false;
  let pendingAltMenuToggle = false;
  let dbForcedReadOnly = false;
  let dbIntentionalReadOnly = false;
  let dbReadOnlyReason = "";
  let dbLockedByApp = "";
  let showLockedByAppInReadOnlyNotice = false;
  let forcedReadOnlyByDatabase: Record<
    string,
    {
      readOnlyReason: string;
      lockedByApp: string;
      intentionalReadOnly: boolean;
    }
  > = {};
  let effectiveReadOnly = false;
  $: prettyPrintJson = dbPath
    ? Boolean($valueFormatPreference[dbPath]?.prettyPrintJson)
    : false;

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
  $: showEditorLoadingOverlay = isLoadingKeys || isRefreshing;
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
    if (typeof display !== "string") return "";
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
  $: dirtyTabIds = new Set(
    tabs
      .filter((tab) => {
        if (tab.type !== "database") return false;
        if (tab.id === activeTabId) return isDirty;
        const state = tabStateMap[tab.id];
        return state ? state.editorValueRaw !== state.originalValueRaw : false;
      })
      .map((tab) => tab.id)
  );
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
  $: effectiveReadOnly = dbForcedReadOnly;
  $: showLockedByAppInReadOnlyNotice =
    Boolean(dbLockedByApp.trim()) &&
    !dbReadOnlyReason.toLowerCase().includes(dbLockedByApp.toLowerCase());
  $: remainingAutoRefreshMs =
    autoRefreshIntervalMs > 0 && nextRefreshAt
      ? Math.max(0, nextRefreshAt - countdownNow)
      : 0;
  $: autoRefreshProgress =
    autoRefreshIntervalMs > 0
      ? Math.max(0, Math.min(1, remainingAutoRefreshMs / autoRefreshIntervalMs))
      : 1;
  $: canSaveValueChanges =
    selectedKey !== null &&
    isValueEditing &&
    !effectiveReadOnly &&
    !isSaving &&
    !valueValidationError;
  $: canSaveAndClosePendingTab = (() => {
    if (!pendingUnsavedClose || isSaving) return false;

    const { tabId } = pendingUnsavedClose;
    if (tabId === activeTabId) return canSaveValueChanges;

    const tab = tabs.find((item) => item.id === tabId);
    if (!tab || tab.type !== "database") return false;
    if (getForcedReadOnlyState(tab.path)) return false;

    const state = tabStateMap[tabId];
    if (!state || state.selectedKey === null || !state.isValueEditing) {
      return false;
    }

    return !getHexValidationError(state.editorValueRaw, "Value");
  })();

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

  function getForcedReadOnlyState(path: string): {
    readOnlyReason: string;
    lockedByApp: string;
    intentionalReadOnly: boolean;
  } | null {
    const state = forcedReadOnlyByDatabase[path];
    if (!state) return null;
    return state;
  }

  function setForcedReadOnlyState(
    path: string,
    forcedReadOnly: boolean,
    readOnlyReason: string,
    lockedByApp: string,
    intentionalReadOnly: boolean
  ) {
    if (!path) return;
    if (!forcedReadOnly) {
      if (!forcedReadOnlyByDatabase[path]) return;
      const { [path]: _, ...rest } = forcedReadOnlyByDatabase;
      forcedReadOnlyByDatabase = rest;
      return;
    }
    forcedReadOnlyByDatabase = {
      ...forcedReadOnlyByDatabase,
      [path]: {
        readOnlyReason,
        lockedByApp,
        intentionalReadOnly,
      },
    };
  }

  function togglePrettyPrintJson() {
    if (!dbPath) return;
    const next = !valueFormatPreference.isPrettyPrintEnabled(dbPath);
    valueFormatPreference.setPrettyPrintForPath(dbPath, next);
    syncEditorDisplayWithRawValue();
  }

  function handleValueEditorChange(
    event: { value: string } | CustomEvent<{ value: string }>
  ) {
    const nextValue =
      "detail" in event ? (event.detail?.value ?? "") : (event.value ?? "");
    editorValue = nextValue;
    editorValueRaw = nextValue;
  }

  async function copyValueToClipboard() {
    if (!selectedKey || showValueLoadingOverlay) return;
    try {
      await navigator.clipboard.writeText(editorValue);
      hasCopiedValue = true;
      if (copyValueFeedbackTimeout) {
        clearTimeout(copyValueFeedbackTimeout);
      }
      copyValueFeedbackTimeout = setTimeout(() => {
        hasCopiedValue = false;
      }, 1600);
    } catch {
      await Dialogs.Error({
        Title: "Copy failed",
        Message: "Unable to copy value to clipboard.",
      });
    }
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
    clearAutoRefreshTimer();
    clearRefreshCountdownTicker();
  }

  function setAutoRefreshInterval(intervalMs: number) {
    autoRefreshIntervalMs = intervalMs;
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
    return Math.max(0, containerWidth);
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

  $: if (!isResizingPane && isDesktopLayout) {
    keyPaneWidth = clampKeyPaneWidth($keyPaneWidthPreference);
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
    keyPaneWidthPreference.set(Math.round(clampKeyPaneWidth(keyPaneWidth)));
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
        moveEvent.clientX - containerRect.left - GRID_GAP_PX / 2;
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
    pendingCreatePath = null;
    dbPath = "";
    prettyPrintJson = false;
    dbForcedReadOnly = false;
    dbIntentionalReadOnly = false;
    dbReadOnlyReason = "";
    dbLockedByApp = "";
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
    pendingUnsavedClose = null;
    isValueEditing = false;
  }

  async function finishOpeningDatabase(
    result: OpenDatabaseResult,
    fallbackPath: string
  ) {
    const canonicalPath = result.canonicalPath || fallbackPath;
    const forcedReadOnly = Boolean(result.forcedReadOnly);

    pendingCreatePath = null;
    setForcedReadOnlyState(
      canonicalPath,
      forcedReadOnly,
      result.readOnlyReason || "",
      result.lockedByApp || "",
      Boolean(result.intentionalReadOnly)
    );
    recentPathsPreference.add(canonicalPath);

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
      await activateTab(existingTab.id);
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
      await activateTab(activeDashboardTabId, { force: true });
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
  }

  async function activateTab(
    tabId: string,
    options: { force?: boolean } = {}
  ) {
    if (tabId === activeTabId && !options.force) return;

    const nextTab = tabs.find((tab) => tab.id === tabId);
    if (!nextTab) return;

    captureCurrentTabState();

    activeTabId = tabId;

    if (nextTab.type === "dashboard") {
      resetEditorViewState();
      return;
    }

    dbPath = nextTab.path;
    const forcedState = getForcedReadOnlyState(dbPath);
    dbForcedReadOnly = Boolean(forcedState);
    dbIntentionalReadOnly = Boolean(forcedState?.intentionalReadOnly);
    dbReadOnlyReason = forcedState?.readOnlyReason ?? "";
    dbLockedByApp = forcedState?.lockedByApp ?? "";
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

  function reorderTab(tabId: string, targetIndex: number) {
    const sourceIndex = tabs.findIndex((tab) => tab.id === tabId);
    if (sourceIndex === -1) return;

    const clampedTargetIndex = Math.max(
      0,
      Math.min(targetIndex, tabs.length - 1)
    );
    if (clampedTargetIndex === sourceIndex) return;

    const nextTabs = [...tabs];
    const [movedTab] = nextTabs.splice(sourceIndex, 1);
    if (!movedTab) return;
    nextTabs.splice(clampedTargetIndex, 0, movedTab);
    tabs = nextTabs;
  }

  function canCloseTab(tab: WorkspaceTab): boolean {
    return !(tabs.length === 1 && tab.type === "dashboard");
  }

  async function executeCloseTab(tabId: string) {
    const tabIndex = tabs.findIndex((tab) => tab.id === tabId);
    if (tabIndex === -1) return;
    const tab = tabs[tabIndex];
    if (!canCloseTab(tab)) return;

    const closingActiveTab = tabId === activeTabId;

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
      await activateTab(dashboardTab.id, { force: true });
      return;
    }

    if (!closingActiveTab) return;

    const nextIndex = Math.max(0, Math.min(tabIndex, tabs.length - 1));
    await activateTab(tabs[nextIndex].id);
  }

  async function requestCloseTab(
    tabId: string,
    options: { closeWindowAfter?: boolean } = {}
  ) {
    const tab = tabs.find((item) => item.id === tabId);
    if (!tab) return;
    if (!canCloseTab(tab)) return;

    const closeWindowAfter = Boolean(options.closeWindowAfter);
    if (dirtyTabIds.has(tabId)) {
      pendingUnsavedClose = { tabId, closeWindowAfter };
      return;
    }

    await executeCloseTab(tabId);
    if (closeWindowAfter) {
      await WindowService.CloseCurrentWindow();
    }
  }

  function cancelPendingUnsavedClose() {
    if (isSaving) return;
    pendingUnsavedClose = null;
  }

  async function closePendingWithoutSaving() {
    if (!pendingUnsavedClose || isSaving) return;
    const { tabId, closeWindowAfter } = pendingUnsavedClose;
    pendingUnsavedClose = null;
    if (closeWindowAfter) {
      await WindowService.CloseCurrentWindow();
      return;
    }
    await executeCloseTab(tabId);
  }

  async function saveAndClosePending() {
    if (!pendingUnsavedClose || isSaving) return;
    const { tabId, closeWindowAfter } = pendingUnsavedClose;
    if (tabId !== activeTabId) {
      await activateTab(tabId);
    }

    await saveValue({ closeEditorOnSuccess: true });
    if (isDirty) return;

    pendingUnsavedClose = null;
    if (closeWindowAfter) {
      await WindowService.CloseCurrentWindow();
      return;
    }
    await executeCloseTab(tabId);
  }

  async function closeActiveTabOrWindow() {
    const activeTab = getActiveTab();
    if (tabs.length <= 1) {
      if (
        activeTab &&
        activeTab.type === "database" &&
        activeTab.id === activeTabId
      ) {
        if (isDirty) {
          pendingUnsavedClose = { tabId: activeTab.id, closeWindowAfter: false };
          return;
        }
        await requestCloseTab(activeTab.id);
        return;
      }
      if (confirmCloseLastTabPreference.get()) {
        pendingCloseLastTab = true;
        return;
      }
      await WindowService.CloseCurrentWindow();
      return;
    }

    if (!activeTab) {
      await WindowService.CloseCurrentWindow();
      return;
    }

    if (canCloseTab(activeTab)) {
      await requestCloseTab(activeTab.id);
      return;
    }

    await WindowService.CloseCurrentWindow();
  }

  function cancelPendingCloseLastTab() {
    pendingCloseLastTab = false;
  }

  async function closeWindowAfterLastTabConfirm() {
    pendingCloseLastTab = false;
    await WindowService.CloseCurrentWindow();
  }

  async function closeWindowAndDisableLastTabConfirm() {
    confirmCloseLastTabPreference.set(false);
    pendingCloseLastTab = false;
    await WindowService.CloseCurrentWindow();
  }

  async function openDatabaseFromPath(path: string, readOnly = false) {
    if (!path || path.trim() === "" || isOpeningDatabase) return;

    pendingCreatePath = null;
    isOpeningDatabase = true;
    errorMessage = "";

    try {
      const result: OpenDatabaseResult = await LevelDBService.OpenDatabase(
        path,
        readOnly
      );
      if (result.ok) {
        await finishOpeningDatabase(result, path);
      } else {
        isOpeningDatabase = false;
        if (result.databaseMissing) {
          pendingCreatePath = result.canonicalPath || path;
          return;
        }
        await Dialogs.Error({
          Title: "Invalid Database",
          Message:
            result.error ||
            "The selected folder is not a valid LevelDB database.",
        });
        return;
      }
    } catch (err) {
      errorMessage = err instanceof Error ? err.message : String(err);
      isOpeningDatabase = false;
      await Dialogs.Error({
        Title: "Error",
        Message: errorMessage,
      });
    } finally {
      isOpeningDatabase = false;
    }
  }

  async function createDatabaseAtPendingPath() {
    if (!pendingCreatePath || isCreatingDatabase) return;

    isCreatingDatabase = true;
    errorMessage = "";

    try {
      const requestedPath = pendingCreatePath;
      const result: OpenDatabaseResult =
        await LevelDBService.CreateDatabase(requestedPath);
      if (result.ok) {
        await finishOpeningDatabase(result, requestedPath);
        return;
      }

      await Dialogs.Error({
        Title: "Create Database Failed",
        Message:
          result.error ||
          "The database could not be created at the selected location.",
      });
    } catch (err) {
      errorMessage = err instanceof Error ? err.message : String(err);
      await Dialogs.Error({
        Title: "Create Database Failed",
        Message: errorMessage,
      });
    } finally {
      isCreatingDatabase = false;
    }
  }

  async function openDatabaseFromDialog(readOnly?: boolean) {
    try {
      const path = await Dialogs.OpenFile({
        CanChooseDirectories: true,
        CanChooseFiles: false,
        Title: "Select LevelDB Database Folder",
      });

      // OpenFile returns string or string[] depending on options; for single dir it's a string
      const selectedPath = Array.isArray(path) ? path[0] : path;
      if (selectedPath) {
        await openDatabaseFromPath(selectedPath, readOnly ?? false);
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
    isLoadingKeys = true;
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
      isLoadingKeys = false;
    }
  }

  async function loadKeys() {
    await reloadDatabase();
  }

  async function refreshDatabase(source: "manual" | "auto") {
    if (!dbPath || isRefreshing || isLoadingKeys) {
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

    isRefreshing = true;
    const refreshStartedAtMs = Date.now();
    try {
      const refreshResult: OpenDatabaseResult =
        await LevelDBService.RefreshDatabase(dbPath);
      const canonicalPath = refreshResult.canonicalPath || dbPath;
      const forcedReadOnly = Boolean(refreshResult.forcedReadOnly);
      setForcedReadOnlyState(
        canonicalPath,
        forcedReadOnly,
        refreshResult.readOnlyReason || "",
        refreshResult.lockedByApp || "",
        Boolean(refreshResult.intentionalReadOnly)
      );
      if (canonicalPath === dbPath) {
        dbForcedReadOnly = forcedReadOnly;
        dbIntentionalReadOnly = Boolean(refreshResult.intentionalReadOnly);
        dbReadOnlyReason = refreshResult.readOnlyReason || "";
        dbLockedByApp = refreshResult.lockedByApp || "";
      }
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
    if (!selectedKey || valueLoading || isSaving || effectiveReadOnly) return;
    isValueEditing = true;
  }

  async function acceptRemoteConflictValue(
    key: string,
    currentValueExists: boolean,
    currentValue: string,
    closeEditorAfterResolve: boolean
  ) {
    if (currentValueExists) {
      originalValueRaw = currentValue;
      editorValueRaw = currentValue;
      syncEditorDisplayWithRawValue();
      isValueEditing = !closeEditorAfterResolve;
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
    if (selectedKey === key) {
      isValueEditing = !closeEditorAfterResolve;
    }
  }

  async function saveValue({
    closeEditorOnSuccess,
  }: {
    closeEditorOnSuccess: boolean;
  }) {
    if (
      !selectedKey ||
      !isValueEditing ||
      isSaving ||
      effectiveReadOnly ||
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
          isValueEditing = !closeEditorOnSuccess;
          return;
        }

        await acceptRemoteConflictValue(
          keyToSave,
          saveResult.currentValueExists,
          saveResult.currentValue,
          closeEditorOnSuccess
        );
        return;
      }

      if (!saveResult.ok) {
        throw new Error("Save failed.");
      }

      originalValueRaw = localValueRaw;
      syncEditorDisplayWithRawValue();
      isValueEditing = !closeEditorOnSuccess;
    } catch (err) {
      await Dialogs.Error({
        Title: "Save failed",
        Message: err instanceof Error ? err.message : String(err),
      });
    } finally {
      isSaving = false;
    }
  }

  async function saveValueOnly() {
    await saveValue({ closeEditorOnSuccess: false });
  }

  async function saveValueAndClose() {
    await saveValue({ closeEditorOnSuccess: true });
  }

  function revertValueChanges() {
    editorValueRaw = originalValueRaw;
    syncEditorDisplayWithRawValue();
    isValueEditing = false;
  }

  function startRename(key: string) {
    if (isRenaming || effectiveReadOnly) return;
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
      effectiveReadOnly ||
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
    if (!editingKey || isRenaming || effectiveReadOnly || renameValidationError)
      return;
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
    if (isDeleting || effectiveReadOnly) return;
    keyPendingDelete = key;
  }

  async function confirmDeleteKey() {
    if (!keyPendingDelete || isDeleting || effectiveReadOnly) return;
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
    const platformHint =
      `${navigator.platform} ${navigator.userAgent}`.toLowerCase();
    isWindowsRuntime = platformHint.includes("win");

    const mediaQuery = window.matchMedia("(min-width: 768px)");
    const updateLayoutMode = () => {
      isDesktopLayout = mediaQuery.matches;
      if (!isDesktopLayout) return;
      syncKeyPaneWidthToViewport();
    };

    updateLayoutMode();

    const onWindowResize = () => {
      syncKeyPaneWidthToViewport();
    };

    mediaQuery.addEventListener("change", updateLayoutMode);
    window.addEventListener("resize", onWindowResize);

    const unregisterCloseFromMenu = Events.On(
      "app:closeActiveTabOrWindow",
      () => {
        void closeActiveTabOrWindow();
      }
    );
    const unregisterSaveFromMenu = Events.On("app:saveValue", () => {
      if (!canSaveValueChanges) {
        return;
      }
      void saveValueOnly();
    });
    const unregisterToggleSettings = Events.On("app:toggleSettings", () => {
      showSettingsDialog = !showSettingsDialog;
    });

    const unregisterShortcutHandlers = [
      shortcutManager.registerHandler(ShortcutCommand.Escape, () => {
        if (showSettingsDialog) {
          showSettingsDialog = false;
          return;
        }
        if (editingKey) {
          resetRenameForm();
          return;
        }
        if (pendingUnsavedClose && !isSaving) {
          pendingUnsavedClose = null;
          return;
        }
        if (pendingCloseLastTab) {
          pendingCloseLastTab = false;
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
        }
      }),
    ];

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.defaultPrevented) {
        return;
      }

      if (isWindowsRuntime) {
        if (
          event.key === "Alt" &&
          !event.repeat &&
          !event.ctrlKey &&
          !event.metaKey &&
          !event.shiftKey
        ) {
          pendingAltMenuToggle = true;
          event.preventDefault();
          return;
        }

        if (pendingAltMenuToggle && event.key !== "Alt" && event.altKey) {
          pendingAltMenuToggle = false;
        }
      }

      if (shortcutManager.handleKeyDown(event)) {
        event.preventDefault();
      }
    };

    const onKeyUp = (event: KeyboardEvent) => {
      if (!isWindowsRuntime || event.key !== "Alt") {
        return;
      }
      const shouldToggleMenu = pendingAltMenuToggle;
      pendingAltMenuToggle = false;
      if (!shouldToggleMenu) {
        return;
      }
      event.preventDefault();
      void WindowService.ToggleCurrentWindowMenuBar();
    };

    const onWindowBlur = () => {
      pendingAltMenuToggle = false;
    };

    window.addEventListener("keydown", onKeyDown);
    window.addEventListener("keyup", onKeyUp);
    window.addEventListener("blur", onWindowBlur);

    return () => {
      mediaQuery.removeEventListener("change", updateLayoutMode);
      window.removeEventListener("resize", onWindowResize);
      window.removeEventListener("keydown", onKeyDown);
      window.removeEventListener("keyup", onKeyUp);
      window.removeEventListener("blur", onWindowBlur);
      unregisterCloseFromMenu();
      unregisterSaveFromMenu();
      unregisterToggleSettings();
      unregisterShortcutHandlers.forEach((unregister) => unregister());
      stopPaneResize();
    };
  });

  onDestroy(() => {
    stopPaneResize();
    stopAutoRefresh();
    if (keySearchDebounceTimeout) {
      clearTimeout(keySearchDebounceTimeout);
    }
    if (copyValueFeedbackTimeout) {
      clearTimeout(copyValueFeedbackTimeout);
    }
  });
</script>

{#if dbPath}
  <div class="editor-layout flex h-screen flex-col">
    <div
      class="titlebar-drag-region shrink-0 border-b border-border/50 bg-background/55 backdrop-blur-sm"
      role="none"
      on:dblclick={handleTitlebarDoubleClick}
    ></div>
    <TabStrip
      {tabs}
      {activeTabId}
      {canCloseTab}
      {dirtyTabIds}
      onTabChange={handleTabValueChange}
      onCloseTab={(tabId) => requestCloseTab(tabId)}
      onAddDashboardTab={addDashboardTab}
      onReorderTab={reorderTab}
    />
    <header
      class="relative z-40 overflow-visible bg-background/60 px-4 pt-3 pb-2 backdrop-blur-sm"
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
            {#if autoRefreshIntervalMs > 0}
              <Badge variant="outline">Auto {autoRefreshLabel}</Badge>
            {/if}
          </div>
        </div>

        <div
          class="flex items-center justify-between gap-2 md:shrink-0 md:justify-end"
        >
          <SplitButton
            variant="outline"
            size="sm"
            disabled={isRefreshing}
            triggerLabel="Auto-refresh options"
            menuClass="min-w-40"
            primaryClass="gap-1.5 border-r-0 pr-2"
            triggerClass="w-8 px-0"
            triggerIconClass="h-3.5 w-3.5"
            on:primary={triggerManualRefresh}
          >
            <span
              class="inline-flex items-center gap-1.5"
              title={getRefreshButtonTitle()}
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
            </span>

            <svelte:fragment slot="menu">
              {#each AUTO_REFRESH_OPTIONS as option}
                <DropdownMenuItem
                  radio={true}
                  checked={option.intervalMs === autoRefreshIntervalMs}
                  disabled={isRefreshing}
                  on:click={() => setAutoRefreshInterval(option.intervalMs)}
                >
                  <span>{option.label}</span>
                </DropdownMenuItem>
              {/each}
            </svelte:fragment>
          </SplitButton>

          <ThreeDotMenu
            class="h-8 w-8"
            triggerLabel="View settings"
            triggerTitle="View settings"
            menuClass="min-w-48"
          >
            <DropdownMenuItem
              slot="menu"
              class={`flex items-center justify-between ${
                prettyPrintJson ? "bg-muted/60" : ""
              }`}
              on:click={(event) => {
                event.preventDefault();
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
            </DropdownMenuItem>
          </ThreeDotMenu>
        </div>
      </div>
    </header>

    <div
      class="relative grid min-h-0 flex-1 gap-4 px-4 pb-4 pt-2"
      aria-busy={showEditorLoadingOverlay}
      style={dbForcedReadOnly
        ? "grid-template-rows: auto minmax(0, 1fr);"
        : "grid-template-rows: minmax(0, 1fr);"}
    >
      {#if dbForcedReadOnly}
        <div
          class="rounded-md border border-warning/30 bg-warning/10 px-3 py-2 text-sm text-warning-foreground dark:border-warning/20 dark:bg-warning/5 dark:text-warning/80"
          role="status"
          aria-live="polite"
        >
          <p class="font-medium">Opened in read-only mode.</p>
          <p class="mt-0.5 text-xs opacity-90">
            {dbIntentionalReadOnly
              ? dbReadOnlyReason ||
                "This database was opened in read-only mode without taking a lock, so editing is disabled."
              : dbReadOnlyReason ||
                "This database is currently in use by another application, so editing is disabled."}
          </p>
          {#if !dbIntentionalReadOnly && showLockedByAppInReadOnlyNotice}
            <p class="mt-0.5 text-xs opacity-90">In use by: {dbLockedByApp}</p>
          {/if}
        </div>
      {/if}

      <div
        bind:this={editorSplitContainer}
        class="relative grid min-h-0 min-w-0 flex-1 gap-4 md:grid-cols-1"
        style={isDesktopLayout
          ? `grid-template-columns: ${keyPaneWidth}px minmax(${MIN_VALUE_PANE_WIDTH}px, 1fr);`
          : undefined}
      >
        {#if isDesktopLayout}
          <button
            type="button"
            class="group absolute bottom-4 top-4 z-10 w-2 -translate-x-1/2 cursor-col-resize rounded-full bg-transparent"
            style={`left: ${keyPaneWidth + GRID_GAP_PX / 2}px;`}
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
                  disabled={effectiveReadOnly}
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
            {#if showCreateForm}
              <div
                class="space-y-2 rounded-md border border-border/70 bg-muted/20 p-2"
              >
                <input
                  class="w-full rounded-md border border-input bg-background px-2 py-1.5 text-sm"
                  placeholder="New key (text or 0x...)"
                  bind:value={newKeyInput}
                  disabled={isCreating || effectiveReadOnly}
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
                  disabled={isCreating || effectiveReadOnly}
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
                      effectiveReadOnly ||
                      !!createKeyValidationError ||
                      !!createValueValidationError}
                    on:click={createKey}
                  >
                    {#if isCreating}
                      <RefreshCcw class="h-3.5 w-3.5 animate-spin" />
                    {:else}
                      <Save class="h-3.5 w-3.5" />
                    {/if}
                    Create key
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
                          class={`relative overflow-hidden flex items-center rounded-md px-2 py-1.5 text-sm transition-colors ${
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
                                disabled={isRenaming || effectiveReadOnly}
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
                            <div
                              class="ml-1 flex shrink-0 items-center gap-0.5"
                            >
                              <Button
                                variant="ghost"
                                size="icon"
                                class="h-7 w-7"
                                disabled={isRenaming ||
                                  effectiveReadOnly ||
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
                                disabled={isRenaming || effectiveReadOnly}
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
                              class="min-w-0 flex-1 text-left transition-[padding] duration-150 group-hover:pr-15 group-focus-within:pr-15"
                              title={key}
                              on:click={() => selectKey(key)}
                            >
                              <span class="block w-full truncate">{key}</span>
                            </button>
                            <div
                              class="pointer-events-none absolute inset-y-0 right-0 flex items-center gap-0.5 pl-8 pr-2 opacity-0 transition-opacity group-hover:pointer-events-auto group-hover:opacity-100 group-focus-within:pointer-events-auto group-focus-within:opacity-100"
                              style={`background: linear-gradient(to right, transparent, ${
                                selectedKey === key
                                  ? "color-mix(in srgb, hsl(var(--primary)) 15%, hsl(var(--card)))"
                                  : "hsl(var(--muted))"
                              } 1.75rem);`}
                            >
                              <Button
                                variant="ghost"
                                size="icon"
                                class="h-7 w-7"
                                disabled={effectiveReadOnly}
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
                                disabled={isDeleting || effectiveReadOnly}
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
                  <SplitButton
                    size="sm"
                    triggerLabel="Save options"
                    disabled={!canSaveValueChanges}
                    primaryClass="gap-1.5 border-r-0 pr-2"
                    triggerClass="w-8 px-0"
                    triggerIconClass="h-3.5 w-3.5"
                    on:primary={() => {
                      void saveValueOnly();
                    }}
                  >
                    <span class="inline-flex items-center gap-1.5">
                      {#if isSaving}
                        <RefreshCcw class="h-3.5 w-3.5 animate-spin" />
                      {:else}
                        <Save class="h-3.5 w-3.5" />
                      {/if}
                      Save
                    </span>
                    <DropdownMenuItem
                      slot="menu"
                      disabled={!canSaveValueChanges}
                      on:click={() => {
                        void saveValueAndClose();
                      }}
                    >
                      Save & Close
                    </DropdownMenuItem>
                  </SplitButton>
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
                      effectiveReadOnly}
                    on:click={startValueEdit}
                  >
                    <Pencil class="h-3.5 w-3.5" />
                    Edit
                  </Button>
                {/if}
                <Button
                  variant="outline"
                  size="icon"
                  disabled={!selectedKey || valueLoading || isSaving}
                  on:click={copyValueToClipboard}
                  title={hasCopiedValue ? "Copied" : "Copy value"}
                  aria-label={hasCopiedValue ? "Copied" : "Copy value"}
                >
                  {#if hasCopiedValue}
                    <Check class="h-3.5 w-3.5" />
                  {:else}
                    <Copy class="h-3.5 w-3.5" />
                  {/if}
                </Button>
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
                <div class="px-1 py-3 text-sm text-muted-foreground">
                  Select a key
                </div>
              {:else if showValueLoadingOverlay}
                <div
                  class="flex h-full min-h-0 flex-1 flex-col rounded-md border border-border/70 bg-muted/20 p-3"
                  role="status"
                  aria-live="polite"
                >
                  <span class="sr-only">Loading value…</span>
                  <div
                    class="mb-3 h-3 w-40 rounded bg-muted/80 motion-safe:animate-pulse motion-reduce:animate-none"
                  ></div>
                  <div class="space-y-2.5" aria-hidden="true">
                    <div
                      class="h-3 w-full rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"
                    ></div>
                    <div
                      class="h-3 w-[95%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"
                    ></div>
                    <div
                      class="h-3 w-[88%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"
                    ></div>
                    <div
                      class="h-3 w-[92%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"
                    ></div>
                    <div
                      class="h-3 w-[83%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"
                    ></div>
                    <div
                      class="h-3 w-[90%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"
                    ></div>
                    <div
                      class="h-3 w-[72%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"
                    ></div>
                    <div
                      class="h-3 w-[86%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"
                    ></div>
                    <div
                      class="h-3 w-[65%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"
                    ></div>
                  </div>
                </div>
              {:else}
                <ValueCodeMirror
                  class={`h-full min-h-0 flex-1 cursor-text select-text rounded-md border border-border/70 transition-colors ${
                    isValueEditing
                      ? "bg-background text-foreground focus-within:border-primary"
                      : "bg-muted/50 text-foreground"
                  }`}
                  value={editorValue}
                  on:change={handleValueEditorChange}
                  readOnly={!isValueEditing || isSaving || effectiveReadOnly}
                />
                {#if valueValidationError}
                  <p class="px-1 text-xs text-destructive">
                    {valueValidationError}
                  </p>
                {/if}
              {/if}
            </div>
          </CardContent>
        </Card>
      </div>

      {#if showEditorLoadingOverlay}
        <div
          class="absolute inset-0 z-20 flex items-center justify-center rounded-lg bg-muted/35 backdrop-blur-md supports-[backdrop-filter]:bg-muted/25"
        >
          <div
            role="status"
            aria-live="polite"
            class="flex items-center gap-2 rounded-full border border-border/80 bg-card/92 px-4 py-2 text-sm font-medium text-foreground shadow-lg ring-1 ring-border/40"
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
      class="titlebar-drag-region shrink-0 border-b border-border/50 bg-background/55 backdrop-blur-sm"
      role="none"
      on:dblclick={handleTitlebarDoubleClick}
    ></div>
    <TabStrip
      {tabs}
      {activeTabId}
      {canCloseTab}
      {dirtyTabIds}
      onTabChange={handleTabValueChange}
      onCloseTab={(tabId) => requestCloseTab(tabId)}
      onAddDashboardTab={addDashboardTab}
      onReorderTab={reorderTab}
    />
    <div class="flex min-h-0 flex-1 items-center justify-center p-6">
      {#if pendingCreatePath}
        <div
          class="flex w-full max-w-2xl flex-col items-center gap-6 px-8 py-10 text-center"
        >
          <div
            class="flex h-20 w-20 items-center justify-center rounded-full border border-border/70 bg-muted/60 text-primary shadow-sm"
          >
            <DatabaseZap class="h-10 w-10" />
          </div>

          <div class="w-full space-y-3">
            <h1 class="text-2xl font-semibold tracking-tight">
              No LevelDB database found
            </h1>
            <p class="text-sm leading-6 text-muted-foreground">
              The selected folder does not contain a LevelDB database yet.
              Create a new database at this location to continue.
            </p>
            <div
              class="w-full max-w-full rounded-lg border border-border/70 bg-muted/35 px-4 py-3"
            >
              <p
                class="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground"
              >
                Selected folder
              </p>
              <p
                class="mt-2 max-w-full break-words font-mono text-sm text-foreground [overflow-wrap:anywhere]"
              >
                {pendingCreatePath}
              </p>
            </div>
          </div>

          <div class="flex flex-wrap items-center justify-center gap-3">
            <Button
              variant="outline"
              disabled={isCreatingDatabase}
              on:click={() => {
                pendingCreatePath = null;
              }}
            >
              Cancel
            </Button>
            <Button
              disabled={isCreatingDatabase}
              on:click={createDatabaseAtPendingPath}
            >
              <Plus class="mr-2 h-4 w-4" />
              {isCreatingDatabase ? "Creating…" : "Create database here"}
            </Button>
          </div>
        </div>
      {:else}
        <Card class="w-full max-w-2xl">
          <CardHeader class="space-y-4">
            <div class="flex items-center justify-between">
              <CardTitle class="flex items-center gap-2 text-2xl">
                <Database class="h-6 w-6 text-primary" />
                LevelDB Editor
              </CardTitle>
            </div>
            <CardDescription>
              Open a LevelDB folder to browse keys and inspect values instantly.
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-6">
            <SplitButton
              class="w-fit"
              disabled={isOpeningDatabase}
              triggerLabel="More open options"
              on:primary={() => {
                void openDatabaseFromDialog();
              }}
            >
              <span class="inline-flex items-center gap-2">
                <FolderOpen class="h-4 w-4" />
                {isOpeningDatabase ? "Opening…" : "Open LevelDB database"}
              </span>

              <DropdownMenuItem
                slot="menu"
                disabled={isOpeningDatabase}
                on:click={() => {
                  void openDatabaseFromDialog(true);
                }}
              >
                Open in read-only mode
              </DropdownMenuItem>
            </SplitButton>

            {#if errorMessage}
              <p class="text-sm text-destructive">{errorMessage}</p>
            {/if}

            {#if $recentPathsPreference.length > 0}
              <section class="space-y-3">
                <h2
                  class="flex items-center gap-2 text-sm font-medium text-muted-foreground"
                >
                  <History class="h-4 w-4" />
                  Recently opened
                </h2>
                <div class="space-y-2">
                  {#each $recentPathsPreference as item}
                    <div class="group flex items-center gap-1">
                      <Button
                        variant="ghost"
                        class="min-w-0 flex-1 justify-start font-normal"
                        on:click={() => openDatabaseFromPath(item.path, false)}
                        title={item.path}
                        disabled={isOpeningDatabase}
                      >
                        <span class="block truncate">{item.label}</span>
                      </Button>

                      <ThreeDotMenu
                        variant="ghost"
                        class="h-7 w-7 text-muted-foreground opacity-0 transition-opacity hover:text-foreground group-hover:opacity-100 group-focus-within:opacity-100"
                        triggerLabel={`More options for ${item.label}`}
                        triggerTitle="More options"
                        menuClass="min-w-56"
                        disabled={isOpeningDatabase}
                      >
                        <DropdownMenuItem
                          slot="menu"
                          disabled={isOpeningDatabase}
                          on:click={() => {
                            void openDatabaseFromPath(item.path, true);
                          }}
                        >
                          Open in read-only mode
                        </DropdownMenuItem>
                      </ThreeDotMenu>

                      <Button
                        variant="ghost"
                        size="icon"
                        class="h-7 w-7 text-muted-foreground opacity-0 transition-opacity hover:text-foreground group-hover:opacity-100 group-focus-within:opacity-100"
                        aria-label={`Remove ${item.label} from recently opened`}
                        title="Remove from recently opened"
                        on:click={() => {
                          recentPathsPreference.remove(item.path);
                        }}
                        disabled={isOpeningDatabase}
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
      {/if}
    </div>
  </div>
{/if}

{#if keyPendingDelete}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-background/55 p-4 backdrop-blur-sm"
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
          Delete <span class="break-all font-semibold">{keyPendingDelete}</span
          >? This action cannot be undone.
        </CardDescription>
      </CardHeader>
      <CardContent class="flex justify-end gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={isDeleting || effectiveReadOnly}
          on:click={() => {
            keyPendingDelete = null;
          }}
        >
          Cancel
        </Button>
        <Button
          variant="destructive"
          size="sm"
          disabled={isDeleting || effectiveReadOnly}
          on:click={confirmDeleteKey}
        >
          {#if isDeleting}
            <RefreshCcw class="mr-1.5 h-3.5 w-3.5 animate-spin" />
          {:else}
            <Trash2 class="mr-1.5 h-3.5 w-3.5" />
          {/if}
          Delete
        </Button>
      </CardContent>
    </Card>
  </div>
{/if}

{#if pendingUnsavedClose}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-background/55 p-4 backdrop-blur-sm"
    role="presentation"
    on:click={cancelPendingUnsavedClose}
  >
    <Card
      class="w-full max-w-md border-border"
      role="dialog"
      aria-modal="true"
      aria-labelledby="close-unsaved-title"
      aria-describedby="close-unsaved-description"
      on:click={(event) => {
        event.stopPropagation();
      }}
    >
      <CardHeader class="space-y-2">
        <CardTitle id="close-unsaved-title">Unsaved changes</CardTitle>
        <CardDescription id="close-unsaved-description">
          You have unsaved value edits. Save before closing this database tab?
        </CardDescription>
      </CardHeader>
      <CardContent class="flex justify-end gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={isSaving}
          on:click={cancelPendingUnsavedClose}
        >
          Cancel
        </Button>
        <Button
          variant="secondary"
          size="sm"
          disabled={isSaving}
          on:click={closePendingWithoutSaving}
        >
          Close without Saving
        </Button>
        <Button
          size="sm"
          disabled={!canSaveAndClosePendingTab}
          on:click={saveAndClosePending}
        >
          {#if isSaving}
            <RefreshCcw class="mr-1.5 h-3.5 w-3.5 animate-spin" />
          {:else}
            <Save class="mr-1.5 h-3.5 w-3.5" />
          {/if}
          Save &amp; Close
        </Button>
      </CardContent>
    </Card>
  </div>
{/if}

{#if pendingCloseLastTab}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-background/55 p-4 backdrop-blur-sm"
    role="presentation"
    on:click={cancelPendingCloseLastTab}
  >
    <Card
      class="w-full max-w-md border-border"
      role="dialog"
      aria-modal="true"
      aria-labelledby="close-last-tab-title"
      aria-describedby="close-last-tab-description"
      on:click={(event) => {
        event.stopPropagation();
      }}
    >
      <CardHeader class="space-y-2">
        <CardTitle id="close-last-tab-title">Close window</CardTitle>
        <CardDescription id="close-last-tab-description">
          Are you sure you want to close this window?
        </CardDescription>
      </CardHeader>
      <CardContent class="flex justify-end gap-2">
        <Button
          variant="outline"
          size="sm"
          on:click={cancelPendingCloseLastTab}
        >
          Cancel
        </Button>
        <SplitButton
          size="sm"
          triggerLabel="Close options"
          primaryClass="pr-2"
          triggerClass="w-8 px-0"
          triggerIconClass="h-3.5 w-3.5"
          on:primary={() => {
            void closeWindowAfterLastTabConfirm();
          }}
        >
          Close
          <DropdownMenuItem
            slot="menu"
            on:click={() => {
              void closeWindowAndDisableLastTabConfirm();
            }}
          >
            Close and don&apos;t ask again
          </DropdownMenuItem>
        </SplitButton>
      </CardContent>
    </Card>
  </div>
{/if}

<SettingsDialog
  open={showSettingsDialog}
  on:close={() => (showSettingsDialog = false)}
/>
