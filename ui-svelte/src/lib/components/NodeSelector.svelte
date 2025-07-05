<script>
  export let nodes;
  export let selectedNodePort;

  $: onlineNodes = nodes.filter(n => n.status === 'online');
</script>

<div class="mb-4">
  <label for="node-selector" class="block text-sm font-medium text-slate-700 mb-1">Target Node</label>
  <select
    id="node-selector"
    bind:value={selectedNodePort}
    class="block w-full pl-3 pr-10 py-2 text-base border-slate-300 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md shadow-sm"
  >
    {#if onlineNodes.length > 0}
      {#each onlineNodes as node (node.id)}
        <option value={node.api_port}>
          {node.id} ({node.is_leader ? 'Leader' : 'Follower'}) - Port {node.api_port}
        </option>
      {/each}
    {:else}
      <option value={null}>No online nodes available</option>
    {/if}
  </select>
</div>