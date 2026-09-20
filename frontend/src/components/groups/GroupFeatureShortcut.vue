<template>
  <RouterLink
    v-if="auth.isAdmin && route.name === 'AdminGroups'"
    to="/admin/group-features"
    class="btn btn-primary fixed bottom-6 right-6 z-20 shadow-lg"
  >{{ t('open') }}</RouterLink>
</template>

<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const auth = useAuthStore()
const { t } = useI18n({
  useScope: 'local', inheritLocale: true,
  messages: { en: { open: 'Configure time-window rates' }, zh: { open: '配置分组时段倍率' } }
})
</script>
