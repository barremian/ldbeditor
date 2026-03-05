<script lang="ts">
  import { DropdownMenu as DropdownMenuPrimitive } from "bits-ui";
  import { cn } from "$lib/utils.js";

  type $$Props = DropdownMenuPrimitive.DropdownTriggerProps;

  let className: $$Props["class"] = undefined;
  export { className as class };
</script>

<DropdownMenuPrimitive.Trigger class={cn(className)} {...$$restProps} let:builder>
  <slot {builder} />
</DropdownMenuPrimitive.Trigger>
