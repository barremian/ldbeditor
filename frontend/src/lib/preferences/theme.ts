/**
 * Theme preference. Keep storage key and valid values in sync with `app.html` inline script.
 */
import { writable } from "svelte/store";
import { getStorageItem, setStorageItem } from "./storage";
import {
  WINDOW_INSTANCE_ID,
  attachPreferenceSync,
  postPreferenceBroadcast,
} from "./sync";

export type ThemePreference = "light" | "dark" | "system";

export const THEME_STORAGE_KEY = "ldbeditor-theme-preference";

const SYSTEM_DARK_QUERY = "(prefers-color-scheme: dark)";
const THEME_SYNC_CHANNEL = "ldbeditor-theme-sync";
const THEME_SYNC_MESSAGE_TYPE = "theme-preference-changed";

let removeSystemThemeListener: (() => void) | null = null;
let removeThemeSyncListeners: (() => void) | null = null;

const isThemePreference = (value: string | null): value is ThemePreference =>
  value === "light" || value === "dark" || value === "system";

const getSystemTheme = (): "light" | "dark" => {
  if (typeof window === "undefined" || !window.matchMedia) return "light";
  return window.matchMedia(SYSTEM_DARK_QUERY).matches ? "dark" : "light";
};

const resolveTheme = (preference: ThemePreference): "light" | "dark" =>
  preference === "system" ? getSystemTheme() : preference;

const applyThemeToDocument = (preference: ThemePreference) => {
  if (typeof document === "undefined") return;
  const root = document.documentElement;
  const effectiveTheme = resolveTheme(preference);

  root.classList.toggle("dark", effectiveTheme === "dark");
  root.dataset.theme = effectiveTheme;
  root.dataset.themePreference = preference;
  root.style.colorScheme = effectiveTheme;
};

const syncSystemThemeListener = (preference: ThemePreference) => {
  if (typeof window === "undefined" || !window.matchMedia) return;

  if (removeSystemThemeListener) {
    removeSystemThemeListener();
    removeSystemThemeListener = null;
  }

  if (preference !== "system") return;

  const mediaQuery = window.matchMedia(SYSTEM_DARK_QUERY);
  const handleSystemThemeChange = () => applyThemeToDocument("system");
  mediaQuery.addEventListener("change", handleSystemThemeChange);
  removeSystemThemeListener = () =>
    mediaQuery.removeEventListener("change", handleSystemThemeChange);
};

const themePreferenceStore = writable<ThemePreference>("system");

const applyThemePreference = (preference: ThemePreference) => {
  themePreferenceStore.set(preference);
  applyThemeToDocument(preference);
  syncSystemThemeListener(preference);
};

const ensureThemeSync = () => {
  attachPreferenceSync<ThemePreference>({
    getRemoveRef: () => removeThemeSyncListeners,
    setRemoveRef: (fn) => {
      removeThemeSyncListeners = fn;
    },
    storageKey: THEME_STORAGE_KEY,
    channelName: THEME_SYNC_CHANNEL,
    messageType: THEME_SYNC_MESSAGE_TYPE,
    parseStorageValue: (raw) =>
      isThemePreference(raw) ? raw : undefined,
    parseMessagePayload: (data) => {
      const p = (data as { preference?: unknown }).preference;
      return typeof p === "string" && isThemePreference(p) ? p : undefined;
    },
    onExternal: (v) => applyThemePreference(v),
  });
};

export const themePreference = {
  subscribe: themePreferenceStore.subscribe,
  set(preference: ThemePreference) {
    ensureThemeSync();
    applyThemePreference(preference);
    setStorageItem(THEME_STORAGE_KEY, preference);
    postPreferenceBroadcast(THEME_SYNC_CHANNEL, {
      type: THEME_SYNC_MESSAGE_TYPE,
      source: WINDOW_INSTANCE_ID,
      preference,
    });
  },
};

export const initializeTheme = () => {
  ensureThemeSync();
  const stored = getStorageItem(THEME_STORAGE_KEY);
  const preference = isThemePreference(stored) ? stored : "system";
  applyThemePreference(preference);
};
