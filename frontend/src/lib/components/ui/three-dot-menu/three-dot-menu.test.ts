import { cleanup, fireEvent, render } from "@testing-library/svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import Host from "./three-dot-menu.test-host.svelte";

afterEach(() => {
  cleanup();
});

async function openMenuAndGetItem(
  getByRole: (role: string, options?: Record<string, unknown>) => HTMLElement,
  findByRole: (
    role: string,
    options?: Record<string, unknown>
  ) => Promise<HTMLElement>
) {
  await fireEvent.click(getByRole("button", { name: "Open options" }));

  return findByRole("menuitem", {
    name: "Open in read-only mode",
  });
}

describe("ThreeDotMenu", () => {
  it("renders trigger with accessible label", () => {
    const { getByRole } = render(Host, {
      onItemClick: vi.fn(),
    });

    expect(getByRole("button", { name: "Open options" })).toBeInTheDocument();
  });

  it("opens menu and renders slotted options", async () => {
    const { getByRole, findByRole } = render(Host, {
      onItemClick: vi.fn(),
    });

    const item = await openMenuAndGetItem(getByRole, findByRole);

    expect(item).toBeInTheDocument();
  });

  it("invokes option handler on click", async () => {
    const onItemClick = vi.fn();
    const { getByRole, findByRole } = render(Host, {
      onItemClick,
    });

    const item = await openMenuAndGetItem(getByRole, findByRole);
    await fireEvent.click(item);

    expect(onItemClick).toHaveBeenCalledTimes(1);
  });

  it("keeps menu open when item prevents default selection", async () => {
    const onItemClick = vi.fn();
    const { getByRole, findByRole } = render(Host, {
      onItemClick,
      persistOnSelect: true,
    });

    const item = await openMenuAndGetItem(getByRole, findByRole);
    await fireEvent.click(item);

    expect(onItemClick).toHaveBeenCalledTimes(1);
    expect(
      await findByRole("menuitem", { name: "Open in read-only mode" })
    ).toBeInTheDocument();
  });

  it("supports keyboard activation with Enter and Space", async () => {
    const onItemClick = vi.fn();
    const { getByRole, findByRole } = render(Host, {
      onItemClick,
    });

    const item = await openMenuAndGetItem(getByRole, findByRole);
    item.focus();
    await fireEvent.keyDown(item, { key: "Enter" });
    await fireEvent.keyDown(item, { key: " " });

    expect(onItemClick).toHaveBeenCalledTimes(2);
  });

  it("honors disabled state on trigger", async () => {
    const onItemClick = vi.fn();
    const { getByRole } = render(Host, {
      onItemClick,
      disabled: true,
    });

    const trigger = getByRole("button", { name: "Open options" });
    expect(trigger).toBeDisabled();

    await fireEvent.click(trigger);
    expect(onItemClick).not.toHaveBeenCalled();
  });
});
