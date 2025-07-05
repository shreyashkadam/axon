<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { getAllKeys } from '$lib/api.js';

  export let selectedNodePort;
  const dispatch = createEventDispatcher();

  let data = {};
  let error = '';
  let isLoading = false;

  const fetchAll = async () => {
    if (!selectedNodePort) {
      data = {};
      error = 'No online node selected.';
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

  function handleKeySelect(k, v) {
    dispatch('keyselect', { key: k, value: v });
  }
</script>

<div class="flex flex-col h-full">
  <div class="flex justify-between items-center mb-4">
    <h2 class="text-xl font-bold">All Stored Keys</h2>
    <button class="bg-slate-600 hover:bg-slate-700 text-white font-bold p-2 rounded-md" on:click={fetchAll} title="Refresh">
      Refresh
    </button>
  </div>
  {#if error}<p class="text-red-600 text-sm mb-2">{error}</p>{/if}
  <div class="flex-grow overflow-y-auto border border-slate-200 rounded-lg">
    <table class="min-w-full divide-y divide-slate-200">
      <thead class="bg-slate-50 sticky top-0">
        <tr>
          <th class="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase">Key</th>
          <th class="px-6 py-3 text-left text-xs font-medium text-slate-500 uppercase">Value</th>
        </tr>
      </thead>
      <tbody class="bg-white divide-y divide-slate-200">
        {#if Object.keys(data).length > 0}
          {#each Object.entries(data) as [k, v] (k)}
            <tr class="hover:bg-indigo-50 cursor-pointer" on:click={() => handleKeySelect(k, v)}>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-slate-900">{k}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-slate-500 truncate">{v}</td>
            </tr>
          {/each}
        {:else}
          <tr>
            <td colspan="2" class="text-center py-10 text-slate-500">
                {isLoading ? 'Loading...' : 'No data in the store.'}
            </td>
          </tr>
        {/if}
      </tbody>
    </table>
  </div>
</div>