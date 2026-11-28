<template>
  <!-- 站长中心的最短路径：做到的打勾；全部做完收成一行，点开再看。 -->
  <div class="hub-steps">
    <button v-if="allDone" type="button" class="hub-steps-done" :aria-expanded="open" @click="emit('toggle')">
      <span class="hub-steps-check"><Icon name="check" size="xs" /></span>
      <span class="min-w-0 flex-1 truncate text-left">{{ tr('都做好了，可以开始赚钱', 'All set, you are ready to earn') }}</span>
      <span class="hub-steps-toggle">{{ open ? tr('收起步骤', 'Hide steps') : tr('查看步骤', 'Show steps') }}<Icon name="chevronDown" size="xs" class="transition-transform" :class="{ 'rotate-180': open }" /></span>
    </button>
    <div class="gc-collapse" :data-open="open ? 'true' : 'false'">
      <ol class="hub-steps-list">
        <li v-for="(step, index) in steps" :key="step.key" class="hub-step" :class="{ 'hub-step-done': step.done }">
          <span class="hub-step-mark" :data-tip="step.done ? tr('这一步已完成', 'Done') : tr('还没做', 'Not done yet')">
            <Icon v-if="step.done" name="check" size="xs" />
            <template v-else>{{ index + 1 }}</template>
          </span>
          <div class="min-w-0 flex-1">
            <p class="hub-step-title">{{ step.title }}</p>
            <p class="hub-step-body">{{ step.body }}</p>
          </div>
          <button v-if="step.action" type="button" class="btn btn-sm shrink-0" :class="step.action.primary && !step.done ? 'btn-primary' : 'btn-secondary'" @click="step.action.run()">{{ step.action.label }}</button>
        </li>
      </ol>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

export interface HubStep {
  key: string
  done: boolean
  title: string
  body: string
  action?: { label: string; run: () => unknown; primary?: boolean }
}

const props = defineProps<{ steps: HubStep[]; open: boolean }>()
const emit = defineEmits<{ toggle: [] }>()
const { locale } = useI18n(); const tr = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const allDone = computed(() => props.steps.every(step => step.done))
</script>

<style scoped>
.hub-steps { display: flex; flex-direction: column; gap: 6px; }
.hub-steps-done { display: flex; align-items: center; gap: 10px; border-radius: var(--surface-radius-control); padding: 8px 10px; font-size: 13px; font-weight: 500; color: var(--surface-text); background: var(--surface-soft); transition: background-color var(--motion-hover) var(--motion-ease-fluid); }
.hub-steps-done:hover { background: var(--surface-hover); }
.hub-steps-check { display: inline-flex; flex: none; align-items: center; justify-content: center; width: 20px; height: 20px; border-radius: 9999px; color: #fff; background: theme('colors.emerald.500'); }
.hub-steps-toggle { display: inline-flex; flex: none; align-items: center; gap: 4px; font-size: 12px; color: var(--surface-muted); }
.hub-steps-list { display: flex; flex-direction: column; gap: 2px; }
.hub-step { display: flex; align-items: center; gap: 12px; border-radius: var(--surface-radius-control); padding: 8px 10px; transition: background-color var(--motion-hover) var(--motion-ease-fluid); }
.hub-step:hover { background: var(--surface-hover); }
.hub-step-mark { display: inline-flex; flex: none; align-items: center; justify-content: center; width: 24px; height: 24px; border: 1.5px solid var(--surface-edge); border-radius: 9999px; font-size: 12px; font-weight: 600; color: var(--surface-muted); }
.hub-step-done .hub-step-mark { border-color: transparent; color: #fff; background: theme('colors.emerald.500'); }
.hub-step-title { font-size: 13px; font-weight: 600; color: var(--surface-text); }
.hub-step-done .hub-step-title { color: var(--surface-muted); }
.hub-step-body { margin-top: 1px; font-size: 12px; line-height: 1.55; color: var(--surface-muted); }
</style>
