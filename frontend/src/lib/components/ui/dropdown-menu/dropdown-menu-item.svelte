<script lang="ts">
  import { DropdownMenu as DropdownMenuPrimitive } from "bits-ui";
  import { Check } from "lucide-svelte";
  import { cn } from "$lib/utils.js";

  type $$Props = DropdownMenuPrimitive.ItemProps & {
    inset?: boolean;
    radio?: boolean;
    checked?: boolean;
    showRadioIndicator?: boolean;
  };

  let className: $$Props["class"] = undefined;
  export let inset: $$Props["inset"] = false;
  export let radio: $$Props["radio"] = false;
  export let checked: $$Props["checked"] = false;
  export let showRadioIndicator: $$Props["showRadioIndicator"] = true;
  export { className as class };
</script>

{#if radio}
  <DropdownMenuPrimitive.Item
    class={cn(
      "focus:bg-accent focus:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50 relative flex cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none",
      inset && "pl-8",
      "w-full justify-between",
      checked && "bg-muted/80",
      className
    )}
    role="menuitemradio"
    aria-checked={checked}
    {...$$restProps}
    on:click
    on:keydown
  >
    <slot />
    {#if showRadioIndicator && checked}
      <Check class="h-3.5 w-3.5" />
    {/if}
  </DropdownMenuPrimitive.Item>
{:else}
  <DropdownMenuPrimitive.Item
    class={cn(
      "focus:bg-accent focus:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50 relative flex cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none",
      inset && "pl-8",
      className
    )}
    {...$$restProps}
    on:click
    on:keydown
  >
    <slot />
  </DropdownMenuPrimitive.Item>
{/if}
