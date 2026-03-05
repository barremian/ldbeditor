<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { MoreHorizontal } from "lucide-svelte";
  import { Button, type ButtonProps } from "$lib/components/ui/button";
  import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuTrigger,
    type ContentProps,
  } from "$lib/components/ui/dropdown-menu";
  import { cn } from "$lib/utils.js";

  type $$Props = {
    class?: string;
    variant?: ButtonProps["variant"];
    size?: ButtonProps["size"];
    disabled?: boolean;
    triggerLabel?: string;
    triggerTitle?: string;
    menuAlign?: ContentProps["align"];
    menuSideOffset?: ContentProps["sideOffset"];
    menuClass?: string;
  };

  type $$Events = {
    triggerclick: MouseEvent;
  };

  let className: $$Props["class"] = undefined;
  export let variant: $$Props["variant"] = "outline";
  export let size: $$Props["size"] = "icon";
  export let disabled: $$Props["disabled"] = false;
  export let triggerLabel: $$Props["triggerLabel"] = "More options";
  export let triggerTitle: $$Props["triggerTitle"] = "More options";
  export let menuAlign: $$Props["menuAlign"] = "end";
  export let menuSideOffset: $$Props["menuSideOffset"] = 4;
  export let menuClass: $$Props["menuClass"] = undefined;
  export { className as class };

  const dispatch = createEventDispatcher<$$Events>();

  function handleTriggerClick(event: MouseEvent) {
    dispatch("triggerclick", event);
  }
</script>

<DropdownMenu>
  <DropdownMenuTrigger asChild let:builder>
    <Button
      builders={[builder]}
      {variant}
      {size}
      {disabled}
      class={cn(className)}
      title={triggerTitle}
      aria-label={triggerLabel}
      on:click={handleTriggerClick}
    >
      <MoreHorizontal class="h-3.5 w-3.5" />
    </Button>
  </DropdownMenuTrigger>

  <DropdownMenuContent
    align={menuAlign}
    sideOffset={menuSideOffset}
    class={cn(menuClass)}
  >
    <slot name="menu" />
  </DropdownMenuContent>
</DropdownMenu>
