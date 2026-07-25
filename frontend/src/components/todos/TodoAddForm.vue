<template>
  <form @submit.prevent="onSubmit" class="overflow-hidden rounded-2xl border border-indigo-100 bg-white shadow-sm transition-colors dark:border-indigo-500/20 dark:bg-slate-800/90">
    <!-- Header -->
    <div class="border-b border-indigo-100 bg-gradient-to-r from-indigo-50 via-white to-violet-50 px-4 py-3 dark:border-indigo-500/20 dark:from-indigo-950/40 dark:via-slate-800 dark:to-violet-950/30 sm:px-5">
      <div class="flex items-center gap-3">
        <span class="grid h-9 w-9 shrink-0 place-items-center rounded-xl bg-indigo-600 text-white shadow-md shadow-indigo-500/20">
          <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 5v14m-7-7h14" />
          </svg>
        </span>
        <div>
          <h2 class="text-sm font-black text-slate-900 dark:text-white">Create a new todo</h2>
          <p class="text-xs text-slate-500 dark:text-slate-400">Start with a title, expand for full control.</p>
        </div>
      </div>
    </div>

    <!-- Body -->
    <div class="p-4 sm:p-5">
      <!-- Title -->
      <label class="block">
        <span class="mb-1.5 block text-xs font-black text-slate-700 dark:text-slate-300">Task title</span>
        <input
          ref="titleInput"
          v-model="title"
          placeholder="What needs to be done?"
          required
          maxlength="200"
          class="w-full rounded-xl border border-gray-300 bg-gray-50 px-3.5 py-3 text-sm font-semibold text-slate-800 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:bg-white focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-900/50 dark:text-slate-100 dark:focus:bg-slate-900 dark:focus:ring-indigo-900/30"
        />
      </label>

      <!-- Expanded details -->
      <div v-if="moreOpen" class="mt-4 space-y-4 border-t border-gray-100 pt-4 dark:border-slate-700/80">
        <!-- Description -->
        <label class="block">
          <span class="mb-1.5 block text-xs font-black text-slate-700 dark:text-slate-300">Description <span class="font-semibold text-slate-400">(optional)</span></span>
          <textarea
            v-model="description"
            rows="2"
            placeholder="Add notes or context for this task..."
            class="w-full resize-y rounded-xl border border-gray-300 bg-gray-50 px-3.5 py-2.5 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-indigo-400 focus:bg-white focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-900/50 dark:text-slate-200 dark:focus:bg-slate-900 dark:focus:ring-indigo-900/30"
          />
        </label>

        <!-- Row 1: Priority + Domain -->
        <div class="grid gap-3 sm:grid-cols-2">
          <div>
            <span class="mb-1.5 block text-xs font-black text-slate-700 dark:text-slate-300">Priority</span>
            <TodoPrioritySelect v-model="priority" />
          </div>
          <div>
            <span class="mb-1.5 block text-xs font-black text-slate-700 dark:text-slate-300">Domain / Category</span>
            <input
              v-model="domain"
              list="add-domain-list"
              placeholder="e.g., Work, Personal"
              class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 transition focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-900/50 dark:text-slate-200 dark:focus:ring-indigo-900/30"
            />
            <datalist id="add-domain-list">
              <option v-for="cat in defaultDomains" :key="cat" :value="cat" />
            </datalist>
          </div>
        </div>

        <!-- Row 2: Start Date + End Date -->
        <div class="grid gap-3 sm:grid-cols-2">
          <div>
            <span class="mb-1.5 block text-xs font-black text-slate-700 dark:text-slate-300">Start date</span>
            <TodoDueDatePicker v-model="startDate" />
          </div>
          <div>
            <span class="mb-1.5 block text-xs font-black text-slate-700 dark:text-slate-300">End date</span>
            <TodoDueDatePicker v-model="endDate" />
          </div>
        </div>

        <!-- Row 3: Recurrence + Color -->
        <div class="grid gap-3 sm:grid-cols-[1fr_auto]">
          <div>
            <span class="mb-1.5 flex items-center gap-1 text-xs font-black text-slate-700 dark:text-slate-300">
              <svg class="h-3.5 w-3.5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><polyline points="17 1 21 5 17 9" /><path d="M3 11V9a4 4 0 014-4h14" /><polyline points="7 23 3 19 7 15" /><path d="M21 13v2a4 4 0 01-4 4H3" /></svg>
              Recurrence
            </span>
            <select
              v-model="rrule"
              class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 transition focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-900/50 dark:text-slate-200 dark:focus:ring-indigo-900/30"
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
              v-if="rrule === 'custom'"
              v-model="customRrule"
              type="text"
              placeholder="e.g., FREQ=WEEKLY;BYDAY=MO,WE,FR"
              class="mt-1.5 w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-1.5 text-xs font-semibold text-slate-700 transition placeholder:text-slate-400 focus:border-indigo-400 focus:outline-none focus:ring-4 focus:ring-indigo-100 dark:border-slate-600 dark:bg-slate-900/50 dark:text-slate-200 dark:focus:ring-indigo-900/30"
            />
          </div>
          <div>
            <span class="mb-1.5 block text-xs font-black text-slate-700 dark:text-slate-300">Color</span>
            <div class="flex flex-wrap gap-1.5">
              <button
                v-for="c in colorOptions"
                :key="c"
                type="button"
                @click="color = color === c ? '' : c"
                class="h-6 w-6 rounded-full border-2 transition-all"
                :class="color === c ? 'border-slate-900 dark:border-white scale-110 shadow-md' : 'border-transparent hover:scale-110'"
                :style="{ backgroundColor: c }"
                :title="c"
              />
            </div>
          </div>
        </div>

        <!-- Row 4: Tags -->
        <div>
          <span class="mb-1.5 block text-xs font-black text-slate-700 dark:text-slate-300">Tags</span>
          <TodoTagInput
            v-model="tags"
            :suggestions="existingTags"
            @remove-tag="removeTag"
            @add-suggestion="addTag"
          />
        </div>
      </div>

      <!-- Footer: toggle + submit -->
      <div class="mt-4 flex flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-between">
        <button
          type="button"
          @click="moreOpen = !moreOpen"
          class="inline-flex items-center justify-center gap-1.5 rounded-lg px-3 py-2 text-xs font-black text-slate-500 transition hover:bg-gray-100 hover:text-slate-700 dark:text-slate-400 dark:hover:bg-slate-700 dark:hover:text-slate-200"
          :aria-expanded="moreOpen"
        >
          <svg class="h-3.5 w-3.5 transition-transform" :class="{ 'rotate-180': moreOpen }" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="m6 9 6 6 6-6" />
          </svg>
          {{ moreOpen ? 'Hide details' : 'Add details, scheduling & more' }}
        </button>
        <button
          type="submit"
          class="inline-flex items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-indigo-600 to-violet-600 px-5 py-2.5 text-sm font-black text-white shadow-lg shadow-indigo-500/25 transition hover:-translate-y-0.5 hover:shadow-xl disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:translate-y-0"
          :disabled="!title.trim()"
        >
          Create todo
          <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="m9 18 6-6-6-6" />
          </svg>
        </button>
      </div>
    </div>
  </form>
</template>

<script setup lang="ts">
import { nextTick, ref, type PropType } from 'vue'
import type { TodoPriority } from '../../types'
import TodoPrioritySelect from './TodoPrioritySelect.vue'
import TodoDueDatePicker from './TodoDueDatePicker.vue'
import TodoTagInput from './TodoTagInput.vue'

const props = defineProps({
  existingTags: { type: Array as PropType<string[]>, default: () => [] },
})

const emit = defineEmits<{
  submit: [data: {
    title: string
    description?: string
    priority?: TodoPriority
    start_date?: string | null
    end_date?: string | null
    domain?: string
    color?: string
    rrule?: string | null
    tags: string[]
  }]
}>()

const title = ref('')
const description = ref('')
const priority = ref<TodoPriority>('medium')
const startDate = ref<string | null>(null)
const endDate = ref<string | null>(null)
const domain = ref('')
const color = ref('')
const rrule = ref('')
const customRrule = ref('')
const tags = ref<string[]>([])
const moreOpen = ref(false)
const titleInput = ref<HTMLInputElement | null>(null)

const defaultDomains = ['Work', 'Personal', 'Health', 'Finance', 'Education', 'Shopping', 'Errands', 'Projects']
const colorOptions = ['#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#8b5cf6', '#ec4899', '#06b6d4', '#f97316', '#6366f1', '#14b8a6']

function onSubmit() {
  if (!title.value.trim()) return
  const finalRrule = rrule.value === 'custom' ? customRrule.value : rrule.value
  emit('submit', {
    title: title.value.trim(),
    description: description.value.trim() || undefined,
    priority: priority.value,
    start_date: startDate.value,
    end_date: endDate.value,
    domain: domain.value.trim() || undefined,
    color: color.value || undefined,
    rrule: finalRrule || undefined,
    tags: tags.value,
  })
  reset()
}

function removeTag(tag: string) {
  tags.value = tags.value.filter(item => item !== tag)
}

function addTag(tag: string) {
  if (!tags.value.includes(tag)) tags.value = [...tags.value, tag]
}

function reset() {
  title.value = ''
  description.value = ''
  priority.value = 'medium'
  startDate.value = null
  endDate.value = null
  domain.value = ''
  color.value = ''
  rrule.value = ''
  customRrule.value = ''
  tags.value = []
  nextTick(() => titleInput.value?.focus())
}
</script>
