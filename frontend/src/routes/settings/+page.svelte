<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { ChevronDown, HandHeart, Palette, Settings2 } from "lucide-svelte";
  import { themePreference, type ThemePreference } from "$lib/theme";
  import {
    confirmCloseLastTabPreference,
    initializeConfirmCloseLastTabPreference,
  } from "$lib/preferences";
  import { Button } from "$lib/components/ui/button";
  import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuTrigger,
    DropdownMenuItem,
  } from "$lib/components/ui/dropdown-menu";

  type NavItem = {
    id: "general" | "appearance" | "donation";
    label: string;
    visible: boolean;
    icon: typeof Palette;
  };

  // Keep General and Donation entries in-place for easy reactivation later.
  const navItems: NavItem[] = [
    { id: "general", label: "General", visible: true, icon: Settings2 },
    { id: "appearance", label: "Appearance", visible: true, icon: Palette },
    { id: "donation", label: "Donation", visible: false, icon: HandHeart },
  ];

  let activeSettingId: NavItem["id"] = "general";

  const themeOptions: { value: ThemePreference; label: string }[] = [
    { value: "system", label: "System" },
    { value: "light", label: "Light" },
    { value: "dark", label: "Dark" },
  ];

  let preferredTheme: ThemePreference = "system";
  $: themeLabel = themeOptions.find((o) => o.value === preferredTheme)?.label ?? "System";
  let unsubscribeThemePreference = () => {};
  let confirmCloseLastTab = true;
  let unsubscribeConfirmCloseLastTab = () => {};

  onMount(() => {
    initializeConfirmCloseLastTabPreference();
    unsubscribeThemePreference = themePreference.subscribe((value) => {
      preferredTheme = value;
    });
    unsubscribeConfirmCloseLastTab = confirmCloseLastTabPreference.subscribe((value) => {
      confirmCloseLastTab = value;
    });
  });

  onDestroy(() => {
    unsubscribeThemePreference();
    unsubscribeConfirmCloseLastTab();
  });
</script>

<div class="flex h-full overflow-hidden text-sm text-foreground">
  <div class="m-3 mr-0 flex w-60 shrink-0 flex-col">
    <div class="mb-2 px-3 py-2">
      <p class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">SETTINGS</p>
      <h1 class="mt-1 text-base font-semibold text-foreground">LevelDB Editor</h1>
    </div>

    <!-- Left nav panel -->
    <nav
      class="flex-1 overflow-y-auto rounded-md border border-border/70 p-2"
      aria-label="Settings navigation"
    >
      {#each navItems as item (item.id)}
        {#if item.visible}
          <button
            type="button"
            class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring {activeSettingId ===
            item.id
              ? 'bg-primary/15 text-primary'
              : 'text-foreground hover:bg-muted'}"
            on:click={() => (activeSettingId = item.id)}
          >
            <svelte:component this={item.icon} class="h-4 w-4 shrink-0" aria-hidden="true" />
            <span>{item.label}</span>
          </button>
        {/if}
      {/each}
    </nav>
  </div>

  <!-- Right content area -->
  <div class="flex-1 overflow-y-auto p-6">
    {#if activeSettingId === "general"}
      <h2 class="mb-4 text-lg font-semibold text-foreground">General</h2>

      <p class="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
        Window
      </p>
      <div class="rounded-lg border border-border/70 p-3">
        <div class="divide-y divide-border/70">
          <div class="flex min-h-[44px] items-center justify-between gap-4 px-1 py-2">
            <span class="text-sm text-foreground">Confirm when closing last tab</span>
            <button
              type="button"
              role="switch"
              aria-checked={confirmCloseLastTab}
              class="shrink-0 rounded-md outline-none ring-offset-background focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              on:click={() =>
                confirmCloseLastTabPreference.set(!confirmCloseLastTab)}
            >
              <span
                class={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border transition-colors ${
                  confirmCloseLastTab
                    ? "border-primary/40 bg-primary"
                    : "border-border bg-muted"
                }`}
                aria-hidden="true"
              >
                <span
                  class={`inline-block h-4 w-4 rounded-full bg-background shadow-sm transition-transform ${
                    confirmCloseLastTab ? "translate-x-4" : "translate-x-0.5"
                  }`}
                ></span>
              </span>
            </button>
          </div>
        </div>
      </div>
    {/if}

    {#if activeSettingId === "appearance"}
      <h2 class="mb-4 text-lg font-semibold text-foreground">Appearance</h2>

      <p class="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
        Display
      </p>
      <div class="rounded-lg border border-border/70 p-3">
        <div class="divide-y divide-border/70">
          <div class="flex min-h-[44px] items-center justify-between gap-4 px-1 py-2">
            <span class="text-sm text-foreground">Theme</span>
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
        </div>
      </div>
    {/if}
  </div>
</div>
