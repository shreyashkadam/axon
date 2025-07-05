<script>
  import { onMount } from 'svelte';
  import { getManagedNodes } from '$lib/api.js';
  import ClusterDashboard from '$lib/components/ClusterDashboard.svelte';
  import KeyValueExplorer from '$lib/components/KeyValueExplorer.svelte';
  import AllKeysViewer from '$lib/components/AllKeysViewer.svelte';
  import AddNodeModal from '$lib/components/AddNodeModal.svelte';
  import Icon from "@iconify/svelte";

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

<div class="min-h-screen bg-slate-50 p-4 sm:p-6 lg:p-8">
  <div class="max-w-7xl mx-auto">
    <header class="flex flex-col sm:flex-row justify-between items-center pb-6 border-b border-slate-200">
      <div class="flex items-center gap-3">
        <Icon icon="codicon:server-process" class="w-8 h-8 text-indigo-600" />
        <h1 class="text-2xl sm:text-3xl font-bold text-slate-900">
          Distributed KV Store Dashboard
        </h1>
      </div>
      <button
        on:click={() => (isModalOpen = true)}
        class="mt-4 sm:mt-0 px-4 py-2 bg-indigo-600 text-white font-semibold rounded-lg shadow-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-400 focus:ring-opacity-75 transition-all duration-200"
      >
        + Add Node
      </button>
    </header>

    <AddNodeModal
      bind:isOpen={isModalOpen}
      onNodeAdded={fetchNodes}
      existingNodeIds={nodes.map(n => n.id)}
    />
    
    <main class="mt-8">
      <ClusterDashboard {nodes} />
      
      <div class="mt-8 grid grid-cols-1 xl:grid-cols-2 gap-8">
        <div class="bg-white p-6 rounded-xl shadow-lg border border-slate-200">
          <KeyValueExplorer 
            {nodes} 
            bind:selectedNodePort
            bind:explorerState
          />
        </div>
        <div class="bg-white p-6 rounded-xl shadow-lg border border-slate-200">
          <AllKeysViewer 
            {selectedNodePort} 
            on:keyselect={handleKeySelect}
          />
        </div>
      </div>
    </main>
  </div>
</div>