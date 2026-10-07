<template>
  <div class="space-y-4 pt-4">
    <div class="rounded-xl border border-primary-100 bg-white/80 p-4 dark:border-primary-500/20 dark:bg-dark-900/70">
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('gotocc.plaza.videoFilters') }}</p>
        <span class="text-xs tabular-nums text-gray-500 dark:text-dark-400">{{ t('gotocc.plaza.modelCount', { count: visibleModels.length }) }}</span>
      </div>
      <div class="grid gap-3 sm:grid-cols-2">
        <div>
          <label :for="`video-vendor-${group.id}`" class="input-label">{{ t('gotocc.plaza.vendor') }}</label>
          <Select :id="`video-vendor-${group.id}`" v-model="vendor" :options="vendorOptions" :searchable="false" />
        </div>
        <div>
          <label :for="`video-billing-${group.id}`" class="input-label">{{ t('gotocc.plaza.billingMethod') }}</label>
          <Select :id="`video-billing-${group.id}`" v-model="billing" :options="billingOptions" :searchable="false" />
        </div>
      </div>
    </div>
    <div v-if="visibleModels.length" class="grid gap-4 md:grid-cols-2 2xl:grid-cols-3">
      <PlazaModelCard v-for="model in visibleModels" :key="`${model.platform}:${model.name}`" :group="group" :model="model" @detail="emit('detail', model)" />
    </div>
    <p v-else class="rounded-xl border border-dashed border-gray-200 p-6 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400">{{ t('modelPlaza.noSearchResult') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
import { BILLING_MODE_PER_REQUEST, BILLING_MODE_VIDEO } from '@/constants/channel'
import PlazaModelCard from './PlazaModelCard.vue'
import { canonicalVendor, vendorLabel } from './vendors'

const props = defineProps<{ group: ModelPlazaGroup }>()
const emit = defineEmits<{ detail: [model: PlazaModel] }>()
const { t } = useI18n()
const vendor = ref('all')
const billing = ref('all')
const vendorOf = (model: PlazaModel) => canonicalVendor(model.info?.vendor ?? model.platform)
const matchesVendor = (model: PlazaModel) => vendor.value === 'all' || vendorOf(model) === vendor.value
const matchesBilling = (model: PlazaModel) => billing.value === 'all' || model.pricing?.billing_mode === billing.value
const visibleModels = computed(() => props.group.models.filter(model => matchesVendor(model) && matchesBilling(model)))
const countedLabel = (label: string, count: number) => `${label} (${count})`

const vendorOptions = computed(() => {
  const matching = props.group.models.filter(matchesBilling)
  const vendors = [...new Set(props.group.models.map(vendorOf))]
  return [
    { value: 'all', label: countedLabel(t('gotocc.plaza.all'), matching.length) },
    ...vendors.map(value => ({ value, label: countedLabel(vendorLabel(value), matching.filter(model => vendorOf(model) === value).length) })),
  ]
})
const billingOptions = computed(() => {
  const matching = props.group.models.filter(matchesVendor)
  return [
    { value: 'all', label: countedLabel(t('gotocc.plaza.all'), matching.length) },
    { value: BILLING_MODE_PER_REQUEST, label: countedLabel(t('gotocc.plaza.billing.perRequest'), matching.filter(model => model.pricing?.billing_mode === BILLING_MODE_PER_REQUEST).length) },
    { value: BILLING_MODE_VIDEO, label: countedLabel(t('gotocc.plaza.billing.perSecond'), matching.filter(model => model.pricing?.billing_mode === BILLING_MODE_VIDEO).length) },
  ]
})
</script>
