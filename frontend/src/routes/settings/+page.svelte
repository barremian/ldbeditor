<script lang="ts">
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

  let preferredTheme = "system";
  let compactDensity = false;
  let showStatusBadges = true;
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
      <select id="theme-select" bind:value={preferredTheme}>
        <option value="system">System</option>
        <option value="light">Light</option>
        <option value="dark">Dark</option>
      </select>
    </div>

    <div class="divider"></div>

    <div class="settings-row checkbox-row">
      <label for="compact-density">Use compact spacing in list views</label>
      <input
        id="compact-density"
        type="checkbox"
        bind:checked={compactDensity}
      />
    </div>

    <div class="divider"></div>

    <div class="settings-row checkbox-row">
      <label for="status-badges">Show status badges in tabs</label>
      <input
        id="status-badges"
        type="checkbox"
        bind:checked={showStatusBadges}
      />
    </div>
  </section>
</main>

<style>
  .preferences-window {
    min-height: 100vh;
    background: #f2f2f7;
    color: #1e1e1e;
    font-size: 13px;
  }

  .preferences-toolbar {
    border-bottom: 1px solid #d8d8de;
    display: flex;
    justify-content: center;
    padding: 8px 24px 10px;
  }

  .toolbar-item {
    align-items: center;
    background: transparent;
    border: none;
    border-radius: 10px;
    color: #4a4a4f;
    cursor: default;
    display: flex;
    flex-direction: column;
    font-size: 13px;
    gap: 5px;
    min-width: 108px;
    padding: 8px 10px;
  }

  .toolbar-item.is-active {
    background: #e8f1ff;
    color: #0a84ff;
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
    color: #222226;
    font-size: 14px;
    line-height: 1.35;
  }

  .settings-row select {
    appearance: none;
    background:
      linear-gradient(45deg, transparent 50%, #636369 50%),
      linear-gradient(135deg, #636369 50%, transparent 50%),
      linear-gradient(to right, #fdfdff, #fdfdff);
    background-position:
      calc(100% - 14px) 50%,
      calc(100% - 9px) 50%,
      0 0;
    background-repeat: no-repeat;
    background-size:
      5px 5px,
      5px 5px,
      100% 100%;
    border: 1px solid #c8c8ce;
    border-radius: 8px;
    color: #222226;
    font-size: 13px;
    min-width: 180px;
    padding: 7px 28px 7px 10px;
  }

  .divider {
    background: #d6d6dc;
    height: 1px;
    width: 100%;
  }

  .checkbox-row input[type="checkbox"] {
    accent-color: #0a84ff;
    height: 16px;
    margin: 0;
    width: 16px;
  }
</style>
