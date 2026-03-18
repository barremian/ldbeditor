import { get, writable } from "svelte/store";
import {
  getStorageItem,
  migrateStorageKey,
  setStorageItem,
} from "./storage";
import {
  WINDOW_INSTANCE_ID,
  attachPreferenceSync,
  postPreferenceBroadcast,
} from "./sync";

export type ValueFormatPrefsMap = Record<
  string,
  { prettyPrintJson: boolean }
>;

const LEGACY_KEY = "value-format-preferences";
const STORAGE_KEY = "ldbeditor-value-format-preferences";
const CHANNEL = "ldbeditor-value-format-sync";
const MSG_TYPE = "value-format-prefs-changed";

let removeSync: (() => void) | null = null;

function normalizePrefsRecord(
  parsed: Record<string, unknown> | null | undefined
): ValueFormatPrefsMap {
  return Object.entries(parsed ?? {}).reduce((acc, [path, preference]) => {
    if (!path || typeof preference !== "object" || preference === null) {
      return acc;
    }
    acc[path] = {
      prettyPrintJson: Boolean(
        (preference as { prettyPrintJson?: unknown }).prettyPrintJson
      ),
    };
    return acc;
  }, {} as ValueFormatPrefsMap);
}

function parsePrefsJson(raw: string | null): ValueFormatPrefsMap {
  if (!raw) return {};
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    return normalizePrefsRecord(parsed);
  } catch {
    return {};
  }
}

function loadFromStorage(): ValueFormatPrefsMap {
  return parsePrefsJson(getStorageItem(STORAGE_KEY));
}

const store = writable<ValueFormatPrefsMap>({});

const persist = (value: ValueFormatPrefsMap) => {
  try {
    setStorageItem(STORAGE_KEY, JSON.stringify(value));
  } catch {
    // ignore
  }
};

const apply = (value: ValueFormatPrefsMap) => {
  store.set(value);
};

const ensureSync = () => {
  attachPreferenceSync<ValueFormatPrefsMap>({
    getRemoveRef: () => removeSync,
    setRemoveRef: (fn) => {
      removeSync = fn;
    },
    storageKey: STORAGE_KEY,
    channelName: CHANNEL,
    messageType: MSG_TYPE,
    parseStorageValue: (raw) => parsePrefsJson(raw),
    parseMessagePayload: (data) => {
      const p = (data as { prefs?: unknown }).prefs;
      if (!p || typeof p !== "object" || p === null) return undefined;
      return normalizePrefsRecord(p as Record<string, unknown>);
    },
    onExternal: (v) => apply(v),
  });
};

export const valueFormatPreference = {
  subscribe: store.subscribe,
  get(): ValueFormatPrefsMap {
    return get(store);
  },
  setAll(prefs: ValueFormatPrefsMap) {
    ensureSync();
    apply(prefs);
    persist(prefs);
    postPreferenceBroadcast(CHANNEL, {
      type: MSG_TYPE,
      source: WINDOW_INSTANCE_ID,
      prefs,
    });
  },
  setPrettyPrintForPath(path: string, prettyPrintJson: boolean) {
    if (!path) return;
    const current = get(store);
    const next: ValueFormatPrefsMap = {
      ...current,
      [path]: { prettyPrintJson },
    };
    valueFormatPreference.setAll(next);
  },
  isPrettyPrintEnabled(path: string): boolean {
    return Boolean(get(store)[path]?.prettyPrintJson);
  },
};

export const initializeValueFormatPreference = () => {
  migrateStorageKey(LEGACY_KEY, STORAGE_KEY);
  ensureSync();
  apply(loadFromStorage());
};
