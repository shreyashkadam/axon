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
            setTimeout(() => {
                loadingStates = { ...loadingStates, [nodeId]: false };
            }, 2000);
        }
    };

    const handleDelete = (nodeId) => {
        if (window.confirm(`Are you sure you want to permanently delete ${nodeId}? This action cannot be undone.`)) {
            handleAction(deleteNode, nodeId);
        }
    };
</script>

<div class="flex flex-wrap justify-center gap-6">
    {#each nodes as node (node.id)}
        {@const isOnline = node.status === 'online'}
        {@const isLoading = loadingStates[node.id]}
        {@const statusBorderColor = isOnline ? 'border-green-500' : 'border-red-500'}
        {@const cardClasses = `bg-white rounded-xl shadow-lg p-5 w-full sm:w-64 transform transition-all duration-300 ease-in-out hover:scale-105 hover:shadow-2xl border-t-4 ${statusBorderColor} ${!isOnline && 'opacity-70'}`}
        {@const leaderGradient = 'bg-gradient-to-br from-amber-400 to-orange-500'}
        {@const followerGradient = 'bg-gradient-to-br from-sky-400 to-blue-500'}
        {@const roleBadgeClasses = `px-3 py-1 text-xs font-bold text-white rounded-full shadow-md ${node.is_leader ? leaderGradient : followerGradient}`}
        
        <div class={cardClasses}>
            <div class="flex justify-between items-center mb-2">
                <h3 class="text-lg font-bold text-slate-800">{node.id}</h3>
                <button
                    on:click={() => handleDelete(node.id)}
                    class="text-slate-400 hover:text-red-500 transition-colors"
                    title={`Decommission ${node.id}`}
                >
                    <Icon icon="codicon:trash" />
                </button>
            </div>
            <p class="text-xs text-slate-500 mb-4">API Port: {node.api_port}</p>

            <div class={`flex items-center justify-center gap-2 font-semibold ${isOnline ? 'text-green-600' : 'text-slate-500'}`}>
                <span class={`w-3 h-3 rounded-full ${isOnline ? 'bg-green-500' : 'bg-red-500'}`}></span>
                {node.status.charAt(0).toUpperCase() + node.status.slice(1)}
            </div>

            {#if isOnline}
                <div class="mt-3 text-center">
                    <span class={roleBadgeClasses}>
                        {node.is_leader ? 'LEADER' : 'Follower'}
                    </span>
                </div>
            {/if}

            <div class="mt-5 flex justify-center gap-3">
                <button on:click={() => handleAction(stopNode, node.id)} disabled={!isOnline || isLoading} class="flex items-center justify-center w-24 px-3 py-2 text-xs font-semibold text-white bg-slate-700 rounded-md shadow-sm hover:bg-slate-800 disabled:bg-slate-300 transition-colors">
                    {#if isLoading}
                        <Icon icon="codicon:loading" class="animate-spin" />
                    {:else}
                        <Icon icon="codicon:stop-circle" class="mr-2" /> Stop
                    {/if}
                </button>
                <button on:click={() => handleAction(startNode, node.id)} disabled={isOnline || isLoading} class="flex items-center justify-center w-24 px-3 py-2 text-xs font-semibold text-white bg-indigo-600 rounded-md shadow-sm hover:bg-indigo-700 disabled:bg-slate-300 transition-colors">
                    {#if isLoading}
                        <Icon icon="codicon:loading" class="animate-spin" />
                    {:else}
                        <Icon icon="codicon:debug-start" class="mr-2" /> Start
                    {/if}
                </button>
            </div>
        </div>
    {/each}
</div>