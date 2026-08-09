<template>
  <ImportCard
    title="Import from Chrome"
    description="Upload a Chrome history file (no extension)"
    noun="history entry"
    noun-plural="history entries"
    color="violet"
    :on-import="onImport"
    @imported="$emit('imported')"
  />
</template>

<script setup lang="ts">
import { importHistory } from '../../lib/api';
import ImportCard, { type ImportSummary } from '../ui/ImportCard.vue';

const props = defineProps<{ selectedUserId: number }>();
defineEmits<{ imported: [] }>();

const onImport = (file: File): Promise<ImportSummary> => importHistory(props.selectedUserId, file);
</script>
