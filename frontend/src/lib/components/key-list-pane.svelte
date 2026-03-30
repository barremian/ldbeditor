<script lang="ts">
  import { createEventDispatcher, tick } from "svelte";
  import { Check, KeyRound, Pencil, Plus, RefreshCcw, Save, Trash2, X } from "lucide-svelte";
  import { Button } from "$lib/components/ui/button";
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from "$lib/components/ui/card";

  export let keys: string[] = [];
  export let filteredKeys: string[] = [];
  export let selectedKey: string | null = null;
  export let effectiveReadOnly = false;
  export let showCreateForm = false;
  export let isCreating = false;
  export let isRenaming = false;
  export let isDeleting = false;
  export let newKeyInput = "";
  export let newValueInput = "";
  export let editingKey: string | null = null;
  export let renameInput = "";
  export let keySearchInput = "";
  export let createKeyValidationError = "";
  export let createValueValidationError = "";
  export let renameValidationError = "";
  export let viewportEl: HTMLDivElement | null = null;

  const dispatch = createEventDispatcher<{
    createKey: void;
    selectKey: { key: string };
    startRename: { key: string };
    confirmRename: void;
    requestDelete: { key: string };
    clearSearch: void;
    cancelCreate: void;
    resetRename: void;
    scroll: void;
  }>();

  let renameInputElement: HTMLInputElement | null = null;
  let lastFocusedRenameKey: string | null = null;

  $: if (editingKey && editingKey !== lastFocusedRenameKey) {
    lastFocusedRenameKey = editingKey;
    void tick().then(() => {
      if (!renameInputElement) return;
      renameInputElement.focus();
      const end = renameInputElement.value.length;
      renameInputElement.setSelectionRange(end, end);
    });
  }

  $: if (!editingKey) {
    lastFocusedRenameKey = null;
  }
</script>

<Card class="flex h-full min-h-0 min-w-0 flex-col">
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
              dispatch("cancelCreate");
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
          <p class="text-xs text-destructive">{createKeyValidationError}</p>
        {/if}
        <textarea
          class="min-h-20 w-full rounded-md border border-input bg-background p-2 font-mono text-sm"
          placeholder="Initial value (text or 0x...)"
          bind:value={newValueInput}
          disabled={isCreating || effectiveReadOnly}
        ></textarea>
        {#if createValueValidationError}
          <p class="text-xs text-destructive">{createValueValidationError}</p>
        {/if}
        <div class="flex items-center gap-2">
          <Button
            size="sm"
            class="gap-1.5"
            disabled={isCreating ||
              effectiveReadOnly ||
              !!createKeyValidationError ||
              !!createValueValidationError}
            on:click={() => dispatch("createKey")}
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
            on:click={() => dispatch("cancelCreate")}
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
          on:click={() => dispatch("clearSearch")}
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
          bind:this={viewportEl}
          on:scroll={() => dispatch("scroll")}
        >
          <ul class="w-full space-y-1 p-2">
            {#each filteredKeys as key}
              <li class="group w-full">
                <div
                  data-key-item-selected={selectedKey === key ? "true" : undefined}
                  class={`relative flex items-center overflow-hidden rounded-md px-2 py-1.5 text-sm transition-colors ${
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
                            dispatch("confirmRename");
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
                          effectiveReadOnly ||
                          !renameInput ||
                          !!renameValidationError}
                        on:click={(event) => {
                          event.stopPropagation();
                          dispatch("confirmRename");
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
                          dispatch("resetRename");
                        }}
                      >
                        <X class="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  {:else}
                    <button
                      class="min-w-0 flex-1 text-left transition-[padding] duration-150 group-hover:pr-15 group-focus-within:pr-15"
                      title={key}
                      on:click={() => dispatch("selectKey", { key })}
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
                          dispatch("startRename", { key });
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
                          dispatch("requestDelete", { key });
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
