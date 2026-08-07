<template>
  <form class="space-y-3 rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800/60" @submit.prevent="submit">
    <label class="block">
      <span class="mb-1 block text-[11px] font-bold text-slate-600 dark:text-slate-400">Paper title</span>
      <input v-model="title" type="text" required placeholder="e.g. SSC Mock Test 1" class="w-full rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200" />
    </label>

    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <span class="text-[11px] font-bold text-slate-600 dark:text-slate-400">Sections (filters + question count)</span>
        <button type="button" class="rounded-md bg-violet-100 px-2 py-0.5 text-[11px] font-bold text-violet-700 transition hover:bg-violet-200 dark:bg-violet-900/30 dark:text-violet-300" @click="addSection">+ Add section</button>
      </div>

      <div v-for="(section, i) in sections" :key="i" class="grid grid-cols-2 gap-2 rounded-lg border border-gray-100 bg-slate-50/60 p-2 sm:grid-cols-7 dark:border-slate-700 dark:bg-slate-900/40">
        <div class="flex flex-col gap-1">
          <span class="text-[10px] font-bold text-slate-500 dark:text-slate-400">Tags</span>
          <div class="flex flex-wrap items-center gap-1 rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200">
            <span
              v-for="tag in section.tags ?? []"
              :key="tag"
              class="flex items-center gap-1 rounded bg-violet-100 px-1.5 py-0.5 text-[10px] font-semibold text-violet-800 dark:bg-violet-900/40 dark:text-violet-200"
            >
              {{ tag }}
              <button type="button" class="text-violet-600 hover:text-violet-800 dark:text-violet-300" @click="removeSectionTag(i, tag)">×</button>
            </span>
            <input
              v-model="sectionTagDrafts[i]"
              type="text"
              list="paper-section-tags"
              placeholder="SSC"
              class="flex-1 min-w-[6ch] border-0 bg-transparent p-0 text-xs focus:outline-none dark:text-slate-200"
              @keydown.enter.prevent="commitSectionTag(i)"
              @keydown.,.prevent="commitSectionTag(i)"
              @blur="commitSectionTag(i)"
            />
          </div>
        </div>
        <select v-model="section.subject" class="rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200">
          <option value="">Any subject</option>
          <option v-for="v in vocabulary?.subjects ?? []" :key="v" :value="v">{{ v }}</option>
        </select>
        <select v-model="section.topic" class="rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200">
          <option value="">Any topic</option>
          <option v-for="v in vocabulary?.topics ?? []" :key="v" :value="v">{{ v }}</option>
        </select>
        <select v-model="section.type" class="rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200">
          <option value="">Any type</option>
          <option value="single_choice">Single choice</option>
          <option value="multiple_choice">Multiple choice</option>
          <option value="input">Input</option>
          <option value="chronology">Chronology</option>
        </select>
        <select v-model="section.difficulty" class="rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200">
          <option value="">Any difficulty</option>
          <option value="easy">Easy</option>
          <option value="medium">Medium</option>
          <option value="hard">Hard</option>
        </select>
        <input v-model.number="section.count" type="number" min="1" max="200" required placeholder="Count" class="rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200" />
        <button type="button" class="rounded-lg px-2 py-1.5 text-xs font-bold text-rose-500 transition hover:bg-rose-50 disabled:opacity-40 dark:hover:bg-rose-900/20" :disabled="sections.length <= 1" @click="sections.splice(i, 1)">Remove</button>
      </div>
      <datalist id="paper-section-tags">
        <option v-for="v in vocabulary?.tags ?? []" :key="v" :value="v" />
      </datalist>
    </div>

    <p class="text-[11px] text-slate-500 dark:text-slate-400">Total requested: <span class="font-black">{{ totalCount }}</span> questions</p>

    <div class="flex justify-end">
      <button type="submit" :disabled="isGenerating" class="rounded-lg bg-violet-600 px-4 py-1.5 text-xs font-bold text-white transition hover:bg-violet-700 disabled:opacity-50">
        {{ isGenerating ? 'Generating…' : 'Generate paper' }}
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import type { QuestionPaperSection, TagVocabulary } from '../../types'

defineProps<{
  vocabulary?: TagVocabulary | null
  isGenerating?: boolean
}>()

const emit = defineEmits<{
  generate: [input: { title: string; sections: QuestionPaperSection[] }]
}>()

const title = ref('')
const sections = reactive<QuestionPaperSection[]>([{ tags: [], subject: '', topic: '', type: undefined, difficulty: undefined, count: 10 } as QuestionPaperSection])
const sectionTagDrafts = ref<string[]>(sections.map(() => ''))

watch(sections, (current) => {
  while (sectionTagDrafts.value.length < current.length) sectionTagDrafts.value.push('')
  while (sectionTagDrafts.value.length > current.length) sectionTagDrafts.value.pop()
}, { deep: true })

const addSection = () => {
  sections.push({ tags: [], subject: '', topic: '', type: undefined, difficulty: undefined, count: 10 } as QuestionPaperSection)
  sectionTagDrafts.value.push('')
}

function commitSectionTag(i: number) {
  const draft = sectionTagDrafts.value[i]
  if (!draft) return
  const value = draft.trim()
  if (!value) {
    sectionTagDrafts.value[i] = ''
    return
  }
  const tags = sections[i].tags ?? []
  if (!tags.includes(value)) {
    sections[i].tags = [...tags, value]
  }
  sectionTagDrafts.value[i] = ''
}

function removeSectionTag(i: number, tag: string) {
  const tags = sections[i].tags ?? []
  sections[i].tags = tags.filter((t) => t !== tag)
}

const totalCount = computed(() => sections.reduce((sum, s) => sum + (s.count || 0), 0))

const submit = () => {
  // Pick up any tag the user typed but didn't commit with Enter.
  sections.forEach((_, i) => commitSectionTag(i))
  const cleaned = sections.map((s) => {
    const out: QuestionPaperSection = { count: s.count }
    if (s.tags && s.tags.length) out.tags = s.tags
    if (s.subject) out.subject = s.subject
    if (s.topic) out.topic = s.topic
    if (s.type) out.type = s.type
    if (s.difficulty) out.difficulty = s.difficulty
    return out
  })
  emit('generate', { title: title.value.trim(), sections: cleaned })
  title.value = ''
}
</script>
