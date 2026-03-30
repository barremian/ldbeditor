<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { Check, Copy, FileText, Pencil, RefreshCcw, Save, X } from "lucide-svelte";
  import { Badge } from "$lib/components/ui/badge";
  import { Button } from "$lib/components/ui/button";
  import { DropdownMenuItem } from "$lib/components/ui/dropdown-menu";
  import { SplitButton } from "$lib/components/ui/split-button";
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from "$lib/components/ui/card";
  import ValueCodeMirror from "$lib/components/value-codemirror.svelte";

  export let selectedKey: string | null = null;
  export let editorValue = "";
  export let isDirty = false;
  export let isValueEditing = false;
  export let effectiveReadOnly = false;
  export let valueLoading = false;
  export let canSaveValueChanges = false;
  export let isSaving = false;
  export let hasCopiedValue = false;
  export let valueValidationError = "";

  const dispatch = createEventDispatcher<{
    edit: void;
    save: void;
    saveAndClose: void;
    revert: void;
    copy: void;
    valueChange: { value: string };
  }>();

  function handleEditorChange(
    event: { value: string } | CustomEvent<{ value: string }>
  ) {
    const value = "detail" in event ? event.detail.value : event.value;
    dispatch("valueChange", { value });
  }
</script>

<Card class="flex h-full min-h-0 min-w-0 w-full flex-col" aria-busy={valueLoading}>
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
            on:primary={() => dispatch("save")}
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
              on:click={() => dispatch("saveAndClose")}
            >
              Save & Close
            </DropdownMenuItem>
          </SplitButton>
          <Button
            variant="outline"
            size="sm"
            class="gap-1.5"
            disabled={isSaving}
            on:click={() => dispatch("revert")}
          >
            <X class="h-3.5 w-3.5" />
            Cancel
          </Button>
        {:else}
          <Button
            size="sm"
            class="gap-1.5"
            disabled={!selectedKey || valueLoading || isSaving || effectiveReadOnly}
            on:click={() => dispatch("edit")}
          >
            <Pencil class="h-3.5 w-3.5" />
            Edit
          </Button>
        {/if}
        <Button
          variant="outline"
          size="icon"
          disabled={!selectedKey || valueLoading || isSaving}
          on:click={() => dispatch("copy")}
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
        <span class="block w-full truncate" title={selectedKey}>{selectedKey}</span>
      {:else}
        Select a key to inspect its value
      {/if}
    </CardDescription>
  </CardHeader>
  <CardContent class="flex min-h-0 flex-1 flex-col gap-2 px-3 pb-3">
    <div class="relative flex min-h-0 flex-1 flex-col">
      {#if selectedKey === null}
        <div class="px-1 py-3 text-sm text-muted-foreground">Select a key</div>
      {:else if valueLoading}
        <div
          class="flex h-full min-h-0 flex-1 flex-col rounded-md border border-border/70 bg-muted/20 p-3"
          role="status"
          aria-live="polite"
        >
          <span class="sr-only">Loading value...</span>
          <div
            class="mb-3 h-3 w-40 rounded bg-muted/80 motion-safe:animate-pulse motion-reduce:animate-none"
          ></div>
          <div class="space-y-2.5" aria-hidden="true">
            <div class="h-3 w-full rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"></div>
            <div class="h-3 w-[95%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"></div>
            <div class="h-3 w-[88%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"></div>
            <div class="h-3 w-[92%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"></div>
            <div class="h-3 w-[83%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"></div>
            <div class="h-3 w-[90%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"></div>
            <div class="h-3 w-[72%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"></div>
            <div class="h-3 w-[86%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"></div>
            <div class="h-3 w-[65%] rounded bg-muted/70 motion-safe:animate-pulse motion-reduce:animate-none"></div>
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
          on:change={handleEditorChange}
          readOnly={!isValueEditing || isSaving || effectiveReadOnly}
        />
        {#if valueValidationError}
          <p class="px-1 text-xs text-destructive">{valueValidationError}</p>
        {/if}
      {/if}
    </div>
  </CardContent>
</Card>
