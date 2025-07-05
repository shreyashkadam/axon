<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->

<script>
  import { addNewNode } from '$lib/api.js';
  
  export let isOpen = false;
  export let onNodeAdded = () => {};
  export let existingNodeIds = [];

  let nodeId = '';
  let error = '';
  let isAdding = false;

  $: {
    if (isOpen) {
      if (nodeId.trim() === '') {
        error = 'Node name cannot be empty.';
      } else if (existingNodeIds.includes(nodeId.trim())) {
        error = 'Node name already exists.';
      } else if (!/^[a-zA-Z0-9_-]+$/.test(nodeId.trim())) {
        error = 'Name can only contain letters, numbers, hyphens, and underscores.';
      } else {
        error = '';
      }
    }
  }

  const handleAdd = async () => {
    if (error || isAdding) return;
    isAdding = true;
    try {
      await addNewNode(nodeId.trim());
      onNodeAdded();
      nodeId = '';
      isOpen = false;
    } catch (err) {
      error = err.response?.data?.error || 'Failed to add node.';
    } finally {
      isAdding = false;
    }
  };

  const handleClose = () => {
    if (!isAdding) {
        isOpen = false;
    }
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 bg-black bg-opacity-50 z-50 flex justify-center items-center" on:click|self={handleClose}>
    <div class="bg-white rounded-lg shadow-2xl p-6 w-full max-w-md" on:click|stopPropagation>
      <h2 class="text-xl font-bold mb-4">Add New Node</h2>
      <div class="space-y-2">
        <label for="node-name" class="block text-sm font-medium text-slate-700">Node Name</label>
        <input
          id="node-name"
          type="text"
          bind:value={nodeId}
          placeholder="e.g., node4"
          class="block w-full px-3 py-2 bg-white border border-slate-300 rounded-md text-sm shadow-sm placeholder-slate-400
                 focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
        />
        {#if error}<p class="text-sm text-red-600 mt-1">{error}</p>{/if}
      </div>
      <div class="mt-6 flex justify-end gap-3">
        <button 
          class="px-4 py-2 bg-white border border-slate-300 rounded-md text-sm font-medium text-slate-700 hover:bg-slate-50 focus:outline-none"
          on:click={handleClose} 
          disabled={isAdding}
        >
          Cancel
        </button>
        <button 
          class="px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:bg-indigo-300 disabled:cursor-not-allowed"
          on:click={handleAdd} 
          disabled={!!error || isAdding}
        >
          {isAdding ? 'Adding...' : 'Add Node'}
        </button>
      </div>
    </div>
  </div>
{/if}