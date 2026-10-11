<template>
  <div class="bg-canvas dark:bg-dark-950" :class="fullViewport ? 'h-dvh overflow-hidden' : 'min-h-screen'">
    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="relative transition-all duration-300"
      :class="[sidebarCollapsed ? 'lg:ml-[72px]' : 'lg:ml-64', fullViewport ? 'flex h-full min-h-0 flex-col' : 'min-h-screen']"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content -->
      <main :class="fullViewport ? 'relative flex min-h-0 flex-1 flex-col' : 'p-4 md:p-6 lg:p-8'">
        <AdminSupportBanner v-if="supportUser" :target="supportUser" class="mb-5" />
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AdminSupportBanner from '@/components/admin/support/AdminSupportBanner.vue'
import { useUserView } from '@/composables/useUserView'
import { adminSupportContext } from '@/utils/adminSupportContext'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

withDefaults(defineProps<{ fullViewport?: boolean }>(), { fullViewport: false })

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const view = useUserView()
const supportUser = computed(() => adminSupportContext.value ? view.user : null)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour, startTeamTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: !adminSupportContext.value
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
  onboardingStore.setTeamGuideCallback(startTeamTour)
})

defineExpose({ replayTour })
</script>
