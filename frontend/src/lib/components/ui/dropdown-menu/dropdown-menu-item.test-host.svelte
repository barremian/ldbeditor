<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
  } from "$lib/components/ui/dropdown-menu";

  export let onItemClick: () => void;
  export let radio = false;
  export let checked = false;
  export let showRadioIndicator = true;
</script>

<DropdownMenu>
  <DropdownMenuTrigger asChild let:builder>
    <Button builders={[builder]} aria-label="Open actions">Open actions</Button>
  </DropdownMenuTrigger>
  <DropdownMenuContent>
    <DropdownMenuItem
      {radio}
      {checked}
      {showRadioIndicator}
      on:click={onItemClick}
    >
      Open in read-only mode
    </DropdownMenuItem>
  </DropdownMenuContent>
</DropdownMenu>
