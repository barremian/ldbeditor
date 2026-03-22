import { ShortcutCommand, type ShortcutCommandId } from "$lib/shortcuts/commands";

export const shortcutConfig = {
  [ShortcutCommand.Escape]: ["Escape"],
} satisfies Partial<Record<ShortcutCommandId, string[]>>;
