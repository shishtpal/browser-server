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

    <BubbleActions @action="onAction" />
  </article>
</template>

<script setup lang="ts">
import type { AIMessage } from '@browser-server/shared-types';
import BubbleActions, { type BubbleActionName } from './BubbleActions.vue';
import ImageAttachmentStrip from './ImageAttachmentStrip.vue';

const props = defineProps<{ message: AIMessage }>();

const emit = defineEmits<{
  copy: [content: string];
  delete: [messageId: string];
  branch: [messageId: string];
}>();

function onAction(name: BubbleActionName) {
  if (name === 'copy') emit('copy', props.message.content);
  else if (name === 'branch') emit('branch', props.message.id);
  else emit('delete', props.message.id);
}
</script>
