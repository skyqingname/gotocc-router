<template>
  <AppLayout appearance="user">
    <div class="start-page" data-test="getting-started">
      <section class="card start-hero">
        <div class="min-w-0">
          <h2 :key="progress.allDone.value ? 'done' : 'todo'" class="start-title gc-reveal">
            {{ progress.allDone.value ? t('consoleGuide.start.doneTitle') : t('consoleGuide.start.title') }}
          </h2>
          <p class="start-lead">{{ progress.allDone.value ? t('consoleGuide.start.doneLead') : t('consoleGuide.start.lead') }}</p>
        </div>
        <div class="start-progress" :aria-label="t('consoleGuide.start.progress', { done: progress.doneCount.value, total: 3 })">
          <span v-if="progress.loaded.value" :key="progress.doneCount.value" class="start-progress-text gc-num gc-reveal">
            {{ t('consoleGuide.start.progress', { done: progress.doneCount.value, total: 3 }) }}
          </span>
          <span v-else class="gc-skeleton inline-block h-4 w-16"></span>
          <span class="start-progress-track"><i :style="{ width: `${(progress.doneCount.value / 3) * 100}%` }"></i></span>
        </div>
      </section>

      <div v-if="!progress.loaded.value" class="flex flex-col gap-3">
        <div v-for="index in 3" :key="index" class="gc-skeleton h-[74px] w-full"></div>
      </div>
      <OnboardingSteps v-else :steps="progress.steps.value" :has-key="progress.hasKey.value" @go="go" />

      <section v-if="progress.allDone.value" class="card start-done" data-test="onboarding-done">
        <CelebrationBurst />
        <div class="min-w-0">
          <h3 class="start-done-title">{{ t('consoleGuide.start.celebrateTitle') }}</h3>
          <p class="start-lead">{{ t('consoleGuide.start.celebrateLead') }}</p>
          <div class="mt-3 flex flex-wrap gap-2">
            <RouterLink to="/dashboard" class="btn btn-primary btn-sm">{{ t('consoleGuide.start.toDashboard') }}</RouterLink>
            <RouterLink to="/model-plaza?embedded=1" class="btn btn-secondary btn-sm">{{ t('consoleGuide.start.toModels') }}</RouterLink>
          </div>
        </div>
      </section>

      <section class="start-extra">
        <button type="button" class="start-extra-link" data-test="onboarding-tour" @click="router.push({ path: '/keys', query: { tour: '1' } })">
          <Icon name="play" size="sm" />{{ t('consoleGuide.start.watchTour') }}
        </button>
        <span class="surface-muted">{{ t('consoleGuide.start.tourHint') }}</span>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import CelebrationBurst from '@/components/gotocc/onboarding/CelebrationBurst.vue'
import OnboardingSteps from '@/components/gotocc/onboarding/OnboardingSteps.vue'
import { ONBOARDING_STEP_ROUTES, useOnboardingProgress, type OnboardingStepId } from '@/composables/useOnboardingProgress'

const { t } = useI18n()
const router = useRouter()
const progress = useOnboardingProgress()

function go(id: OnboardingStepId) {
  void router.push(ONBOARDING_STEP_ROUTES[id])
}

onMounted(() => { void progress.refresh() })
</script>

<style scoped>
.start-page { display: flex; max-width: 52rem; flex-direction: column; gap: 14px; margin-inline: auto; width: 100%; }
.start-hero { display: flex; flex-wrap: wrap; align-items: flex-end; justify-content: space-between; gap: 16px 24px; padding: 22px 24px; overflow: hidden; }
.start-hero::before {
  content: ''; position: absolute; inset: 0; pointer-events: none; border-radius: inherit;
  background: radial-gradient(28rem 12rem at 0% 0%, rgb(99 102 241 / 14%), transparent 70%), radial-gradient(20rem 10rem at 100% 0%, rgb(139 92 246 / 12%), transparent 70%);
}
.start-title { position: relative; font-size: 22px; font-weight: 700; letter-spacing: -.01em; color: var(--surface-text); }
.start-lead { position: relative; margin-top: 4px; font-size: 13px; line-height: 1.7; color: var(--surface-muted); }
.start-progress { position: relative; display: flex; min-width: 12rem; flex-direction: column; align-items: flex-end; gap: 8px; }
.start-progress-text { font-size: 13px; font-weight: 600; color: var(--surface-text); }
.start-progress-track { display: block; height: 6px; width: 100%; overflow: hidden; border-radius: 9999px; background: var(--surface-soft); }
.start-progress-track > i {
  display: block; height: 100%; border-radius: inherit;
  background: linear-gradient(90deg, theme('colors.primary.500'), theme('colors.accent.500'));
  transition: width var(--motion-collapse) var(--motion-ease-fluid);
}
.start-done { display: flex; align-items: center; gap: 18px; padding: 18px 22px; }
.start-done-title { font-size: 17px; font-weight: 700; color: var(--surface-text); }
.start-extra { display: flex; flex-wrap: wrap; align-items: center; gap: 8px 12px; padding-inline: 4px; font-size: 12px; }
.start-extra-link { display: inline-flex; align-items: center; gap: 6px; font-weight: 600; color: theme('colors.primary.600'); }
.dark .start-extra-link { color: theme('colors.primary.300'); }
.start-extra-link:hover { text-decoration: underline; }
@media (max-width: 640px) {
  .start-hero { padding: 18px; }
  .start-progress { width: 100%; align-items: flex-start; }
  .start-done { flex-direction: column; align-items: flex-start; }
}
</style>
