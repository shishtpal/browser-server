<template>
  <section class="mt-4 border-t border-slate-200 pt-3 dark:border-white/10">
    <button
      class="flex w-full items-center gap-2 px-2 py-1.5 text-xs font-semibold text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-slate-400 dark:hover:bg-white/5 dark:hover:text-white"
      type="button"
      @click="$emit('toggle')"
    >
      <ChevronRight
        class="h-3.5 w-3.5 shrink-0 transition-transform"
        :class="{ 'rotate-90': open }"
        :stroke-width="2.5"
        aria-hidden="true"
      />
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
            <ArchiveRestore class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { AIConversation } from '@browser-server/shared-types';
import { ArchiveRestore, ChevronRight } from '@lucide/vue';
import { formatRelativeTime } from './chatFormat';

defineProps<{
  items: AIConversation[];
  open: boolean;
}>();

defineEmits<{
  toggle: [];
  select: [id: string];
  restore: [conversation: AIConversation];
}>();
</script>
