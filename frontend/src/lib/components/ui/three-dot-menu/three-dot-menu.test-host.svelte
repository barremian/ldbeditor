<script lang="ts">
  import { DropdownMenuItem } from "$lib/components/ui/dropdown-menu";
  import { ThreeDotMenu } from "$lib/components/ui/three-dot-menu";

  export let onItemClick: () => void;
  export let disabled = false;
  export let persistOnSelect = false;

  function handleItemClick(event: Event) {
    if (persistOnSelect) {
      event.preventDefault();
    }
    onItemClick();
  }
</script>

<ThreeDotMenu triggerLabel="Open options" triggerTitle="Open options" {disabled}>
  <DropdownMenuItem slot="menu" on:click={handleItemClick}>
    Open in read-only mode
  </DropdownMenuItem>
</ThreeDotMenu>
