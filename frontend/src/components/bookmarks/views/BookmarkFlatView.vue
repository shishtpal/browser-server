<template>
  <div>
    <!-- Desktop table (md+) -->
    <div
      class="hidden overflow-x-auto rounded-xl border border-gray-200/80 bg-white/90 shadow-sm transition-colors md:block dark:border-slate-700/80 dark:bg-slate-800/90"
    >
      <table
        class="w-full table-fixed divide-y divide-gray-200 transition-colors dark:divide-slate-700"
      >
        <colgroup>
          <col class="w-10" />
          <col class="w-[22%]" />
          <col class="w-[24%]" />
          <col class="w-[16%]" />
          <col class="w-[12%]" />
          <col class="w-[16%]" />
          <col class="w-20" />
        </colgroup>
        <thead class="bg-gray-50 transition-colors dark:bg-slate-800/80">
          <tr>
            <th v-for="col in columns" :key="col.label" class="px-3 py-3" :class="col.headerClass">
              {{ col.label }}
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 transition-colors dark:divide-slate-700/50">
          <BookmarkTableRow
            v-for="b in bookmarks"
            :key="b.id"
            :bookmark="b"
            @edit="$emit('edit', $event)"
            @delete="$emit('delete', $event)"
            @filter-tag="$emit('filter-tag', $event)"
          />
        </tbody>
      </table>
    </div>

    <!-- Mobile cards (md-) -->
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 md:hidden">
      <BookmarkCard
        v-for="b in bookmarks"
        :key="b.id"
        :bookmark="b"
        @edit="$emit('edit', $event)"
        @delete="$emit('delete', $event)"
        @filter-tag="$emit('filter-tag', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { BookmarkResponse } from '../../../types';
import BookmarkCard from './BookmarkCard.vue';
import BookmarkTableRow from './BookmarkTableRow.vue';

defineProps<{ bookmarks: BookmarkResponse[] }>();

defineEmits<{
  edit: [bookmark: BookmarkResponse];
  delete: [id: number];
  'filter-tag': [tag: string];
}>();

const headerCellClass =
  'px-3 py-3 text-left text-[10px] font-black uppercase tracking-wide text-slate-500 transition-colors dark:text-slate-400';

const columns = [
  { label: '', headerClass: '' },
  { label: 'Title', headerClass: headerCellClass },
  { label: 'URL', headerClass: headerCellClass },
  { label: 'Description', headerClass: headerCellClass },
  { label: 'Folder', headerClass: headerCellClass },
  { label: 'Tags', headerClass: headerCellClass },
  { label: 'Actions', headerClass: `${headerCellClass} text-right` },
];
</script>
