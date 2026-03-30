import { writable } from "svelte/store";

export const AUTO_REFRESH_OPTIONS = [
  { label: "Off", intervalMs: 0 },
  { label: "5s", intervalMs: 5000 },
  { label: "15s", intervalMs: 15000 },
  { label: "30s", intervalMs: 30000 },
  { label: "60s", intervalMs: 60000 },
] as const;

export type AutoRefreshState = {
  intervalMs: number;
  nextRefreshAt: number | null;
  countdownNow: number;
};

type AutoRefreshOptions = {
  onTick: () => void | Promise<void>;
  countdownTickMs?: number;
};

const DEFAULT_COUNTDOWN_TICK_MS = 200;

export function createAutoRefreshController(options: AutoRefreshOptions) {
  const countdownTickMs =
    options.countdownTickMs ?? DEFAULT_COUNTDOWN_TICK_MS;

  const store = writable<AutoRefreshState>({
    intervalMs: 0,
    nextRefreshAt: null,
    countdownNow: Date.now(),
  });

  let state: AutoRefreshState = {
    intervalMs: 0,
    nextRefreshAt: null,
    countdownNow: Date.now(),
  };
  let autoRefreshTimeout: ReturnType<typeof setTimeout> | null = null;
  let refreshCountdownInterval: ReturnType<typeof setInterval> | null = null;

  const setState = (updater: (current: AutoRefreshState) => AutoRefreshState) => {
    state = updater(state);
    store.set(state);
  };

  const clearTimer = () => {
    if (!autoRefreshTimeout) return;
    clearTimeout(autoRefreshTimeout);
    autoRefreshTimeout = null;
  };

  const clearCountdownTicker = () => {
    if (!refreshCountdownInterval) return;
    clearInterval(refreshCountdownInterval);
    refreshCountdownInterval = null;
  };

  const startCountdownTicker = (dbPath: string) => {
    clearCountdownTicker();
    if (!dbPath || state.intervalMs <= 0) return;

    setState((current) => ({
      ...current,
      countdownNow: Date.now(),
    }));

    refreshCountdownInterval = setInterval(() => {
      setState((current) => ({
        ...current,
        countdownNow: Date.now(),
      }));
    }, countdownTickMs);
  };

  const schedule = (dbPath: string) => {
    clearTimer();
    clearCountdownTicker();
    if (!dbPath || state.intervalMs <= 0) {
      setState((current) => ({ ...current, nextRefreshAt: null }));
      return;
    }

    const scheduledAt = Date.now();
    setState((current) => ({
      ...current,
      nextRefreshAt: scheduledAt + current.intervalMs,
      countdownNow: scheduledAt,
    }));

    startCountdownTicker(dbPath);
    autoRefreshTimeout = setTimeout(() => {
      void options.onTick();
    }, state.intervalMs);
  };

  const setIntervalMs = (intervalMs: number, dbPath: string) => {
    setState((current) => ({
      ...current,
      intervalMs,
    }));

    if (intervalMs <= 0 || !dbPath) {
      clearTimer();
      clearCountdownTicker();
      setState((current) => ({ ...current, nextRefreshAt: null }));
      return;
    }

    schedule(dbPath);
  };

  const pause = () => {
    clearTimer();
    clearCountdownTicker();
    setState((current) => ({
      ...current,
      nextRefreshAt: null,
      countdownNow: Date.now(),
    }));
  };

  const stop = () => {
    clearTimer();
    clearCountdownTicker();
    setState((current) => ({
      ...current,
      intervalMs: 0,
      nextRefreshAt: null,
    }));
  };

  const destroy = () => {
    clearTimer();
    clearCountdownTicker();
  };

  return {
    subscribe: store.subscribe,
    getState: () => state,
    setIntervalMs,
    schedule,
    pause,
    stop,
    destroy,
  };
}

export function getAutoRefreshLabel(intervalMs: number): string {
  return (
    AUTO_REFRESH_OPTIONS.find((option) => option.intervalMs === intervalMs)
      ?.label ?? "Custom"
  );
}

export function getRemainingAutoRefreshMs(state: AutoRefreshState): number {
  if (state.intervalMs <= 0 || !state.nextRefreshAt) {
    return 0;
  }

  return Math.max(0, state.nextRefreshAt - state.countdownNow);
}

export function getAutoRefreshProgress(state: AutoRefreshState): number {
  if (state.intervalMs <= 0) return 1;
  const remainingMs = getRemainingAutoRefreshMs(state);
  return Math.max(0, Math.min(1, remainingMs / state.intervalMs));
}
