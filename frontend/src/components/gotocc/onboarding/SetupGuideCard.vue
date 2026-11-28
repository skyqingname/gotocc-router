<template>
  <div v-if="rendered" class="gc-collapse" :data-open="visible" :inert="!visible || undefined" @transitionend.self="onCollapsed">
    <div>
      <section class="card setup-card gc-reveal" data-test="setup-guide-card" :aria-label="t('consoleGuide.setup.title')">
        <header class="setup-head">
          <div class="min-w-0">
            <h2 class="setup-title">{{ t('consoleGuide.setup.title') }}</h2>
            <p class="setup-lead">{{ t('consoleGuide.setup.lead') }}</p>
          </div>
          <div class="setup-side">
            <span :key="progress.doneCount.value" class="setup-count gc-num gc-reveal">{{ t('consoleGuide.start.progress', { done: progress.doneCount.value, total: 3 }) }}</span>
            <button type="button" class="setup-collapse" :aria-label="t('consoleGuide.setup.collapse')" :title="t('consoleGuide.setup.collapse')" data-test="setup-guide-collapse" @click="progress.dismiss()">
              <Icon name="chevronUp" size="sm" />
            </button>
          </div>
        </header>
        <span class="setup-track"><i :style="{ width: `${(progress.doneCount.value / 3) * 100}%` }"></i></span>
        <OnboardingSteps :steps="progress.steps.value" :has-key="progress.hasKey.value" compact @go="go" />
        <RouterLink to="/getting-started" class="setup-more">{{ t('consoleGuide.setup.openPage') }}<Icon name="arrowRight" size="xs" /></RouterLink>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import OnboardingSteps from '@/components/gotocc/onboarding/OnboardingSteps.vue'
import { ONBOARDING_STEP_ROUTES, useOnboardingProgress, type OnboardingStepId } from '@/composables/useOnboardingProgress'

// 仪表盘顶部的设置引导：三步没做完且没收起时出现；收起后从头像菜单「新手导览」或 ⌘K 搜「导览」重新打开。
const { t } = useI18n()
const router = useRouter()
const progress = useOnboardingProgress()
const visible = computed(() => progress.loaded.value && !progress.allDone.value && !progress.dismissed.value)

// 收起时先播完高度动画再移除，免得在仪表盘顶部留下一段空白间距。
const rendered = ref(false)
watch(visible, (value) => { if (value) rendered.value = true }, { immediate: true })
function onCollapsed(event: TransitionEvent) {
  if (event.propertyName === 'grid-template-rows' && !visible.value) rendered.value = false
}

function go(id: OnboardingStepId) {
  void router.push(ONBOARDING_STEP_ROUTES[id])
}

onMounted(() => { void progress.refresh() })
</script>

<style scoped>
.setup-card { display: flex; flex-direction: column; gap: 12px; padding: 16px 18px; }
.setup-card::before {
  content: ''; position: absolute; inset: 0; pointer-events: none; border-radius: inherit;
  background: radial-gradient(24rem 9rem at 0% 0%, rgb(99 102 241 / 12%), transparent 70%);
}
.setup-head { position: relative; display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.setup-title { font-size: 15px; font-weight: 700; color: var(--surface-text); }
.setup-lead { margin-top: 2px; font-size: 12px; color: var(--surface-muted); }
.setup-side { display: flex; flex: none; align-items: center; gap: 6px; }
.setup-count { font-size: 12px; font-weight: 600; color: var(--surface-muted); }
.setup-collapse {
  display: grid; height: 28px; width: 28px; place-items: center; border-radius: var(--surface-radius-tag); color: var(--surface-muted);
  transition: background-color var(--motion-fast) var(--motion-ease), color var(--motion-fast) var(--motion-ease);
}
.setup-collapse:hover { background: var(--surface-hover); color: var(--surface-text); }
.setup-track { position: relative; display: block; height: 5px; overflow: hidden; border-radius: 9999px; background: var(--surface-soft); }
.setup-track > i {
  display: block; height: 100%; border-radius: inherit;
  background: linear-gradient(90deg, theme('colors.primary.500'), theme('colors.accent.500'));
  transition: width var(--motion-collapse) var(--motion-ease-fluid);
}
.setup-more { position: relative; display: inline-flex; align-items: center; gap: 4px; align-self: flex-start; font-size: 12px; font-weight: 600; color: theme('colors.primary.600'); }
.dark .setup-more { color: theme('colors.primary.300'); }
.setup-more:hover { text-decoration: underline; }
</style>
