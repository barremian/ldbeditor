import type { Tabs as TabsPrimitive } from "bits-ui";
import Content from "./tabs-content.svelte";
import List from "./tabs-list.svelte";
import Root from "./tabs.svelte";
import Trigger from "./tabs-trigger.svelte";

type Props = TabsPrimitive.Props;
type ListProps = TabsPrimitive.ListProps;
type TriggerProps = TabsPrimitive.TriggerProps;
type ContentProps = TabsPrimitive.ContentProps;

export {
  Root,
  List,
  Trigger,
  Content,
  type Props,
  type ListProps,
  type TriggerProps,
  type ContentProps,
  //
  Root as Tabs,
  List as TabsList,
  Trigger as TabsTrigger,
  Content as TabsContent,
};
