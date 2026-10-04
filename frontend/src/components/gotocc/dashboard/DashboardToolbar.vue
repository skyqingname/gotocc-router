<template>
  <div class="flex flex-wrap items-center gap-2">
    <!-- 筛选面板：密钥、模型、分组，与使用记录页的同名筛选含义一致 -->
    <div ref="filterRef" class="relative">
      <button type="button" class="btn btn-secondary btn-sm" :aria-expanded="filterOpen" @click="filterOpen = !filterOpen">
        <Icon name="filter" size="sm" />
        {{ t('gotocc.dashboard.filters.title') }}
        <span v-if="activeFilterCount > 0" class="rounded-full bg-primary-600 px-1.5 text-[10px] font-semibold leading-4 text-white">{{ activeFilterCount }}</span>
      </button>
      <Transition name="dropdown-fade">
        <div v-if="filterOpen" class="dropdown left-0 mt-2 w-72 space-y-3 p-4 sm:left-auto sm:right-0">
          <div>
            <label class="input-label">{{ t('gotocc.dashboard.filters.apiKey') }}</label>
            <Select v-model="filters.api_key_id" :options="keyOptions" searchable @change="applyFilters" />
          </div>
          <div>
            <label class="input-label">{{ t('gotocc.dashboard.filters.model') }}</label>
            <Select v-model="filters.model" :options="modelOptions" searchable @change="applyFilters" />
          </div>
          <div>
            <label class="input-label">{{ t('gotocc.dashboard.filters.group') }}</label>
            <Select v-model="filters.group_id" :options="groupOptions" searchable @change="applyFilters" />
          </div>
          <button type="button" class="btn btn-ghost btn-sm w-full" :disabled="activeFilterCount === 0" @click="resetFilters">
            {{ t('common.reset') }}
          </button>
        </div>
      </Transition>
    </div>
    <button
      type="button"
      class="btn btn-secondary btn-sm px-2.5"
      :disabled="refreshing"
      :title="t('common.refresh')"
      :aria-label="t('common.refresh')"
      @click="emit('refresh')"
    >
      <Icon name="refresh" size="sm" :class="{ 'animate-spin': refreshing }" />
    </button>
    <DateRangePicker :start-date="pickerStart" :end-date="pickerEnd" @change="selectCustomRange" />
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { injectUsageState } from './usageState'

// 刷新覆盖整个仪表盘，由页面统一处理。
defineProps<{ refreshing: boolean }>()
const emit = defineEmits<{ (event: 'refresh'): void }>()

const { t } = useI18n()
const {
  filters,
  activeFilterCount,
  keyOptions,
  modelOptions,
  groupOptions,
  pickerStart,
  pickerEnd,
  applyFilters,
  resetFilters,
  selectCustomRange,
} = injectUsageState()

const filterOpen = ref(false)
const filterRef = ref<HTMLElement | null>(null)

const handleClickOutside = (event: MouseEvent) => {
  if (filterRef.value && !filterRef.value.contains(event.target as Node)) filterOpen.value = false
}

onMounted(() => document.addEventListener('click', handleClickOutside))
onBeforeUnmount(() => document.removeEventListener('click', handleClickOutside))
</script>
