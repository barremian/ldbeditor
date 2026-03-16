export const ShortcutCommand = {
  CloseActiveTabOrWindow: "closeActiveTabOrWindow",
  SaveValue: "saveValue",
  Escape: "escape",
} as const;

export type ShortcutCommandId =
  (typeof ShortcutCommand)[keyof typeof ShortcutCommand];
