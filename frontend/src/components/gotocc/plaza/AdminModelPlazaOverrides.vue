<template>
  <div class="space-y-3 border-t border-gray-100 pt-5 dark:border-dark-700">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="min-w-0">
        <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('gotocc.plaza.admin.title') }}</p>
        <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('gotocc.plaza.admin.description') }}</p>
      </div>
      <button type="button" class="btn btn-primary btn-sm" :disabled="saving || loading" @click="save">
        <Icon name="check" size="sm" />{{ t('gotocc.plaza.admin.save') }}
      </button>
    </div>
    <input v-model.trim="search" type="search" class="input max-w-xs" :placeholder="t('gotocc.plaza.searchPlaceholder')" />
    <div v-if="loading" class="py-6 text-center text-sm text-gray-500">{{ t('common.loading') }}</div>
    <div v-else class="max-h-[32rem] divide-y divide-gray-100 overflow-y-auto rounded-xl border border-gray-200 dark:divide-dark-700 dark:border-dark-700">
      <div v-for="row in visibleRows" :key="row.key" class="grid gap-3 p-3 lg:grid-cols-[14rem_10rem_1fr]">
        <div class="min-w-0">
          <p class="flex items-center gap-2">
            <ModelIcon :model="row.name" size="16px" />
            <span class="truncate text-sm font-medium text-gray-900 dark:text-white" :title="row.name">{{ row.name }}</span>
          </p>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
            {{ row.source ? t('gotocc.plaza.admin.fromCatalog') : t('gotocc.plaza.admin.notInCatalog') }}
          </p>
          <div class="mt-2 flex flex-wrap gap-2">
            <label v-for="purpose in PURPOSES" :key="purpose" class="flex items-center gap-1 text-xs text-gray-600 dark:text-dark-300">
              <input v-model="drafts[row.key].purposes" type="checkbox" :value="purpose" class="h-3.5 w-3.5 rounded border-gray-300 text-primary-600" />
              {{ t(`gotocc.plaza.purposes.${purpose}`) }}
            </label>
          </div>
        </div>
        <div>
          <input v-model.trim="drafts[row.key].vendor" class="input text-sm" list="gc-plaza-vendors" :placeholder="row.autoVendor" />
        </div>
        <textarea v-model.trim="drafts[row.key].description" rows="2" class="input text-sm" :placeholder="row.autoDescription || t('gotocc.plaza.admin.descriptionPlaceholder')"></textarea>
      </div>
      <p v-if="visibleRows.length === 0" class="py-6 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('modelPlaza.noSearchResult') }}</p>
    </div>
    <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('gotocc.plaza.admin.hint') }}</p>
    <datalist id="gc-plaza-vendors">
      <option v-for="(label, id) in VENDOR_LABELS" :key="id" :value="id">{{ label }}</option>
    </datalist>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { getModelPlaza } from '@/api/modelPlaza'
import { getModelPlazaOverrides, updateModelPlazaOverrides, type PlazaModelOverride } from '@/api/admin/modelPlazaOverrides'
import { useAppStore } from '@/stores/app'
import { PURPOSES, VENDOR_LABELS, vendorLabel } from './vendors'

// 后台按模型填写的展示信息：留空沿用 models.dev 的自动值；占位文字显示当前展示的值。
const { t } = useI18n()
const appStore = useAppStore()

interface Row {
  key: string
  name: string
  source: boolean
  autoVendor: string
  autoDescription: string
}

const rows = ref<Row[]>([])
const drafts = reactive<Record<string, PlazaModelOverride>>({})
const search = ref('')
const loading = ref(true)
const saving = ref(false)

const visibleRows = computed(() => {
  const keyword = search.value.toLowerCase()
  return rows.value.filter((row) => !keyword || row.name.toLowerCase().includes(keyword))
})

const load = async () => {
  loading.value = true
  const [plaza, overrides] = await Promise.all([getModelPlaza(), getModelPlazaOverrides()]).finally(() => {
    loading.value = false
  })
  const seen = new Map<string, Row>()
  for (const group of plaza.groups) {
    for (const model of group.models) {
      const key = model.name.toLowerCase()
      if (seen.has(key)) continue
      const override = overrides[key]
      seen.set(key, {
        key,
        name: model.name,
        source: model.info?.source === 'models.dev',
        // 已有覆盖时展示值即覆盖值，占位只在未覆盖时有意义。
        autoVendor: override?.vendor ? '' : vendorLabel(model.info?.vendor ?? model.platform),
        autoDescription: override?.description ? '' : model.info?.description ?? '',
      })
      drafts[key] = { description: override?.description ?? '', vendor: override?.vendor ?? '', purposes: [...(override?.purposes ?? [])] }
    }
  }
  rows.value = [...seen.values()].sort((a, b) => a.name.localeCompare(b.name))
}

const save = async () => {
  saving.value = true
  await updateModelPlazaOverrides({ ...drafts }).finally(() => {
    saving.value = false
  })
  appStore.showSuccess(t('gotocc.plaza.admin.saved'))
  await load()
}

onMounted(() => void load())
</script>
