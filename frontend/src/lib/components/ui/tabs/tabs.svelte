<script lang="ts">
  import { Tabs as TabsPrimitive } from "bits-ui";
  import { cn } from "$lib/utils.js";

  type $$Props = TabsPrimitive.Props;

  let className: $$Props["class"] = undefined;
  export { className as class };
</script>

<TabsPrimitive.Root class={cn("w-full", className)} {...$$restProps}>
  <slot />
</TabsPrimitive.Root>
