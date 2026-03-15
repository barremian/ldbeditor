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

  const SCROLL_STEP_PX = 240;
  const EDGE_TOLERANCE_PX = 2;

  let viewportEl: HTMLDivElement | undefined = undefined;
  let hasHorizontalOverflow = false;
  let canScrollLeft = false;
  let canScrollRight = false;
  let cleanupViewportBindings: (() => void) | null = null;

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

  onDestroy(() => {
    cleanupViewportBindings?.();
  });
</script>

<div class="relative bg-background/60 pt-2 backdrop-blur-sm">
  <ScrollArea
    orientation="horizontal"
    class="w-full"
    scrollbarXClasses="z-20 h-2.5 border-t border-border/60 bg-muted/20 hover:bg-muted/35 active:bg-muted/45"
    bind:viewportEl
  >
    <Tabs value={activeTabId} onValueChange={onTabChange} class="w-full">
      <TabsList
        class={`relative h-auto w-max min-w-full items-end border-b border-border !border-x-0 !border-t-0 bg-transparent p-0 rounded-none ${
          hasHorizontalOverflow ? "px-8" : ""
        }`}
      >
        {#each tabs as tab, index (tab.id)}
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
                  void onCloseTab(tab.id);
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
