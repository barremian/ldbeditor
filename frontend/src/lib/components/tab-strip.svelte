<script lang="ts">
  import { onDestroy, tick } from "svelte";
  import { ChevronLeft, ChevronRight, Plus, X } from "lucide-svelte";
  import { Button } from "$lib/components/ui/button";
  import { ScrollArea } from "$lib/components/ui/scroll-area";
  import { Tabs, TabsList, TabsTrigger } from "$lib/components/ui/tabs";

  type WorkspaceTab =
    | { id: string; type: "dashboard"; title: string }
    | { id: string; type: "database"; title: string; path: string };

  export let tabs: WorkspaceTab[] = [];
  export let activeTabId = "";
  export let canCloseTab: (tab: WorkspaceTab) => boolean = () => false;
  export let onTabChange: (tabId: string | undefined) => void = () => {};
  export let onCloseTab: (tabId: string) => void | Promise<void> = () => {};
  export let onAddDashboardTab: () => void | Promise<void> = () => {};
  export let onReorderTab: (tabId: string, targetIndex: number) => void | Promise<void> =
    () => {};

  const SCROLL_STEP_PX = 240;
  const EDGE_TOLERANCE_PX = 2;
  const DRAG_START_THRESHOLD_PX = 4;

  let viewportEl: HTMLDivElement | undefined = undefined;
  let hasHorizontalOverflow = false;
  let canScrollLeft = false;
  let canScrollRight = false;
  let cleanupViewportBindings: (() => void) | null = null;
  let tabElements: Record<string, HTMLDivElement> = {};
  let dragPointerId: number | null = null;
  let dragStartX = 0;
  let pointerX = 0;
  let draggedTabId: string | null = null;
  let draggedTabIndex = -1;
  let isDragCandidate = false;
  let isDraggingTab = false;
  let isHoveringOriginalSlot = false;
  let dropIndex: number | null = null;
  let dropBoundarySlot: number | null = null;
  let suppressClickAfterDrag = false;

  function registerTabElement(node: HTMLDivElement, tabId: string) {
    tabElements[tabId] = node;
    return {
      update(nextTabId: string) {
        if (nextTabId === tabId) return;
        delete tabElements[tabId];
        tabId = nextTabId;
        tabElements[tabId] = node;
      },
      destroy() {
        delete tabElements[tabId];
      },
    };
  }

  function resetDragState(options: { preserveSuppressedClick?: boolean } = {}) {
    const preserveSuppressedClick = options.preserveSuppressedClick ?? false;
    dragPointerId = null;
    dragStartX = 0;
    pointerX = 0;
    draggedTabId = null;
    draggedTabIndex = -1;
    isDragCandidate = false;
    isDraggingTab = false;
    isHoveringOriginalSlot = false;
    dropIndex = null;
    dropBoundarySlot = null;
    if (!preserveSuppressedClick) {
      suppressClickAfterDrag = false;
    }
  }

  function getTabLayouts() {
    return tabs
      .map((tab, index) => {
        const element = tabElements[tab.id];
        if (!element) return null;
        const rect = element.getBoundingClientRect();
        return {
          index,
          midpoint: rect.left + rect.width / 2,
        };
      })
      .filter((layout): layout is { index: number; midpoint: number } => layout !== null);
  }

  function updateDropTarget() {
    if (!isDraggingTab || draggedTabIndex < 0) {
      dropIndex = null;
      dropBoundarySlot = null;
      isHoveringOriginalSlot = false;
      return;
    }

    const layouts = getTabLayouts();
    if (layouts.length !== tabs.length) {
      dropIndex = null;
      dropBoundarySlot = null;
      isHoveringOriginalSlot = false;
      return;
    }

    const leftNeighborMidpoint =
      draggedTabIndex > 0
        ? layouts[draggedTabIndex - 1].midpoint
        : Number.NEGATIVE_INFINITY;
    const rightNeighborMidpoint =
      draggedTabIndex < layouts.length - 1
        ? layouts[draggedTabIndex + 1].midpoint
        : Number.POSITIVE_INFINITY;

    if (pointerX > leftNeighborMidpoint && pointerX < rightNeighborMidpoint) {
      dropIndex = null;
      dropBoundarySlot = null;
      isHoveringOriginalSlot = true;
      return;
    }

    isHoveringOriginalSlot = false;

    const remainingLayouts = layouts.filter(
      (layout) => layout.index !== draggedTabIndex
    );

    let insertionIndex = remainingLayouts.length;
    for (let index = 0; index < remainingLayouts.length; index += 1) {
      if (pointerX < remainingLayouts[index].midpoint) {
        insertionIndex = index;
        break;
      }
    }

    dropIndex = insertionIndex;
    dropBoundarySlot =
      insertionIndex <= draggedTabIndex ? insertionIndex : insertionIndex + 1;
  }

  function handleTabPointerDown(event: PointerEvent, tabId: string, tabIndex: number) {
    if (!event.isPrimary || event.button !== 0) return;

    const target = event.target as HTMLElement | null;
    if (target?.closest("[data-tab-close-button='true']")) return;

    const currentTarget = event.currentTarget as HTMLElement | null;
    if (!currentTarget) return;

    currentTarget.setPointerCapture(event.pointerId);
    dragPointerId = event.pointerId;
    dragStartX = event.clientX;
    pointerX = event.clientX;
    draggedTabId = tabId;
    draggedTabIndex = tabIndex;
    isDragCandidate = true;
    isDraggingTab = false;
    isHoveringOriginalSlot = false;
    dropIndex = null;
    dropBoundarySlot = null;
    suppressClickAfterDrag = false;
  }

  function handleTabPointerMove(event: PointerEvent) {
    if (dragPointerId !== event.pointerId || !isDragCandidate) return;

    pointerX = event.clientX;
    if (!isDraggingTab) {
      if (Math.abs(pointerX - dragStartX) < DRAG_START_THRESHOLD_PX) {
        return;
      }
      isDraggingTab = true;
    }

    event.preventDefault();
    updateDropTarget();
  }

  function handleTabPointerUp(event: PointerEvent) {
    if (dragPointerId !== event.pointerId) return;

    const currentTarget = event.currentTarget as HTMLElement | null;
    if (currentTarget?.hasPointerCapture(event.pointerId)) {
      currentTarget.releasePointerCapture(event.pointerId);
    }

    const reorderedTabId = draggedTabId;
    const targetIndex = dropIndex;
    const sourceIndex = draggedTabIndex;
    const shouldCommit =
      isDraggingTab &&
      reorderedTabId !== null &&
      targetIndex !== null &&
      sourceIndex !== -1 &&
      targetIndex !== sourceIndex;

    suppressClickAfterDrag = isDraggingTab;
    resetDragState({ preserveSuppressedClick: true });

    if (!shouldCommit) return;
    void onReorderTab(reorderedTabId, targetIndex);
  }

  function handleTabPointerCancel(event: PointerEvent) {
    if (dragPointerId !== event.pointerId) return;
    const currentTarget = event.currentTarget as HTMLElement | null;
    if (currentTarget?.hasPointerCapture(event.pointerId)) {
      currentTarget.releasePointerCapture(event.pointerId);
    }
    suppressClickAfterDrag = isDraggingTab;
    resetDragState({ preserveSuppressedClick: true });
  }

  function handleTabsValueChangeWithDragGuard(tabId: string | undefined) {
    if (suppressClickAfterDrag) {
      suppressClickAfterDrag = false;
      return;
    }
    onTabChange(tabId);
  }

  function updateOverflowState() {
    const viewport = viewportEl;
    if (!viewport) {
      hasHorizontalOverflow = false;
      canScrollLeft = false;
      canScrollRight = false;
      return;
    }
    const maxScrollLeft = Math.max(0, viewport.scrollWidth - viewport.clientWidth);
    hasHorizontalOverflow = maxScrollLeft > EDGE_TOLERANCE_PX;
    canScrollLeft = hasHorizontalOverflow && viewport.scrollLeft > EDGE_TOLERANCE_PX;
    canScrollRight =
      hasHorizontalOverflow &&
      viewport.scrollLeft < maxScrollLeft - EDGE_TOLERANCE_PX;
  }

  function bindViewport() {
    cleanupViewportBindings?.();
    cleanupViewportBindings = null;

    const viewport = viewportEl;
    if (!viewport) return;

    const onScroll = () => updateOverflowState();
    const resizeObserver = new ResizeObserver(() => updateOverflowState());
    resizeObserver.observe(viewport);

    viewport.addEventListener("scroll", onScroll, { passive: true });
    cleanupViewportBindings = () => {
      viewport.removeEventListener("scroll", onScroll);
      resizeObserver.disconnect();
    };

    void tick().then(updateOverflowState);
  }

  function scrollTabs(delta: number) {
    if (!viewportEl) return;
    viewportEl.scrollBy({ left: delta, behavior: "smooth" });
  }

  $: viewportEl, bindViewport();
  $: tabs, void tick().then(updateOverflowState);
  $: activeTabId, void tick().then(updateOverflowState);
  $: if (draggedTabId && !tabs.some((tab) => tab.id === draggedTabId)) {
    resetDragState();
  }

  onDestroy(() => {
    cleanupViewportBindings?.();
    resetDragState();
  });
</script>

<div class="relative bg-background/60 pt-2 backdrop-blur-sm">
  <ScrollArea
    orientation="horizontal"
    class="w-full"
    scrollbarXClasses="z-20 h-2.5 border-t border-border/60 bg-muted/20 hover:bg-muted/35 active:bg-muted/45"
    bind:viewportEl
  >
    <Tabs value={activeTabId} onValueChange={handleTabsValueChangeWithDragGuard} class="w-full">
      <TabsList
        class={`relative h-auto w-max min-w-full items-end border-b border-border !border-x-0 !border-t-0 bg-transparent p-0 rounded-none ${
          hasHorizontalOverflow ? "px-8" : ""
        }`}
      >
        {#each tabs as tab, index (tab.id)}
          {#if dropBoundarySlot === index}
            <div class="-mb-px h-7 w-0 shrink-0 pointer-events-none">
              <span class="relative -left-px block h-full w-0.5 rounded-full bg-primary/70"></span>
            </div>
          {/if}
          <div
            class={`-mb-px flex max-w-[264px] items-center gap-1 rounded-t-md border border-transparent pl-1 pr-1 ${
              activeTabId === tab.id
                ? "relative z-10 border-t-border border-l-border border-r-border border-b-transparent bg-background text-foreground"
                : `text-muted-foreground hover:bg-muted/40 ${
                    (index < tabs.length - 1 && activeTabId !== tabs[index + 1].id) ||
                    index === tabs.length - 1
                      ? "border-r-border/60"
                      : ""
                  }`
            } ${
              isDraggingTab && draggedTabId === tab.id
                ? isHoveringOriginalSlot
                  ? "opacity-75 ring-1 ring-ring/40 transition-none"
                  : "opacity-90 transition-none"
                : ""
            } ${isDraggingTab ? "cursor-grabbing" : "cursor-grab"}`}
            use:registerTabElement={tab.id}
            aria-grabbed={isDraggingTab && draggedTabId === tab.id}
            on:pointerdown={(event) => handleTabPointerDown(event, tab.id, index)}
            on:pointermove={handleTabPointerMove}
            on:pointerup={handleTabPointerUp}
            on:pointercancel={handleTabPointerCancel}
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
                data-tab-close-button="true"
                title={`Close ${tab.title}`}
                on:click={(event) => {
                  event.stopPropagation();
                  void onCloseTab(tab.id);
                }}
              >
                <X class="h-3.5 w-3.5" />
              </Button>
            {/if}
          </div>
        {/each}
        {#if dropBoundarySlot === tabs.length}
          <div class="-mb-px h-7 w-0 shrink-0 pointer-events-none">
            <span class="relative -left-px block h-full w-0.5 rounded-full bg-primary/70"></span>
          </div>
        {/if}
        <Button
          variant="ghost"
          size="icon"
          class="h-7 w-7 rounded-t-md text-muted-foreground hover:bg-muted/40 hover:text-foreground"
          title="New dashboard tab"
          on:click={() => {
            void onAddDashboardTab();
          }}
        >
          <Plus class="h-4 w-4" />
        </Button>
      </TabsList>
    </Tabs>
  </ScrollArea>

  {#if hasHorizontalOverflow}
    <div class="pointer-events-none absolute inset-x-0 top-2 z-20 flex items-start justify-between px-1">
      <Button
        variant="ghost"
        size="icon"
        class="pointer-events-auto h-6 w-6 rounded-md bg-background/85 text-muted-foreground shadow-sm backdrop-blur hover:text-foreground"
        title="Scroll tabs left"
        aria-label="Scroll tabs left"
        disabled={!canScrollLeft}
        on:click={() => scrollTabs(-SCROLL_STEP_PX)}
      >
        <ChevronLeft class="h-3.5 w-3.5" />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        class="pointer-events-auto h-6 w-6 rounded-md bg-background/85 text-muted-foreground shadow-sm backdrop-blur hover:text-foreground"
        title="Scroll tabs right"
        aria-label="Scroll tabs right"
        disabled={!canScrollRight}
        on:click={() => scrollTabs(SCROLL_STEP_PX)}
      >
        <ChevronRight class="h-3.5 w-3.5" />
      </Button>
    </div>
  {/if}
</div>
