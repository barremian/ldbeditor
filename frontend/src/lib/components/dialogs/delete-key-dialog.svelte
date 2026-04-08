<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { RefreshCcw, Trash2 } from "lucide-svelte";
  import { Button } from "$lib/components/ui/button";
  import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "$lib/components/ui/card";

  export let keyPendingDelete: string | null = null;
  export let isDeleting = false;
  export let effectiveReadOnly = false;

  const dispatch = createEventDispatcher<{
    cancel: void;
    confirm: void;
  }>();
</script>

{#if keyPendingDelete}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-background/55 p-4 backdrop-blur-sm"
    role="presentation"
    on:click={() => {
      if (!isDeleting) {
        dispatch("cancel");
      }
    }}
  >
    <Card
      class="w-full max-w-md border-destructive/40"
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-key-title"
      aria-describedby="delete-key-description"
      on:click={(event) => event.stopPropagation()}
    >
      <CardHeader class="space-y-2">
        <CardTitle id="delete-key-title">Delete key?</CardTitle>
        <CardDescription id="delete-key-description">
          Delete <span class="break-all font-semibold">{keyPendingDelete}</span>?
          This action cannot be undone.
        </CardDescription>
      </CardHeader>
      <CardContent class="flex justify-end gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={isDeleting || effectiveReadOnly}
          on:click={() => dispatch("cancel")}
        >
          Cancel
        </Button>
        <Button
          variant="destructive"
          size="sm"
          disabled={isDeleting || effectiveReadOnly}
          on:click={() => dispatch("confirm")}
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
