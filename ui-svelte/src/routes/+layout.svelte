<script>
  import "../app.css";
  import Icon from "@iconify/svelte";
</script>

<div class="flex flex-col min-h-screen">
  <main class="flex-grow">
    <slot />
  </main>

  <footer class="text-center p-4 mt-8 text-slate-500 text-sm">
    <div class="mb-2">
      Made with ❤️ by Shreyash Kadam
    </div>
    <div class="flex justify-center items-center gap-4">
      <a href="https://www.linkedin.com/in/shreyash-kadam/" target="_blank" rel="noopener noreferrer" class="text-slate-500 hover:text-blue-700 transition-colors">
        <Icon icon="mdi:linkedin" class="w-6 h-6" />
      </a>
      <a href="https://github.com/shreyashkadam" target="_blank" rel="noopener noreferrer" class="text-slate-500 hover:text-slate-900 transition-colors">
        <Icon icon="mdi:github" class="w-6 h-6" />
      </a>
    </div>
  </footer>
</div>