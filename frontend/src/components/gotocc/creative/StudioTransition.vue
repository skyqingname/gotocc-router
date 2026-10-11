<template><Transition :name="name"><slot /></Transition></template>
<script setup lang="ts">
defineProps<{ name: string }>()
</script>
