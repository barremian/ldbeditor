export const WINDOW_INSTANCE_ID =
  typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random()}`;

export function attachPreferenceSync<T>(params: {
  getRemoveRef: () => (() => void) | null;
  setRemoveRef: (fn: (() => void) | null) => void;
  storageKey: string;
  channelName: string;
  messageType: string;
  parseStorageValue: (raw: string | null) => T | undefined;
  parseMessagePayload: (data: unknown) => T | undefined;
  onExternal: (value: T) => void;
}): void {
  if (typeof window === "undefined" || params.getRemoveRef()) return;

  const cleanups: Array<() => void> = [];

  const onStorage = (event: StorageEvent) => {
    if (event.key !== params.storageKey) return;
    const parsed = params.parseStorageValue(event.newValue);
    if (parsed !== undefined) params.onExternal(parsed);
  };
  window.addEventListener("storage", onStorage);
  cleanups.push(() => window.removeEventListener("storage", onStorage));

  if (typeof BroadcastChannel !== "undefined") {
    try {
      const channel = new BroadcastChannel(params.channelName);
      const onMessage = (event: MessageEvent) => {
        const data = event.data;
        if (
          !data ||
          typeof data !== "object" ||
          (data as { type?: string }).type !== params.messageType ||
          (data as { source?: string }).source === WINDOW_INSTANCE_ID
        ) {
          return;
        }
        const v = params.parseMessagePayload(data);
        if (v !== undefined) params.onExternal(v);
      };
      channel.addEventListener("message", onMessage);
      cleanups.push(() => {
        channel.removeEventListener("message", onMessage);
        channel.close();
      });
    } catch {
      // Ignore BroadcastChannel failures.
    }
  }

  params.setRemoveRef(() => {
    for (const c of cleanups) c();
    params.setRemoveRef(null);
  });
}

export function postPreferenceBroadcast(
  channelName: string,
  message: Record<string, unknown>
): void {
  if (typeof BroadcastChannel === "undefined") return;
  try {
    const channel = new BroadcastChannel(channelName);
    channel.postMessage(message);
    channel.close();
  } catch {
    // Ignore.
  }
}
