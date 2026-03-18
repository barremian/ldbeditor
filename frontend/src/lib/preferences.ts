import { get, writable } from "svelte/store";

const CONFIRM_CLOSE_LAST_TAB_STORAGE_KEY = "ldbeditor-confirm-close-last-tab";
const PREFERENCES_SYNC_CHANNEL = "ldbeditor-preferences-sync";
const CONFIRM_CLOSE_LAST_TAB_MESSAGE_TYPE = "confirm-close-last-tab-changed";

let removePreferenceSyncListeners: (() => void) | null = null;

const windowInstanceId =
  typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random()}`;

const parseStoredBoolean = (value: string | null): boolean | null => {
  if (value === "true") return true;
  if (value === "false") return false;
  return null;
};

const getStoredConfirmCloseLastTab = (): boolean => {
  if (typeof window === "undefined") return true;
  try {
    const stored = window.localStorage.getItem(CONFIRM_CLOSE_LAST_TAB_STORAGE_KEY);
    const parsed = parseStoredBoolean(stored);
    return parsed ?? true;
  } catch {
    return true;
  }
};

const persistConfirmCloseLastTab = (value: boolean) => {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(CONFIRM_CLOSE_LAST_TAB_STORAGE_KEY, String(value));
  } catch {
    // Ignore localStorage write failures.
  }
};

type ConfirmCloseLastTabMessage = {
  type: typeof CONFIRM_CLOSE_LAST_TAB_MESSAGE_TYPE;
  source: string;
  value: boolean;
};

const broadcastConfirmCloseLastTab = (value: boolean) => {
  if (typeof BroadcastChannel === "undefined") return;

  try {
    const channel = new BroadcastChannel(PREFERENCES_SYNC_CHANNEL);
    const message: ConfirmCloseLastTabMessage = {
      type: CONFIRM_CLOSE_LAST_TAB_MESSAGE_TYPE,
      source: windowInstanceId,
      value,
    };
    channel.postMessage(message);
    channel.close();
  } catch {
    // Ignore BroadcastChannel failures.
  }
};

const confirmCloseLastTabStore = writable<boolean>(true);

const applyConfirmCloseLastTab = (value: boolean) => {
  confirmCloseLastTabStore.set(value);
};

const initializePreferenceSync = () => {
  if (typeof window === "undefined" || removePreferenceSyncListeners) return;

  const cleanupFunctions: Array<() => void> = [];

  const onStorage = (event: StorageEvent) => {
    if (event.key !== CONFIRM_CLOSE_LAST_TAB_STORAGE_KEY) return;
    const parsed = parseStoredBoolean(event.newValue);
    if (parsed !== null) {
      applyConfirmCloseLastTab(parsed);
    }
  };
  window.addEventListener("storage", onStorage);
  cleanupFunctions.push(() => window.removeEventListener("storage", onStorage));

  if (typeof BroadcastChannel !== "undefined") {
    try {
      const channel = new BroadcastChannel(PREFERENCES_SYNC_CHANNEL);
      const onMessage = (event: MessageEvent<ConfirmCloseLastTabMessage>) => {
        const message = event.data;
        if (
          !message ||
          message.type !== CONFIRM_CLOSE_LAST_TAB_MESSAGE_TYPE ||
          message.source === windowInstanceId ||
          typeof message.value !== "boolean"
        ) {
          return;
        }
        applyConfirmCloseLastTab(message.value);
      };
      channel.addEventListener("message", onMessage);
      cleanupFunctions.push(() => {
        channel.removeEventListener("message", onMessage);
        channel.close();
      });
    } catch {
      // Ignore BroadcastChannel listener failures.
    }
  }

  removePreferenceSyncListeners = () => {
    for (const cleanup of cleanupFunctions) {
      cleanup();
    }
    removePreferenceSyncListeners = null;
  };
};

export const confirmCloseLastTabPreference = {
  subscribe: confirmCloseLastTabStore.subscribe,
  set(value: boolean) {
    initializePreferenceSync();
    applyConfirmCloseLastTab(value);
    persistConfirmCloseLastTab(value);
    broadcastConfirmCloseLastTab(value);
  },
  /** Read current value synchronously (e.g. inside async handlers). */
  get(): boolean {
    return get(confirmCloseLastTabStore);
  },
};

export const initializeConfirmCloseLastTabPreference = () => {
  initializePreferenceSync();
  applyConfirmCloseLastTab(getStoredConfirmCloseLastTab());
};
