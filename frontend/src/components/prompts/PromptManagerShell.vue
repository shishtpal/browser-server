<template>
  <Teleport v-if="mounted" to="body">
    <Transition name="pm-fade">
      <div v-if="open"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-0 backdrop-blur-sm sm:p-4"
        @click.self="$emit('close')">
        <div
          class="chat-shell flex h-full w-full flex-col overflow-hidden bg-white text-slate-900 shadow-2xl shadow-black/30 dark:bg-slate-950 dark:text-slate-100 sm:h-[92vh] sm:max-w-[1400px] sm:rounded-2xl sm:border sm:border-slate-200 sm:dark:border-white/10"
          role="dialog" aria-modal="true" aria-label="Prompt Manager">
          <slot />
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'

defineProps<{ open: boolean }>()
defineEmits<{ close: [] }>()

const mounted = ref(false)
onMounted(() => {
  mounted.value = true
})
</script>

<style scoped>
.pm-fade-enter-active,
.pm-fade-leave-active {
  transition: opacity 0.18s ease;
}

.pm-fade-enter-from,
.pm-fade-leave-to {
  opacity: 0;
}
</style>
