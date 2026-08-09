<template>
  <ImportCard
    title="Import from Chrome"
    description="Upload a bookmarks HTML export file"
    accept=".html,.htm"
    noun="bookmark"
    color="amber"
    :on-import="onImport"
    @imported="$emit('imported')"
  />
</template>

<script setup lang="ts">
import { importBookmarks } from '../../lib/api';
import ImportCard, { type ImportSummary } from '../ui/ImportCard.vue';

const props = defineProps<{ selectedUserId: number }>();
defineEmits<{ imported: [] }>();

const onImport = (file: File): Promise<ImportSummary> =>
  importBookmarks(props.selectedUserId, file);
</script>
