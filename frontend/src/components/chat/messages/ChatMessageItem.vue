<template>
  <!-- Dispatches on message role; keeps every event signature shared. -->
  <UserBubble
    v-if="message.role === 'user'"
    :message="message"
    @copy="$emit('copy', $event)"
    @delete="$emit('delete', $event)"
    @branch="$emit('branch', $event)"
  />

  <AssistantBubble
    v-else-if="message.role === 'assistant'"
    :message="message"
    :show-thinking="showThinking"
    @copy="$emit('copy', $event)"
    @delete="$emit('delete', $event)"
    @branch="$emit('branch', $event)"
  />

  <ToolMessageCard
    v-else-if="message.role === 'tool'"
    :message="message"
    @copy="$emit('copy', $event)"
    @delete="$emit('delete', $event)"
    @tool-decision="
      (callId, approved, comment) => $emit('tool-decision', callId, approved, comment)
    "
  />

  <!-- Unknown roles render as plain assistant-style text -->
  <article
    v-else
    class="max-w-[96%] rounded-xl rounded-bl-sm border border-slate-200/80 bg-white px-4 py-3 text-[0.92em] break-words whitespace-pre-wrap shadow-sm sm:max-w-[92%] dark:border-white/10 dark:bg-slate-900/70"
  >
    {{ message.content }}
  </article>
</template>

<script setup lang="ts">
import type { AIMessage } from '@browser-server/shared-types';
import UserBubble from './UserBubble.vue';
import AssistantBubble from './AssistantBubble.vue';
import ToolMessageCard from './ToolMessageCard.vue';

withDefaults(
  defineProps<{
    message: AIMessage;
    showThinking?: boolean;
  }>(),
  { showThinking: true },
);

defineEmits<{
  copy: [content: string];
  delete: [messageId: string];
  branch: [messageId: string];
  'tool-decision': [callId: string, approved: boolean, comment: string];
}>();
</script>
