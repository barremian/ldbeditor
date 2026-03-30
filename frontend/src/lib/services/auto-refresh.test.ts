import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  createAutoRefreshController,
  getRemainingAutoRefreshMs,
  type AutoRefreshState,
} from "./auto-refresh";

describe("auto-refresh controller", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("preserves the configured interval when paused", () => {
    const onTick = vi.fn();
    const controller = createAutoRefreshController({ onTick });
    let state: AutoRefreshState | undefined;
    const unsubscribe = controller.subscribe((value) => {
      state = value;
    });

    controller.setIntervalMs(5000, "/tmp/db");
    expect(state?.intervalMs).toBe(5000);
    expect(state?.nextRefreshAt).not.toBeNull();

    controller.pause();
    expect(state?.intervalMs).toBe(5000);
    expect(state?.nextRefreshAt).toBeNull();

    vi.advanceTimersByTime(5000);
    expect(onTick).not.toHaveBeenCalled();

    unsubscribe();
    controller.destroy();
  });

  it("restarts the countdown when rescheduled after a pause", () => {
    const controller = createAutoRefreshController({ onTick: vi.fn() });
    let state: AutoRefreshState | undefined;
    const unsubscribe = controller.subscribe((value) => {
      state = value;
    });

    controller.setIntervalMs(5000, "/tmp/db");
    controller.pause();
    controller.schedule("/tmp/db");

    const initialRemainingMs = getRemainingAutoRefreshMs(state!);
    vi.advanceTimersByTime(1000);

    expect(getRemainingAutoRefreshMs(state!)).toBeLessThan(initialRemainingMs);

    unsubscribe();
    controller.destroy();
  });
});
