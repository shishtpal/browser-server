<template>
  <Modal :open="!!todo" title="Todo details" @close="$emit('close')">
    <div v-if="todo" class="max-h-[75vh] space-y-4 overflow-y-auto overscroll-contain pr-1">
      <!-- Header: priority dot + title + badges -->
      <div class="flex items-start gap-3">
        <span
          class="mt-1.5 h-3 w-3 shrink-0 rounded-full"
          :class="todoDotClass(todo)"
          aria-hidden="true"
        />
        <div class="min-w-0 flex-1">
          <h2
            class="text-base leading-tight font-black break-words text-slate-900 dark:text-white"
            :class="{ 'line-through opacity-60': todo.status === 'completed' }"
          >
            {{ todo.title }}
          </h2>
          <div class="mt-1.5 flex flex-wrap items-center gap-1.5">
            <span
              class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold tracking-wider uppercase"
              :class="statusBadgeClass"
            >
              {{ statusText }}
            </span>
            <TodoPriorityBadge :priority="todo.priority" />
            <span
              v-if="todo.pinned"
              class="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-bold tracking-wider text-amber-700 uppercase dark:bg-amber-900/30 dark:text-amber-400"
            >
              <Pin class="h-2.5 w-2.5" :stroke-width="2.5" aria-hidden="true" />
              Pinned
            </span>
          </div>
        </div>
      </div>

      <!-- Color bar -->
      <div
        v-if="todo.color"
        class="h-1 w-full rounded-full"
        :style="{ backgroundColor: todo.color }"
        aria-hidden="true"
      />

      <!-- Screenshot -->
      <a
        v-if="screenshotUrl"
        :href="screenshotUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="group block"
        title="Open screenshot in a new tab"
      >
        <span
          class="relative block overflow-hidden rounded-lg border border-gray-200 dark:border-slate-600"
        >
          <img
            :src="screenshotUrl"
            alt="Todo screenshot"
            class="max-h-56 w-full object-contain transition group-hover:opacity-90 sm:max-h-64"
          />
          <ExternalLink
            class="absolute top-2 right-2 h-4 w-4 rounded bg-slate-900/50 p-0.5 text-white opacity-70 transition group-hover:opacity-100"
            aria-hidden="true"
          />
        </span>
      </a>

      <!-- Description -->
      <p
        v-if="todo.description"
        class="text-sm leading-relaxed break-words text-slate-600 dark:text-slate-300"
        v-html="linkifyDescription(todo.description)"
      ></p>

      <!-- Metadata grid -->
      <div class="grid grid-cols-2 gap-2.5">
        <div
          v-for="item in metaItems"
          :key="item.label"
          class="flex items-start gap-2.5 rounded-lg bg-slate-50 px-3 py-2.5 dark:bg-slate-700/50"
        >
          <component
            :is="item.icon"
            class="mt-0.5 h-3.5 w-3.5 shrink-0 text-slate-400 dark:text-slate-500"
            :stroke-width="2.25"
            aria-hidden="true"
          />
          <div class="min-w-0">
            <span
              class="block text-[10px] font-bold tracking-wider text-slate-400 uppercase dark:text-slate-500"
            >
              {{ item.label }}
            </span>
            <span
              class="block truncate text-xs font-semibold text-slate-700 dark:text-slate-200"
              :title="item.value"
            >
              {{ item.value }}
            </span>
          </div>
        </div>
      </div>

      <!-- Tags -->
      <div v-if="todo.tags?.length">
        <span
          class="mb-1.5 flex items-center gap-1 text-[10px] font-bold tracking-wider text-slate-400 uppercase dark:text-slate-500"
        >
          <Tags class="h-3 w-3" :stroke-width="2.25" aria-hidden="true" />
          Tags
        </span>
        <div class="flex flex-wrap gap-1.5">
          <span
            v-for="tag in todo.tags"
            :key="tag"
            class="inline-flex items-center rounded-full bg-violet-100 px-2.5 py-0.5 text-xs font-bold text-violet-700 dark:bg-violet-900/30 dark:text-violet-400"
          >
            {{ tag }}
          </span>
        </div>
      </div>

      <!-- Actions -->
      <div class="flex items-center justify-end gap-2 pt-1">
        <Button variant="secondary" size="sm" @click="$emit('close')">Close</Button>
        <Button variant="gradient-violet" size="sm" @click="$emit('edit', todo)">
          <span class="inline-flex items-center gap-1.5">
            <Pencil class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
            Edit
          </span>
        </Button>
      </div>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { format } from 'date-fns';
import {
  Calendar,
  CalendarClock,
  Clock,
  ExternalLink,
  Globe,
  Pencil,
  Pin,
  Repeat,
  Tags,
  type LucideIcon,
} from '@lucide/vue';
import type { Todo } from '../../types';
import { linkifyDescription } from '../../lib/descriptionLinks';
import { getScreenshotUrl } from '../../lib/api/todos';
import Button from '../ui/Button.vue';
import Modal from '../ui/Modal.vue';
import TodoPriorityBadge from '../todos/TodoPriorityBadge.vue';
import { formatRrule, statusLabel, STATUS_META, todoDotClass } from '../todos/todoFormat';

const props = defineProps<{
  todo: Todo | null;
}>();

defineEmits<{
  (e: 'close'): void;
  (e: 'edit', todo: Todo): void;
}>();

const screenshotUrl = computed(() =>
  props.todo?.screenshot_path ? getScreenshotUrl(props.todo.id) : '',
);

const statusText = computed(() => (props.todo ? statusLabel(props.todo.status) : ''));

const statusBadgeClass = computed(
  () => STATUS_META[props.todo?.status as keyof typeof STATUS_META]?.badgeClass ?? '',
);

function formatDate(raw: string | null | undefined): string {
  if (!raw) return '—';
  try {
    const d = new Date(raw.includes('T') ? raw : raw + 'T00:00:00');
    return format(d, 'MMM d, yyyy');
  } catch {
    return raw;
  }
}

const metaItems = computed(() => {
  const t = props.todo;
  const items: { label: string; value: string; icon: LucideIcon }[] = [];
  if (!t) return items;
  if (t.start_date) items.push({ label: 'Start', value: formatDate(t.start_date), icon: Calendar });
  if (t.end_date) items.push({ label: 'End', value: formatDate(t.end_date), icon: CalendarClock });
  if (t.domain) items.push({ label: 'Domain', value: t.domain, icon: Globe });
  if (t.rrule) items.push({ label: 'Recurrence', value: formatRrule(t.rrule), icon: Repeat });
  items.push({ label: 'Created', value: formatDate(t.created_at), icon: Clock });
  items.push({ label: 'Updated', value: formatDate(t.updated_at), icon: Clock });
  return items;
});
</script>
