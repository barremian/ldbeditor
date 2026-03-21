import { get, writable } from "svelte/store";
import {
  getStorageItem,
  parseStoredFiniteNumber,
  setStorageItem,
} from "./storage";
import {
  WINDOW_INSTANCE_ID,
  attachPreferenceSync,
  postPreferenceBroadcast,
} from "./sync";

export const EDITOR_FONT_SIZE_STORAGE_KEY = "ldbeditor-editor-font-size";

export const EDITOR_FONT_SIZE_VALUES = [12, 13, 14, 15, 16, 18] as const;
export type EditorFontSizeValue = (typeof EDITOR_FONT_SIZE_VALUES)[number];

const DEFAULT_EDITOR_FONT_SIZE: EditorFontSizeValue = 14;

const PREFERENCES_SYNC_CHANNEL = "ldbeditor-preferences-sync";
const EDITOR_FONT_SIZE_MESSAGE_TYPE = "editor-font-size-changed";

let removePreferenceSyncListeners: (() => void) | null = null;

function isEditorFontSizeValue(n: number): n is EditorFontSizeValue {
  return (EDITOR_FONT_SIZE_VALUES as readonly number[]).includes(n);
}

export const EDITOR_FONT_SIZE_OPTIONS: { value: EditorFontSizeValue; label: string }[] =
  [
    { value: 12, label: "12 px" },
    { value: 13, label: "13 px" },
    { value: 14, label: "14 px (Default)" },
    { value: 15, label: "15 px" },
    { value: 16, label: "16 px" },
    { value: 18, label: "18 px" },
  ];

const getStoredEditorFontSize = (): EditorFontSizeValue => {
  const stored = getStorageItem(EDITOR_FONT_SIZE_STORAGE_KEY);
  const parsed = parseStoredFiniteNumber(stored);
  if (parsed === null) return DEFAULT_EDITOR_FONT_SIZE;
  return isEditorFontSizeValue(parsed) ? parsed : DEFAULT_EDITOR_FONT_SIZE;
};

const editorFontSizeStore = writable<EditorFontSizeValue>(
  DEFAULT_EDITOR_FONT_SIZE
);

const applyEditorFontSize = (sizePx: EditorFontSizeValue) => {
  editorFontSizeStore.set(sizePx);
  if (typeof document === "undefined") return;
  document.documentElement.style.setProperty(
    "--editor-font-size",
    `${sizePx / 16}rem`
  );
};

const ensureEditorFontSizeSync = () => {
  attachPreferenceSync<EditorFontSizeValue>({
    getRemoveRef: () => removePreferenceSyncListeners,
    setRemoveRef: (fn) => {
      removePreferenceSyncListeners = fn;
    },
    storageKey: EDITOR_FONT_SIZE_STORAGE_KEY,
    channelName: PREFERENCES_SYNC_CHANNEL,
    messageType: EDITOR_FONT_SIZE_MESSAGE_TYPE,
    parseStorageValue: (raw) => {
      const p = parseStoredFiniteNumber(raw);
      if (p === null) return undefined;
      return isEditorFontSizeValue(p) ? p : undefined;
    },
    parseMessagePayload: (data) => {
      const v = (data as { value?: unknown }).value;
      return typeof v === "number" && isEditorFontSizeValue(v) ? v : undefined;
    },
    onExternal: (v) => applyEditorFontSize(v),
  });
};

export const editorFontSizePreference = {
  subscribe: editorFontSizeStore.subscribe,
  set(value: EditorFontSizeValue) {
    ensureEditorFontSizeSync();
    applyEditorFontSize(value);
    setStorageItem(EDITOR_FONT_SIZE_STORAGE_KEY, String(value));
    postPreferenceBroadcast(PREFERENCES_SYNC_CHANNEL, {
      type: EDITOR_FONT_SIZE_MESSAGE_TYPE,
      source: WINDOW_INSTANCE_ID,
      value,
    });
  },
  get(): EditorFontSizeValue {
    return get(editorFontSizeStore);
  },
};

export const initializeEditorFontSizePreference = () => {
  ensureEditorFontSizeSync();
  applyEditorFontSize(getStoredEditorFontSize());
};
