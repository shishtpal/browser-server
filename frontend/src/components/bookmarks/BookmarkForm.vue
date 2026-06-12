<template>
  <form @submit.prevent="onSubmit" class="rounded-xl border border-gray-200 bg-white p-3 shadow-sm transition-colors dark:border-white/10 dark:bg-slate-800/90">
    <div class="flex items-center gap-2">
      <InputField v-model="newTitle" type="text" placeholder="Title" required flex color="cyan" />
      <InputField v-model="newUrl" type="url" placeholder="URL" required flex color="cyan" />
      <InputField v-model="newDescription" type="text" placeholder="Description" class="hidden lg:block" flex color="cyan" />
      <InputField v-model="newTags" type="text" placeholder="Tags: comma, separated" flex color="cyan" />
      <Button type="submit" variant="gradient-cyan" size="sm">Add</Button>
    </div>
  </form>
</template>

<script setup lang="ts">
import InputField from '../ui/InputField.vue'
import Button from '../ui/Button.vue'

const newTitle = defineModel<string>('newTitle', { required: true })
const newUrl = defineModel<string>('newUrl', { required: true })
const newDescription = defineModel<string>('newDescription', { required: true })
const newTags = defineModel<string>('newTags', { required: true })

const emit = defineEmits<{
  submit: []
}>()

const onSubmit = () => emit('submit')
</script>
