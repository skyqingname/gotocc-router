<template>
  <div class="studio-frame flex h-dvh flex-col overflow-hidden bg-white dark:bg-dark-950">
    <AdminSupportBanner v-if="supportUser" :target="supportUser" class="flex-shrink-0" />
    <main class="relative min-h-0 flex-1"><slot /></main>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useUserView } from '@/composables/useUserView'
import { adminSupportContext } from '@/utils/adminSupportContext'
import AdminSupportBanner from '@/components/admin/support/AdminSupportBanner.vue'
const view = useUserView()
const supportUser = computed(() => adminSupportContext.value ? view.user : null)
</script>
