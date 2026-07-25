<template>
  <form
    @submit.prevent="onSubmit"
    @keydown.meta.enter.prevent="onSubmit"
    @keydown.ctrl.enter.prevent="onSubmit"
    class="@container relative isolate overflow-hidden rounded-2xl border border-slate-200/80 bg-white shadow-xs
           dark:border-white/10 dark:bg-slate-900"
  >
    <!-- Ambient glow (v4 radial gradient syntax) -->
    <div
      aria-hidden="true"
      class="pointer-events-none absolute inset-x-0 -top-20 h-32 bg-radial-[at_50%_0%] from-indigo-500/15 to-transparent to-70%"
    />

    <!-- ── Header ─────────────────────────────────────────── -->
    <header class="flex items-center gap-2.5 border-b border-slate-200/70 px-3 py-2.5 @sm:px-4 dark:border-white/10">
      <span class="grid size-7 shrink-0 place-items-center rounded-lg bg-linear-to-br from-indigo-500 to-violet-600 text-white shadow-sm shadow-indigo-500/30">
        <svg class="size-4" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 5v14m-7-7h14" />
        </svg>
      </span>

      <h2 class="truncate text-sm font-semibold text-slate-900 dark:text-slate-100">New todo</h2>

      <kbd class="ml-auto hidden rounded-md border border-slate-200 px-1.5 py-0.5 font-mono text-[10px] text-slate-400 @sm:block dark:border-white/10 dark:text-slate-500">
        ⌘ ↵
      </kbd>
    </header>

    <!-- ── Quick add row ──────────────────────────────────── -->
    <div class="p-3 @sm:p-4">
      <div class="flex flex-col gap-2 @sm:flex-row">
        <div class="relative flex-1">
          <label :for="ids.title" class="sr-only">Task title</label>
          <input
            :id="ids.title"
            ref="titleInput"
            v-model="title"
            :class="cls.input"
            class="peer h-10 pr-16 font-medium"
            placeholder="What needs to be done?"
            maxlength="200"
            autocomplete="off"
            enterkeyhint="done"
            required
          />
          <span
            class="pointer-events-none absolute inset-y-0 right-2.5 grid place-items-center text-[10px] tabular-nums text-slate-400 opacity-0 transition-opacity peer-focus:opacity-100"
          >
            {{ title.length }}/200
          </span>
        </div>

        <button type="submit" :disabled="!canSubmit" :class="cls.primary">
          <svg class="size-4" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 5v14m-7-7h14" />
          </svg>
          Create
        </button>
      </div>

      <!-- ── Chip bar: toggle · quick dates · summary ──────── -->
      <div class="mt-2 flex flex-wrap items-center gap-1.5">
        <button
          type="button"
          :class="[cls.chip, moreOpen && cls.chipOn]"
          :aria-expanded="moreOpen"
          :aria-controls="ids.details"
          @click="moreOpen = !moreOpen"
        >
          <svg
            class="size-3.5 transition-transform duration-200"
            :class="{ 'rotate-180': moreOpen }"
            fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24" aria-hidden="true"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="m6 9 6 6 6-6" />
          </svg>
          Details
        </button>

        <span class="mx-0.5 h-4 w-px bg-slate-200 dark:bg-white/10" />

        <button
          v-for="q in quickDates"
          :key="q.label"
          type="button"
          :class="[cls.chip, endDate === q.value && cls.chipOn]"
          @click="endDate = endDate === q.value ? null : q.value"
        >
          {{ q.label }}
        </button>

        <!-- Collapsed summary so nothing is silently hidden -->
        <template v-if="!moreOpen">
          <span v-for="s in summary" :key="s.key" :class="cls.badge">
            <span v-if="s.dot" class="size-2 rounded-full" :style="{ backgroundColor: s.dot }" />
            {{ s.text }}
          </span>
        </template>

        <button v-if="isDirty" type="button" :class="[cls.chip, 'ml-auto']" @click="reset">Clear</button>
      </div>

      <!-- ── Collapsible details (0fr → 1fr animation) ─────── -->
      <div
        class="-mx-1 grid transition-[grid-template-rows] duration-300 ease-out motion-reduce:transition-none"
        :class="moreOpen ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'"
      >
        <div
          :id="ids.details"
          :inert="!moreOpen ? true : undefined"
          class="overflow-hidden px-1 pb-1"
        >
          <div
            class="mt-3 space-y-3 border-t border-slate-200/70 pt-3 transition-opacity duration-200 dark:border-white/10"
            :class="moreOpen ? 'opacity-100' : 'opacity-0'"
          >
            <!-- Description (auto-growing textarea, v4 field-sizing) -->
            <div>
              <label :for="ids.desc" :class="cls.label">Description</label>
              <textarea
                :id="ids.desc"
                v-model="description"
                :class="cls.input"
                class="field-sizing-content max-h-40 min-h-16 resize-none py-2 leading-relaxed"
                placeholder="Notes, context, links…"
              />
            </div>

            <!-- Priority · Domain · Start · End -->
            <div class="grid gap-3 @md:grid-cols-2 @3xl:grid-cols-4">
              <div>
                <span :class="cls.label">Priority</span>
                <TodoPrioritySelect v-model="priority" />
              </div>

              <div>
                <label :for="ids.domain" :class="cls.label">Domain</label>
                <input
                  :id="ids.domain"
                  v-model="domain"
                  :list="ids.domains"
                  :class="cls.input"
                  class="h-9"
                  placeholder="Work, Personal…"
                  autocomplete="off"
                />
                <datalist :id="ids.domains">
                  <option v-for="c in defaultDomains" :key="c" :value="c" />
                </datalist>
              </div>

              <div>
                <span :class="cls.label">Start date</span>
                <TodoDueDatePicker v-model="startDate" />
              </div>

              <div>
                <span :class="cls.label">End date</span>
                <TodoDueDatePicker v-model="endDate" />
              </div>
            </div>

            <!-- Recurrence · Color -->
            <div class="grid gap-3 @md:grid-cols-[minmax(0,1fr)_auto]">
              <div>
                <label :for="ids.rrule" :class="cls.label">
                  <svg class="size-3 text-indigo-500" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24" aria-hidden="true">
                    <polyline points="17 1 21 5 17 9" /><path d="M3 11V9a4 4 0 0 1 4-4h14" />
                    <polyline points="7 23 3 19 7 15" /><path d="M21 13v2a4 4 0 0 1-4 4H3" />
                  </svg>
                  Recurrence
                </label>

                <div class="relative">
                  <select :id="ids.rrule" v-model="rrule" :class="cls.input" class="h-9 appearance-none pr-8">
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
                    class="size-6 rounded-full bg-(--sw) ring-offset-2 ring-offset-white outline-hidden transition
                           hover:scale-110 focus-visible:ring-2 focus-visible:ring-slate-400 dark:ring-offset-slate-900"
                    :class="color === c ? 'scale-110 ring-2 ring-slate-900 dark:ring-white' : 'ring-1 ring-black/10 dark:ring-white/15'"
                    @click="color = color === c ? '' : c"
                  />
                </div>
              </fieldset>
            </div>

            <!-- Tags -->
            <div>
              <span :class="cls.label">Tags</span>
              <TodoTagInput
                v-model="tags"
                :suggestions="existingTags"
                @remove-tag="removeTag"
                @add-suggestion="addTag"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  </form>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, useId, type PropType } from 'vue'
import type { TodoPriority } from '../../types'
import TodoPrioritySelect from './TodoPrioritySelect.vue'
import TodoDueDatePicker from './TodoDueDatePicker.vue'
import TodoTagInput from './TodoTagInput.vue'

defineProps({
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

/* ---------- shared class tokens (keeps the template readable) ---------- */
const cls = {
  label:
    'mb-1 flex items-center gap-1 text-[11px] font-semibold tracking-wide text-slate-500 uppercase dark:text-slate-400',
  input:
    'w-full rounded-lg border border-slate-200 bg-slate-50 px-3 text-sm text-slate-800 shadow-xs outline-hidden transition ' +
    'placeholder:text-slate-400 focus:border-indigo-400 focus:bg-white focus:ring-2 focus:ring-indigo-500/20 ' +
    'dark:border-white/10 dark:bg-white/5 dark:text-slate-100 dark:focus:border-indigo-400/60 dark:focus:bg-white/10',
  primary:
    'inline-flex h-10 w-full shrink-0 items-center justify-center gap-1.5 rounded-xl bg-linear-to-b from-indigo-500 to-indigo-600 ' +
    'px-4 text-sm font-semibold text-white shadow-sm shadow-indigo-600/25 outline-hidden transition ' +
    'hover:to-indigo-700 active:scale-[.98] focus-visible:ring-2 focus-visible:ring-indigo-500/50 focus-visible:ring-offset-2 ' +
    'disabled:pointer-events-none disabled:opacity-40 @sm:w-auto dark:focus-visible:ring-offset-slate-900',
  chip:
    'inline-flex items-center gap-1 rounded-full border border-slate-200 px-2.5 py-1 text-xs font-medium text-slate-600 ' +
    'outline-hidden transition hover:border-slate-300 hover:bg-slate-50 focus-visible:ring-2 focus-visible:ring-indigo-500/30 ' +
    'dark:border-white/10 dark:text-slate-300 dark:hover:bg-white/5',
  chipOn:
    'border-indigo-300 bg-indigo-50 text-indigo-700 dark:border-indigo-400/30 dark:bg-indigo-500/15 dark:text-indigo-300',
  badge:
    'inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-1 text-[11px] font-medium text-slate-600 dark:bg-white/5 dark:text-slate-300',
}

const ids = {
  title: useId(), desc: useId(), domain: useId(),
  domains: useId(), rrule: useId(), details: useId(),
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
const moreOpen = ref(false)
const titleInput = ref<HTMLInputElement | null>(null)

const defaultDomains = ['Work', 'Personal', 'Health', 'Finance', 'Education', 'Shopping', 'Errands', 'Projects']
const colorOptions = ['#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#8b5cf6', '#ec4899', '#06b6d4', '#f97316', '#6366f1', '#14b8a6']

const rruleOptions = [
  { value: '', label: 'No repeat' },
  { value: 'FREQ=DAILY', label: 'Daily' },
  { value: 'FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR', label: 'Every weekday' },
  { value: 'FREQ=WEEKLY', label: 'Weekly' },
  { value: 'FREQ=WEEKLY;INTERVAL=2', label: 'Every 2 weeks' },
  { value: 'FREQ=MONTHLY', label: 'Monthly' },
  { value: 'FREQ=YEARLY', label: 'Yearly' },
  { value: 'custom', label: 'Custom…' },
]

const iso = (offset: number) => {
  const d = new Date()
  d.setDate(d.getDate() + offset)
  return d.toISOString().slice(0, 10)
}
const quickDates = [
  { label: 'Today', value: iso(0) },
  { label: 'Tomorrow', value: iso(1) },
  { label: 'Next week', value: iso(7) },
]

/* ---------- derived ---------- */
const canSubmit = computed(() => title.value.trim().length > 0)

const isDirty = computed(() =>
  !!(title.value || description.value || domain.value || color.value || rrule.value ||
    startDate.value || endDate.value || tags.value.length || priority.value !== 'medium'),
)

const summary = computed(() => {
  const out: { key: string; text: string; dot?: string }[] = []
  if (priority.value !== 'medium') out.push({ key: 'p', text: `${priority.value} priority` })
  if (domain.value) out.push({ key: 'd', text: domain.value })
  if (startDate.value) out.push({ key: 's', text: `From ${startDate.value}` })
  if (endDate.value && !quickDates.some(q => q.value === endDate.value)) out.push({ key: 'e', text: `Due ${endDate.value}` })
  if (rrule.value) out.push({ key: 'r', text: 'Repeats' })
  if (tags.value.length) out.push({ key: 't', text: `${tags.value.length} tag${tags.value.length > 1 ? 's' : ''}` })
  if (color.value) out.push({ key: 'c', text: 'Color', dot: color.value })
  return out
})

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
