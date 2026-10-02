<template>
  <div data-testid="gotocc-home" class="gotocc-home min-h-screen overflow-x-hidden bg-white text-gray-950 dark:bg-dark-950 dark:text-white">
    <header class="sticky top-0 z-30 border-b border-gray-200/80 bg-white/95 px-4 backdrop-blur dark:border-dark-800 dark:bg-dark-950/95 sm:px-6">
      <nav class="mx-auto flex h-16 max-w-7xl items-center justify-between gap-3">
        <RouterLink to="/home" class="flex min-w-0 items-center gap-2.5" aria-label="GotoCC home">
          <span class="flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-md border border-gray-200 bg-white dark:border-dark-700">
            <img :src="siteLogo || '/logo.png'" alt="GotoCC" class="h-full w-full object-contain" />
          </span>
          <span class="truncate text-sm font-semibold">{{ siteName }}</span>
        </RouterLink>

        <div class="flex shrink-0 items-center gap-1.5 sm:gap-2">
          <RouterLink to="/model-plaza" class="hidden px-3 py-2 text-sm font-medium text-gray-600 transition hover:text-sky-700 dark:text-dark-300 dark:hover:text-sky-300 sm:block">
            {{ t('home.nav.models') }}
          </RouterLink>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="hidden px-3 py-2 text-sm font-medium text-gray-600 transition hover:text-sky-700 dark:text-dark-300 dark:hover:text-sky-300 md:block">
            {{ t('home.docs') }}
          </a>
          <LocaleSwitcher />
          <button type="button" class="icon-button" :title="isDark ? t('home.switchToLight') : t('home.switchToDark')" @click="$emit('toggle-theme')">
            <Icon :name="isDark ? 'sun' : 'moon'" size="md" />
          </button>
          <RouterLink :to="isAuthenticated ? dashboardPath : '/login'" class="inline-flex min-h-9 items-center justify-center rounded-md bg-gray-950 px-3 py-2 text-xs font-semibold text-white transition hover:bg-gray-800 dark:bg-white dark:text-dark-950 dark:hover:bg-gray-200 sm:px-4 sm:text-sm">
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </RouterLink>
        </div>
      </nav>
    </header>

    <main>
      <section class="hero-grid border-b border-gray-200 px-4 pb-12 pt-14 dark:border-dark-800 sm:px-6 sm:pb-16 sm:pt-20">
        <div class="mx-auto max-w-6xl text-center">
          <div class="hero-reveal mx-auto flex w-fit items-center gap-2 rounded-full border border-sky-200 bg-sky-50 px-3 py-1.5 text-xs font-semibold text-sky-700 dark:border-sky-900 dark:bg-sky-950/50 dark:text-sky-300">
            <span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
            {{ t('home.heroBadge') }}
          </div>
          <img :src="siteLogo || '/logo.png'" alt="GotoCC" class="hero-reveal mx-auto mt-7 h-20 w-20 rounded-md object-contain sm:h-24 sm:w-24" />
          <p class="hero-reveal mt-4 text-sm font-semibold text-sky-700 dark:text-sky-300">{{ siteName }}</p>
          <h1 class="hero-reveal mx-auto mt-3 max-w-4xl text-4xl font-bold leading-tight sm:text-5xl lg:text-6xl">
            {{ t('home.heroTitle') }}
          </h1>
          <p class="hero-reveal mx-auto mt-5 max-w-2xl text-base leading-7 text-gray-600 dark:text-dark-300 sm:text-lg">
            {{ heroDescription }}
          </p>
          <div class="hero-reveal mt-8 flex flex-col items-stretch justify-center gap-3 sm:flex-row sm:items-center">
            <RouterLink :to="isAuthenticated ? dashboardPath : '/login'" class="inline-flex min-h-11 items-center justify-center gap-2 rounded-md bg-sky-600 px-6 py-3 text-sm font-semibold text-white transition hover:bg-sky-700">
              {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
              <Icon name="arrowRight" size="sm" />
            </RouterLink>
            <RouterLink to="/model-plaza" class="inline-flex min-h-11 items-center justify-center gap-2 rounded-md border border-gray-300 bg-white px-6 py-3 text-sm font-semibold text-gray-900 transition hover:border-sky-400 hover:text-sky-700 dark:border-dark-700 dark:bg-dark-900 dark:text-white dark:hover:border-sky-600 dark:hover:text-sky-300">
              {{ t('home.exploreMarketplace') }}
              <Icon name="grid" size="sm" />
            </RouterLink>
          </div>
        </div>
      </section>

      <section class="border-b border-gray-200 bg-gray-50 px-4 py-7 dark:border-dark-800 dark:bg-dark-900/50 sm:px-6">
        <div class="mx-auto grid max-w-6xl grid-cols-2 divide-x divide-y divide-gray-200 border border-gray-200 bg-white dark:divide-dark-700 dark:border-dark-700 dark:bg-dark-950 lg:grid-cols-4 lg:divide-y-0">
          <div v-for="stat in statsCards" :key="stat.key" class="min-w-0 px-4 py-5 text-center sm:px-6">
            <p class="truncate text-2xl font-semibold tabular-nums sm:text-3xl" :title="stat.value">{{ stat.value }}</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400 sm:text-sm">{{ stat.label }}</p>
          </div>
        </div>
        <p v-if="statsError" class="mx-auto mt-3 max-w-6xl text-center text-xs text-gray-500 dark:text-dark-400">{{ t('home.stats.unavailable') }}</p>
      </section>

      <section class="px-4 py-14 sm:px-6 sm:py-16">
        <div class="mx-auto max-w-6xl">
          <div class="text-center">
            <p class="section-label">{{ t('home.providers.description') }}</p>
            <h2 class="mt-2 text-2xl font-bold sm:text-3xl">{{ t('home.providers.title') }}</h2>
          </div>
          <div class="mt-8 flex min-h-16 flex-wrap items-center justify-center gap-x-7 gap-y-4 border-y border-gray-200 py-5 dark:border-dark-800">
            <div v-for="provider in providers" :key="provider.platform" class="flex h-9 items-center gap-2 text-gray-700 dark:text-dark-200">
              <span class="flex h-8 w-8 items-center justify-center rounded-md bg-gray-100 dark:bg-dark-800" :class="platformIconClass(provider.platform)">
                <PlatformIcon :platform="provider.platform" size="lg" />
              </span>
              <span class="text-sm font-semibold">{{ provider.label }}</span>
              <span class="text-xs text-gray-400">{{ provider.modelCount }}</span>
            </div>
            <div v-if="modelsLoading" class="text-sm text-gray-500">{{ t('common.loading') }}</div>
            <div v-else-if="providers.length === 0" class="text-sm text-gray-500">{{ modelsError ? t('home.providers.unavailable') : t('home.providers.empty') }}</div>
          </div>
        </div>
      </section>

      <section class="border-y border-gray-200 bg-gray-50 px-4 py-14 dark:border-dark-800 dark:bg-dark-900/40 sm:px-6 sm:py-16">
        <div class="mx-auto max-w-6xl">
          <div class="grid gap-px overflow-hidden rounded-md border border-gray-200 bg-gray-200 dark:border-dark-700 dark:bg-dark-700 md:grid-cols-2">
            <article v-for="feature in features" :key="feature.title" class="min-h-52 bg-white p-6 dark:bg-dark-950 sm:p-7">
              <span class="flex h-10 w-10 items-center justify-center rounded-md" :class="feature.iconClass">
                <Icon :name="feature.icon" size="md" />
              </span>
              <h3 class="mt-5 text-lg font-semibold">{{ feature.title }}</h3>
              <p class="mt-2 max-w-md text-sm leading-6 text-gray-600 dark:text-dark-300">{{ feature.description }}</p>
            </article>
          </div>
        </div>
      </section>

      <section class="px-4 py-14 sm:px-6 sm:py-16">
        <div class="mx-auto max-w-6xl">
          <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
            <div>
              <p class="section-label">Model Plaza</p>
              <h2 class="mt-2 text-2xl font-bold sm:text-3xl">{{ t('home.providers.title') }}</h2>
            </div>
            <RouterLink to="/model-plaza" class="inline-flex items-center gap-2 text-sm font-semibold text-sky-700 hover:text-sky-800 dark:text-sky-300 dark:hover:text-sky-200">
              {{ t('home.viewAll') }}<Icon name="arrowRight" size="sm" />
            </RouterLink>
          </div>

          <div v-if="modelRows.length" class="mt-7 divide-y divide-gray-200 border-y border-gray-200 dark:divide-dark-800 dark:border-dark-800">
            <div v-for="row in modelRows" :key="row.platform" class="grid gap-3 py-5 sm:grid-cols-[160px_1fr] sm:items-center">
              <div class="flex items-center gap-2 font-semibold">
                <PlatformIcon :platform="row.platform" size="md" :class="platformIconClass(row.platform)" />
                {{ row.label }}
              </div>
              <div class="flex min-w-0 flex-wrap gap-2">
                <span v-for="model in row.models" :key="model" class="max-w-full truncate rounded border border-gray-200 bg-gray-50 px-2.5 py-1 font-mono text-xs text-gray-700 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-200" :title="model">{{ model }}</span>
              </div>
            </div>
          </div>
          <div v-else class="mt-7 border-y border-gray-200 py-10 text-center text-sm text-gray-500 dark:border-dark-800">
            {{ modelsLoading ? t('common.loading') : (modelsError ? t('home.providers.unavailable') : t('home.providers.empty')) }}
          </div>
        </div>
      </section>

      <section class="border-t border-gray-200 px-4 py-14 dark:border-dark-800 sm:px-6 sm:py-16">
        <div class="mx-auto max-w-6xl">
          <div class="grid gap-9 md:grid-cols-3">
            <article v-for="step in steps" :key="step.index" class="min-w-0">
              <span class="flex h-8 w-8 items-center justify-center rounded-full bg-sky-100 text-sm font-semibold text-sky-700 dark:bg-sky-950 dark:text-sky-300">{{ step.index }}</span>
              <h3 class="mt-4 text-lg font-semibold">{{ step.title }}</h3>
              <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ step.description }}</p>
              <div v-if="step.index === '03'" class="mt-5 max-w-xs rounded-md border border-gray-200 bg-gray-50 px-3 py-2.5 font-mono text-xs text-gray-600 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-300">GOTOCC_API_KEY</div>
            </article>
          </div>
        </div>
      </section>

      <section class="bg-gray-950 px-4 py-14 text-center text-white sm:px-6 sm:py-16 dark:bg-black">
        <h2 class="text-2xl font-bold sm:text-3xl">{{ t('home.cta.title') }}</h2>
        <p class="mx-auto mt-3 max-w-xl text-sm leading-6 text-gray-300 sm:text-base">{{ t('home.cta.description') }}</p>
        <RouterLink :to="isAuthenticated ? dashboardPath : '/login'" class="mt-7 inline-flex min-h-11 items-center justify-center gap-2 rounded-md bg-sky-500 px-7 py-3 text-sm font-semibold text-white transition hover:bg-sky-400">
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.cta.button') }}
          <Icon name="arrowRight" size="sm" />
        </RouterLink>
      </section>
    </main>

    <footer class="border-t border-gray-200 px-4 py-8 dark:border-dark-800 sm:px-6">
      <div class="mx-auto flex max-w-6xl flex-col items-center justify-between gap-4 text-center text-sm text-gray-500 dark:text-dark-400 sm:flex-row sm:text-left">
        <p>&copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</p>
        <div class="flex items-center gap-5">
          <RouterLink to="/model-plaza" class="transition hover:text-sky-700 dark:hover:text-sky-300">{{ t('home.nav.models') }}</RouterLink>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="transition hover:text-sky-700 dark:hover:text-sky-300">{{ t('home.docs') }}</a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { getHomepageModelPlaza, getHomepageStats, type HomepageStats } from '@/api/home'
import type { ModelPlazaGroup } from '@/api/modelPlaza'
import type { GroupPlatform } from '@/types'
import { platformIconClass, platformLabel } from '@/utils/platformColors'

const props = defineProps<{
  siteName: string
  siteLogo: string
  siteSubtitle: string
  docUrl: string
  isAuthenticated: boolean
  dashboardPath: string
  isDark: boolean
}>()

defineEmits<{ (event: 'toggle-theme'): void }>()

const { t } = useI18n()
const stats = ref<HomepageStats | null>(null)
const groups = ref<ModelPlazaGroup[]>([])
const statsLoading = ref(true)
const modelsLoading = ref(true)
const statsError = ref(false)
const modelsError = ref(false)
const currentYear = new Date().getFullYear()
const heroDescription = computed(() => props.siteSubtitle || t('home.heroDescription'))

const totalModelCount = computed(() => new Set(groups.value.flatMap((group) => group.models.map((model) => model.name))).size)
const statsCards = computed(() => [
  { key: 'today', label: t('home.stats.todayTokens'), value: statsLoading.value ? '...' : formatCount(stats.value?.today_tokens) },
  { key: 'total', label: t('home.stats.totalTokens'), value: statsLoading.value ? '...' : formatCount(stats.value?.total_tokens) },
  { key: 'users', label: t('home.stats.totalUsers'), value: statsLoading.value ? '...' : formatCount(stats.value?.total_users, false) },
  { key: 'models', label: t('home.stats.supportedModels'), value: modelsLoading.value ? '...' : formatCount(totalModelCount.value, false) },
])

const platformGroups = computed(() => {
  const map = new Map<GroupPlatform, ModelPlazaGroup[]>()
  for (const group of groups.value) {
    if (!isGroupPlatform(group.platform)) continue
    map.set(group.platform, [...(map.get(group.platform) || []), group])
  }
  return map
})

const providers = computed(() => Array.from(platformGroups.value.entries()).map(([platform, entries]) => ({
  platform,
  label: platformLabel(platform),
  modelCount: new Set(entries.flatMap((group) => group.models.map((model) => model.name))).size,
})))

const modelRows = computed(() => providers.value.slice(0, 5).map((provider) => ({
  platform: provider.platform,
  label: provider.label,
  models: Array.from(new Set((platformGroups.value.get(provider.platform) || []).flatMap((group) => group.models.map((model) => model.name)))).slice(0, 6),
})).filter((row) => row.models.length > 0))

const features = computed(() => [
  { icon: 'server' as const, iconClass: 'bg-sky-100 text-sky-700 dark:bg-sky-950 dark:text-sky-300', title: t('home.features.unifiedGateway'), description: t('home.features.unifiedGatewayDesc') },
  { icon: 'refresh' as const, iconClass: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300', title: t('home.features.multiAccount'), description: t('home.features.multiAccountDesc') },
  { icon: 'chart' as const, iconClass: 'bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-300', title: t('home.features.balanceQuota'), description: t('home.features.balanceQuotaDesc') },
  { icon: 'terminal' as const, iconClass: 'bg-gray-200 text-gray-800 dark:bg-dark-800 dark:text-dark-200', title: t('home.features.dataPolicies'), description: t('home.features.dataPoliciesDesc') },
])

const steps = computed(() => [
  { index: '01', title: t('home.steps.signup.title'), description: t('home.steps.signup.description') },
  { index: '02', title: t('home.steps.browse.title'), description: t('home.steps.browse.description') },
  { index: '03', title: t('home.steps.apiKey.title'), description: t('home.steps.apiKey.description') },
])

function isGroupPlatform(value: string): value is GroupPlatform {
  return ['anthropic', 'openai', 'gemini', 'antigravity', 'grok', 'composite'].includes(value)
}

function formatCount(value: number | undefined, compact = true): string {
  const normalized = Math.max(0, Number(value || 0))
  if (!compact || normalized < 10000) return normalized.toLocaleString()
  return new Intl.NumberFormat(undefined, { notation: 'compact', maximumFractionDigits: 1 }).format(normalized)
}

onMounted(async () => {
  const [statsResult, modelsResult] = await Promise.allSettled([getHomepageStats(), getHomepageModelPlaza()])
  if (statsResult.status === 'fulfilled') stats.value = statsResult.value
  else statsError.value = true
  if (modelsResult.status === 'fulfilled') groups.value = modelsResult.value.groups
  else modelsError.value = true
  statsLoading.value = false
  modelsLoading.value = false
})
</script>

<style scoped>
.hero-grid {
  background-color: #ffffff;
  background-image:
    linear-gradient(rgba(14, 165, 233, 0.07) 1px, transparent 1px),
    linear-gradient(90deg, rgba(14, 165, 233, 0.07) 1px, transparent 1px);
  background-size: 40px 40px;
}

.dark .hero-grid {
  background-color: #09090b;
  background-image:
    linear-gradient(rgba(56, 189, 248, 0.07) 1px, transparent 1px),
    linear-gradient(90deg, rgba(56, 189, 248, 0.07) 1px, transparent 1px);
}

.icon-button {
  display: inline-flex;
  width: 2.25rem;
  height: 2.25rem;
  align-items: center;
  justify-content: center;
  border-radius: 0.375rem;
  color: rgb(107 114 128);
  transition: background-color 150ms ease, color 150ms ease;
}

.icon-button:hover {
  background: rgb(243 244 246);
  color: rgb(3 105 161);
}

.dark .icon-button:hover {
  background: rgb(39 39 42);
  color: rgb(125 211 252);
}

.section-label {
  font-size: 0.75rem;
  line-height: 1rem;
  font-weight: 700;
  text-transform: uppercase;
  color: rgb(3 105 161);
}

.dark .section-label {
  color: rgb(125 211 252);
}

.hero-reveal {
  animation: hero-reveal 420ms ease both;
}

.hero-reveal:nth-child(2) { animation-delay: 50ms; }
.hero-reveal:nth-child(3) { animation-delay: 90ms; }
.hero-reveal:nth-child(4) { animation-delay: 130ms; }
.hero-reveal:nth-child(5) { animation-delay: 170ms; }
.hero-reveal:nth-child(6) { animation-delay: 210ms; }

@keyframes hero-reveal {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

@media (prefers-reduced-motion: reduce) {
  .hero-reveal { animation: none; }
}
</style>
