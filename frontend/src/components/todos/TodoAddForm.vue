<template>
  <form
    @submit.prevent="onSubmit"
    @keydown.meta.enter.prevent="onSubmit"
    @keydown.ctrl.enter.prevent="onSubmit"
    class="@container relative isolate overflow-hidden rounded-2xl border border-slate-200/80 bg-white shadow-xs
           dark:border-white/10 dark:bg-slate-900"
  >
    <!-- Ambient glow -->
    <div
      aria-hidden="true"
      class="pointer-events-none absolute inset-x-0 -top-20 h-32 bg-radial-[at_50%_0%] from-indigo-500/15 to-transparent to-70%"
    />

    <!-- ── Header ─────────────────────────────────────────── -->
    <header class="flex items-center gap-3 px-4 pt-4 pb-3 @sm:px-5">
      <span class="grid size-8 shrink-0 place-items-center rounded-xl bg-linear-to-br from-indigo-500 to-violet-600 text-white shadow-sm shadow-indigo-500/30">
        <svg class="size-4" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 5v14m-7-7h14" />
        </svg>
      </span>

      <div class="min-w-0 leading-tight">
        <h2 class="truncate text-sm font-semibold text-slate-900 dark:text-slate-100">New todo</h2>
        <p class="truncate text-xs text-slate-400 dark:text-slate-500">Enter a title, add details as needed.</p>
      </div>

      <!-- Priority segmented control -->
      <fieldset class="ml-auto flex shrink-0 items-center gap-0.5 rounded-full bg-slate-100 p-0.5 dark:bg-white/5">
        <legend class="sr-only">Priority</legend>
        <button
          v-for="p in priorityOptions"
          :key="p.value"
          type="button"
          :aria-pressed="priority === p.value"
          :class="[cls.priorityBtn, priority === p.value ? cls.priorityBtnOn : cls.priorityBtnOff]"
          @click="priority = p.value"
        >
          <span class="size-1.5 rounded-full" :style="{ backgroundColor: p.dot }" />
          {{ p.label }}
        </button>
      </fieldset>
    </header>

    <!-- ── Body ───────────────────────────────────────────── -->
    <div class="px-4 pb-4 @sm:px-5">
      <!-- Title -->
      <div class="relative">
        <label :for="ids.title" class="sr-only">Task title</label>
        <input
          :id="ids.title"
          ref="titleInput"
          v-model="title"
          :class="cls.input"
          class="peer h-11 pr-14 text-[15px] font-medium"
          placeholder="What needs to be done?"
          maxlength="200"
          autocomplete="off"
          enterkeyhint="done"
          required
        />
        <span class="pointer-events-none absolute inset-y-0 right-3 grid place-items-center text-[10px] tabular-nums text-slate-400">
          {{ title.length }}/200
        </span>
      </div>

      <!-- Collapsible details -->
      <div
        class="grid transition-[grid-template-rows] duration-300 ease-out motion-reduce:transition-none"
        :class="moreOpen ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'"
      >
        <div :id="ids.details" :inert="!moreOpen ? true : undefined" class="overflow-hidden">
          <div
            class="space-y-3.5 pt-3.5 transition-opacity duration-200"
            :class="moreOpen ? 'opacity-100' : 'opacity-0'"
          >
            <!-- Description -->
            <div>
              <label :for="ids.desc" :class="cls.label">
                Description <span class="normal-case text-slate-300 dark:text-slate-600">· optional</span>
              </label>
              <textarea
                :id="ids.desc"
                v-model="description"
                :class="cls.input"
                class="field-sizing-content max-h-32 min-h-11 resize-none py-2.5 leading-relaxed"
                placeholder="Notes or context…"
              />
            </div>

            <!-- Category · Recurrence -->
            <div class="grid gap-3 @sm:grid-cols-2">
              <div>
                <label :for="ids.domain" :class="cls.label">Category</label>
                <input
                  :id="ids.domain"
                  v-model="domain"
                  :list="ids.domains"
                  :class="cls.input"
                  class="h-10"
                  placeholder="Work, Personal…"
                  autocomplete="off"
                />
                <datalist :id="ids.domains">
                  <option v-for="c in defaultDomains" :key="c" :value="c" />
                </datalist>
              </div>

              <div>
                <label :for="ids.rrule" :class="cls.label">
                  <svg class="size-3 text-indigo-500" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24" aria-hidden="true">
                    <polyline points="17 1 21 5 17 9" /><path d="M3 11V9a4 4 0 0 1 4-4h14" />
                    <polyline points="7 23 3 19 7 15" /><path d="M21 13v2a4 4 0 0 1-4 4H3" />
                  </svg>
                  Recurrence
                </label>
                <div class="relative">
                  <select :id="ids.rrule" v-model="rrule" :class="cls.input" class="h-10 appearance-none pr-8">
                    <option v-for="o in rruleOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
                  </select>
                  <svg class="pointer-events-none absolute inset-y-0 right-2.5 my-auto size-4 text-slate-400" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24" aria-hidden="true">
                    <path stroke-linecap="round" stroke-linejoin="round" d="m6 9 6 6 6-6" />
                  </svg>
                </div>
                <input
                  v-if="rrule === 'custom'"
                  v-model="customRrule"
                  :class="cls.input"
                  class="mt-1.5 h-8 font-mono text-xs"
                  placeholder="FREQ=WEEKLY;BYDAY=MO,WE,FR"
                />
              </div>
            </div>

            <!-- Start · End date -->
            <div class="grid gap-3 @sm:grid-cols-2">
              <div>
                <span :class="cls.label">Start date</span>
                <TodoDueDatePicker v-model="startDate" />
              </div>
              <div>
                <span :class="cls.label">End date</span>
                <TodoDueDatePicker v-model="endDate" />
              </div>
            </div>

            <!-- Color -->
            <fieldset>
              <legend :class="cls.label">Color</legend>
              <div class="flex flex-wrap gap-1.5 pt-0.5">
                <button
                  v-for="c in colorOptions"
                  :key="c"
                  type="button"
                  :aria-pressed="color === c"
                  :title="c"
                  :style="{ '--sw': c }"
                  class="size-6 rounded-full bg-(--sw) outline-hidden transition hover:scale-110 focus-visible:ring-2 focus-visible:ring-slate-400"
                  :class="color === c ? 'scale-110 ring-2 ring-offset-2 ring-slate-900 dark:ring-white dark:ring-offset-slate-900' : 'ring-1 ring-black/10 dark:ring-white/15'"
                  @click="color = color === c ? '' : c"
                />
              </div>
            </fieldset>

            <!-- Tags -->
            <div>
              <label :for="ids.tags" :class="cls.label">Tags</label>
              <input
                :id="ids.tags"
                v-model="tagDraft"
                :class="cls.input"
                class="h-10"
                placeholder="Type and press Enter…"
                autocomplete="off"
                @keydown.enter.prevent="commitTagDraft"
              />
              <div v-if="tags.length || existingTags.length" class="mt-1.5 flex flex-wrap gap-1.5">
                <span v-for="t in tags" :key="t" :class="cls.tagOn">
                  {{ t }}
                  <button type="button" class="opacity-60 hover:opacity-100" @click="removeTag(t)" :aria-label="`Remove ${t}`">
                    <svg class="size-3" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24"><path stroke-linecap="round" d="M18 6 6 18M6 6l12 12" /></svg>
                  </button>
                </span>
                <button
                  v-for="t in suggestibleTags"
                  :key="t"
                  type="button"
                  :class="cls.tagOff"
                  @click="addTag(t)"
                >
                  + {{ t }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="mt-3.5 flex items-center justify-between">
        <button type="button" :class="cls.toggle" @click="moreOpen = !moreOpen">
          <svg
            class="size-3.5 transition-transform duration-200"
            :class="{ 'rotate-180': !moreOpen }"
            fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24" aria-hidden="true"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="m18 15-6-6-6 6" />
          </svg>
          {{ moreOpen ? 'Hide details' : 'Show details' }}
        </button>

        <button type="submit" :disabled="!canSubmit" :class="cls.primary">
          Create
          <svg class="size-4" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 12h14m-6-6 6 6-6 6" />
          </svg>
        </button>
      </div>
    </div>
  </form>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, useId, type PropType } from 'vue'
import type { TodoPriority } from '../../types'
import TodoDueDatePicker from './TodoDueDatePicker.vue'

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

/* ---------- shared class tokens ---------- */
const cls = {
  label:
    'mb-1 flex items-center gap-1 text-[11px] font-semibold tracking-wide text-slate-500 uppercase dark:text-slate-400',
  input:
    'w-full rounded-lg border border-slate-200 bg-slate-50/70 px-3 text-sm text-slate-800 shadow-xs outline-hidden transition ' +
    'placeholder:text-slate-400 focus:border-indigo-400 focus:bg-white focus:ring-2 focus:ring-indigo-500/20 ' +
    'dark:border-white/10 dark:bg-white/5 dark:text-slate-100 dark:focus:border-indigo-400/60 dark:focus:bg-white/10',
  primary:
    'inline-flex h-9 items-center justify-center gap-1.5 rounded-full bg-linear-to-b from-indigo-500 to-violet-600 ' +
    'px-4 text-sm font-semibold text-white shadow-sm shadow-indigo-600/25 outline-hidden transition ' +
    'hover:to-violet-700 active:scale-[.98] focus-visible:ring-2 focus-visible:ring-indigo-500/50 focus-visible:ring-offset-2 ' +
    'disabled:pointer-events-none disabled:opacity-40 dark:focus-visible:ring-offset-slate-900',
  toggle:
    'inline-flex items-center gap-1 text-xs font-medium text-slate-500 outline-hidden transition ' +
    'hover:text-slate-700 focus-visible:ring-2 focus-visible:ring-indigo-500/30 rounded-md px-1 -mx-1 dark:text-slate-400 dark:hover:text-slate-200',
  priorityBtn:
    'inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium outline-hidden transition ' +
    'focus-visible:ring-2 focus-visible:ring-indigo-500/30',
  priorityBtnOn: 'bg-white text-slate-900 shadow-xs dark:bg-slate-800 dark:text-slate-100',
  priorityBtnOff: 'text-slate-400 hover:text-slate-600 dark:text-slate-500 dark:hover:text-slate-300',
  tagOn:
    'inline-flex items-center gap-1 rounded-full bg-indigo-50 px-2.5 py-1 text-xs font-medium text-indigo-700 ' +
    'dark:bg-indigo-500/15 dark:text-indigo-300',
  tagOff:
    'inline-flex items-center rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-500 transition ' +
    'hover:bg-slate-200 dark:bg-white/5 dark:text-slate-400 dark:hover:bg-white/10',
}

const ids = {
  title: useId(), desc: useId(), domain: useId(),
  domains: useId(), rrule: useId(), details: useId(), tags: useId(),
}

/* ---------- state ---------- */
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
const tagDraft = ref('')
const moreOpen = ref(false)
const titleInput = ref<HTMLInputElement | null>(null)

const priorityOptions: { value: TodoPriority; label: string; dot: string }[] = [
  { value: 'low', label: 'Low', dot: '#94a3b8' },
  { value: 'medium', label: 'Med', dot: '#3b82f6' },
  { value: 'high', label: 'High', dot: '#f59e0b' },
  { value: 'urgent', label: 'Urgent', dot: '#ef4444' },
]

const defaultDomains = ['Work', 'Personal', 'Health', 'Finance', 'Education', 'Shopping', 'Errands', 'Projects']
const colorOptions = ['#6366f1', '#3b82f6', '#06b6d4', '#10b981', '#22c55e', '#f59e0b', '#f97316', '#ef4444', '#ec4899', '#8b5cf6']

const rruleOptions = [
  { value: '', label: 'None' },
  { value: 'FREQ=DAILY', label: 'Daily' },
  { value: 'FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR', label: 'Every weekday' },
  { value: 'FREQ=WEEKLY', label: 'Weekly' },
  { value: 'FREQ=WEEKLY;INTERVAL=2', label: 'Every 2 weeks' },
  { value: 'FREQ=MONTHLY', label: 'Monthly' },
  { value: 'FREQ=YEARLY', label: 'Yearly' },
  { value: 'custom', label: 'Custom…' },
]

/* ---------- derived ---------- */
const canSubmit = computed(() => title.value.trim().length > 0)

const suggestibleTags = computed(() =>
  props.existingTags.filter(t => !tags.value.includes(t)),
)

/* ---------- actions ---------- */
function onSubmit() {
  if (!canSubmit.value) return
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

const removeTag = (tag: string) => (tags.value = tags.value.filter(t => t !== tag))
const addTag = (tag: string) => { if (!tags.value.includes(tag)) tags.value = [...tags.value, tag] }
function commitTagDraft() {
  const t = tagDraft.value.trim()
  if (t) addTag(t)
  tagDraft.value = ''
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
  tagDraft.value = ''
  nextTick(() => titleInput.value?.focus())
}
</script>