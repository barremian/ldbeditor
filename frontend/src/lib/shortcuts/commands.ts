export const ShortcutCommand = {
  Escape: "escape",
} as const;

export type ShortcutCommandId =
  (typeof ShortcutCommand)[keyof typeof ShortcutCommand];
