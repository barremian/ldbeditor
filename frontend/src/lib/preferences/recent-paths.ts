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

export type RecentPathItem = { path: string; label: string };

const MAX_RECENT = 5;
const LEGACY_KEY = "recent-leveldb-paths";
const STORAGE_KEY = "ldbeditor-recent-leveldb-paths";
const CHANNEL = "ldbeditor-recent-paths-sync";
const MSG_TYPE = "recent-paths-changed";

let removeSync: (() => void) | null = null;

function parseRecentJson(raw: string | null): RecentPathItem[] {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed
      .filter(
        (x): x is RecentPathItem =>
          typeof x === "object" &&
          x !== null &&
          typeof (x as RecentPathItem).path === "string" &&
          typeof (x as RecentPathItem).label === "string"
      )
      .slice(0, MAX_RECENT);
  } catch {
    return [];
  }
}

const store = writable<RecentPathItem[]>([]);

const persist = (paths: RecentPathItem[]) => {
  const trimmed = paths.slice(0, MAX_RECENT);
  try {
    setStorageItem(STORAGE_KEY, JSON.stringify(trimmed));
  } catch {
    // ignore
  }
};

const apply = (paths: RecentPathItem[]) => {
  store.set(paths.slice(0, MAX_RECENT));
};

const ensureSync = () => {
  attachPreferenceSync<RecentPathItem[]>({
    getRemoveRef: () => removeSync,
    setRemoveRef: (fn) => {
      removeSync = fn;
    },
    storageKey: STORAGE_KEY,
    channelName: CHANNEL,
    messageType: MSG_TYPE,
    parseStorageValue: (raw) => parseRecentJson(raw),
    parseMessagePayload: (data) => {
      const p = (data as { paths?: unknown }).paths;
      if (!Array.isArray(p)) return undefined;
      return parseRecentJson(JSON.stringify(p));
    },
    onExternal: (v) => apply(v),
  });
};

export const recentPathsPreference = {
  subscribe: store.subscribe,
  get(): RecentPathItem[] {
    return get(store);
  },
  setPaths(paths: RecentPathItem[]) {
    ensureSync();
    const next = paths.slice(0, MAX_RECENT);
    apply(next);
    persist(next);
    postPreferenceBroadcast(CHANNEL, {
      type: MSG_TYPE,
      source: WINDOW_INSTANCE_ID,
      paths: next,
    });
  },
  add(path: string) {
    const label = path.split(/[/\\]/).pop() || path;
    const current = get(store).filter((p) => p.path !== path);
    recentPathsPreference.setPaths([{ path, label }, ...current]);
  },
  remove(path: string) {
    recentPathsPreference.setPaths(get(store).filter((p) => p.path !== path));
  },
};

export const initializeRecentPathsPreference = () => {
  migrateStorageKey(LEGACY_KEY, STORAGE_KEY);
  ensureSync();
  apply(parseRecentJson(getStorageItem(STORAGE_KEY)));
};

export { MAX_RECENT as MAX_RECENT_PATHS };
