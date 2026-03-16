import { ShortcutCommand, type ShortcutCommandId } from "$lib/shortcuts/commands";

export const shortcutConfig = {
  [ShortcutCommand.CloseActiveTabOrWindow]: ["CmdOrCtrl+W"],
  [ShortcutCommand.SaveValue]: ["CmdOrCtrl+S"],
  [ShortcutCommand.Escape]: ["Escape"],
} satisfies Partial<Record<ShortcutCommandId, string[]>>;
