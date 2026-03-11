<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { ChevronDown } from "lucide-svelte";
  import { themePreference, type ThemePreference } from "$lib/theme";
  import { Button } from "$lib/components/ui/button";
  import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuTrigger,
    DropdownMenuItem,
  } from "$lib/components/ui/dropdown-menu";

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

  const themeOptions: { value: ThemePreference; label: string }[] = [
    { value: "system", label: "System" },
    { value: "light", label: "Light" },
    { value: "dark", label: "Dark" },
  ];

  let preferredTheme: ThemePreference = "system";
  $: themeLabel = themeOptions.find((o) => o.value === preferredTheme)?.label ?? "System";
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
      <span class="settings-label">Theme</span>
      <DropdownMenu>
        <DropdownMenuTrigger asChild let:builder>
          <Button
            builders={[builder]}
            variant="outline"
            class="min-w-[180px] justify-between font-normal"
          >
            {themeLabel}
            <ChevronDown class="ml-2 h-3.5 w-3.5 opacity-60" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" class="min-w-[180px]">
          {#each themeOptions as option (option.value)}
            <DropdownMenuItem
              radio
              checked={preferredTheme === option.value}
              on:click={() => themePreference.set(option.value)}
            >
              {option.label}
            </DropdownMenuItem>
          {/each}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>

  </section>
</main>

<style>
  .preferences-window {
    min-height: 100vh;
    background: linear-gradient(
      180deg,
      hsl(var(--background)) 0%,
      hsl(var(--muted) / 0.42) 100%
    );
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
    transition:
      background-color 120ms ease,
      color 120ms ease,
      border-color 120ms ease;
  }

  .toolbar-item:focus-visible {
    outline: 2px solid hsl(var(--ring));
    outline-offset: 1px;
  }

  .toolbar-item.is-active {
    background: hsl(var(--accent) / 0.72);
    border: 1px solid hsl(var(--border) / 0.9);
    color: hsl(var(--foreground));
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

  .settings-label {
    color: hsl(var(--foreground));
    font-size: 14px;
    line-height: 1.35;
  }

</style>
