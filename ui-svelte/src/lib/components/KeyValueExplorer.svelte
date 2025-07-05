<script>
  import NodeSelector from './NodeSelector.svelte';
  import { getKey, putKey, deleteKey } from '$lib/api.js';
  import Icon from '@iconify/svelte';

  export let nodes;
  export let selectedNodePort;
  export let explorerState;

  let response = { text: '{ "status": "Awaiting operation..." }', type: 'info' };
  let isLoading = false;

  const handleAction = async (actionFn) => {
    if (!selectedNodePort || isLoading) return;
    isLoading = true;
    response = { text: '{ "status": "Processing..." }', type: 'info' };
    try {
      const res = await actionFn();
      response = { text: JSON.stringify(res.data, null, 2), type: 'success' };
    } catch (error) {
      response = { text: JSON.stringify(error.response?.data || { error: error.message }, null, 2), type: 'error' };
    } finally {
      isLoading = false;
    }
  };
</script>

<div class="flex flex-col h-full">
  <h2 class="text-xl font-bold mb-4">Key-Value Explorer</h2>
  <NodeSelector {nodes} bind:selectedNodePort />
  <div class="space-y-4">
    <div>
      <label for="kv-key" class="block text-sm font-medium text-slate-700">Key</label>
      <input id="kv-key" type="text" bind:value={explorerState.key} class="mt-1 block w-full rounded-md border-slate-300 shadow-sm" />
    </div>
    <div>
      <label for="kv-value" class="block text-sm font-medium text-slate-700">Value</label>
      <textarea id="kv-value" bind:value={explorerState.value} rows="4" class="mt-1 block w-full rounded-md border-slate-300 shadow-sm"></textarea>
    </div>
  </div>
  <div class="mt-4 grid grid-cols-1 sm:grid-cols-3 gap-3">
    <button on:click={() => handleAction(() => getKey(selectedNodePort, explorerState.key))} disabled={isLoading || !explorerState.key} class="bg-blue-500 hover:bg-blue-600 text-white font-bold py-2 px-4 rounded-md disabled:bg-slate-300">
      GET
    </button>
    <button on:click={() => handleAction(() => putKey(selectedNodePort, explorerState.key, explorerState.value))} disabled={isLoading || !explorerState.key} class="bg-green-500 hover:bg-green-600 text-white font-bold py-2 px-4 rounded-md disabled:bg-slate-300">
      PUT
    </button>
    <button on:click={() => handleAction(() => deleteKey(selectedNodePort, explorerState.key))} disabled={isLoading || !explorerState.key} class="bg-red-500 hover:bg-red-600 text-white font-bold py-2 px-4 rounded-md disabled:bg-slate-300">
      DELETE
    </button>
  </div>
  <div class="mt-4 flex-grow">
    <span class="block text-sm font-medium text-slate-700">Response</span>
    <pre class="mt-1 p-4 rounded-md text-sm bg-slate-100 overflow-auto h-40"><code>{response.text}</code></pre>
  </div>
</div>