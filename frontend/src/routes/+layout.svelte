<script lang="ts">
  import { onMount } from "svelte";
  import "../app.css";
  import { initializeAllPreferences } from "$lib/preferences";

  onMount(() => {
    initializeAllPreferences();
    const platformHint = `${navigator.platform} ${navigator.userAgent}`.toLowerCase();
    const isDarwin = platformHint.includes("mac");
    document.body.classList.toggle("platform-darwin", isDarwin);
  });
</script>

<slot />
