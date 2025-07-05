<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { getAllKeys } from '$lib/api.js';
  import Icon from '@iconify/svelte';

  export let selectedNodePort;
  const dispatch = createEventDispatcher();

  let data = {};
  let error = '';
  let isLoading = false;
  let searchTerm = '';

  const fetchAll = async () => {
    if (!selectedNodePort) {
      data = {};
      error = 'No online node selected to query.';
      return;
    }
    isLoading = true;
    error = '';
    try {
      const res = await getAllKeys(selectedNodePort);
      data = res.data || {};
    } catch (err) {
      error = `Error: ${err.response?.data?.error || err.message}`;
      data = {};
    } finally {
      isLoading = false;
    }
  };

  onMount(fetchAll);
  
  $: if (selectedNodePort) {
      fetchAll();
  }

  $: filteredData = Object.entries(data).filter(([k]) => 
    k.toLowerCase().includes(searchTerm.toLowerCase())
  );

  function handleKeySelect(k, v) {
    dispatch('keyselect', { key: k, value: v });
  }
</script>

<div class="flex flex-col h-full">
  <h2 class="text-xl font-bold mb-4">All Stored Keys</h2>
  <div class="flex gap-2 mb-4">
      <input
        type="text"
        placeholder="Search keys..."
        bind:value={searchTerm}
        class="block w-full rounded-md border-slate-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"
      />
    <button class="flex-shrink-0 bg-slate-600 hover:bg-slate-700 text-white font-bold p-2 rounded-md shadow-sm transition-colors" on:click={fetchAll} title="Refresh All Keys">
      <Icon icon="codicon:sync" class={isLoading ? 'animate-spin' : ''} />
    </button>
  </div>
  {#if error}<p class="text-red-600 text-sm mb-2">{error}</p>{/if}
  <div class="flex-grow overflow-y-auto border border-slate-200 rounded-lg">
    <table class="min-w-full divide-y divide-slate-200">
      <thead class="bg-slate-50 sticky top-0">
        <tr>
          <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Key</th>
          <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">Value</th>
        </tr>
      </thead>
      <tbody class="bg-white divide-y divide-slate-200">
        {#if filteredData.length > 0}
          {#each filteredData as [k, v] (k)}
            <tr class="hover:bg-indigo-50 cursor-pointer" on:click={() => handleKeySelect(k, v)}>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-slate-900">{k}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-slate-500 truncate">{v}</td>
            </tr>
          {/each}
        {:else}
          <tr>
              <td colspan="2" class="text-center py-16 text-slate-500">
                  <div class="flex flex-col items-center gap-2">
                      <Icon icon="codicon:package" class="w-8 h-8 text-slate-400" />
                      <span>{isLoading ? 'Loading...' : 'No data in the store.'}</span>
                  </div>
              </td>
          </tr>
        {/if}
      </tbody>
    </table>
  </div>
</div>