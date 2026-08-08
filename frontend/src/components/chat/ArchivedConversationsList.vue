<template>
  <section class="mt-4 border-t border-slate-200 pt-3 dark:border-white/10">
    <button
      class="flex w-full items-center gap-2 px-2 py-1.5 text-xs font-semibold text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-slate-400 dark:hover:bg-white/5 dark:hover:text-white"
      type="button"
      @click="$emit('toggle')"
    >
      <svg
        class="h-3.5 w-3.5 transition-transform"
        :class="{ 'rotate-90': open }"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
      </svg>
      Archived
      <span
        v-if="items.length > 0"
        class="ml-auto rounded-full bg-slate-200 px-1.5 py-0.5 text-[10px] font-bold text-slate-600 dark:bg-white/10 dark:text-slate-300"
        >{{ items.length }}</span
      >
    </button>

    <div v-if="open" class="mt-1 max-h-100 space-y-1 overflow-y-auto pr-1">
      <div
        v-if="items.length === 0"
        class="px-2 py-3 text-center text-xs text-slate-400 dark:text-slate-500"
      >
        No archived conversations
      </div>
      <div
        v-for="conversation in items"
        :key="'archived-' + conversation.id"
        class="group relative cursor-pointer rounded-lg border border-slate-200 bg-slate-50/50 p-3 transition hover:bg-slate-100 dark:border-white/5 dark:bg-white/5 dark:hover:bg-white/10"
        @click="$emit('select', conversation.id)"
      >
        <span
          class="block truncate text-[0.82rem] font-semibold text-slate-400 dark:text-slate-500"
        >
          {{ conversation.title }}
        </span>
        <span
          class="mt-0.5 block truncate font-mono text-[0.66rem] text-slate-400 dark:text-slate-500"
        >
          {{ conversation.model }} · {{ formatRelativeTime(conversation.updated_at) }}
        </span>
        <div class="absolute top-2 right-2 hidden gap-1 group-hover:flex">
          <button
            class="rounded p-1 text-slate-400 hover:bg-green-100 hover:text-green-600 dark:hover:bg-green-900/30 dark:hover:text-green-400"
            title="Restore"
            type="button"
            @click.stop="$emit('restore', conversation)"
          >
            <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
              />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { AIConversation } from '@browser-server/shared-types';

defineProps<{
  items: AIConversation[];
  open: boolean;
}>();

defineEmits<{
  toggle: [];
  select: [id: string];
  restore: [conversation: AIConversation];
}>();

function formatRelativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return 'just now';
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d ago`;
  return new Date(iso).toLocaleDateString();
}
</script>
