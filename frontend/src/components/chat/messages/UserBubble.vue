<template>
  <article
    class="group relative ml-auto max-w-[90%] rounded-xl rounded-br-sm bg-indigo-600 px-3.5 py-2.5 text-white shadow-sm ring-1 ring-indigo-500/40 ring-inset focus-within:outline-none sm:max-w-[82%] dark:bg-indigo-600 dark:ring-indigo-400/30"
  >
    <ImageAttachmentStrip
      :attachments="message.attachments ?? []"
      :conversation-id="message.conversation_id"
    />

    <pre class="font-sans text-[0.92em] leading-[1.6] break-words whitespace-pre-wrap">{{
      message.content
    }}</pre>

    <BubbleActions
      :include="includeActions"
      :active="speakActive ? ['speak'] : []"
      :busy="speakBusy ? ['speak'] : []"
      @action="onAction"
    />
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { AIMessage } from '@browser-server/shared-types';
import BubbleActions, { type BubbleActionName } from './BubbleActions.vue';
import ImageAttachmentStrip from './ImageAttachmentStrip.vue';

const props = withDefaults(
  defineProps<{
    message: AIMessage;
    ttsAvailable?: boolean;
    speakBusy?: boolean;
    speakActive?: boolean;
  }>(),
  { ttsAvailable: true, speakBusy: false, speakActive: false },
);

const includeActions = computed<BubbleActionName[]>(() =>
  props.ttsAvailable ? ['copy', 'speak', 'branch', 'delete'] : ['copy', 'branch', 'delete'],
);

const emit = defineEmits<{
  copy: [content: string];
  delete: [messageId: string];
  branch: [messageId: string];
  speak: [payload: { messageId: string; content: string }];
}>();

function onAction(name: BubbleActionName) {
  if (name === 'copy') emit('copy', props.message.content);
  else if (name === 'branch') emit('branch', props.message.id);
  else if (name === 'speak')
    emit('speak', { messageId: props.message.id, content: props.message.content });
  else emit('delete', props.message.id);
}
</script>
