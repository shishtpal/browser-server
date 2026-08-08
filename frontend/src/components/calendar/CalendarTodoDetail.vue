<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="todo"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 backdrop-blur-sm"
        @click.self="emit('close')"
      >
        <Transition
          enter-active-class="transition duration-200 ease-out"
          enter-from-class="opacity-0 scale-95"
          enter-to-class="opacity-100 scale-100"
          leave-active-class="transition duration-150 ease-in"
          leave-from-class="opacity-100 scale-100"
          leave-to-class="opacity-0 scale-95"
        >
          <div
            v-if="todo"
            class="relative w-full max-w-md rounded-2xl border border-gray-200 bg-white p-6 shadow-2xl shadow-gray-900/30 dark:border-white/10 dark:bg-slate-800 dark:shadow-slate-950/30"
          >
            <!-- Close button -->
            <button
              type="button"
              class="absolute top-4 right-4 grid h-8 w-8 place-items-center rounded-lg bg-gray-100 text-slate-500 transition hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-400 dark:hover:bg-slate-600"
              aria-label="Close"
              @click="emit('close')"
            >
              &times;
            </button>

            <!-- Header: Priority dot + Title -->
            <div class="flex items-start gap-3 pr-10">
              <span class="mt-1.5 h-3 w-3 shrink-0 rounded-full" :class="priorityDotClass" />
              <div class="min-w-0 flex-1">
                <h2
                  class="text-lg leading-tight font-black text-slate-900 dark:text-white"
                  :class="{ 'line-through opacity-60': todo.status === 'completed' }"
                >
                  {{ todo.title }}
                </h2>
                <div class="mt-1 flex flex-wrap items-center gap-2">
                  <span
                    class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold tracking-wider uppercase"
                    :class="statusBadgeClass"
                  >
                    {{ todo.status }}
                  </span>
                  <span
                    class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold tracking-wider uppercase"
                    :class="priorityBadgeClass"
                  >
                    {{ todo.priority }}
                  </span>
                  <span
                    v-if="todo.pinned"
                    class="inline-flex items-center gap-0.5 rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-bold tracking-wider text-amber-700 uppercase dark:bg-amber-900/30 dark:text-amber-400"
                  >
                    <svg class="h-2.5 w-2.5" fill="currentColor" viewBox="0 0 20 20">
                      <path
                        d="M9.828.722a.5.5 0 01.354 0l.707.293a.5.5 0 01.293.354l.354 3.535 3.535.354a.5.5 0 01.354.293l.293.707a.5.5 0 010 .354l-2.828 2.828 1.06 5.303a.5.5 0 01-.146.457l-.5.5a.5.5 0 01-.561.085L10 13.414l-3.243 1.871a.5.5 0 01-.56-.085l-.5-.5a.5.5 0 01-.147-.457l1.06-5.303L3.783 6.112a.5.5 0 010-.354l.293-.707a.5.5 0 01.354-.293l3.535-.354.354-3.535a.5.5 0 01.293-.354l.707-.293z"
                      />
                    </svg>
                    Pinned
                  </span>
                </div>
              </div>
            </div>

            <!-- Color bar -->
            <div
              v-if="todo.color"
              class="mt-3 h-1 w-full rounded-full"
              :style="{ backgroundColor: todo.color }"
            />

            <!-- Screenshot -->
            <a
              v-if="todo.screenshot_path"
              :href="screenshotUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="mt-4 block"
            >
              <img
                :src="screenshotUrl"
                alt="Todo screenshot"
                class="block max-h-64 w-full rounded-lg border border-gray-200 object-contain transition hover:opacity-90 dark:border-slate-600"
              />
            </a>

            <!-- Description -->
            <div v-if="todo.description" class="mt-4">
              <p
                class="text-sm leading-relaxed text-slate-600 dark:text-slate-300"
                v-html="linkifyDescription(todo.description)"
              ></p>
            </div>

            <!-- Metadata grid -->
            <div class="mt-5 grid grid-cols-2 gap-3">
              <!-- Start Date -->
              <div
                v-if="todo.start_date"
                class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/50"
              >
                <span
                  class="block text-[10px] font-bold tracking-wider text-slate-400 uppercase dark:text-slate-500"
                  >Start</span
                >
                <span class="text-sm font-semibold text-slate-700 dark:text-slate-200">{{
                  formatDate(todo.start_date)
                }}</span>
              </div>

              <!-- End Date -->
              <div
                v-if="todo.end_date"
                class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/50"
              >
                <span
                  class="block text-[10px] font-bold tracking-wider text-slate-400 uppercase dark:text-slate-500"
                  >End</span
                >
                <span class="text-sm font-semibold text-slate-700 dark:text-slate-200">{{
                  formatDate(todo.end_date)
                }}</span>
              </div>

              <!-- Domain -->
              <div v-if="todo.domain" class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/50">
                <span
                  class="block text-[10px] font-bold tracking-wider text-slate-400 uppercase dark:text-slate-500"
                  >Domain</span
                >
                <span class="text-sm font-semibold text-slate-700 dark:text-slate-200">{{
                  todo.domain
                }}</span>
              </div>

              <!-- Recurrence -->
              <div v-if="todo.rrule" class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/50">
                <span
                  class="block text-[10px] font-bold tracking-wider text-slate-400 uppercase dark:text-slate-500"
                  >Recurrence</span
                >
                <span class="text-sm font-semibold text-slate-700 dark:text-slate-200">{{
                  formatRrule(todo.rrule)
                }}</span>
              </div>

              <!-- Created -->
              <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/50">
                <span
                  class="block text-[10px] font-bold tracking-wider text-slate-400 uppercase dark:text-slate-500"
                  >Created</span
                >
                <span class="text-sm font-semibold text-slate-700 dark:text-slate-200">{{
                  formatDate(todo.created_at)
                }}</span>
              </div>

              <!-- Updated -->
              <div class="rounded-lg bg-slate-50 px-3 py-2 dark:bg-slate-700/50">
                <span
                  class="block text-[10px] font-bold tracking-wider text-slate-400 uppercase dark:text-slate-500"
                  >Updated</span
                >
                <span class="text-sm font-semibold text-slate-700 dark:text-slate-200">{{
                  formatDate(todo.updated_at)
                }}</span>
              </div>
            </div>

            <!-- Tags -->
            <div v-if="todo.tags && todo.tags.length > 0" class="mt-4">
              <span
                class="mb-1.5 block text-[10px] font-bold tracking-wider text-slate-400 uppercase dark:text-slate-500"
                >Tags</span
              >
              <div class="flex flex-wrap gap-1.5">
                <span
                  v-for="tag in todo.tags"
                  :key="tag"
                  class="inline-flex items-center rounded-full bg-indigo-100 px-2.5 py-0.5 text-xs font-bold text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400"
                >
                  {{ tag }}
                </span>
              </div>
            </div>

            <!-- Actions -->
            <div class="mt-6 flex items-center justify-end gap-2">
              <button
                type="button"
                class="rounded-lg border border-gray-200 bg-white px-3.5 py-2 text-xs font-bold text-slate-600 shadow-sm transition hover:bg-gray-50 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-slate-600"
                @click="emit('close')"
              >
                Close
              </button>
              <button
                type="button"
                class="rounded-lg bg-indigo-600 px-3.5 py-2 text-xs font-bold text-white shadow-sm transition hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600"
                @click="emit('edit', todo)"
              >
                Edit
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import type { Todo } from '../../types';
import { computed } from 'vue';
import { format } from 'date-fns';
import { linkifyDescription } from '../../lib/descriptionLinks';
import { getScreenshotUrl } from '../../lib/api/todos';

const props = defineProps<{
  todo: Todo | null;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'edit', todo: Todo): void;
}>();

const screenshotUrl = computed(() =>
  props.todo?.screenshot_path ? getScreenshotUrl(props.todo.id) : '',
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

function formatRrule(rrule: string): string {
  const map: Record<string, string> = {
    'FREQ=DAILY': 'Daily',
    'FREQ=WEEKLY': 'Weekly',
    'FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR': 'Every Weekday',
    'FREQ=WEEKLY;INTERVAL=2': 'Every 2 Weeks',
    'FREQ=MONTHLY': 'Monthly',
    'FREQ=YEARLY': 'Yearly',
  };
  return map[rrule] || rrule;
}

const priorityDotClass = computed(() => {
  if (!props.todo) return '';
  if (props.todo.status === 'completed') return 'bg-slate-300 dark:bg-slate-600';
  const map: Record<string, string> = {
    low: 'bg-slate-400 dark:bg-slate-500',
    medium: 'bg-blue-500 dark:bg-blue-400',
    high: 'bg-amber-500 dark:bg-amber-400',
    urgent: 'bg-red-500 dark:bg-red-400',
  };
  return map[props.todo.priority] || 'bg-slate-400';
});

const statusBadgeClass = computed(() => {
  if (!props.todo) return '';
  const map: Record<string, string> = {
    pending: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
    completed: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400',
    archived: 'bg-slate-100 text-slate-500 dark:bg-slate-700 dark:text-slate-400',
  };
  return map[props.todo.status] || '';
});

const priorityBadgeClass = computed(() => {
  if (!props.todo) return '';
  const map: Record<string, string> = {
    low: 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300',
    medium: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
    high: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
    urgent: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  };
  return map[props.todo.priority] || '';
});
</script>
