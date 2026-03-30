<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { DropdownMenuItem } from "$lib/components/ui/dropdown-menu";
  import { SplitButton } from "$lib/components/ui/split-button";
  import { Button } from "$lib/components/ui/button";
  import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "$lib/components/ui/card";

  export let open = false;

  const dispatch = createEventDispatcher<{
    cancel: void;
    confirm: void;
    confirmAndDisable: void;
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
      aria-labelledby="close-last-tab-title"
      aria-describedby="close-last-tab-description"
      on:click={(event) => event.stopPropagation()}
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
          on:click={() => dispatch("cancel")}
        >
          Cancel
        </Button>
        <SplitButton
          size="sm"
          triggerLabel="Close options"
          primaryClass="pr-2"
          triggerClass="w-8 px-0"
          triggerIconClass="h-3.5 w-3.5"
          on:primary={() => dispatch("confirm")}
        >
          Close
          <DropdownMenuItem
            slot="menu"
            on:click={() => dispatch("confirmAndDisable")}
          >
            Close and don&apos;t ask again
          </DropdownMenuItem>
        </SplitButton>
      </CardContent>
    </Card>
  </div>
{/if}
