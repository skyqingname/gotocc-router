<template>
  <div class="studio-frame flex h-dvh flex-col overflow-hidden bg-white dark:bg-dark-950">
    <AppHeader>
      <template #leading>
        <div class="flex min-w-0 items-center gap-4">
          <router-link :to="homePath" class="flex min-w-0 items-center gap-2.5" :aria-label="app.siteName">
            <img :src="app.siteLogo || '/logo.svg'" alt="" class="h-8 w-8 flex-shrink-0 rounded-lg object-contain" />
            <span class="max-w-48 truncate text-base font-semibold text-gray-900 dark:text-white max-sm:max-w-20">{{ app.siteName }}</span>
          </router-link>
          <router-link :to="homePath" :aria-label="t('studioGotocc.backHome')" :title="t('studioGotocc.backHome')" class="flex h-8 w-8 flex-shrink-0 items-center justify-center text-gray-500 hover:text-primary-600"><Icon name="home" size="md" /></router-link>
        </div>
      </template>
    </AppHeader>
    <AdminSupportBanner v-if="supportUser" :target="supportUser" class="flex-shrink-0" />
    <main class="relative min-h-0 flex-1"><slot /></main>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore, useAuthStore } from '@/stores'
import { useUserView } from '@/composables/useUserView'
import { adminSupportContext } from '@/utils/adminSupportContext'
import AppHeader from '@/components/layout/AppHeader.vue'
import AdminSupportBanner from '@/components/admin/support/AdminSupportBanner.vue'
import Icon from '@/components/icons/Icon.vue'
const { t } = useI18n()
const app = useAppStore()
const auth = useAuthStore()
const view = useUserView()
const homePath = computed(() => auth.isAdmin ? '/admin/dashboard' : '/dashboard')
const supportUser = computed(() => adminSupportContext.value ? view.user : null)
</script>
