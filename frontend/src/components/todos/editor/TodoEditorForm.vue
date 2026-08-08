<template>
  <form id="todo-editor" class="flex flex-col gap-4" @submit.prevent="onSubmit">
    <!-- Title -->
    <FormField label="Title" required>
      <input
        v-model="form.title"
        type="text"
        required
        class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-indigo-900/30"
        placeholder="What needs to be done?"
      />
    </FormField>

    <!-- Description -->
    <FormField label="Description">
      <textarea
        v-model="form.description"
        rows="2"
        class="w-full resize-none rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-indigo-900/30"
        placeholder="Add details..."
      ></textarea>
    </FormField>

    <!-- Dates -->
    <div class="grid grid-cols-2 gap-3">
      <FormField label="Start date">
        <input
          v-model="form.start_date"
          type="date"
          class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30"
        />
      </FormField>
      <FormField label="End date">
        <input
          v-model="form.end_date"
          type="date"
          class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30"
        />
      </FormField>
    </div>

    <!-- Priority + Status -->
    <div class="grid grid-cols-2 gap-3">
      <FormField label="Priority">
        <select v-model="form.priority" :class="selectClass">
          <option v-for="p in PRIORITY_ORDER" :key="p" :value="p">
            {{ PRIORITY_META[p].label }}
          </option>
        </select>
      </FormField>
      <FormField v-if="isEdit" label="Status">
        <select v-model="form.status" :class="selectClass">
          <option value="pending">Pending</option>
          <option value="in_progress">In Progress</option>
          <option value="completed">Completed</option>
          <option value="archived">Archived</option>
        </select>
      </FormField>
    </div>

    <!-- Domain -->
    <FormField label="Domain / Category">
      <div class="relative">
        <Globe
          class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          v-model="form.domain"
          type="text"
          list="todo-domain-list"
          placeholder="e.g., Work, Personal, Health"
          class="w-full rounded-lg border border-gray-300 bg-gray-50 py-2 pr-3 pl-9 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-indigo-900/30"
        />
        <datalist id="todo-domain-list">
          <option v-for="cat in DEFAULT_DOMAINS" :key="cat" :value="cat" />
        </datalist>
      </div>
    </FormField>

    <!-- Recurrence + Color -->
    <div class="grid grid-cols-[1fr_auto] items-end gap-3">
      <FormField label="Recurrence">
        <div class="relative">
          <Repeat
            class="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
            aria-hidden="true"
          />
          <select v-model="form.rrule" :class="[selectClass, 'pl-8']">
            <option v-for="opt in RRULE_OPTIONS" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
        </div>
        <input
          v-if="form.rrule === 'custom'"
          v-model="customRrule"
          type="text"
          placeholder="e.g., FREQ=WEEKLY;BYDAY=MO,WE,FR"
          class="mt-1.5 w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-xs font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-indigo-900/30"
        />
      </FormField>
      <FormField label="Color">
        <div class="flex flex-wrap gap-1.5">
          <button
            v-for="c in COLOR_OPTIONS"
            :key="c"
            type="button"
            class="grid h-7 w-7 place-items-center rounded-full border-2 transition-all"
            :class="
              form.color === c
                ? 'scale-110 border-slate-900 dark:border-white'
                : 'border-transparent hover:scale-105'
            "
            :style="{ backgroundColor: c }"
            :aria-label="`Set color ${c}`"
            :aria-pressed="form.color === c"
            @click="form.color = form.color === c ? '' : c"
          >
            <Check
              v-if="form.color === c"
              class="h-3 w-3 text-white drop-shadow"
              :stroke-width="3.5"
              aria-hidden="true"
            />
          </button>
        </div>
      </FormField>
    </div>

    <!-- Tags -->
    <FormField label="Tags" help-text="Press Enter or comma to add.">
      <div
        class="flex flex-wrap items-center gap-1.5 rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 shadow-sm transition focus-within:border-indigo-400 focus-within:ring-4 focus-within:ring-indigo-100 dark:border-slate-600 dark:bg-slate-800 dark:focus-within:border-indigo-500 dark:focus-within:ring-indigo-900/30"
      >
        <span
          v-for="tag in tags"
          :key="tag"
          class="inline-flex items-center gap-1 rounded-full bg-indigo-100 px-2 py-0.5 text-xs font-bold text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400"
        >
          {{ tag }}
          <button
            type="button"
            class="hover:text-indigo-900 dark:hover:text-indigo-200"
            :aria-label="`Remove tag ${tag}`"
            @click="removeTag(tag)"
          >
            <X class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          </button>
        </span>
        <input
          v-model="tagInput"
          type="text"
          placeholder="Add a tag..."
          class="min-w-[10ch] flex-1 border-0 bg-transparent p-0.5 text-sm font-semibold text-slate-700 placeholder:font-normal focus:outline-none dark:text-slate-200"
          @keydown.enter.prevent="addTag"
          @keydown.,.prevent="addTag"
        />
      </div>
    </FormField>

    <!-- Pinned (edit only) -->
    <label
      v-if="isEdit"
      class="flex cursor-pointer items-center gap-2.5 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2.5 dark:border-slate-700 dark:bg-slate-800/50"
    >
      <input
        v-model="form.pinned"
        type="checkbox"
        class="h-4 w-4 rounded border-gray-300 text-amber-600 focus:ring-amber-500 dark:border-slate-600 dark:bg-slate-800"
      />
      <Pin class="h-3.5 w-3.5 text-amber-500" :stroke-width="2.25" aria-hidden="true" />
      <span class="text-xs font-bold text-slate-600 dark:text-slate-300">
        Pinned to the top of the list
      </span>
    </label>
  </form>
</template>

<script setup lang="ts">
import type { CreateTodoInput, Todo, TodoPriority, TodoStatus } from '../../../types';
import { computed, ref, watch } from 'vue';
import { Check, Globe, Pin, Repeat, X } from '@lucide/vue';
import FormField from '../../ui/FormField.vue';
import { PRIORITY_META, PRIORITY_ORDER, RRULE_OPTIONS } from '../todoFormat';

const props = withDefaults(
  defineProps<{
    editingTodo?: Todo | null;
    initialDueDate?: string;
    userId: number;
    /** Set to false in create mode (user picker might be unselected at first). */
    disabled?: boolean;
  }>(),
  { editingTodo: null, initialDueDate: '', disabled: false },
);

const emit = defineEmits<{
  submit: [data: CreateTodoInput];
  update: [id: number, data: Partial<Todo>];
}>();

const DEFAULT_DOMAINS = [
  'Work',
  'Personal',
  'Health',
  'Finance',
  'Education',
  'Shopping',
  'Errands',
  'Projects',
];

const COLOR_OPTIONS = [
  '#3b82f6',
  '#ef4444',
  '#22c55e',
  '#f59e0b',
  '#8b5cf6',
  '#ec4899',
  '#06b6d4',
  '#f97316',
  '#6366f1',
  '#14b8a6',
];

const selectClass =
  'w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30';

const isEdit = computed(() => !!props.editingTodo);

const blankForm = () => ({
  title: '',
  description: '',
  start_date: props.initialDueDate || '',
  end_date: '',
  priority: 'medium' as TodoPriority,
  status: 'pending' as TodoStatus,
  domain: '',
  color: '',
  rrule: '',
  pinned: false,
});

const form = ref(blankForm());
const customRrule = ref('');
const tags = ref<string[]>([]);
const tagInput = ref('');

/** Reset/prefill when the editing target (or create date) changes. */
watch(
  () => [props.editingTodo?.id, props.initialDueDate] as const,
  () => {
    const t = props.editingTodo;
    if (t) {
      form.value = {
        title: t.title,
        description: t.description ?? '',
        start_date: t.start_date || '',
        end_date: t.end_date || '',
        priority: t.priority,
        status: t.status,
        domain: t.domain || '',
        color: t.color || '',
        rrule: t.rrule || '',
        pinned: t.pinned,
      };
      tags.value = [...(t.tags || [])];
      // A saved rule that isn't a preset shows in the custom field.
      const isPreset = RRULE_OPTIONS.some((o) => o.value === t.rrule && o.value !== 'custom');
      if (t.rrule && !isPreset) {
        form.value.rrule = 'custom';
        customRrule.value = t.rrule;
      } else {
        form.value.rrule = t.rrule || '';
        customRrule.value = '';
      }
    } else {
      form.value = blankForm();
      tags.value = [];
      customRrule.value = '';
    }
    tagInput.value = '';
  },
  { immediate: true },
);

function addTag() {
  const tag = tagInput.value.trim();
  if (tag && !tags.value.includes(tag)) {
    tags.value = [...tags.value, tag];
  }
  tagInput.value = '';
}

function removeTag(tag: string) {
  tags.value = tags.value.filter((t) => t !== tag);
}

function finalRrule() {
  return (form.value.rrule === 'custom' ? customRrule.value : form.value.rrule).trim();
}

function onSubmit() {
  if (props.disabled) return;
  addTag(); // pick up a half-typed tag
  const title = form.value.title.trim();
  if (!title) return;

  if (props.editingTodo) {
    emit('update', props.editingTodo.id, {
      title,
      description: form.value.description.trim(),
      start_date: form.value.start_date || null,
      end_date: form.value.end_date || null,
      priority: form.value.priority,
      status: form.value.status,
      domain: form.value.domain.trim() || undefined,
      color: form.value.color || undefined,
      rrule: finalRrule() || undefined,
      pinned: form.value.pinned,
      tags: tags.value,
    });
  } else {
    emit('submit', {
      user_id: props.userId,
      title,
      description: form.value.description.trim() || undefined,
      start_date: form.value.start_date || null,
      end_date: form.value.end_date || null,
      priority: form.value.priority,
      domain: form.value.domain.trim() || undefined,
      color: form.value.color || undefined,
      rrule: finalRrule() || undefined,
      tags: tags.value,
    });
  }
}

/** Submit programmatically (the parent modal owns the action buttons). */
defineExpose({ submit: onSubmit });
</script>
