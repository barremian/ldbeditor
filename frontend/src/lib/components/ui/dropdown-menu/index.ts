import type { DropdownMenu as DropdownMenuPrimitive } from "bits-ui";
import Root from "./dropdown-menu.svelte";
import Trigger from "./dropdown-menu-trigger.svelte";
import Content from "./dropdown-menu-content.svelte";
import Item from "./dropdown-menu-item.svelte";
import Separator from "./dropdown-menu-separator.svelte";

type Props = DropdownMenuPrimitive.Props;
type TriggerProps = DropdownMenuPrimitive.DropdownTriggerProps;
type ContentProps = DropdownMenuPrimitive.ContentProps;
type ItemProps = DropdownMenuPrimitive.ItemProps;
type SeparatorProps = DropdownMenuPrimitive.SeparatorProps;

export {
  Root,
  Trigger,
  Content,
  Item,
  Separator,
  type Props,
  type TriggerProps,
  type ContentProps,
  type ItemProps,
  type SeparatorProps,
  //
  Root as DropdownMenu,
  Trigger as DropdownMenuTrigger,
  Content as DropdownMenuContent,
  Item as DropdownMenuItem,
  Separator as DropdownMenuSeparator,
};
