<script lang="ts">
  import { ScrollArea as ScrollAreaPrimitive } from "bits-ui";
  import { Scrollbar } from "./index.js";
  import { cn } from "$lib/utils.js";

  type $$Props = ScrollAreaPrimitive.Props & {
    orientation?: "vertical" | "horizontal" | "both";
    scrollbarXClasses?: string;
    scrollbarYClasses?: string;
    contentClass?: string;
    contentStyle?: string;
    viewportEl?: HTMLDivElement | undefined;
  };

  let className: $$Props["class"] = undefined;
  export { className as class };
  export let orientation = "vertical";
  export let scrollbarXClasses: string = "";
  export let scrollbarYClasses: string = "";
  export let contentClass: string = "";
  export let contentStyle: string = "display: block; width: 100%; min-width: 100%;";
  export let viewportEl: HTMLDivElement | undefined = undefined;
</script>

<ScrollAreaPrimitive.Root
  {...$$restProps}
  class={cn("relative overflow-hidden", className)}
>
  <ScrollAreaPrimitive.Viewport bind:el={viewportEl} class="h-full w-full rounded-[inherit]">
    <ScrollAreaPrimitive.Content asChild let:builder>
      <div
        {...builder}
        use:builder.action
        class={cn("block w-full min-w-full", contentClass)}
        style={contentStyle}
      >
        <slot />
      </div>
    </ScrollAreaPrimitive.Content>
  </ScrollAreaPrimitive.Viewport>
  {#if orientation === "vertical" || orientation === "both"}
    <Scrollbar orientation="vertical" class={scrollbarYClasses} />
  {/if}
  {#if orientation === "horizontal" || orientation === "both"}
    <Scrollbar orientation="horizontal" class={scrollbarXClasses} />
  {/if}
  <ScrollAreaPrimitive.Corner />
</ScrollAreaPrimitive.Root>
