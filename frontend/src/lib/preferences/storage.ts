/** All persisted preference keys use this prefix (see also app.html inline theme script). */
export const STORAGE_KEY_PREFIX = "ldbeditor-";

export function storageKey(shortName: string): string {
  const name = shortName.startsWith(STORAGE_KEY_PREFIX)
    ? shortName
    : `${STORAGE_KEY_PREFIX}${shortName}`;
  return name;
}

export function getStorageItem(key: string): string | null {
  if (typeof window === "undefined") return null;
  try {
    return window.localStorage.getItem(key);
  } catch {
    return null;
  }
}

export function setStorageItem(key: string, value: string): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(key, value);
  } catch {
    // Ignore localStorage write failures.
  }
}

export function removeStorageItem(key: string): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.removeItem(key);
  } catch {
    // Ignore.
  }
}

/** Copy value from oldKey to newKey if newKey is empty, then remove oldKey. */
export function migrateStorageKey(oldKey: string, newKey: string): void {
  if (oldKey === newKey) return;
  const existingNew = getStorageItem(newKey);
  if (existingNew !== null && existingNew !== "") {
    removeStorageItem(oldKey);
    return;
  }
  const oldVal = getStorageItem(oldKey);
  if (oldVal === null) return;
  setStorageItem(newKey, oldVal);
  removeStorageItem(oldKey);
}

export function parseStoredBoolean(value: string | null): boolean | null {
  if (value === "true") return true;
  if (value === "false") return false;
  return null;
}

export function parseStoredFiniteNumber(value: string | null): number | null {
  if (value === null || value === "") return null;
  const n = Number(value);
  return Number.isFinite(n) ? n : null;
}
