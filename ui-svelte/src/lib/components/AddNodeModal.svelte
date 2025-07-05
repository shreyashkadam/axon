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
</script>

{#if isOpen}
  <div class="fixed inset-0 bg-black bg-opacity-50 z-50 flex justify-center items-center" on:click={() => isOpen = false}>
    <div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-md" on:click|stopPropagation>
      <h2 class="text-xl font-bold mb-4">Add New Node</h2>
      <div class="space-y-2">
        <label for="node-name" class="block text-sm font-medium text-slate-700">Node Name</label>
        <input
          id="node-name"
          type="text"
          bind:value={nodeId}
          placeholder="e.g., node4"
          class="block w-full px-3 py-2 border border-slate-300 rounded-md"
        />
        {#if error}<p class="text-sm text-red-600 mt-1">{error}</p>{/if}
      </div>
      <div class="mt-6 flex justify-end gap-3">
        <button
          class="px-4 py-2 bg-white border border-slate-300 rounded-md text-sm font-medium hover:bg-slate-50"
          on:click={() => isOpen = false}
          disabled={isAdding}
        >
          Cancel
        </button>
        <button
          class="px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700 disabled:bg-indigo-300"
          on:click={handleAdd}
          disabled={!!error || isAdding}
        >
          {isAdding ? 'Adding...' : 'Add Node'}
        </button>
      </div>
    </div>
  </div>
{/if}