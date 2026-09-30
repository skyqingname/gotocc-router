<template>
  <section class="overflow-hidden rounded-xl border border-gray-200 dark:border-dark-600" :aria-label="t('admin.users.inviter.title')">
    <div class="border-b border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-600 dark:bg-dark-800">
      <div class="flex items-center justify-between gap-3">
        <h3 class="text-sm font-semibold text-gray-900 dark:text-gray-100">{{ t('admin.users.inviter.title') }}</h3>

      </div>
      <p v-if="loading" class="mt-2 text-sm text-gray-500">{{ t('admin.users.inviter.loading') }}</p>
      <div v-else-if="loadError" class="mt-2 flex items-center justify-between gap-3 text-sm text-red-600 dark:text-red-400">
        <span role="alert">{{ loadError }}</span>
        <button type="button" class="shrink-0 font-medium underline" @click="load">{{ t('admin.users.inviter.retry') }}</button>
      </div>
      <div v-else-if="state?.inviter" class="mt-2">
        <div class="flex flex-wrap items-baseline gap-x-2 gap-y-1">
          <span class="text-sm text-gray-500">{{ t('admin.users.inviter.current') }}</span>
          <span class="text-sm font-medium text-gray-900 dark:text-gray-100">{{ state.inviter.username || state.inviter.email }}</span>
          <span class="font-mono text-xs text-gray-500">#{{ state.inviter.id }}</span>
        </div>
        <p class="mt-1 break-all text-xs text-gray-500">{{ state.inviter.email }}</p>
      </div>
      <p v-else class="mt-2 text-sm text-gray-500">{{ t('admin.users.inviter.unbound') }}</p>
    </div>

    <div v-if="state && !loading" class="space-y-3 p-4">
      <div>
        <label for="user-inviter-code" class="input-label">{{ t('admin.users.inviter.codeLabel') }}</label>
        <div class="flex gap-2">
          <input id="user-inviter-code" v-model="code" class="input min-w-0 flex-1 font-mono" type="text" autocomplete="off"
            :disabled="disabled" :placeholder="t('admin.users.inviter.codePlaceholder')" @keydown.enter.prevent="resolve" />
          <button type="button" class="btn btn-secondary shrink-0" :disabled="disabled || resolving || !code.trim()" @click="resolve">
            {{ resolving ? t('admin.users.inviter.resolving') : t('admin.users.inviter.resolve') }}
          </button>
        </div>
        <p class="input-hint">{{ t('admin.users.inviter.codeHint') }}</p>
      </div>
      <p v-if="resolveError" role="alert" class="text-sm text-red-600 dark:text-red-400">{{ resolveError }}</p>
      <div v-if="resolved" class="rounded-lg border border-primary-200 bg-primary-50 px-3 py-2.5 dark:border-primary-800 dark:bg-primary-950/30" aria-live="polite">
        <p class="text-xs font-medium text-primary-700 dark:text-primary-300">{{ t(hasChange ? 'admin.users.inviter.next' : 'admin.users.inviter.matched') }}</p>
        <div class="mt-1 flex flex-wrap items-baseline gap-x-2">
          <span class="text-sm font-semibold text-gray-900 dark:text-gray-100">{{ resolved.username || resolved.email }}</span>
          <span class="font-mono text-xs text-gray-500">#{{ resolved.id }}</span>
        </div>
        <p class="mt-0.5 break-all text-xs text-gray-500">{{ resolved.email }}</p>
      </div>
      <p v-else-if="dirty" class="text-xs text-amber-700 dark:text-amber-300">{{ t('admin.users.inviter.resolveFirst') }}</p>
      <p class="text-xs leading-relaxed text-gray-500">{{ t('admin.users.inviter.effect') }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { AffiliateInviterChange, AffiliateInviterState, AffiliateInviterUser } from '@/api/admin/affiliates'

const props = defineProps<{ userId: number, active: boolean, disabled: boolean }>()
const emit = defineEmits<{ 'update:change': [value: AffiliateInviterChange | null], validity: [valid: boolean] }>()
const { t } = useI18n()
const state = ref<AffiliateInviterState | null>(null)
const code = ref('')
const loading = ref(false)
const resolving = ref(false)
const loadError = ref('')
const resolveError = ref('')
const resolved = ref<AffiliateInviterUser | null>(null)
let loadSequence = 0
let resolveSequence = 0

const normalizedCode = computed(() => code.value.trim().toUpperCase())
const dirty = computed(() => state.value !== null && (
  normalizedCode.value !== state.value.code.toUpperCase()
))
const hasChange = computed(() => dirty.value || (resolved.value !== null && resolved.value.id !== state.value?.inviter?.id))
const load = async () => {
  const sequence = ++loadSequence
  ++resolveSequence
  state.value = null
  resolved.value = null
  resolving.value = false
  loadError.value = ''
  resolveError.value = ''
  emit('update:change', null)
  emit('validity', true)
  if (!props.active) return
  loading.value = true
  try {
    const result = await adminAPI.affiliates.getInviter(props.userId)
    if (sequence !== loadSequence) return
    state.value = result
    code.value = result.code
  } catch (error: any) {
    if (sequence === loadSequence) loadError.value = error.message
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}

watch(code, () => {
  ++resolveSequence
  resolved.value = null
  resolveError.value = ''
  resolving.value = false
  emit('update:change', null)
  emit('validity', !dirty.value)
})

const resolve = async () => {
  if (!state.value || !normalizedCode.value || props.disabled || resolving.value) return
  const sequence = ++resolveSequence
  const ownerVersion = state.value.version
  const value = normalizedCode.value
  resolving.value = true
  resolveError.value = ''
  resolved.value = null
  emit('update:change', null)
  emit('validity', !dirty.value)
  try {
    const result = await adminAPI.affiliates.resolveInviterCode(value)
    if (sequence !== resolveSequence) return
    if (result.id === props.userId) {
      resolveError.value = t('admin.users.inviter.self')
      emit('validity', false)
      return
    }
    resolved.value = result
    emit('update:change', hasChange.value ? { code: value, resolved_user_id: result.id, expected_version: ownerVersion } : null)
    emit('validity', true)
  } catch (error: any) {
    if (sequence !== resolveSequence) return
    resolveError.value = error.reason === 'AFFILIATE_CODE_INVALID' ? t('admin.users.inviter.notFound') : error.message
    emit('validity', !dirty.value)
  } finally {
    if (sequence === resolveSequence) resolving.value = false
  }
}

watch(() => [props.active, props.userId], load, { immediate: true })
onBeforeUnmount(() => { ++loadSequence; ++resolveSequence })
</script>
