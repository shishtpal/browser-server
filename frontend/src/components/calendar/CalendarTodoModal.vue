<template>
  <Modal :open="open" :title="editingTodo ? 'Edit Todo' : 'New Todo'" @close="close">
    <form class="flex flex-col gap-3" @submit.prevent="onSubmit">
      <!-- Title -->
      <div>
        <label
          class="mb-1 block text-[10px] font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
          >Title</label
        >
        <input
          v-model="form.title"
          type="text"
          required
          class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-indigo-900/30"
          placeholder="What needs to be done?"
        />
      </div>

      <!-- Description -->
      <div>
        <label
          class="mb-1 block text-[10px] font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
        >
          Description
        </label>
        <textarea
          v-model="form.description"
          rows="2"
          class="w-full resize-none rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-indigo-900/30"
          placeholder="Add details..."
        ></textarea>
      </div>

      <!-- Start Date + End Date -->
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label
            class="mb-1 block text-[10px] font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
          >
            Start Date
          </label>
          <input
            v-model="form.start_date"
            type="date"
            class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30"
          />
        </div>
        <div>
          <label
            class="mb-1 block text-[10px] font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
          >
            End Date
          </label>
          <input
            v-model="form.end_date"
            type="date"
            class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30"
          />
        </div>
      </div>

      <!-- Priority + Status -->
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label
            class="mb-1 block text-[10px] font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
          >
            Priority
          </label>
          <select
            v-model="form.priority"
            class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30"
          >
            <option value="low">Low</option>
            <option value="medium">Medium</option>
            <option value="high">High</option>
            <option value="urgent">Urgent</option>
          </select>
        </div>
        <div v-if="editingTodo">
          <label
            class="mb-1 block text-[10px] font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
          >
            Status
          </label>
          <select
            v-model="form.status"
            class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30"
          >
            <option value="pending">Pending</option>
            <option value="in_progress">In Progress</option>
            <option value="completed">Completed</option>
            <option value="archived">Archived</option>
          </select>
        </div>
      </div>

      <!-- Domain (Category) -->
      <div>
        <label
          class="mb-1 block text-[10px] font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
        >
          <span class="flex items-center gap-1">
            <svg
              class="h-3 w-3"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <circle cx="12" cy="12" r="10" />
              <path
                d="M2 12h20M12 2a15.3 15.3 0 014 10 15.3 15.3 0 01-4 10 15.3 15.3 0 01-4-10 15.3 15.3 0 014-10z"
              />
            </svg>
            Domain / Category
          </span>
        </label>
        <input
          v-model="form.domain"
          type="text"
          list="domain-list"
          placeholder="e.g., Work, Personal, Health"
          class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-indigo-900/30"
        />
        <datalist id="domain-list">
          <option v-for="cat in defaultDomains" :key="cat" :value="cat" />
        </datalist>
      </div>

      <!-- Color + Recurrence -->
      <div class="grid grid-cols-[1fr_auto] items-end gap-3">
        <div>
          <label
            class="mb-1 block text-[10px] font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
          >
            <span class="flex items-center gap-1">
              <svg
                class="h-3 w-3"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <polyline points="17 1 21 5 17 9" />
                <path d="M3 11V9a4 4 0 014-4h14" />
                <polyline points="7 23 3 19 7 15" />
                <path d="M21 13v2a4 4 0 01-4 4H3" />
              </svg>
              Recurrence
            </span>
          </label>
          <select
            v-model="form.rrule"
            class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-indigo-900/30"
          >
            <option value="">None</option>
            <option value="FREQ=DAILY">Daily</option>
            <option value="FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR">Every Weekday</option>
            <option value="FREQ=WEEKLY">Weekly</option>
            <option value="FREQ=WEEKLY;INTERVAL=2">Every 2 Weeks</option>
            <option value="FREQ=MONTHLY">Monthly</option>
            <option value="FREQ=YEARLY">Yearly</option>
            <option value="custom">Custom...</option>
          </select>
          <input
            v-if="form.rrule === 'custom'"
            v-model="customRrule"
            type="text"
            placeholder="e.g., FREQ=WEEKLY;BYDAY=MO,WE,FR"
            class="mt-1.5 w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-xs font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-indigo-900/30"
          />
        </div>
        <div>
          <label
            class="mb-1 block text-[10px] font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
            >Color</label
          >
          <div class="flex flex-wrap gap-1">
            <button
              v-for="c in colorOptions"
              :key="c"
              type="button"
              @click="form.color = form.color === c ? '' : c"
              class="h-6 w-6 rounded-full border-2 transition-all"
              :class="
                form.color === c
                  ? 'scale-110 border-slate-900 dark:border-white'
                  : 'border-transparent hover:scale-105'
              "
              :style="{ backgroundColor: c }"
            />
          </div>
        </div>
      </div>

      <!-- Tags -->
      <div>
        <label
          class="mb-1 block text-[10px] font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
        >
          <span class="flex items-center gap-1">
            <svg
              class="h-3 w-3"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path
                d="M20.59 13.41l-7.17 7.17a2 2 0 01-2.83 0L2 12V7h5l6.59 6.59a2 2 0 010 2.82z"
              />
              <line x1="7" y1="7" x2="7.01" y2="7" />
            </svg>
            Tags
          </span>
        </label>
        <div class="flex gap-2">
          <input
            v-model="tagInput"
            type="text"
            placeholder="Add a tag..."
            class="flex-1 rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-indigo-900/30"
            @keydown.enter.prevent="addTag"
          />
          <Button type="button" variant="secondary" size="sm" @click="addTag">Add</Button>
        </div>
        <div v-if="tags.length > 0" class="mt-1.5 flex flex-wrap gap-1">
          <span
            v-for="tag in tags"
            :key="tag"
            class="inline-flex items-center gap-1 rounded-full bg-indigo-100 px-2 py-0.5 text-xs font-bold text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400"
          >
            {{ tag }}
            <button
              type="button"
              @click="removeTag(tag)"
              class="hover:text-indigo-900 dark:hover:text-indigo-200"
            >
              <svg
                class="h-3 w-3"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </span>
        </div>
      </div>

      <!-- Pinned toggle (edit mode) -->
      <div
        v-if="editingTodo"
        class="flex items-center gap-4 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 dark:border-slate-700 dark:bg-slate-800/50"
      >
        <label class="flex cursor-pointer items-center gap-2">
          <input
            v-model="form.pinned"
            type="checkbox"
            class="h-4 w-4 rounded border-gray-300 text-amber-600 focus:ring-amber-500 dark:border-slate-600 dark:bg-slate-800"
          />
          <span class="text-xs font-bold text-slate-600 dark:text-slate-300">Pinned</span>
        </label>
      </div>

      <!-- Actions -->
      <div class="flex items-center justify-between pt-2">
        <Button v-if="editingTodo" type="button" variant="danger" size="sm" @click="emit('delete')"
          >Delete</Button
        >
        <div v-else></div>
        <div class="flex items-center gap-2">
          <Button type="button" variant="secondary" size="sm" @click="close">Cancel</Button>
          <Button type="submit" variant="gradient-violet" size="sm" :loading="saving">{{
            editingTodo ? 'Save' : 'Create'
          }}</Button>
        </div>
      </div>
    </form>
  </Modal>
</template>

<script setup lang="ts">
import type { Todo, CreateTodoInput, TodoPriority, TodoStatus } from '../../types';
import { ref, watch } from 'vue';
import Modal from '../ui/Modal.vue';
import Button from '../ui/Button.vue';

const props = defineProps<{
  open: boolean;
  editingTodo?: Todo | null;
  initialDueDate?: string;
  userId: number;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'submit', data: CreateTodoInput): void;
  (e: 'update', id: number, data: Partial<Todo>): void;
  (e: 'delete'): void;
}>();

const defaultDomains = [
  'Work',
  'Personal',
  'Health',
  'Finance',
  'Education',
  'Shopping',
  'Errands',
  'Projects',
];
const colorOptions = [
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

const form = ref({
  title: '',
  description: '',
  start_date: '',
  end_date: '',
  priority: 'medium' as TodoPriority,
  status: 'pending' as TodoStatus,
  domain: '',
  color: '',
  rrule: '',
  pinned: false,
});

const customRrule = ref('');
const tags = ref<string[]>([]);
const tagInput = ref('');
const saving = ref(false);

watch(
  () => props.open,
  (open) => {
    if (open) {
      if (props.editingTodo) {
        form.value.title = props.editingTodo.title;
        form.value.description = props.editingTodo.description;
        form.value.start_date = props.editingTodo.start_date || '';
        form.value.end_date = props.editingTodo.end_date || '';
        form.value.priority = props.editingTodo.priority;
        form.value.status = props.editingTodo.status;
        form.value.domain = props.editingTodo.domain || '';
        form.value.color = props.editingTodo.color || '';
        form.value.rrule = props.editingTodo.rrule || '';
        form.value.pinned = props.editingTodo.pinned;
        tags.value = [...(props.editingTodo.tags || [])];
        customRrule.value = '';
      } else {
        form.value.title = '';
        form.value.description = '';
        form.value.start_date = props.initialDueDate || '';
        form.value.end_date = '';
        form.value.priority = 'medium';
        form.value.status = 'pending';
        form.value.domain = '';
        form.value.color = '';
        form.value.rrule = '';
        form.value.pinned = false;
        tags.value = [];
        customRrule.value = '';
      }
      tagInput.value = '';
    }
  },
);

function addTag() {
  const tag = tagInput.value.trim();
  if (tag && !tags.value.includes(tag)) {
    tags.value = [...tags.value, tag];
    tagInput.value = '';
  }
}

function removeTag(tag: string) {
  tags.value = tags.value.filter((t) => t !== tag);
}

function close() {
  emit('close');
}

async function onSubmit() {
  if (!form.value.title.trim()) return;
  saving.value = true;
  const finalRrule = form.value.rrule === 'custom' ? customRrule.value : form.value.rrule;
  try {
    if (props.editingTodo) {
      emit('update', props.editingTodo.id, {
        title: form.value.title.trim(),
        description: form.value.description.trim(),
        start_date: form.value.start_date || null,
        end_date: form.value.end_date || null,
        priority: form.value.priority,
        status: form.value.status,
        domain: form.value.domain.trim() || undefined,
        color: form.value.color || undefined,
        rrule: finalRrule || undefined,
        pinned: form.value.pinned,
        tags: tags.value,
      });
    } else {
      const payload: CreateTodoInput = {
        user_id: props.userId,
        title: form.value.title.trim(),
        description: form.value.description.trim() || undefined,
        start_date: form.value.start_date || null,
        end_date: form.value.end_date || null,
        priority: form.value.priority,
        domain: form.value.domain.trim() || undefined,
        color: form.value.color || undefined,
        rrule: finalRrule || undefined,
        tags: tags.value,
      };
      emit('submit', payload);
    }
    close();
  } finally {
    saving.value = false;
  }
}
</script>
