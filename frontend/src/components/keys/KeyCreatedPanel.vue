<template>
  <div class="created space-y-4" data-test="key-created">
    <p class="created-lead">
      <span class="created-check" aria-hidden="true"><Icon name="check" size="sm" :stroke-width="2.5" /></span>
      {{ t('keyConsole.created.lead') }}
    </p>

    <div>
      <div class="created-label">{{ t('keys.apiKey') }}</div>
      <div class="created-field">
        <DecodeText :value="apiKey.key" revealed arrive class="min-w-0 flex-1 text-[13px] leading-6" data-test="key-created-secret" />
        <button type="button" class="created-icon-btn" :aria-label="t('keyConsole.detail.copyKey')" data-test="key-created-copy" @click="copy(apiKey.key, 'key')">
          <Icon :name="copied === 'key' ? 'check' : 'copy'" size="sm" />
        </button>
      </div>
    </div>

    <div>
      <div class="created-label">{{ t('keyConsole.created.baseUrl') }}</div>
      <div class="created-field">
        <code :key="baseUrl" class="gc-reveal min-w-0 flex-1 break-all font-mono text-[13px]">{{ baseUrl }}</code>
        <button type="button" class="created-icon-btn" :aria-label="t('keyConsole.endpoint.copy')" @click="copy(baseUrl, 'base')">
          <Icon :name="copied === 'base' ? 'check' : 'copy'" size="sm" />
        </button>
      </div>
    </div>

    <div v-if="clients.length">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div v-segmented class="segmented inline-flex max-w-full overflow-x-auto" role="tablist" :aria-label="t('keyConsole.created.client')">
          <button
            v-for="client in clients"
            :key="client.id"
            type="button"
            role="tab"
            :aria-selected="client.id === clientId"
            class="segmented-item h-8 whitespace-nowrap px-3 text-sm font-medium"
            :class="client.id === clientId && 'segmented-item-active'"
            @click="clientId = client.id"
          >{{ t(`keyConsole.created.clients.${client.id}`) }}</button>
        </div>
        <div v-segmented class="segmented inline-flex" role="tablist" :aria-label="t('keyConsole.created.shell')">
          <button
            v-for="option in shells"
            :key="option"
            type="button"
            role="tab"
            :aria-selected="option === shell"
            class="segmented-item h-7 px-2.5 text-xs font-medium"
            :class="option === shell && 'segmented-item-active'"
            @click="shell = option"
          >{{ t(`keyConsole.created.shells.${option}`) }}</button>
        </div>
      </div>
      <div class="created-snippet">
        <pre :key="snippet" class="gc-reveal" data-test="key-created-snippet">{{ snippet }}</pre>
        <button type="button" class="created-icon-btn created-snippet-copy" :aria-label="t('keyConsole.created.copySnippet')" @click="copy(snippet, 'snippet')">
          <Icon :name="copied === 'snippet' ? 'check' : 'copy'" size="sm" />
        </button>
      </div>
      <p class="mt-2 text-xs surface-muted">{{ t('keyConsole.created.moreConfig') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import DecodeText from '@/components/gotocc/motion/DecodeText.vue'
import { vSegmented } from '@/components/gotocc/vSegmented'
import { useClipboard } from '@/composables/useClipboard'
import { envClientsFor, envSnippet, type EnvClient, type EnvShell } from '@/utils/keyEnvSnippets'
import type { ApiKey } from '@/types'

const props = defineProps<{ apiKey: ApiKey; baseUrl: string }>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const shells: EnvShell[] = ['unix', 'powershell']
const shell = ref<EnvShell>(navigator.userAgent.includes('Windows') ? 'powershell' : 'unix')
const platform = computed(() => props.apiKey.group?.platform ?? null)
const clients = computed(() => envClientsFor(platform.value, props.apiKey.group?.allow_messages_dispatch ?? false))
const clientId = ref<EnvClient['id']>(clients.value[0]?.id ?? 'claude')
const client = computed(() => clients.value.find(item => item.id === clientId.value) ?? clients.value[0])
const snippet = computed(() => envSnippet(client.value, shell.value, props.baseUrl, platform.value, props.apiKey.key))

const copied = ref<'' | 'key' | 'base' | 'snippet'>('')
let copiedTimer: number | undefined

async function copy(text: string, what: 'key' | 'base' | 'snippet') {
  if (!(await copyToClipboard(text, t('keyConsole.copied')))) return
  copied.value = what
  window.clearTimeout(copiedTimer)
  copiedTimer = window.setTimeout(() => { copied.value = '' }, 1600)
}
</script>

<style scoped>
.created-lead { display: flex; align-items: center; gap: 8px; font-size: 13px; color: var(--surface-muted); }
.created-check {
  display: inline-flex; height: 22px; width: 22px; flex: none; align-items: center; justify-content: center;
  border-radius: 9999px; background: theme('colors.emerald.500'); color: white;
  animation: created-pop 420ms var(--motion-ease-fluid) both;
}
@keyframes created-pop { from { transform: scale(.4); opacity: 0; } }
.created-label { margin-bottom: 6px; font-size: 12px; font-weight: 600; color: var(--surface-muted); }
.created-field {
  display: flex; align-items: center; gap: 6px; padding: 9px 8px 9px 12px;
  border: 1px solid var(--surface-edge); border-radius: var(--surface-radius-control);
  background: var(--surface-soft); color: var(--surface-text);
}
.created-icon-btn {
  display: inline-flex; flex: none; height: 28px; width: 28px; align-items: center; justify-content: center;
  border-radius: var(--surface-radius-tag); color: var(--surface-muted);
  transition: background-color var(--motion-fast) var(--motion-ease), color var(--motion-fast) var(--motion-ease);
}
.created-icon-btn:hover { background: var(--surface-hover); color: var(--surface-text); }
.created-snippet { position: relative; margin-top: 8px; }
.created-snippet pre {
  margin: 0; padding: 12px 44px 12px 14px; overflow-x: auto;
  border-radius: var(--surface-radius-control); background: #16141f; color: #e9e7f5;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; line-height: 1.7;
  white-space: pre-wrap; word-break: break-all;
}
.created-snippet-copy { position: absolute; right: 6px; top: 6px; color: #b9b5d0; }
.created-snippet-copy:hover { background: rgb(255 255 255 / 10%); color: white; }
</style>
