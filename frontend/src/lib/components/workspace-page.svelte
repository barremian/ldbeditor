<script lang="ts">
  import { onDestroy, onMount, tick } from "svelte";
  import { Dialogs, Events, Window } from "@wailsio/runtime";
  import * as LevelDBService from "../../../bindings/ldbeditor/leveldbservice";
  import { OpenDatabaseResult } from "../../../bindings/ldbeditor/models";
  import * as WindowService from "../../../bindings/ldbeditor/windowservice";
  import DashboardView from "$lib/components/dashboard-view.svelte";
  import CloseLastTabDialog from "$lib/components/dialogs/close-last-tab-dialog.svelte";
  import DeleteKeyDialog from "$lib/components/dialogs/delete-key-dialog.svelte";
  import UnsavedChangesDialog from "$lib/components/dialogs/unsaved-changes-dialog.svelte";
  import EditorHeader from "$lib/components/editor-header.svelte";
  import KeyListPane from "$lib/components/key-list-pane.svelte";
  import SettingsDialog from "$lib/components/settings-dialog.svelte";
  import TabStrip from "$lib/components/tab-strip.svelte";
  import ValuePane from "$lib/components/value-pane.svelte";
  import {
    confirmCloseLastTabPreference,
    keyPaneWidthPreference,
    recentPathsPreference,
    valueFormatPreference,
  } from "$lib/preferences";
  import { ShortcutCommand } from "$lib/shortcuts/commands";
  import { shortcutConfig } from "$lib/shortcuts/config";
  import { createShortcutManager } from "$lib/shortcuts/manager";
  import {
    createAutoRefreshController,
    getAutoRefreshLabel,
    getAutoRefreshProgress,
    getRemainingAutoRefreshMs,
    type AutoRefreshState,
  } from "$lib/services/auto-refresh";
  import {
    getTabLabel,
    type PendingUnsavedClose,
    type TabViewState,
    type WorkspaceTab,
  } from "$lib/types/workspace";
  import { formatValueForDisplay } from "$lib/utils/format";
  import { getHexValidationError } from "$lib/utils/validation";
  import { RefreshCcw } from "lucide-svelte";

  const KEY_SEARCH_DEBOUNCE_MS = 300;
  const DEFAULT_KEY_PANE_WIDTH = 320;
  const MIN_KEY_PANE_WIDTH = 300;
  const MIN_VALUE_PANE_WIDTH = 320;
  const GRID_GAP_PX = 16;
  const MIN_REFRESH_FEEDBACK_MS = 150;
  const shortcutManager = createShortcutManager(shortcutConfig);
  const createEmptyAutoRefreshState = (): AutoRefreshState => ({
    intervalMs: 0,
    nextRefreshAt: null,
    countdownNow: Date.now(),
  });

  type AutoRefreshController = ReturnType<typeof createAutoRefreshController>;

  let autoRefreshState: AutoRefreshState = createEmptyAutoRefreshState();
  let activeAutoRefreshUnsubscribe: (() => void) | null = null;
  let tabAutoRefreshMap: Record<string, AutoRefreshController> = {};
  let tabAutoRefreshStateMap: Record<string, AutoRefreshState> = {};
  let tabAutoRefreshCleanupMap: Record<string, () => void> = {};

  let isOpeningDatabase = false;
  let isCreatingDatabase = false;
  let isLoadingKeys = false;
  let errorMessage = "";
  let pendingCreatePath: string | null = null;
  let nextTabId = 2;
  let tabs: WorkspaceTab[] = [
    { id: "tab-1", type: "dashboard", title: "Dashboard" },
  ];
  let activeTabId = tabs[0].id;

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
  let isRefreshing = false;
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
  let prettyPrintJson = false;

  let tabStateMap: Record<string, TabViewState> = {};
  let keyListViewportEl: HTMLDivElement | null = null;
  let pendingKeyListScrollTop: number | null | undefined = undefined;
  let showEditorLoadingOverlay = false;
  let showValueLoadingOverlay = false;
  let editorLoadingOverlayText = "Loading keys...";
  let autoRefreshIntervalMs = 0;
  let autoRefreshLabel = "Off";
  let remainingAutoRefreshMs = 0;
  let autoRefreshProgress = 1;
  let autoRefreshProgressByTab: Record<string, number> = {};
  let canSaveValueChanges = false;
  let canSaveAndClosePendingTab = false;

  $: prettyPrintJson = dbPath
    ? Boolean($valueFormatPreference[dbPath]?.prettyPrintJson)
    : false;
  $: showEditorLoadingOverlay = isLoadingKeys || isRefreshing;
  $: showValueLoadingOverlay = valueLoading;
  $: editorLoadingOverlayText = isRefreshing
    ? "Refreshing database..."
    : "Loading keys...";
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
  $: effectiveReadOnly = dbForcedReadOnly;
  $: showLockedByAppInReadOnlyNotice =
    Boolean(dbLockedByApp.trim()) &&
    !dbReadOnlyReason.toLowerCase().includes(dbLockedByApp.toLowerCase());
  $: autoRefreshIntervalMs = autoRefreshState.intervalMs;
  $: remainingAutoRefreshMs = getRemainingAutoRefreshMs(autoRefreshState);
  $: autoRefreshProgress = getAutoRefreshProgress(autoRefreshState);
  $: autoRefreshLabel = getAutoRefreshLabel(autoRefreshIntervalMs);
  $: autoRefreshProgressByTab = (() => {
    const nextProgressByTab: Record<string, number> = {};

    tabs.forEach((tab) => {
      if (tab.type !== "database") return;

      const state =
        tabAutoRefreshStateMap[tab.id] ??
        (tabStateMap[tab.id]
          ? {
              intervalMs: tabStateMap[tab.id].autoRefreshIntervalMs,
              nextRefreshAt: null,
              countdownNow: Date.now(),
            }
          : null);

      if (!state || state.intervalMs <= 0) return;
      nextProgressByTab[tab.id] = getAutoRefreshProgress(state);
    });

    return nextProgressByTab;
  })();
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

  function syncEditorDisplayWithRawValue() {
    editorValue = formatValueForDisplay(editorValueRaw, prettyPrintJson);
  }

  function getForcedReadOnlyState(path: string): {
    readOnlyReason: string;
    lockedByApp: string;
    intentionalReadOnly: boolean;
  } | null {
    return forcedReadOnlyByDatabase[path] ?? null;
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
      const { [path]: _removed, ...rest } = forcedReadOnlyByDatabase;
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
    if (!dbPath || isValueEditing || isSaving) return;
    const next = !valueFormatPreference.isPrettyPrintEnabled(dbPath);
    valueFormatPreference.setPrettyPrintForPath(dbPath, next);
    syncEditorDisplayWithRawValue();
  }

  function handleValueEditorChange(event: CustomEvent<{ value: string }>) {
    const nextValue = event.detail?.value ?? "";
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

  function setAutoRefreshInterval(intervalMs: number) {
    const activeTab = getActiveTab();
    if (!activeTab || activeTab.type !== "database") return;

    const controller = getTabAutoRefreshController(activeTab.id);
    if (!controller) return;

    controller.setIntervalMs(intervalMs, activeTab.path);

    if (tabStateMap[activeTab.id]) {
      tabStateMap = {
        ...tabStateMap,
        [activeTab.id]: {
          ...tabStateMap[activeTab.id],
          autoRefreshIntervalMs: intervalMs,
        },
      };
    }
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

  function getActiveTab() {
    return tabs.find((tab) => tab.id === activeTabId) ?? null;
  }

  function findDatabaseTabByPath(path: string) {
    return (
      tabs.find((tab) => tab.type === "database" && tab.path === path) ?? null
    );
  }

  function getDatabaseTabById(tabId: string) {
    const tab = tabs.find((candidate) => candidate.id === tabId);
    return tab?.type === "database" ? tab : null;
  }

  function getTabAutoRefreshController(tabId: string) {
    const existing = tabAutoRefreshMap[tabId];
    if (existing) return existing;

    const tab = getDatabaseTabById(tabId);
    if (!tab) return null;

    const controller = createAutoRefreshController({
      onTick: () => void runAutoRefreshTick(tabId),
    });
    const cleanup = controller.subscribe((value) => {
      tabAutoRefreshStateMap = {
        ...tabAutoRefreshStateMap,
        [tabId]: value,
      };
    });
    const savedIntervalMs = tabStateMap[tabId]?.autoRefreshIntervalMs ?? 0;
    if (savedIntervalMs > 0) {
      controller.setIntervalMs(savedIntervalMs, tab.path);
    }

    tabAutoRefreshMap = {
      ...tabAutoRefreshMap,
      [tabId]: controller,
    };
    tabAutoRefreshCleanupMap = {
      ...tabAutoRefreshCleanupMap,
      [tabId]: cleanup,
    };
    return controller;
  }

  function subscribeToActiveAutoRefresh(tabId: string | null) {
    if (activeAutoRefreshUnsubscribe) {
      activeAutoRefreshUnsubscribe();
      activeAutoRefreshUnsubscribe = null;
    }

    if (!tabId) {
      autoRefreshState = createEmptyAutoRefreshState();
      return;
    }

    const controller = getTabAutoRefreshController(tabId);
    if (!controller) {
      autoRefreshState = createEmptyAutoRefreshState();
      return;
    }

    activeAutoRefreshUnsubscribe = controller.subscribe((value) => {
      autoRefreshState = value;
    });
  }

  function getActiveAutoRefreshController() {
    const activeTab = getActiveTab();
    if (!activeTab || activeTab.type !== "database") {
      return null;
    }
    return getTabAutoRefreshController(activeTab.id);
  }

  function destroyTabAutoRefreshController(tabId: string) {
    const controller = tabAutoRefreshMap[tabId];
    if (!controller) return;
    tabAutoRefreshCleanupMap[tabId]?.();

    if (tabId === activeTabId) {
      subscribeToActiveAutoRefresh(null);
    }

    controller.stop();
    controller.destroy();

    const { [tabId]: _removed, ...rest } = tabAutoRefreshMap;
    tabAutoRefreshMap = rest;
    const { [tabId]: _removedCleanup, ...remainingCleanup } =
      tabAutoRefreshCleanupMap;
    tabAutoRefreshCleanupMap = remainingCleanup;
    const { [tabId]: _removedState, ...remainingState } = tabAutoRefreshStateMap;
    tabAutoRefreshStateMap = remainingState;
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

    tabStateMap = {
      ...tabStateMap,
      [activeTab.id]: {
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
      autoRefreshIntervalMs:
        getTabAutoRefreshController(activeTab.id)?.getState().intervalMs ?? 0,
      },
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
    pendingCreatePath = null;
    dbPath = "";
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
      subscribeToActiveAutoRefresh(null);
      resetEditorViewState();
      return;
    }

    const autoRefreshController = getTabAutoRefreshController(tabId);
    subscribeToActiveAutoRefresh(tabId);

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
      if (
        autoRefreshController &&
        autoRefreshController.getState().intervalMs > 0 &&
        !autoRefreshController.getState().nextRefreshAt
      ) {
        autoRefreshController.schedule(dbPath);
      }
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
      destroyTabAutoRefreshController(tabId);
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
    await executeCloseTab(tabId);
    if (closeWindowAfter) {
      await WindowService.CloseCurrentWindow();
    }
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
    await executeCloseTab(tabId);
    if (closeWindowAfter) {
      await WindowService.CloseCurrentWindow();
    }
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
      } else if (result.databaseMissing) {
        pendingCreatePath = result.canonicalPath || path;
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
    const autoRefreshController =
      source === "auto" ? getActiveAutoRefreshController() : null;

    if (!dbPath || isRefreshing || isLoadingKeys) {
      if (source === "auto" && dbPath && autoRefreshIntervalMs > 0) {
        autoRefreshController?.schedule(dbPath);
      }
      return;
    }

    if (isDirty && !(await confirmDiscardUnsavedChanges())) {
      if (source === "auto" && autoRefreshIntervalMs > 0) {
        autoRefreshController?.schedule(dbPath);
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
        autoRefreshController?.schedule(dbPath);
      }
    }
  }

  function createEmptyTabViewState(autoRefreshIntervalMs: number): TabViewState {
    return {
      keys: [],
      selectedKey: null,
      keySearchInput: "",
      debouncedKeySearch: "",
      keyListScrollTop: null,
      originalValueRaw: "",
      editorValueRaw: "",
      isValueEditing: false,
      autoRefreshIntervalMs,
    };
  }

  async function refreshBackgroundTab(tabId: string) {
    const tab = getDatabaseTabById(tabId);
    const autoRefreshController = tabAutoRefreshMap[tabId];
    if (!tab || !autoRefreshController) return;

    const intervalMs = autoRefreshController.getState().intervalMs;
    if (intervalMs <= 0) return;

    const currentState =
      tabStateMap[tabId] ?? createEmptyTabViewState(intervalMs);

    try {
      const refreshResult: OpenDatabaseResult =
        await LevelDBService.RefreshDatabase(tab.path);
      const canonicalPath = refreshResult.canonicalPath || tab.path;
      setForcedReadOnlyState(
        canonicalPath,
        Boolean(refreshResult.forcedReadOnly),
        refreshResult.readOnlyReason || "",
        refreshResult.lockedByApp || "",
        Boolean(refreshResult.intentionalReadOnly)
      );

      if (!getDatabaseTabById(tabId) || tabId === activeTabId) {
        return;
      }

      const nextKeys = (await LevelDBService.GetKeys(tab.path)) ?? [];
      const nextSelectedKey =
        currentState.selectedKey && nextKeys.includes(currentState.selectedKey)
          ? currentState.selectedKey
          : null;

      let nextOriginalValueRaw = currentState.originalValueRaw;
      let nextEditorValueRaw = currentState.editorValueRaw;
      let nextIsValueEditing =
        currentState.isValueEditing && nextSelectedKey !== null;

      if (!nextSelectedKey) {
        nextOriginalValueRaw = "";
        nextEditorValueRaw = "";
        nextIsValueEditing = false;
      } else if (!currentState.isValueEditing) {
        try {
          const nextValue = await LevelDBService.GetValue(
            tab.path,
            nextSelectedKey
          );
          nextOriginalValueRaw = nextValue ?? "";
          nextEditorValueRaw = nextOriginalValueRaw;
        } catch (err) {
          const message =
            err instanceof Error ? err.message : String(err);
          nextOriginalValueRaw = `Error: ${message}`;
          nextEditorValueRaw = nextOriginalValueRaw;
        }
      }

      if (!getDatabaseTabById(tabId) || tabId === activeTabId) {
        return;
      }

      tabStateMap = {
        ...tabStateMap,
        [tabId]: {
          ...currentState,
          keys: nextKeys,
          selectedKey: nextSelectedKey,
          originalValueRaw: nextOriginalValueRaw,
          editorValueRaw: nextEditorValueRaw,
          isValueEditing: nextIsValueEditing,
          autoRefreshIntervalMs: intervalMs,
        },
      };
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      await Dialogs.Error({
        Title: "Auto-refresh failed",
        Message: `Could not refresh the background database tab. ${message}`.trim(),
      });
    } finally {
      const nextTab = getDatabaseTabById(tabId);
      const nextController = tabAutoRefreshMap[tabId];
      if (
        nextTab &&
        nextController &&
        nextController.getState().intervalMs > 0
      ) {
        nextController.schedule(nextTab.path);
      }
    }
  }

  async function runAutoRefreshTick(tabId: string) {
    if (tabId === activeTabId) {
      if (!dbPath || autoRefreshIntervalMs <= 0) return;
      await refreshDatabase("auto");
      return;
    }

    await refreshBackgroundTab(tabId);
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
      editorValue = formatValueForDisplay(editorValueRaw, prettyPrintJson);
    } catch (err) {
      if (requestId !== valueRequestId || selectedKey !== key) return;
      const message = `Error: ${err instanceof Error ? err.message : String(err)}`;
      originalValueRaw = message;
      editorValueRaw = message;
      editorValue = formatValueForDisplay(editorValueRaw, prettyPrintJson);
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
    subscribeToActiveAutoRefresh(null);
    Object.values(tabAutoRefreshCleanupMap).forEach((cleanup) => cleanup());
    Object.values(tabAutoRefreshMap).forEach((controller) => {
      controller.stop();
      controller.destroy();
    });
    tabAutoRefreshMap = {};
    tabAutoRefreshCleanupMap = {};
    tabAutoRefreshStateMap = {};

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
      {autoRefreshProgressByTab}
      onTabChange={handleTabValueChange}
      onCloseTab={(tabId) => requestCloseTab(tabId)}
      onAddDashboardTab={addDashboardTab}
      onReorderTab={reorderTab}
    />
    <EditorHeader
      {dbPath}
      {isRefreshing}
      {autoRefreshIntervalMs}
      {autoRefreshProgress}
      {prettyPrintJson}
      canTogglePrettyPrint={!isValueEditing && !isSaving}
      refreshButtonTitle={getRefreshButtonTitle()}
      on:refresh={triggerManualRefresh}
      on:setAutoRefresh={(event) =>
        setAutoRefreshInterval(event.detail.intervalMs)}
      on:togglePrettyPrint={togglePrettyPrintJson}
    />

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

        <div
          class="flex min-h-0 min-w-0"
          style={isDesktopLayout
            ? `min-width: ${MIN_KEY_PANE_WIDTH}px;`
            : undefined}
        >
          <KeyListPane
            {keys}
            {filteredKeys}
            {selectedKey}
            {effectiveReadOnly}
            {isCreating}
            {isRenaming}
            {isDeleting}
            {createKeyValidationError}
            {createValueValidationError}
            {renameValidationError}
            bind:showCreateForm
            bind:newKeyInput
            bind:newValueInput
            bind:renameInput
            bind:keySearchInput
            bind:viewportEl={keyListViewportEl}
            {editingKey}
            on:createKey={createKey}
            on:selectKey={(event) => selectKey(event.detail.key)}
            on:startRename={(event) => startRename(event.detail.key)}
            on:confirmRename={renameEditingKey}
            on:requestDelete={(event) => requestDeleteKey(event.detail.key)}
            on:clearSearch={clearKeySearch}
            on:cancelCreate={resetCreateForm}
            on:resetRename={resetRenameForm}
            on:scroll={handleKeyListScroll}
          />
        </div>

        <div
          class="flex min-h-0 min-w-0"
          style={isDesktopLayout
            ? `min-width: ${MIN_VALUE_PANE_WIDTH}px;`
            : undefined}
        >
          <ValuePane
            {selectedKey}
            {editorValue}
            {isDirty}
            {isValueEditing}
            {effectiveReadOnly}
            valueLoading={showValueLoadingOverlay}
            {canSaveValueChanges}
            {isSaving}
            {hasCopiedValue}
            {valueValidationError}
            on:edit={startValueEdit}
            on:save={() => void saveValueOnly()}
            on:saveAndClose={() => void saveValueAndClose()}
            on:revert={revertValueChanges}
            on:copy={() => void copyValueToClipboard()}
            on:valueChange={handleValueEditorChange}
          />
        </div>
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
      {autoRefreshProgressByTab}
      onTabChange={handleTabValueChange}
      onCloseTab={(tabId) => requestCloseTab(tabId)}
      onAddDashboardTab={addDashboardTab}
      onReorderTab={reorderTab}
    />
    <DashboardView
      {pendingCreatePath}
      {isOpeningDatabase}
      {isCreatingDatabase}
      {errorMessage}
      recentPaths={$recentPathsPreference}
      on:open={() => void openDatabaseFromDialog()}
      on:openReadOnly={() => void openDatabaseFromDialog(true)}
      on:create={() => void createDatabaseAtPendingPath()}
      on:cancelCreate={() => {
        pendingCreatePath = null;
      }}
      on:removeRecent={(event) => {
        recentPathsPreference.remove(event.detail.path);
      }}
      on:openRecent={(event) =>
        void openDatabaseFromPath(event.detail.path, false)}
      on:openRecentReadOnly={(event) =>
        void openDatabaseFromPath(event.detail.path, true)}
    />
  </div>
{/if}

<DeleteKeyDialog
  {keyPendingDelete}
  {isDeleting}
  {effectiveReadOnly}
  on:cancel={() => {
    keyPendingDelete = null;
  }}
  on:confirm={() => void confirmDeleteKey()}
/>

<UnsavedChangesDialog
  open={Boolean(pendingUnsavedClose)}
  {isSaving}
  canSaveAndClose={canSaveAndClosePendingTab}
  on:cancel={cancelPendingUnsavedClose}
  on:closeWithoutSaving={() => void closePendingWithoutSaving()}
  on:saveAndClose={() => void saveAndClosePending()}
/>

<CloseLastTabDialog
  open={pendingCloseLastTab}
  on:cancel={cancelPendingCloseLastTab}
  on:confirm={() => void closeWindowAfterLastTabConfirm()}
  on:confirmAndDisable={() => void closeWindowAndDisableLastTabConfirm()}
/>

<SettingsDialog
  open={showSettingsDialog}
  on:close={() => (showSettingsDialog = false)}
/>
