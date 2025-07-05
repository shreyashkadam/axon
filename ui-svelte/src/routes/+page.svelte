<script>
  import { onMount } from 'svelte';
  import { getManagedNodes } from '$lib/api.js';
  import ClusterDashboard from '$lib/components/ClusterDashboard.svelte';

  let nodes = [];

  const fetchNodes = async () => {
    try {
      const res = await getManagedNodes();
      nodes = Object.values(res.data).sort((a, b) => a.id.localeCompare(b.id));
    } catch (error) {
      console.error("Failed to fetch managed nodes:", error);
      nodes = [];
    }
  };

  onMount(() => {
    fetchNodes();
    const interval = setInterval(fetchNodes, 2000);
    return () => clearInterval(interval);
  });
</script>

<div class="p-8">
  <header class="text-center mb-8">
    <h1 class="text-4xl font-bold">Distributed KV Store Dashboard</h1>
  </header>

  <main>
    <ClusterDashboard {nodes} />
  </main>
</div>