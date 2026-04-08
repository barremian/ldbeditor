<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { RefreshCcw, Save } from "lucide-svelte";
  import { Button } from "$lib/components/ui/button";
  import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "$lib/components/ui/card";

  export let open = false;
  export let isSaving = false;
  export let canSaveAndClose = false;

  const dispatch = createEventDispatcher<{
    cancel: void;
    closeWithoutSaving: void;
    saveAndClose: void;
  }>();
</script>

{#if open}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-background/55 p-4 backdrop-blur-sm"
    role="presentation"
    on:click={() => dispatch("cancel")}
  >
    <Card
      class="w-full max-w-md border-border"
      role="dialog"
      aria-modal="true"
      aria-labelledby="close-unsaved-title"
      aria-describedby="close-unsaved-description"
      on:click={(event) => event.stopPropagation()}
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
          on:click={() => dispatch("cancel")}
        >
          Cancel
        </Button>
        <Button
          variant="secondary"
          size="sm"
          disabled={isSaving}
          on:click={() => dispatch("closeWithoutSaving")}
        >
          Close without Saving
        </Button>
        <Button
          size="sm"
          disabled={!canSaveAndClose}
          on:click={() => dispatch("saveAndClose")}
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
