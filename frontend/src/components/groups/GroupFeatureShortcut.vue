<template>
  <nav
    v-if="auth.isAdmin && route.name === 'AdminGroups'"
    class="fixed bottom-6 right-6 z-20 flex flex-wrap gap-2"
    :aria-label="t('label')"
  >
    <RouterLink to="/admin/group-features" class="btn btn-primary shadow-lg">{{ t('rates') }}</RouterLink>
    <RouterLink to="/admin/video-models" class="btn btn-secondary shadow-lg">{{ t('video') }}</RouterLink>
  </nav>
</template>

<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const auth = useAuthStore()
const { t } = useI18n({
  useScope: 'local', inheritLocale: true,
  messages: {
    en: { label: 'Group configuration', rates: 'Time-window rates', video: 'Video channel models' },
    zh: { label: '分组配置', rates: '分组时段倍率', video: 'Video 渠道模型' }
  }
})
</script>
