<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { ChevronDown } from "lucide-svelte";
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
    primaryDisabled?: boolean;
    menuDisabled?: boolean;
    primaryLabel?: string;
    triggerLabel?: string;
    menuAlign?: ContentProps["align"];
    menuSideOffset?: ContentProps["sideOffset"];
    primaryClass?: string;
    triggerClass?: string;
    menuClass?: string;
    triggerIconClass?: string;
  };

  type $$Events = {
    primary: MouseEvent;
  };

  let className: $$Props["class"] = undefined;
  export let variant: $$Props["variant"] = "default";
  export let size: $$Props["size"] = "default";
  export let disabled: $$Props["disabled"] = false;
  export let primaryDisabled: $$Props["primaryDisabled"] = false;
  export let menuDisabled: $$Props["menuDisabled"] = false;
  export let primaryLabel: $$Props["primaryLabel"] = "Action";
  export let triggerLabel: $$Props["triggerLabel"] = "More actions";
  export let menuAlign: $$Props["menuAlign"] = "end";
  export let menuSideOffset: $$Props["menuSideOffset"] = 4;
  export let primaryClass: $$Props["primaryClass"] = undefined;
  export let triggerClass: $$Props["triggerClass"] = undefined;
  export let menuClass: $$Props["menuClass"] = undefined;
  export let triggerIconClass: $$Props["triggerIconClass"] = "h-4 w-4";
  export { className as class };

  const dispatch = createEventDispatcher<$$Events>();

  function handlePrimaryClick(event: MouseEvent) {
    dispatch("primary", event);
  }
</script>

<div class={cn("inline-flex rounded-md shadow-sm", className)} role="group">
  <Button
    {variant}
    {size}
    disabled={disabled || primaryDisabled}
    class={cn("rounded-r-none", primaryClass)}
    on:click={handlePrimaryClick}
  >
    <slot>{primaryLabel}</slot>
  </Button>

  <DropdownMenu>
    <DropdownMenuTrigger asChild let:builder>
      <Button
        builders={[builder]}
        {variant}
        {size}
        disabled={disabled || menuDisabled}
        class={cn("rounded-l-none border-l border-border/60 px-2", triggerClass)}
        aria-label={triggerLabel}
      >
        <ChevronDown class={triggerIconClass} />
      </Button>
    </DropdownMenuTrigger>

    <DropdownMenuContent
      align={menuAlign}
      sideOffset={menuSideOffset}
      class={menuClass}
    >
      <slot name="menu" />
    </DropdownMenuContent>
  </DropdownMenu>
</div>
