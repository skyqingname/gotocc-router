<template>
  <Transition :name="name" @before-enter="restoreElement" @before-leave="leaveElement"><slot /></Transition>
</template>
<script setup lang="ts">
defineProps<{ name: string }>()
function leaveElement(element: Element) { element.setAttribute('inert', '') }
function restoreElement(element: Element) { element.removeAttribute('inert') }
</script>
