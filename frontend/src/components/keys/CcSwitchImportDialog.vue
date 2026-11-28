<template>
  <BaseDialog :show="show" :title="t('keyConsole.ccs.title')" width="normal" @close="emit('close')">
    <div v-if="apiKey" class="space-y-5" data-test="ccs-dialog">
      <fieldset>
        <legend class="input-label">{{ t('keyConsole.ccs.app') }}</legend>
        <div class="ccs-apps" role="radiogroup" :aria-label="t('keyConsole.ccs.app')">
          <button
            v-for="app in apps"
            :key="app.id"
            type="button"
            role="radio"
            class="ccs-app"
            :aria-checked="app.id === appId"
            :data-app="app.id"
            @click="selectApp(app.id)"
          >
            <span class="ccs-app-icon" :class="app.icon ? platformBadgeLightClass(app.icon) : 'ccs-app-icon-letter'" aria-hidden="true">
              <PlatformIcon v-if="app.icon" :platform="app.icon" size="md" />
              <span v-else>{{ app.label.charAt(0) }}</span>
            </span>
            <span class="ccs-app-label">{{ app.label }}</span>
          </button>
        </div>
        <p v-if="protocolIssue" :key="appId" class="ccs-issue gc-reveal" data-test="ccs-protocol-issue">
          <Icon name="exclamationTriangle" size="sm" class="mt-0.5 flex-none" />
          <span>{{ t(`keyConsole.ccs.issues.${protocolIssue}`, { app: app.label, group: apiKey.group?.name ?? '' }) }}</span>
        </p>
      </fieldset>

      <div>
        <label class="input-label" for="ccs-name">{{ t('keyConsole.ccs.name') }}</label>
        <input id="ccs-name" v-model="name" type="text" class="input" maxlength="64" @input="nameTouched = true" />
      </div>

      <div class="space-y-3">
        <div class="ccs-models-status" :data-state="modelState" data-test="ccs-models-status">
          <template v-if="modelState === 'loading'">
            <span class="gc-skeleton inline-block h-3 w-3 rounded-full"></span>
            <span>{{ t('keyConsole.models.loading') }}</span>
          </template>
          <template v-else-if="modelState === 'error'">
            <span class="text-red-500">{{ t('keyConsole.models.failed') }}</span>
            <button type="button" class="ccs-link" @click="loadModels">{{ t('keyConsole.models.retry') }}</button>
          </template>
          <template v-else-if="modelState === 'ready' && models.length === 0">
            <span>{{ t('keyConsole.models.empty') }}</span>
          </template>
          <template v-else-if="modelState === 'ready'">
            <span>{{ t('keyConsole.models.available', { count: models.length }) }}</span>
            <button type="button" class="ccs-link" @click="loadModels">{{ t('keyConsole.models.refresh') }}</button>
          </template>
        </div>
        <datalist :id="listId">
          <option v-for="model in models" :key="model" :value="model" />
        </datalist>
        <div>
          <label class="input-label" for="ccs-model">{{ t('keyConsole.ccs.mainModel') }} <span class="text-red-500">*</span></label>
          <input id="ccs-model" v-model="modelValues.model" type="text" class="input font-mono" :list="listId" :placeholder="t('keyConsole.ccs.modelPlaceholder')" autocomplete="off" data-test="ccs-model" />
        </div>
        <div v-if="extraFields.length" class="ccs-extra">
          <div v-for="field in extraFields" :key="field">
            <label class="input-label" :for="`ccs-${field}`">{{ t(`keyConsole.ccs.fields.${field}`) }}</label>
            <input :id="`ccs-${field}`" v-model="modelValues[field]" type="text" class="input font-mono" :list="listId" :placeholder="t('keyConsole.ccs.optional')" autocomplete="off" />
          </div>
        </div>
      </div>

      <div class="ccs-endpoint">
        <span class="surface-muted">{{ t('keyConsole.ccs.endpoint') }}</span>
        <code :key="endpoint" class="gc-reveal break-all font-mono" data-test="ccs-endpoint">{{ endpoint }}</code>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="!ready" data-test="ccs-open" @click="openCcSwitch">
          <Icon name="externalLink" size="sm" />
          {{ t('keyConsole.ccs.open') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { useKeyModels } from '@/composables/useKeyModels'
import { markCcSwitchImported } from '@/composables/useOnboardingProgress'
import { platformBadgeLightClass } from '@/utils/platformColors'
import {
  CC_SWITCH_APPS,
  buildCcSwitchImportDeeplink,
  ccSwitchApp,
  ccSwitchEndpoint,
  ccSwitchProtocolIssue,
  defaultCcSwitchAppId,
  defaultCcSwitchModel,
  type CcSwitchModelField
} from '@/utils/ccswitchImport'
import type { ApiKey } from '@/types'

const props = defineProps<{
  show: boolean
  apiKey: ApiKey | null
  baseUrl: string
  homepage: string
}>()
const emit = defineEmits<{ close: [] }>()

const { t } = useI18n()
const appStore = useAppStore()
const { models, state: modelState, load, reset } = useKeyModels()

const apps = CC_SWITCH_APPS
const listId = 'ccs-model-options'
const appId = ref(apps[0].id)
const name = ref('')
const nameTouched = ref(false)
const modelValues = reactive<Record<CcSwitchModelField, string>>({ model: '', haikuModel: '', sonnetModel: '', opusModel: '' })

const app = computed(() => ccSwitchApp(appId.value))
const platform = computed(() => props.apiKey?.group?.platform ?? null)
const extraFields = computed(() => app.value.model_fields.filter(field => field !== 'model'))
const protocolIssue = computed(() => ccSwitchProtocolIssue(app.value.api_format, platform.value, props.apiKey?.group?.allow_messages_dispatch ?? false))
const endpoint = computed(() => ccSwitchEndpoint(app.value, props.baseUrl, platform.value))
const ready = computed(() => name.value.trim() !== '' && modelValues.model.trim() !== '')

function preselectModel() {
  const preferred = defaultCcSwitchModel(appId.value)
  if (!modelValues.model && preferred && models.value.includes(preferred)) modelValues.model = preferred
}

function selectApp(id: string) {
  if (id === appId.value) return
  appId.value = id
  if (!nameTouched.value) name.value = app.value.default_name
  for (const field of Object.keys(modelValues) as CcSwitchModelField[]) modelValues[field] = ''
  preselectModel()
}

async function loadModels() {
  if (!props.apiKey) return
  await load(props.apiKey.key)
  preselectModel()
}

watch(() => props.show, open => {
  if (!open || !props.apiKey) {
    reset()
    return
  }
  appId.value = defaultCcSwitchAppId(platform.value)
  name.value = app.value.default_name
  nameTouched.value = false
  for (const field of Object.keys(modelValues) as CcSwitchModelField[]) modelValues[field] = ''
  void loadModels()
}, { immediate: true })

function openCcSwitch() {
  if (!props.apiKey) return
  const deeplink = buildCcSwitchImportDeeplink({
    app: app.value,
    baseUrl: props.baseUrl,
    homepage: props.homepage,
    platform: platform.value,
    name: name.value.trim(),
    apiKey: props.apiKey.key,
    models: modelValues
  })
  window.open(deeplink, '_self')
  markCcSwitchImported()
  // Still focused shortly after means no protocol handler took the link.
  setTimeout(() => {
    if (document.hasFocus()) appStore.showError(t('keys.ccSwitchNotInstalled'))
  }, 100)
  emit('close')
}
</script>

<style scoped>
.ccs-apps { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; }
.ccs-app {
  display: flex; min-width: 0; align-items: center; gap: 8px; padding: 8px 10px;
  border: 1px solid var(--surface-edge); border-radius: var(--surface-radius-control);
  background: var(--surface-soft); text-align: left;
  transition: border-color var(--motion-fast) var(--motion-ease), background-color var(--motion-fast) var(--motion-ease), box-shadow var(--motion-fast) var(--motion-ease);
}
.ccs-app:hover { border-color: theme('colors.primary.300'); }
.ccs-app[aria-checked='true'] {
  border-color: theme('colors.primary.500');
  background: rgb(99 102 241 / 8%);
  box-shadow: 0 0 0 1px theme('colors.primary.500');
}
.ccs-app:focus-visible { outline: 2px solid theme('colors.primary.500'); outline-offset: 2px; }
.ccs-app-icon { display: flex; height: 28px; width: 28px; flex: none; align-items: center; justify-content: center; border-radius: var(--surface-radius-tag); }
.ccs-app-icon-letter { background: linear-gradient(135deg, theme('colors.primary.500'), theme('colors.accent.500')); color: white; font-size: 13px; font-weight: 700; }
.ccs-app-label { min-width: 0; font-size: 13px; font-weight: 600; line-height: 1.25; color: var(--surface-text); }
.ccs-issue {
  display: flex; gap: 6px; margin-top: 10px; padding: 8px 10px; font-size: 12px; line-height: 1.6;
  border-radius: var(--surface-radius-control); background: rgb(245 158 11 / 10%); color: theme('colors.amber.700');
}
.dark .ccs-issue { color: theme('colors.amber.300'); }
.ccs-models-status { display: flex; min-height: 20px; align-items: center; gap: 8px; font-size: 12px; color: var(--surface-muted); }
.ccs-link { font-weight: 600; color: theme('colors.primary.600'); }
.dark .ccs-link { color: theme('colors.primary.300'); }
.ccs-link:hover { text-decoration: underline; }
.ccs-extra { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; }
.ccs-endpoint { display: flex; flex-direction: column; gap: 2px; padding: 8px 10px; border-radius: var(--surface-radius-control); background: var(--surface-soft); font-size: 12px; }
@media (max-width: 640px) {
  .ccs-apps { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .ccs-extra { grid-template-columns: minmax(0, 1fr); }
}
</style>
