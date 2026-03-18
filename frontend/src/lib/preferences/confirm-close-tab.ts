import { get, writable } from "svelte/store";
import { getStorageItem, parseStoredBoolean, setStorageItem } from "./storage";
import {
  WINDOW_INSTANCE_ID,
  attachPreferenceSync,
  postPreferenceBroadcast,
} from "./sync";

const CONFIRM_CLOSE_LAST_TAB_STORAGE_KEY = "ldbeditor-confirm-close-last-tab";
const PREFERENCES_SYNC_CHANNEL = "ldbeditor-preferences-sync";
const CONFIRM_CLOSE_LAST_TAB_MESSAGE_TYPE = "confirm-close-last-tab-changed";

let removePreferenceSyncListeners: (() => void) | null = null;

const getStoredConfirmCloseLastTab = (): boolean => {
  const stored = getStorageItem(CONFIRM_CLOSE_LAST_TAB_STORAGE_KEY);
  const parsed = parseStoredBoolean(stored);
  return parsed ?? true;
};

const confirmCloseLastTabStore = writable<boolean>(true);

const applyConfirmCloseLastTab = (value: boolean) => {
  confirmCloseLastTabStore.set(value);
};

const ensureConfirmCloseSync = () => {
  attachPreferenceSync<boolean>({
    getRemoveRef: () => removePreferenceSyncListeners,
    setRemoveRef: (fn) => {
      removePreferenceSyncListeners = fn;
    },
    storageKey: CONFIRM_CLOSE_LAST_TAB_STORAGE_KEY,
    channelName: PREFERENCES_SYNC_CHANNEL,
    messageType: CONFIRM_CLOSE_LAST_TAB_MESSAGE_TYPE,
    parseStorageValue: (raw) => {
      const p = parseStoredBoolean(raw);
      return p !== null ? p : undefined;
    },
    parseMessagePayload: (data) => {
      const v = (data as { value?: unknown }).value;
      return typeof v === "boolean" ? v : undefined;
    },
    onExternal: (v) => applyConfirmCloseLastTab(v),
  });
};

export const confirmCloseLastTabPreference = {
  subscribe: confirmCloseLastTabStore.subscribe,
  set(value: boolean) {
    ensureConfirmCloseSync();
    applyConfirmCloseLastTab(value);
    setStorageItem(CONFIRM_CLOSE_LAST_TAB_STORAGE_KEY, String(value));
    postPreferenceBroadcast(PREFERENCES_SYNC_CHANNEL, {
      type: CONFIRM_CLOSE_LAST_TAB_MESSAGE_TYPE,
      source: WINDOW_INSTANCE_ID,
      value,
    });
  },
  get(): boolean {
    return get(confirmCloseLastTabStore);
  },
};

export const initializeConfirmCloseLastTabPreference = () => {
  ensureConfirmCloseSync();
  applyConfirmCloseLastTab(getStoredConfirmCloseLastTab());
};
