<script>
  import { onMount } from 'svelte';
  import { getManagedNodes } from '$lib/api.js';
  import ClusterDashboard from '$lib/components/ClusterDashboard.svelte';
  import KeyValueExplorer from '$lib/components/KeyValueExplorer.svelte';

  let nodes = [];
  let selectedNodePort = null;
  let explorerState = { key: '', value: '' };

  const fetchNodes = async () => {
    try {
      const res = await getManagedNodes();
      const managedNodes = Object.values(res.data).sort((a, b) => a.id.localeCompare(b.id));
      nodes = managedNodes;

      const onlineNodes = managedNodes.filter(n => n.status === 'online');

      if (!selectedNodePort && onlineNodes.length > 0) {
        selectedNodePort = onlineNodes[0].api_port;
      } else if (onlineNodes.length === 0) {
        selectedNodePort = null;
      }
    } catch (error) {
      console.error("Failed to fetch managed nodes:", error);
      nodes = [];
      selectedNodePort = null;
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
    <div class="mb-8">
      <ClusterDashboard {nodes} />
    </div>
    
    <div class="bg-white p-6 rounded-lg shadow-md">
        <KeyValueExplorer
            {nodes}
            bind:selectedNodePort
            bind:explorerState
        />
    </div>
  </main>
</div>