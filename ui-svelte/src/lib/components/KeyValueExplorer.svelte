<script>
  import NodeSelector from './NodeSelector.svelte';
  import { getKey, putKey, deleteKey } from '$lib/api.js';
  import Icon from '@iconify/svelte';

  export let nodes;
  export let selectedNodePort;
  export let explorerState;

  let response = { text: '{ "status": "Awaiting operation..." }', type: 'info' };
  let isLoading = false;
  let loadingAction = '';

  const handleAction = async (actionFn, actionName) => {
    if (!selectedNodePort || isLoading) return;
    isLoading = true;
    loadingAction = actionName;
    response = { text: '{ "status": "Processing..." }', type: 'info' };
    try {
      const res = await actionFn();
      response = { text: JSON.stringify(res.data, null, 2), type: 'success' };
    } catch (error) {
      response = { text: JSON.stringify(error.response?.data || { error: error.message }, null, 2), type: 'error' };
    } finally {
      isLoading = false;
      loadingAction = '';
    }
  };
</script>

<div class="flex flex-col h-full">
  <h2 class="text-xl font-bold mb-4">Key-Value Explorer</h2>
  <NodeSelector {nodes} bind:selectedNodePort />
  <div class="space-y-4">
    <div>
      <label for="kv-key" class="block text-sm font-medium text-slate-700">Key</label>
      <input id="kv-key" type="text" bind:value={explorerState.key} class="mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50" />
    </div>
    <div>
      <label for="kv-value" class="block text-sm font-medium text-slate-700">Value</label>
      <textarea id="kv-value" bind:value={explorerState.value} rows="4" class="mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"></textarea>
    </div>
  </div>
  <div class="mt-4 grid grid-cols-1 sm:grid-cols-3 gap-3">
    <button on:click={() => handleAction(() => getKey(selectedNodePort, explorerState.key), 'GET')} disabled={isLoading || !explorerState.key} class="flex justify-center items-center gap-2 w-full bg-blue-500 hover:bg-blue-600 text-white font-bold py-2 px-4 rounded-md shadow-sm transition-colors disabled:bg-slate-300">
      {#if isLoading && loadingAction === 'GET'} <Icon icon="codicon:loading" class="animate-spin"/> {:else} <Icon icon="codicon:search" /> {/if} GET
    </button>
    <button on:click={() => handleAction(() => putKey(selectedNodePort, explorerState.key, explorerState.value), 'PUT')} disabled={isLoading || !explorerState.key} class="flex justify-center items-center gap-2 w-full bg-green-500 hover:bg-green-600 text-white font-bold py-2 px-4 rounded-md shadow-sm transition-colors disabled:bg-slate-300">
        {#if isLoading && loadingAction === 'PUT'} <Icon icon="codicon:loading" class="animate-spin"/> {:else} <Icon icon="codicon:save" /> {/if} PUT
    </button>
    <button on:click={() => handleAction(() => deleteKey(selectedNodePort, explorerState.key), 'DELETE')} disabled={isLoading || !explorerState.key} class="flex justify-center items-center gap-2 w-full bg-red-500 hover:bg-red-600 text-white font-bold py-2 px-4 rounded-md shadow-sm transition-colors disabled:bg-slate-300">
        {#if isLoading && loadingAction === 'DELETE'} <Icon icon="codicon:loading" class="animate-spin"/> {:else} <Icon icon="codicon:trash" /> {/if} DELETE
    </button>
  </div>
  <div class="mt-4 flex-grow flex flex-col">
      <span class="block text-sm font-medium text-slate-700">Response</span>
      <div class="mt-1 p-1 rounded-md text-sm overflow-auto h-40 bg-slate-50 border">
          <pre class="p-4 m-0 h-full bg-slate-50 text-sm whitespace-pre-wrap"><code>{response.text}</code></pre>
      </div>
  </div>
</div>