<template>
  <div class="grid flex-1 place-items-center p-6 text-center">
    <div class="max-w-sm">
      <div
        class="mx-auto mb-4 grid h-16 w-16 place-items-center rounded-2xl bg-slate-100 dark:bg-white/5"
      >
        <Bot class="h-8 w-8 text-slate-400" :stroke-width="1.75" aria-hidden="true" />
      </div>
      <h2 class="text-xl font-black">AI is disabled</h2>
      <p class="mt-2 text-sm text-slate-500 dark:text-slate-400">
        Create
        <code class="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-xs dark:bg-white/10">
          bs-ai-config.json
        </code>
        and
        <code class="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-xs dark:bg-white/10">
          bs-ai-models.json
        </code>
        next to the server, then restart the server.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Bot } from '@lucide/vue';
</script>
