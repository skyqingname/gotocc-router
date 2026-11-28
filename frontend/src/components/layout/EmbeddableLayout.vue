<template>
  <!-- 页面单独打开时套控制台外框；作为别的页面的选项卡内容嵌入时只输出内容本身。 -->
  <slot v-if="embedded" />
  <AppLayout v-else :appearance="appearance" :workspace="workspace"><slot /></AppLayout>
</template>

<script setup lang="ts">
import AppLayout from './AppLayout.vue'

defineProps<{ embedded?: boolean; appearance?: 'user' | 'admin'; workspace?: boolean }>()
</script>
