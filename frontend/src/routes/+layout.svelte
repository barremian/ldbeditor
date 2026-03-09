<script lang="ts">
  import { onMount } from "svelte";
  import "../app.css";
  import { initializeTheme } from "$lib/theme";

  onMount(() => {
    initializeTheme();
    const platformHint = `${navigator.platform} ${navigator.userAgent}`.toLowerCase();
    const isDarwin = platformHint.includes("mac");
    document.body.classList.toggle("platform-darwin", isDarwin);
  });
</script>

<slot />
