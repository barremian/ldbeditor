import { writable } from "svelte/store";

export type ThemePreference = "light" | "dark" | "system";

const THEME_STORAGE_KEY = "ldbeditor-theme-preference";
const SYSTEM_DARK_QUERY = "(prefers-color-scheme: dark)";
const THEME_SYNC_CHANNEL = "ldbeditor-theme-sync";
const THEME_SYNC_MESSAGE_TYPE = "theme-preference-changed";

let removeSystemThemeListener: (() => void) | null = null;
let removeThemeSyncListeners: (() => void) | null = null;

const windowInstanceId =
  typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random()}`;

const isThemePreference = (value: string | null): value is ThemePreference =>
  value === "light" || value === "dark" || value === "system";

const getStoredThemePreference = (): ThemePreference => {
  if (typeof window === "undefined") return "system";
  try {
    const stored = window.localStorage.getItem(THEME_STORAGE_KEY);
    return isThemePreference(stored) ? stored : "system";
  } catch {
    return "system";
  }
};

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

type ThemeSyncMessage = {
  type: typeof THEME_SYNC_MESSAGE_TYPE;
  source: string;
  preference: ThemePreference;
};

const applyThemePreference = (preference: ThemePreference) => {
  themePreferenceStore.set(preference);
  applyThemeToDocument(preference);
  syncSystemThemeListener(preference);
};

const persistThemePreference = (preference: ThemePreference) => {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(THEME_STORAGE_KEY, preference);
  } catch {
    // Ignore localStorage write failures.
  }
};

const broadcastThemePreference = (preference: ThemePreference) => {
  if (typeof BroadcastChannel === "undefined") return;

  try {
    const channel = new BroadcastChannel(THEME_SYNC_CHANNEL);
    const message: ThemeSyncMessage = {
      type: THEME_SYNC_MESSAGE_TYPE,
      source: windowInstanceId,
      preference,
    };
    channel.postMessage(message);
    channel.close();
  } catch {
    // Ignore BroadcastChannel failures.
  }
};

const initializeThemeSync = () => {
  if (typeof window === "undefined" || removeThemeSyncListeners) return;

  const cleanupFunctions: Array<() => void> = [];
  const applyExternalPreference = (preference: ThemePreference) => {
    applyThemePreference(preference);
  };

  const onStorage = (event: StorageEvent) => {
    if (event.key !== THEME_STORAGE_KEY || !isThemePreference(event.newValue)) return;
    applyExternalPreference(event.newValue);
  };
  window.addEventListener("storage", onStorage);
  cleanupFunctions.push(() => window.removeEventListener("storage", onStorage));

  if (typeof BroadcastChannel !== "undefined") {
    try {
      const channel = new BroadcastChannel(THEME_SYNC_CHANNEL);
      const onMessage = (event: MessageEvent<ThemeSyncMessage>) => {
        const message = event.data;
        if (
          !message ||
          message.type !== THEME_SYNC_MESSAGE_TYPE ||
          message.source === windowInstanceId ||
          !isThemePreference(message.preference)
        ) {
          return;
        }

        applyExternalPreference(message.preference);
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

  removeThemeSyncListeners = () => {
    for (const cleanup of cleanupFunctions) {
      cleanup();
    }
    removeThemeSyncListeners = null;
  };
};

export const themePreference = {
  subscribe: themePreferenceStore.subscribe,
  set(preference: ThemePreference) {
    initializeThemeSync();
    applyThemePreference(preference);
    persistThemePreference(preference);
    broadcastThemePreference(preference);
  },
};

export const initializeTheme = () => {
  initializeThemeSync();
  const preference = getStoredThemePreference();
  applyThemePreference(preference);
};

