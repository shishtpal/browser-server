<template>
  <div>
    <div class="mb-3 flex flex-wrap items-center gap-1.5">
      <input
        :value="searchQuery"
        type="search"
        placeholder="Search questions…"
        class="w-full rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-xs sm:max-w-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        @input="$emit('update:searchQuery', ($event.target as HTMLInputElement).value)"
        @keyup.enter="$emit('applyFilters')"
      />
      <select
        :value="filterType"
        class="rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        @change="
          $emit('update:filterType', ($event.target as HTMLSelectElement).value);
          $emit('applyFilters');
        "
      >
        <option value="">All types</option>
        <option value="single_choice">Single choice</option>
        <option value="multiple_choice">Multiple choice</option>
        <option value="input">Input</option>
        <option value="chronology">Chronology</option>
      </select>
      <select
        :value="filterDifficulty"
        class="rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        @change="
          $emit('update:filterDifficulty', ($event.target as HTMLSelectElement).value);
          $emit('applyFilters');
        "
      >
        <option value="">All difficulties</option>
        <option value="easy">Easy</option>
        <option value="medium">Medium</option>
        <option value="hard">Hard</option>
      </select>
      <div class="relative">
        <button
          type="button"
          class="flex items-center gap-1 rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
          @click="tagPickerOpen = !tagPickerOpen"
        >
          <span>Tags {{ filterTags.length ? `(${filterTags.length})` : "" }}</span>
        </button>
        <div
          v-if="tagPickerOpen"
          class="absolute z-10 mt-1 w-56 rounded-lg border border-gray-300 bg-white p-2 shadow-lg dark:border-slate-600 dark:bg-slate-800"
        >
          <div class="mb-2 flex flex-wrap gap-1">
            <span
              v-for="tag in filterTags"
              :key="tag"
              class="flex items-center gap-1 rounded bg-violet-100 px-2 py-0.5 text-[10px] font-semibold text-violet-800 dark:bg-violet-900/40 dark:text-violet-200"
            >
              {{ tag }}
              <button
                type="button"
                class="text-violet-600 hover:text-violet-800 dark:text-violet-300"
                @click.stop="toggleTag(tag)"
              >
                ×
              </button>
            </span>
          </div>
          <div class="mb-2 flex gap-1">
            <input
              v-model="newTagDraft"
              type="text"
              placeholder="Add tag…"
              class="flex-1 rounded border border-gray-300 bg-white px-2 py-1 text-xs dark:border-slate-600 dark:bg-slate-700 dark:text-slate-200"
              @keydown.enter.prevent="addDraftTag"
              @keydown.,.prevent="addDraftTag"
            />
            <button
              type="button"
              class="rounded bg-slate-900 px-2 py-1 text-xs font-bold text-white dark:bg-white dark:text-slate-900"
              @click="addDraftTag"
            >
              +
            </button>
          </div>
          <div v-if="availableTags.length" class="max-h-40 space-y-0.5 overflow-y-auto">
            <label
              v-for="tag in availableTags"
              :key="tag"
              class="flex items-center gap-2 rounded px-1 py-0.5 text-xs hover:bg-slate-100 dark:hover:bg-slate-700"
            >
              <input type="checkbox" :checked="filterTags.includes(tag)" @change="toggleTag(tag)" />
              <span>{{ tag }}</span>
            </label>
          </div>
          <p v-else class="text-[10px] text-slate-400">No tags yet. Add some above.</p>
          <button
            type="button"
            class="mt-2 w-full rounded bg-slate-900 px-2 py-1 text-xs font-bold text-white dark:bg-white dark:text-slate-900"
            @click="applyAndClose"
          >
            Apply
          </button>
        </div>
      </div>
      <select
        v-if="vocabulary?.subjects?.length"
        :value="filterSubject"
        class="rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-xs dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        @change="
          $emit('update:filterSubject', ($event.target as HTMLSelectElement).value);
          $emit('applyFilters');
        "
      >
        <option value="">All subjects</option>
        <option v-for="subject in vocabulary.subjects" :key="subject" :value="subject">
          {{ subject }}
        </option>
      </select>
      <button
        type="button"
        class="rounded-lg bg-slate-900 px-3 py-1.5 text-xs font-bold text-white transition hover:bg-slate-700 dark:bg-white dark:text-slate-900 dark:hover:bg-slate-200"
        @click="$emit('applyFilters')"
      >
        Apply
      </button>
    </div>

    <EmptyState
      v-if="questions.length === 0"
      title="No questions yet"
      description="Add your first question using the form above."
      icon="search"
      color="violet"
    />

    <div v-else class="space-y-3">
      <QuestionCard
        v-for="q in questions"
        :key="q.id"
        :question="q"
        @edit="$emit('edit', $event)"
        @delete="$emit('delete', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import EmptyState from "../ui/EmptyState.vue";
import QuestionCard from "./QuestionCard.vue";
import type { QuestionResponse, TagVocabulary } from "../../types";

const props = defineProps<{
  questions: QuestionResponse[];
  vocabulary?: TagVocabulary | null;
  searchQuery: string;
  filterType: string;
  filterDifficulty: string;
  filterTags: string[];
  filterSubject: string;
}>();

const emit = defineEmits<{
  "update:searchQuery": [value: string];
  "update:filterType": [value: string];
  "update:filterDifficulty": [value: string];
  "update:filterTags": [value: string[]];
  "update:filterSubject": [value: string];
  applyFilters: [];
  edit: [question: QuestionResponse];
  delete: [id: number];
}>();

const tagPickerOpen = ref(false);
const newTagDraft = ref("");

const availableTags = computed(() => props.vocabulary?.tags ?? []);

function toggleTag(tag: string) {
  const next = props.filterTags.includes(tag)
    ? props.filterTags.filter((t) => t !== tag)
    : [...props.filterTags, tag];
  emit("update:filterTags", next);
}

function addDraftTag() {
  const value = newTagDraft.value.trim();
  if (!value) return;
  if (!props.filterTags.includes(value)) {
    emit("update:filterTags", [...props.filterTags, value]);
  }
  newTagDraft.value = "";
}

function applyAndClose() {
  emit("applyFilters");
  tagPickerOpen.value = false;
}
</script>
