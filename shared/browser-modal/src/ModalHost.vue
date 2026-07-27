<script setup lang="ts">
import { stack } from './service'
import ModalDialog from './ModalDialog.vue'
</script>

<template>
  <Teleport to="body">
    <ModalDialog
      v-for="(item, i) in stack"
      :key="item.id"
      :instance="item"
      :is-top="i === stack.length - 1"
    />
  </Teleport>
</template>
