<script lang="ts">
  import { DropdownMenu as DropdownMenuPrimitive } from "bits-ui";
  import { flyAndScale, cn } from "$lib/utils.js";

  type $$Props = DropdownMenuPrimitive.ContentProps;

  let className: $$Props["class"] = undefined;
  export let sideOffset: $$Props["sideOffset"] = 4;
  export { className as class };
</script>

<DropdownMenuPrimitive.Content
  transition={flyAndScale}
  {sideOffset}
  class={cn(
    "bg-popover text-popover-foreground z-50 min-w-[8rem] overflow-hidden rounded-md border p-1 shadow-md",
    className
  )}
  {...$$restProps}
>
  <slot />
</DropdownMenuPrimitive.Content>
