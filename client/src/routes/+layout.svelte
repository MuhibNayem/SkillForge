<script lang="ts">
  import { onMount } from "svelte";
  import { initAuth } from "$lib/stores/auth";
  import "../app.css";
  let { children } = $props();

  onMount(() => {
    initAuth();
  });
</script>

{@render children()}
