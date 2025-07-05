<script>
    import { startNode, stopNode, deleteNode } from '$lib/api.js';
    import Icon from '@iconify/svelte';

    export let nodes = [];
    let loadingStates = {};

    const handleAction = async (actionFn, nodeId) => {
        loadingStates = { ...loadingStates, [nodeId]: true };
        try {
            await actionFn(nodeId);
        } catch (error) {
            console.error(`Action failed for node ${nodeId}:`, error);
        } finally {
            // Add a small delay so the user can see the loading state
            setTimeout(() => {
                loadingStates = { ...loadingStates, [nodeId]: false };
            }, 1000);
        }
    };

    const handleDelete = (nodeId) => {
        if (window.confirm(`Are you sure you want to permanently delete ${nodeId}?`)) {
            handleAction(deleteNode, nodeId);
        }
    };
</script>

<div class="flex flex-wrap justify-center gap-6">
    {#each nodes as node (node.id)}
        {@const isOnline = node.status === 'online'}
        {@const isLoading = loadingStates[node.id]}
        <div class="bg-white rounded-lg shadow-md p-5 w-64">
            <div class="flex justify-between items-center mb-2">
                <h3 class="text-lg font-bold">{node.id}</h3>
                <button
                    on:click={() => handleDelete(node.id)}
                    class="text-slate-400 hover:text-red-500"
                    title={`Decommission ${node.id}`}
                >
                    <Icon icon="codicon:trash" />
                </button>
            </div>
            <p class="text-sm text-slate-500 mb-4">API Port: {node.api_port}</p>

            <div class="flex items-center gap-2">
                <span class={`w-3 h-3 rounded-full ${isOnline ? 'bg-green-500' : 'bg-red-500'}`}></span>
                {isOnline ? 'Online' : 'Offline'}
                {#if isOnline && node.is_leader}
                    <span class="ml-2 px-2 py-0.5 text-xs font-semibold text-white bg-amber-500 rounded-full">LEADER</span>
                {/if}
            </div>

            <div class="mt-4 flex justify-center gap-2">
                <button on:click={() => handleAction(stopNode, node.id)} disabled={!isOnline || isLoading} class="px-4 py-2 text-xs font-semibold text-white bg-slate-600 rounded-md hover:bg-slate-700 disabled:bg-slate-300">
                    Stop
                </button>
                <button on:click={() => handleAction(startNode, node.id)} disabled={isOnline || isLoading} class="px-4 py-2 text-xs font-semibold text-white bg-indigo-600 rounded-md hover:bg-indigo-700 disabled:bg-slate-300">
                    Start
                </button>
            </div>
        </div>
    {/each}
</div>