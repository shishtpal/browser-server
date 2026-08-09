<template>
  <Teleport to="body">
    <Transition name="chat-toast">
      <div
        v-if="toast"
        :key="toast.id"
        class="pointer-events-none fixed bottom-6 left-1/2 z-50 -translate-x-1/2 rounded-lg px-4 py-2.5 text-sm font-medium shadow-lg"
        :class="
          toast.kind === 'branch'
            ? 'bg-indigo-600 text-white shadow-indigo-600/30'
            : 'bg-slate-900 text-white dark:bg-white dark:text-slate-900'
        "
        role="status"
      >
        <span class="inline-flex items-center gap-2">
          <GitBranch
            v-if="toast.kind === 'branch'"
            class="h-4 w-4"
            :stroke-width="2.25"
            aria-hidden="true"
          />
          <Check v-else class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
          {{ toast.kind === 'branch' ? 'Branched into a new conversation' : 'Copied to clipboard' }}
        </span>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { Check, GitBranch } from '@lucide/vue';

defineProps<{
  toast: { kind: 'copy' | 'branch'; id: number } | null;
}>();
</script>

<style scoped>
.chat-toast-enter-active,
.chat-toast-leave-active {
  transition: all 0.25s ease;
}
.chat-toast-enter-from,
.chat-toast-leave-to {
  opacity: 0;
  transform: translate(-50%, 10px);
}
</style>
