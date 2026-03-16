import type { ShortcutCommandId } from "$lib/shortcuts/commands";

type ShortcutHandler = (
  context: { commandId: ShortcutCommandId; event?: KeyboardEvent }
) => void | Promise<void>;

type ShortcutMap = Partial<Record<ShortcutCommandId, string[]>>;

type ShortcutManager = {
  registerHandler: (
    commandId: ShortcutCommandId,
    handler: ShortcutHandler
  ) => () => void;
  executeCommand: (
    commandId: ShortcutCommandId,
    context?: { event?: KeyboardEvent }
  ) => void;
  handleKeyDown: (event: KeyboardEvent) => boolean;
};

function normalizeNonModifierToken(token: string): string {
  if (token.length === 1) {
    return token.toUpperCase();
  }
  if (token.toLowerCase() === "esc") {
    return "Escape";
  }
  if (token.toLowerCase() === "space") {
    return " ";
  }
  return token[0].toUpperCase() + token.slice(1).toLowerCase();
}

function normalizeCombo(combo: string): string {
  const parts = combo
    .split("+")
    .map((part) => part.trim())
    .filter(Boolean);

  if (parts.length === 0) {
    return "";
  }

  let hasCtrl = false;
  let hasMeta = false;
  let hasAlt = false;
  let hasShift = false;
  let key = "";

  for (const part of parts) {
    const lower = part.toLowerCase();
    if (lower === "ctrl" || lower === "control") {
      hasCtrl = true;
      continue;
    }
    if (lower === "cmd" || lower === "command" || lower === "meta") {
      hasMeta = true;
      continue;
    }
    if (lower === "cmdorctrl") {
      // Handled in expandComboVariants.
      continue;
    }
    if (lower === "alt" || lower === "option") {
      hasAlt = true;
      continue;
    }
    if (lower === "shift") {
      hasShift = true;
      continue;
    }
    key = normalizeNonModifierToken(part);
  }

  const prefix: string[] = [];
  if (hasCtrl) prefix.push("Ctrl");
  if (hasMeta) prefix.push("Meta");
  if (hasAlt) prefix.push("Alt");
  if (hasShift) prefix.push("Shift");
  if (key) prefix.push(key);
  return prefix.join("+");
}

function expandComboVariants(combo: string): string[] {
  const trimmed = combo.trim();
  if (!trimmed) return [];
  if (!/cmdorctrl/i.test(trimmed)) {
    return [normalizeCombo(trimmed)].filter(Boolean);
  }

  const ctrlCombo = normalizeCombo(trimmed.replace(/cmdorctrl/gi, "Ctrl"));
  const metaCombo = normalizeCombo(trimmed.replace(/cmdorctrl/gi, "Meta"));
  return Array.from(new Set([ctrlCombo, metaCombo].filter(Boolean)));
}

function comboFromEvent(event: KeyboardEvent): string {
  const prefix: string[] = [];
  if (event.ctrlKey) prefix.push("Ctrl");
  if (event.metaKey) prefix.push("Meta");
  if (event.altKey) prefix.push("Alt");
  if (event.shiftKey) prefix.push("Shift");
  prefix.push(normalizeNonModifierToken(event.key));
  return prefix.join("+");
}

export function createShortcutManager(shortcutMap: ShortcutMap): ShortcutManager {
  const comboToCommand = new Map<string, ShortcutCommandId>();
  const handlers = new Map<ShortcutCommandId, Set<ShortcutHandler>>();

  for (const [commandId, combos] of Object.entries(shortcutMap) as Array<
    [ShortcutCommandId, string[] | undefined]
  >) {
    if (!combos) continue;
    for (const combo of combos) {
      for (const variant of expandComboVariants(combo)) {
        comboToCommand.set(variant, commandId);
      }
    }
  }

  function executeCommand(
    commandId: ShortcutCommandId,
    context: { event?: KeyboardEvent } = {}
  ) {
    const commandHandlers = handlers.get(commandId);
    if (!commandHandlers || commandHandlers.size === 0) {
      return;
    }
    for (const handler of commandHandlers) {
      void handler({ commandId, event: context.event });
    }
  }

  return {
    registerHandler(commandId, handler) {
      const existing = handlers.get(commandId) ?? new Set<ShortcutHandler>();
      existing.add(handler);
      handlers.set(commandId, existing);
      return () => {
        const current = handlers.get(commandId);
        if (!current) return;
        current.delete(handler);
        if (current.size === 0) {
          handlers.delete(commandId);
        }
      };
    },
    executeCommand,
    handleKeyDown(event) {
      const combo = comboFromEvent(event);
      const commandId = comboToCommand.get(combo);
      if (!commandId) {
        return false;
      }
      executeCommand(commandId, { event });
      return true;
    },
  };
}
