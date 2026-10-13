<template>
  <BaseDialog :show="show" :title="t('admin.smartRoutingPolicy.title')" width="wide" @close="close">
    <p v-if="loading" class="py-8 text-center text-sm text-gray-500">{{ t('common.loading') }}</p>
    <div v-else class="space-y-6">
      <p v-if="error" role="alert" class="text-sm text-red-600 dark:text-red-400">{{ error }}</p>
      <section class="rounded-xl border border-primary-200 bg-primary-50/60 p-4 dark:border-primary-800/50 dark:bg-primary-900/10">
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0">
            <h4 id="user-routing-permission-label" class="text-sm font-semibold text-gray-900 dark:text-gray-100">{{ t('admin.smartRoutingPolicy.allowUserOverride') }}</h4>
            <p id="user-routing-permission-hint" class="mt-1 text-xs leading-relaxed text-gray-600 dark:text-dark-300">{{ t('admin.smartRoutingPolicy.allowUserOverrideHint') }}</p>
          </div>
          <button type="button" role="switch" data-test="user-routing-permission" :aria-checked="!!policy.allow_user_override" aria-labelledby="user-routing-permission-label" aria-describedby="user-routing-permission-hint" :disabled="saving || !loaded" class="flex min-h-11 min-w-11 shrink-0 items-center justify-center rounded-lg focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500" @click="policy.allow_user_override = !policy.allow_user_override">
            <span class="relative block h-6 w-11 rounded-full transition-colors" :class="policy.allow_user_override ? 'bg-primary-600' : 'bg-gray-300 dark:bg-dark-500'"><span class="absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform" :class="policy.allow_user_override ? 'translate-x-5' : 'translate-x-0'" /></span>
          </button>
        </div>
        <p class="mt-3 text-xs font-medium text-primary-700 dark:text-primary-300">{{ t(policy.allow_user_override ? 'admin.smartRoutingPolicy.userOverrideEnabled' : 'admin.smartRoutingPolicy.userOverrideDisabled') }}</p>
      </section>
      <section class="space-y-2">
        <h4 class="text-sm font-semibold">{{ t('admin.smartRoutingPolicy.defaultOrder') }}</h4>
        <AutoGroupOrderEditor v-model="policy.default_group_order" :groups="groups" :disabled="saving || !loaded" />
      </section>
      <section class="space-y-3">
        <div class="flex items-center justify-between gap-3">
          <h4 class="text-sm font-semibold">{{ t('admin.smartRoutingPolicy.modelRules') }}</h4>
          <button type="button" class="btn btn-secondary" :disabled="saving || !loaded || policy.model_rules.length >= 256" @click="policy.model_rules.push({ model: '', group_ids: [] })"><Icon name="plus" size="sm" class="mr-2" />{{ t('admin.smartRoutingPolicy.addRule') }}</button>
        </div>
        <div v-for="(rule, index) in policy.model_rules" :key="index" class="space-y-2 border-t border-gray-200 pt-3 dark:border-dark-600">
          <div class="flex items-end gap-2">
            <label class="min-w-0 flex-1 text-sm">
              <span class="mb-1 block">{{ t('admin.smartRoutingPolicy.model') }}</span>
              <input v-model="rule.model" class="input w-full" maxlength="256" :disabled="saving" :placeholder="t('admin.smartRoutingPolicy.modelPlaceholder')" />
            </label>
            <button type="button" class="btn btn-secondary h-11 w-11 shrink-0 p-0" :disabled="saving" :title="t('admin.smartRoutingPolicy.removeRule')" :aria-label="t('admin.smartRoutingPolicy.removeRule')" @click="policy.model_rules.splice(index, 1)"><Icon name="trash" size="sm" /></button>
          </div>
          <AutoGroupOrderEditor v-model="rule.group_ids" :groups="groups" :disabled="saving" />
        </div>
      </section>
    </div>
    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="saving" @click="close">{{ t('common.cancel') }}</button>
      <button type="button" class="btn btn-primary" :disabled="loading || saving || !loaded || policy.model_rules.some(rule => !rule.model.trim())" @click="save">{{ t('common.save') }}</button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AdminGroup } from '@/types'
import { getAllIncludingInactive } from '@/api/admin/groups'
import { getAutoRoutingPolicy, updateAutoRoutingPolicy, type AutoGroupRoutingPolicy } from '@/api/admin/apiKeys'
import { useAppStore } from '@/stores/app'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import AutoGroupOrderEditor from './AutoGroupOrderEditor.vue'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: []; saved: [] }>()
const { t } = useI18n()
const app = useAppStore()
const policy = ref<AutoGroupRoutingPolicy>({ allow_user_override: false, default_group_order: [], model_rules: [] })
const groups = ref<AdminGroup[]>([])
const loading = ref(false)
const saving = ref(false)
const loaded = ref(false)
const error = ref('')
let generation = 0
watch(() => props.show, async (show) => {
  const current = ++generation
  if (!show) return
  loading.value = true
  loaded.value = false
  policy.value = { allow_user_override: false, default_group_order: [], model_rules: [] }
  error.value = ''
  try {
    const [data, available] = await Promise.all([getAutoRoutingPolicy(), getAllIncludingInactive()])
    if (current !== generation) return
    policy.value = data
    groups.value = available
    loaded.value = true
  } catch {
    if (current === generation) error.value = t('admin.smartRoutingPolicy.loadFailed')
  } finally {
    if (current === generation) loading.value = false
  }
}, { immediate: true })
function close() { if (!saving.value) emit('close') }
async function save() {
  saving.value = true
  error.value = ''
  try {
    await updateAutoRoutingPolicy(policy.value)
    app.showSuccess(t('admin.smartRoutingPolicy.saved'))
    emit('saved')
    emit('close')
  } catch {
    error.value = t('admin.smartRoutingPolicy.saveFailed')
  } finally { saving.value = false }
}
</script>
