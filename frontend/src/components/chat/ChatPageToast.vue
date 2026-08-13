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
            : toast.kind === 'error'
              ? 'bg-red-600 text-white shadow-red-600/30'
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
          <CircleAlert
            v-else-if="toast.kind === 'error'"
            class="h-4 w-4"
            :stroke-width="2.25"
            aria-hidden="true"
          />
          <Check v-else class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
          {{ toastMessage }}
        </span>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Check, CircleAlert, GitBranch } from '@lucide/vue';

export type ChatToast = { kind: 'copy' | 'branch' | 'error'; id: number; message?: string };

const props = defineProps<{
  toast: ChatToast | null;
}>();

const toastMessage = computed(() => {
  if (!props.toast) return '';
  if (props.toast.kind === 'branch') return 'Branched into a new conversation';
  if (props.toast.kind === 'error') return props.toast.message || 'Something went wrong';
  return 'Copied to clipboard';
});
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
