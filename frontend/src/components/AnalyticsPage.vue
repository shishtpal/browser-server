<template>
  <div class="mx-auto max-w-full px-3 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Domain time" title="Usage" color="rose">
      <template #stats>
        <StatCard :value="totalDuration" label="Duration" variant="primary" color="rose" />
        <StatCard :value="domainCount" label="Domains" variant="dark" color="rose" />
      </template>
      <template #controls>
        <UserSelector id="analytics-user" v-model="selectedUserId" :users="users" color="rose" />
      </template>
    </PageHeader>

    <SelectUserPrompt
      title="Select a user to view usage analytics"
      :users-count="users.length"
      :selected-user-id="selectedUserId"
    />

    <LoadingSpinner v-if="isLoading" message="Loading usage data..." color="rose" />
    <ErrorBanner v-else-if="error" :message="error" :on-retry="load" />

    <div v-else-if="selectedUserId">
      <UsageToolbar
        v-model:date-preset="datePreset"
        v-model:custom-start="customStart"
        v-model:custom-end="customEnd"
        v-model:group-by="groupBy"
      />

      <EmptyState
        v-if="isEmpty"
        title="No activity tracked"
        description="Your browsing time will appear here once the extension starts tracking."
        icon="chart"
        color="rose"
      />

      <div v-else class="space-y-4 sm:space-y-6">
        <DomainBreakdown :domains="summary?.domains ?? []" />

        <UsageTrendChart
          v-if="summary?.timeline.length"
          :points="summary.timeline"
          :labels="timelineLabels"
          :max-value="maxTimelineValue"
          :group-by="groupBy"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { useUser } from '../composables/useUser';
import { useAnalytics } from './analytics/composables/useAnalytics';
import PageHeader from './ui/PageHeader.vue';
import StatCard from './ui/StatCard.vue';
import UserSelector from './ui/UserSelector.vue';
import LoadingSpinner from './ui/LoadingSpinner.vue';
import ErrorBanner from './ui/ErrorBanner.vue';
import EmptyState from './ui/EmptyState.vue';
import SelectUserPrompt from './ui/SelectUserPrompt.vue';
import UsageToolbar from './analytics/UsageToolbar.vue';
import DomainBreakdown from './analytics/DomainBreakdown.vue';
import UsageTrendChart from './analytics/UsageTrendChart.vue';

const { users, currentUserId, setUser, clearUser } = useUser();
const selectedUserId = ref<number | null>(currentUserId.value);

const {
  summary,
  isLoading,
  error,
  datePreset,
  customStart,
  customEnd,
  groupBy,
  totalDuration,
  domainCount,
  maxTimelineValue,
  timelineLabels,
  isEmpty,
  load,
} = useAnalytics(selectedUserId);

watch(selectedUserId, (id) => {
  if (id) setUser(id);
  else clearUser();
});

// Data loads automatically via the immediate watcher inside useAnalytics.
</script>
