<template>
  <svg v-if="name in paths" :class="sizes[size]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true"><path :d="paths[name as keyof typeof paths]" stroke-linecap="round" stroke-linejoin="round" /></svg>
  <Icon v-else :name="name as ExistingIcon" :size="size" />
</template>
<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'
type ExistingIcon = InstanceType<typeof Icon>['$props']['name']
withDefaults(defineProps<{ name: ExistingIcon | keyof typeof paths; size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' }>(), { size: 'md' })
const sizes = { xs: 'w-3 h-3', sm: 'w-4 h-4', md: 'w-5 h-5', lg: 'w-6 h-6', xl: 'w-8 h-8' }
const paths = {
  selection: 'M4 4h16v16H4z',
  eraser: 'm16 3 5 5-11 11H5l-3-3L15 3zM8 10l6 6M10 21h11',
  undo: 'M8 4 3 9l5 5M3 9h10a7 7 0 0 1 0 14',
  circle: 'M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0',
  square: 'M4 4h16v16H4z',
}
</script>
