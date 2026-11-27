<template>
  <ol class="ob-steps" :class="{ 'ob-steps-compact': compact }">
    <li
      v-for="(step, index) in steps"
      :key="step.id"
      class="ob-step"
      :data-state="stateOf(index)"
      :data-step="step.id"
    >
      <span class="ob-step-mark gc-num">
        <Icon v-if="step.done" :key="'done'" name="check" size="sm" :stroke-width="2.6" class="ob-step-check" />
        <span v-else>{{ index + 1 }}</span>
      </span>
      <div class="ob-step-copy">
        <h3 class="ob-step-title">{{ t(`consoleGuide.start.steps.${step.id}.title`) }}</h3>
        <p v-if="!compact" class="ob-step-desc">{{ t(`consoleGuide.start.steps.${step.id}.desc`) }}</p>
      </div>
      <button
        type="button"
        class="btn btn-sm ob-step-action"
        :class="stateOf(index) === 'current' ? 'btn-primary' : 'btn-secondary'"
        :disabled="step.id === 'ccswitch' && !hasKey"
        :title="step.id === 'ccswitch' && !hasKey ? t('consoleGuide.start.needKey') : undefined"
        :data-test="`onboarding-go-${step.id}`"
        @click="emit('go', step.id)"
      >
        {{ step.done ? t(`consoleGuide.start.steps.${step.id}.again`) : t(`consoleGuide.start.steps.${step.id}.action`) }}
      </button>
    </li>
  </ol>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { OnboardingStepId } from '@/composables/useOnboardingProgress'

const props = defineProps<{
  steps: Array<{ id: OnboardingStepId; done: boolean }>
  hasKey: boolean
  compact?: boolean
}>()
const emit = defineEmits<{ go: [id: OnboardingStepId] }>()
const { t } = useI18n()

/** 第一个没完成的步骤是「当前」，其余未完成的是「待办」。 */
function stateOf(index: number): 'done' | 'current' | 'todo' {
  if (props.steps[index].done) return 'done'
  return props.steps.findIndex(step => !step.done) === index ? 'current' : 'todo'
}
</script>

<style scoped>
.ob-steps { display: flex; flex-direction: column; gap: 10px; }
.ob-step {
  display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: 14px;
  padding: 14px 16px; border: 1px solid var(--surface-edge); border-radius: var(--surface-radius-card);
  background: var(--surface-solid);
  transition: border-color var(--motion-normal) var(--motion-ease), box-shadow var(--motion-normal) var(--motion-ease);
}
.ob-step[data-state='current'] { border-color: rgb(99 102 241 / 45%); box-shadow: 0 0 0 3px rgb(99 102 241 / 10%); }
.ob-step-mark {
  display: grid; height: 32px; width: 32px; place-items: center; border-radius: 9999px;
  border: 1.5px solid var(--surface-edge); font-size: 13px; font-weight: 700; color: var(--surface-muted);
  transition: background-color var(--motion-normal) var(--motion-ease), color var(--motion-normal) var(--motion-ease), border-color var(--motion-normal) var(--motion-ease);
}
.ob-step[data-state='current'] .ob-step-mark { border-color: theme('colors.primary.500'); color: theme('colors.primary.600'); }
.dark .ob-step[data-state='current'] .ob-step-mark { color: theme('colors.primary.300'); }
.ob-step[data-state='done'] .ob-step-mark { border-color: transparent; background: theme('colors.emerald.500'); color: white; }
.ob-step-check { animation: ob-check 420ms var(--motion-ease-fluid) both; }
@keyframes ob-check { from { transform: scale(.4); opacity: 0; } }
.ob-step-title { font-size: 14px; font-weight: 650; color: var(--surface-text); }
.ob-step[data-state='done'] .ob-step-title { color: var(--surface-muted); }
.ob-step-desc { margin-top: 2px; font-size: 13px; line-height: 1.6; color: var(--surface-muted); }
.ob-steps-compact { gap: 8px; }
.ob-steps-compact .ob-step { padding: 10px 12px; gap: 12px; background: var(--surface-soft); }
.ob-steps-compact .ob-step-mark { height: 26px; width: 26px; font-size: 12px; }
.ob-steps-compact .ob-step-title { font-size: 13px; }
@media (max-width: 640px) {
  .ob-step { grid-template-columns: auto minmax(0, 1fr); }
  .ob-step-action { grid-column: 2; justify-self: start; }
}
@media (prefers-reduced-motion: reduce) { .ob-step-check { animation: none; } }
</style>
