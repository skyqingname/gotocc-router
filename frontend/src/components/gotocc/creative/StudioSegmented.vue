<template>
  <div v-segmented class="segmented inline-flex" :class="block && 'w-full'" role="group" :aria-label="ariaLabel">
    <button v-for="option in options" :key="option.value" type="button" class="segmented-item h-8 min-w-0 px-3 text-sm font-medium" :class="[block && 'flex-1', modelValue === option.value && 'segmented-item-active']" :aria-pressed="modelValue === option.value" @click="$emit('update:modelValue', option.value)">{{ option.label }}</button>
  </div>
</template>
<script setup lang="ts">
import { vSegmented } from '@/components/gotocc/vSegmented'
defineProps<{ modelValue: string; options: { value: string; label: string }[]; ariaLabel: string; block?: boolean }>()
defineEmits<{ 'update:modelValue': [value: string] }>()
</script>
