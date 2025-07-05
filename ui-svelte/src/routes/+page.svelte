<script>
  import { onMount } from 'svelte';
  import { getManagedNodes } from '$lib/api.js';
  import ClusterDashboard from '$lib/components/ClusterDashboard.svelte';
  import KeyValueExplorer from '$lib/components/KeyValueExplorer.svelte';
  import AllKeysViewer from '$lib/components/AllKeysViewer.svelte';
  import AddNodeModal from '$lib/components/AddNodeModal.svelte';

  let nodes = [];
  let selectedNodePort = null;
  let isModalOpen = false;
  let explorerState = { key: '', value: '' };

  const fetchNodes = async () => {
    try {
      const res = await getManagedNodes();
      const managedNodes = Object.values(res.data).sort((a, b) => a.id.localeCompare(b.id));
      nodes = managedNodes;

      const onlineNodes = managedNodes.filter(n => n.status === 'online');

      if (selectedNodePort && !onlineNodes.some(n => n.api_port === selectedNodePort)) {
        selectedNodePort = onlineNodes.length > 0 ? onlineNodes[0].api_port : null;
      } else if (!selectedNodePort && onlineNodes.length > 0) {
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

  function handleKeySelect(event) {
    const { key, value } = event.detail;
    explorerState = { key, value };
  }
</script>

<div class="p-8">
  <header class="flex justify-between items-center mb-8">
    <h1 class="text-4xl font-bold">Distributed KV Store Dashboard</h1>
    <button
      on:click={() => (isModalOpen = true)}
      class="px-4 py-2 bg-indigo-600 text-white font-semibold rounded-lg shadow-md hover:bg-indigo-700"
    >
      + Add Node
    </button>
  </header>

  <AddNodeModal
    bind:isOpen={isModalOpen}
    onNodeAdded={fetchNodes}
    existingNodeIds={nodes.map(n => n.id)}
  />
  
  <main>
    <div class="mb-8">
      <ClusterDashboard {nodes} />
    </div>
    
    <div class="grid grid-cols-1 xl:grid-cols-2 gap-8">
        <div class="bg-white p-6 rounded-lg shadow-md">
            <KeyValueExplorer
                {nodes}
                bind:selectedNodePort
                bind:explorerState
            />
        </div>
        <div class="bg-white p-6 rounded-lg shadow-md">
            <AllKeysViewer
                {selectedNodePort}
                on:keyselect={handleKeySelect}
            />
        </div>
    </div>
  </main>
</div>