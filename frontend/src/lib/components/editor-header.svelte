<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { Database, RefreshCcw } from "lucide-svelte";
  import { Badge } from "$lib/components/ui/badge";
  import { DropdownMenuItem } from "$lib/components/ui/dropdown-menu";
  import { SplitButton } from "$lib/components/ui/split-button";
  import { ThreeDotMenu } from "$lib/components/ui/three-dot-menu";
  import {
    AUTO_REFRESH_OPTIONS,
    getAutoRefreshLabel,
  } from "$lib/services/auto-refresh";

  const REFRESH_RING_RADIUS = 6;
  const REFRESH_RING_CIRCUMFERENCE = 2 * Math.PI * REFRESH_RING_RADIUS;

  export let dbPath = "";
  export let isRefreshing = false;
  export let autoRefreshIntervalMs = 0;
  export let autoRefreshProgress = 1;
  export let prettyPrintJson = false;
  export let canTogglePrettyPrint = true;
  export let refreshButtonTitle = "Refresh now";

  const dispatch = createEventDispatcher<{
    refresh: void;
    setAutoRefresh: { intervalMs: number };
    togglePrettyPrint: void;
  }>();

  $: autoRefreshLabel = getAutoRefreshLabel(autoRefreshIntervalMs);
  $: dbPathLabel = dbPath.split(/[/\\]/).pop() || dbPath;
</script>

<header
  class="relative z-40 overflow-visible bg-background/60 px-4 pb-2 pt-3 backdrop-blur-sm"
>
  <div
    class="flex flex-col gap-2 md:flex-row md:items-center md:justify-between"
  >
    <div class="flex min-w-0 flex-1 items-center gap-2">
      <div class="flex min-w-0 flex-1 items-center gap-2">
        <Badge
          variant="secondary"
          class="min-w-0 max-w-full shrink"
          title={dbPath}
        >
          <Database class="mr-1.5 h-3.5 w-3.5 shrink-0" />
          <span class="min-w-0 truncate">{dbPathLabel}</span>
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
        on:primary={() => dispatch("refresh")}
      >
        <span
          class="inline-flex items-center gap-1.5"
          title={refreshButtonTitle}
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
          {isRefreshing ? "Refreshing..." : "Refresh"}
        </span>

        <svelte:fragment slot="menu">
          {#each AUTO_REFRESH_OPTIONS as option}
            <DropdownMenuItem
              radio={true}
              checked={option.intervalMs === autoRefreshIntervalMs}
              disabled={isRefreshing}
              on:click={() =>
                dispatch("setAutoRefresh", { intervalMs: option.intervalMs })}
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
          disabled={!canTogglePrettyPrint}
          class={`flex items-center justify-between ${
            prettyPrintJson ? "bg-muted/60" : ""
          }`}
          on:click={(event) => {
            event.preventDefault();
            dispatch("togglePrettyPrint");
          }}
        >
          <span>Pretty JSON</span>
          <span
            class={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border transition-colors ${
              prettyPrintJson
                ? "border-primary/40 bg-primary"
                : "border-border bg-muted"
            } ${canTogglePrettyPrint ? "" : "opacity-50"}`}
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
