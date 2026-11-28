<template>
  <section class="endpoint-bar" :aria-label="t('keyConsole.endpoint.title')">
    <div class="endpoint-bar-head">
      <span class="endpoint-bar-title">{{ t('keyConsole.endpoint.title') }}</span>
      <span class="endpoint-help" tabindex="0" :aria-label="t('keyConsole.endpoint.helpLabel')">
        <Icon name="questionCircle" size="sm" />
        <span class="endpoint-help-panel surface-panel" role="tooltip">
          <span v-for="endpoint in endpoints" :key="endpoint.url" class="block">
            <span class="font-semibold text-gray-900 dark:text-white">{{ endpointName(endpoint) }}</span>
            <span class="mt-0.5 block font-mono text-[11px] surface-muted">{{ endpoint.url }}</span>
            <span class="mt-1 block">{{ endpoint.primary ? t('keyConsole.endpoint.primaryHint') : endpoint.description }}</span>
          </span>
        </span>
      </span>
    </div>

    <div class="endpoint-bar-body">
      <div class="endpoint-url">
        <code :key="current.url" class="gc-reveal endpoint-url-text" :title="current.url">{{ current.url }}</code>
        <button
          type="button"
          class="endpoint-icon-btn"
          :class="copied && 'text-emerald-500'"
          :aria-label="t('keyConsole.endpoint.copy')"
          data-test="endpoint-copy"
          @click="copy"
        >
          <Icon :name="copied ? 'check' : 'copy'" size="sm" :stroke-width="copied ? 2.2 : 1.5" />
        </button>
      </div>

      <div class="endpoint-controls">
        <div v-if="endpoints.length > 1" v-segmented class="segmented endpoint-switch" role="radiogroup" :aria-label="t('keyConsole.endpoint.switch')">
          <button
            v-for="endpoint in endpoints"
            :key="endpoint.url"
            type="button"
            role="radio"
            :aria-checked="endpoint.url === current.url"
            class="segmented-item endpoint-option"
            :class="endpoint.url === current.url && 'segmented-item-active'"
            data-test="endpoint-option"
            @click="select(endpoint.url)"
          >
            <span>{{ endpointName(endpoint) }}</span>
            <span v-if="probes[endpoint.url]" :key="probeLabel(endpoint.url)" class="endpoint-latency gc-reveal gc-num" :data-state="probes[endpoint.url].state">
              {{ probeLabel(endpoint.url) }}
            </span>
          </button>
        </div>
        <button type="button" class="btn btn-secondary btn-sm endpoint-test" :disabled="probing" data-test="endpoint-probe" @click="runProbe">
          <svg v-if="probing" class="h-3.5 w-3.5 animate-spin" viewBox="0 0 24 24" fill="none" aria-hidden="true"><circle cx="12" cy="12" r="9" stroke="currentColor" stroke-width="3" class="opacity-25" /><path d="M21 12a9 9 0 0 0-9-9" stroke="currentColor" stroke-width="3" stroke-linecap="round" /></svg>
          <Icon v-else name="bolt" size="sm" />
          <span>{{ probing ? t('keyConsole.endpoint.probing') : t('keyConsole.endpoint.probe') }}</span>
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, toRef } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { vSegmented } from '@/components/gotocc/vSegmented'
import { useClipboard } from '@/composables/useClipboard'
import { useApiEndpoint, type ApiEndpointOption } from '@/composables/useApiEndpoint'
import { useAppStore } from '@/stores/app'
import type { PublicSettings } from '@/types'

const props = defineProps<{ settings: PublicSettings | null }>()

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()
const { endpoints, current, select, probes, probing, probeAll } = useApiEndpoint(toRef(props, 'settings'))

const copied = ref(false)
let copiedTimer: number | undefined

const endpointName = (endpoint: ApiEndpointOption) => endpoint.primary ? t('keyConsole.endpoint.primary') : endpoint.name

function probeLabel(url: string): string {
  const probe = probes[url]
  if (probe.state === 'probing') return t('keyConsole.endpoint.probingShort')
  if (probe.state === 'failed') return t('keyConsole.endpoint.failedShort')
  return `${probe.ms} ms`
}

async function copy() {
  if (!(await copyToClipboard(current.value.url, t('keyConsole.copied')))) return
  copied.value = true
  window.clearTimeout(copiedTimer)
  copiedTimer = window.setTimeout(() => { copied.value = false }, 1600)
}

async function runProbe() {
  const best = await probeAll()
  if (best) appStore.showSuccess(t('keyConsole.endpoint.switched', { name: endpointName(best.endpoint), ms: best.ms }))
  else appStore.showError(t('keyConsole.endpoint.allFailed'))
}
</script>

<style scoped>
.endpoint-bar { display: flex; min-width: 0; flex-direction: column; gap: 8px; }
/* The help panel is anchored to the whole title row, so it never reaches past the bar. */
.endpoint-bar-head { position: relative; display: flex; align-items: center; gap: 6px; }
.endpoint-bar-title { font-size: 12px; font-weight: 600; letter-spacing: .02em; color: var(--surface-muted); }
.endpoint-help { display: inline-flex; color: var(--surface-muted); cursor: help; outline: none; }
.endpoint-help:is(:hover, :focus-visible) { color: theme('colors.primary.600'); }
.endpoint-help-panel {
  position: absolute; left: 0; top: calc(100% + 8px); z-index: 30;
  display: flex; width: min(22rem, 100%); flex-direction: column; gap: 10px;
  padding: 12px 14px; font-size: 12px; line-height: 1.6; color: var(--surface-text);
  opacity: 0; transform: translateY(-4px); pointer-events: none;
  transition: opacity var(--motion-normal) var(--motion-ease), transform var(--motion-normal) var(--motion-ease);
}
.endpoint-help:is(:hover, :focus-visible) .endpoint-help-panel { opacity: 1; transform: none; }
.endpoint-bar-body { display: flex; min-width: 0; flex-wrap: wrap; align-items: center; gap: 10px 16px; }
.endpoint-url { display: flex; min-width: 0; flex: 1 1 16rem; align-items: center; gap: 6px; }
.endpoint-url-text {
  display: block; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  font-size: 15px; font-weight: 600; color: var(--surface-text);
}
.endpoint-icon-btn {
  display: inline-flex; flex: none; height: 28px; width: 28px; align-items: center; justify-content: center;
  border-radius: var(--surface-radius-tag); color: var(--surface-muted);
  transition: background-color var(--motion-fast) var(--motion-ease), color var(--motion-fast) var(--motion-ease);
}
.endpoint-icon-btn:hover { background: var(--surface-hover); color: var(--surface-text); }
.endpoint-controls { display: flex; min-width: 0; max-width: 100%; flex-wrap: wrap; align-items: center; gap: 8px; }
.endpoint-switch { display: inline-flex; max-width: 100%; overflow-x: auto; scrollbar-width: none; }
.endpoint-option { display: inline-flex; height: 30px; align-items: center; gap: 6px; padding-inline: 12px; font-size: 13px; font-weight: 500; white-space: nowrap; }
.endpoint-latency { font-size: 11px; font-weight: 600; color: theme('colors.emerald.600'); }
.dark .endpoint-latency { color: theme('colors.emerald.400'); }
.endpoint-latency[data-state='probing'] { color: var(--surface-muted); }
.endpoint-latency[data-state='failed'] { color: theme('colors.red.500'); }
.endpoint-test { display: inline-flex; align-items: center; gap: 6px; white-space: nowrap; }
</style>
