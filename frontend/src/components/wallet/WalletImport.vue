<template>
  <ImportCard
    title="Import from browser"
    description="Upload a Chrome-based passwords CSV export"
    accept=".csv,text/csv"
    noun="entry"
    noun-plural="entries"
    color="emerald"
    :on-import="onImport"
    @imported="$emit('imported')"
  />
</template>

<script setup lang="ts">
import { importWallet } from '../../lib/api';
import ImportCard, { type ImportSummary } from '../ui/ImportCard.vue';

const props = defineProps<{ selectedUserId: number }>();
defineEmits<{ imported: [] }>();

const onImport = (file: File): Promise<ImportSummary> => importWallet(props.selectedUserId, file);
</script>
