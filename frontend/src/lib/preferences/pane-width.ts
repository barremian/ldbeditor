import { get, writable } from "svelte/store";
import {
  getStorageItem,
  migrateStorageKey,
  parseStoredFiniteNumber,
  setStorageItem,
} from "./storage";
import {
  WINDOW_INSTANCE_ID,
  attachPreferenceSync,
  postPreferenceBroadcast,
} from "./sync";

const LEGACY_KEY = "editor-key-pane-width";
const STORAGE_KEY = "ldbeditor-editor-key-pane-width";
const DEFAULT_WIDTH = 320;
const CHANNEL = "ldbeditor-pane-width-sync";
const MSG_TYPE = "key-pane-width-changed";

let removeSync: (() => void) | null = null;

const store = writable<number>(DEFAULT_WIDTH);

const persist = (w: number) => {
  setStorageItem(STORAGE_KEY, String(Math.round(w)));
};

const apply = (w: number) => {
  store.set(w);
};

const ensureSync = () => {
  attachPreferenceSync<number>({
    getRemoveRef: () => removeSync,
    setRemoveRef: (fn) => {
      removeSync = fn;
    },
    storageKey: STORAGE_KEY,
    channelName: CHANNEL,
    messageType: MSG_TYPE,
    parseStorageValue: (raw) => {
      const n = parseStoredFiniteNumber(raw);
      return n !== null ? n : undefined;
    },
    parseMessagePayload: (data) => {
      const w = (data as { width?: unknown }).width;
      return typeof w === "number" && Number.isFinite(w) ? w : undefined;
    },
    onExternal: (v) => apply(v),
  });
};

export const keyPaneWidthPreference = {
  subscribe: store.subscribe,
  get(): number {
    return get(store);
  },
  set(width: number) {
    ensureSync();
    apply(width);
    persist(width);
    postPreferenceBroadcast(CHANNEL, {
      type: MSG_TYPE,
      source: WINDOW_INSTANCE_ID,
      width,
    });
  },
  get defaultWidth() {
    return DEFAULT_WIDTH;
  },
};

export const initializeKeyPaneWidthPreference = () => {
  migrateStorageKey(LEGACY_KEY, STORAGE_KEY);
  ensureSync();
  const raw = getStorageItem(STORAGE_KEY);
  const n = parseStoredFiniteNumber(raw);
  apply(n !== null ? n : DEFAULT_WIDTH);
};
