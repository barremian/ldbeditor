<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { themePreference, type ThemePreference } from "$lib/theme";

  type ToolbarItem = {
    id: "general" | "appearance" | "donation";
    label: string;
    visible: boolean;
    isActive: boolean;
  };

  // Keep General and Donation entries in-place for easy reactivation later.
  const toolbarItems: ToolbarItem[] = [
    { id: "general", label: "General", visible: false, isActive: false },
    { id: "appearance", label: "Appearance", visible: true, isActive: true },
    { id: "donation", label: "Donation", visible: false, isActive: false },
  ];

  let preferredTheme: ThemePreference = "system";
  let unsubscribeThemePreference = () => {};

  onMount(() => {
    unsubscribeThemePreference = themePreference.subscribe((value) => {
      preferredTheme = value;
    });
  });

  onDestroy(() => {
    unsubscribeThemePreference();
  });
</script>

<main class="preferences-window">
  <section class="preferences-toolbar">
    {#each toolbarItems as item (item.id)}
      {#if item.visible}
        <button
          type="button"
          class="toolbar-item"
          class:is-active={item.isActive}
        >
          <span class="icon-wrap" aria-hidden="true">
            <svg viewBox="0 0 24 24" focusable="false">
              <path
                d="M8.5 4h7l1.6 2.2 2.6.7v3l-2.6.7L15.5 13h-7L6.9 10.6 4.3 9.9v-3l2.6-.7L8.5 4zM12 9.2a2.4 2.4 0 1 0 0 4.8 2.4 2.4 0 0 0 0-4.8zM7.2 14.4l1.8 1.2h6l1.8-1.2 1.2 1.8-1.8 1.2v2.1h-2.2l-.9-1.4h-2.2l-.9 1.4H7.8v-2.1L6 16.2l1.2-1.8z"
              />
            </svg>
          </span>
          <span>{item.label}</span>
        </button>
      {/if}
    {/each}
  </section>

  <section class="preferences-content">
    <div class="settings-row">
      <label for="theme-select">Theme</label>
      <select
        id="theme-select"
        bind:value={preferredTheme}
        on:change={() => themePreference.set(preferredTheme)}
      >
        <option value="system">System</option>
        <option value="light">Light</option>
        <option value="dark">Dark</option>
      </select>
    </div>

  </section>
</main>

<style>
  .preferences-window {
    min-height: 100vh;
    background: hsl(var(--muted) / 0.55);
    color: hsl(var(--foreground));
    font-size: 13px;
  }

  .preferences-toolbar {
    border-bottom: 1px solid hsl(var(--border));
    display: flex;
    justify-content: center;
    padding: 8px 24px 10px;
  }

  .toolbar-item {
    align-items: center;
    background: transparent;
    border: none;
    border-radius: 10px;
    color: hsl(var(--muted-foreground));
    cursor: default;
    display: flex;
    flex-direction: column;
    font-size: 13px;
    gap: 5px;
    min-width: 108px;
    padding: 8px 10px;
  }

  .toolbar-item:focus-visible {
    outline: 2px solid hsl(var(--ring));
    outline-offset: 1px;
  }

  .toolbar-item.is-active {
    background: hsl(var(--primary) / 0.14);
    color: hsl(var(--primary));
  }

  .icon-wrap {
    align-items: center;
    display: inline-flex;
    justify-content: center;
  }

  .icon-wrap svg {
    fill: currentColor;
    height: 22px;
    width: 22px;
  }

  .preferences-content {
    margin: 0 auto;
    max-width: 780px;
    padding: 26px 34px;
  }

  .settings-row {
    align-items: center;
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 24px;
    min-height: 44px;
  }

  .settings-row label {
    color: hsl(var(--foreground));
    font-size: 14px;
    line-height: 1.35;
  }

  .settings-row select {
    appearance: none;
    background:
      linear-gradient(45deg, transparent 50%, hsl(var(--muted-foreground)) 50%),
      linear-gradient(135deg, hsl(var(--muted-foreground)) 50%, transparent 50%),
      linear-gradient(to right, hsl(var(--background)), hsl(var(--background)));
    background-position:
      calc(100% - 14px) 50%,
      calc(100% - 9px) 50%,
      0 0;
    background-repeat: no-repeat;
    background-size:
      5px 5px,
      5px 5px,
      100% 100%;
    border: 1px solid hsl(var(--input));
    border-radius: 8px;
    color: hsl(var(--foreground));
    font-size: 13px;
    min-width: 180px;
    padding: 7px 28px 7px 10px;
    transition:
      border-color 120ms ease,
      box-shadow 120ms ease,
      background-color 120ms ease;
  }

  .settings-row select:focus-visible {
    border-color: hsl(var(--ring));
    box-shadow: 0 0 0 2px hsl(var(--ring) / 0.25);
    outline: none;
  }

</style>
