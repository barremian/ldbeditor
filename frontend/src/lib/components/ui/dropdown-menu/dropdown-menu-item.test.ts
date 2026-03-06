import { cleanup, fireEvent, render } from "@testing-library/svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import Host from "./dropdown-menu-item.test-host.svelte";

afterEach(() => {
  cleanup();
});

async function openActionsMenuAndGetItem(
  getByRole: (role: string, options?: Record<string, unknown>) => HTMLElement,
  findByRole: (
    role: string,
    options?: Record<string, unknown>
  ) => Promise<HTMLElement>,
  role: "menuitem" | "menuitemradio" = "menuitem"
) {
  await fireEvent.click(getByRole("button", { name: "Open actions" }));

  return findByRole(role, {
    name: "Open in read-only mode",
  });
}

describe("DropdownMenuItem wrapper", () => {
  it("forwards click events to consumers", async () => {
    const onItemClick = vi.fn();

    const { getByRole, findByRole } = render(Host, {
      onItemClick,
    });

    const item = await openActionsMenuAndGetItem(getByRole, findByRole);

    await fireEvent.click(item);

    expect(onItemClick).toHaveBeenCalledTimes(1);
  });

  it("activates via Enter key", async () => {
    const onItemClick = vi.fn();

    const { getByRole, findByRole } = render(Host, {
      onItemClick,
    });

    const item = await openActionsMenuAndGetItem(getByRole, findByRole);

    item.focus();
    await fireEvent.keyDown(item, { key: "Enter" });

    expect(onItemClick).toHaveBeenCalledTimes(1);
  });

  it("activates via Space key", async () => {
    const onItemClick = vi.fn();

    const { getByRole, findByRole } = render(Host, {
      onItemClick,
    });

    const item = await openActionsMenuAndGetItem(getByRole, findByRole);

    item.focus();
    await fireEvent.keyDown(item, { key: " " });

    expect(onItemClick).toHaveBeenCalledTimes(1);
  });

  it("renders as menuitemradio with checked state", async () => {
    const { getByRole, findByRole } = render(Host, {
      onItemClick: vi.fn(),
      radio: true,
      checked: true,
    });

    const item = await openActionsMenuAndGetItem(
      getByRole,
      findByRole,
      "menuitemradio"
    );

    expect(item).toHaveAttribute("aria-checked", "true");
    expect(item.querySelector("svg")).not.toBeNull();
  });

  it("renders radio item without indicator when disabled by prop", async () => {
    const { getByRole, findByRole } = render(Host, {
      onItemClick: vi.fn(),
      radio: true,
      checked: true,
      showRadioIndicator: false,
    });

    const item = await openActionsMenuAndGetItem(
      getByRole,
      findByRole,
      "menuitemradio"
    );

    expect(item).toHaveAttribute("aria-checked", "true");
    expect(item.querySelector("svg")).toBeNull();
  });
});
