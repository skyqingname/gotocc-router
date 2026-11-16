<template>
  <!-- 筛选条：日期与模型在点「查询」时生效，「重置」恢复默认范围并立即查询。窄屏下各项换行铺满。 -->
  <div class="card p-4">
    <div class="filter-bar">
      <div class="filter-field">
        <label class="input-label">{{ t('gotocc.taskCenter.filters.startDate') }}</label>
        <DateTimePicker v-model="start" type="date" :max="end" :aria-label="t('gotocc.taskCenter.filters.startDate')" />
      </div>
      <div class="filter-field">
        <label class="input-label">{{ t('gotocc.taskCenter.filters.endDate') }}</label>
        <DateTimePicker v-model="end" type="date" :min="start" :aria-label="t('gotocc.taskCenter.filters.endDate')" />
      </div>
      <div class="filter-field">
        <label class="input-label">{{ t('gotocc.taskCenter.filters.model') }}</label>
        <Select v-model="model" :options="modelOptions" clearable :placeholder="t('gotocc.taskCenter.filters.allModels')" :aria-label="t('gotocc.taskCenter.filters.model')" />
      </div>
      <div class="filter-actions">
        <button type="button" class="btn btn-secondary" @click="emit('reset')">{{ t('gotocc.taskCenter.filters.reset') }}</button>
        <button type="button" class="btn btn-primary" @click="emit('query')">
          <Icon name="search" size="sm" class="mr-1.5" />
          {{ t('gotocc.taskCenter.filters.query') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import DateTimePicker from '@/components/common/DateTimePicker.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{ models: string[] }>()
const emit = defineEmits<{ (event: 'reset'): void; (event: 'query'): void }>()

const start = defineModel<string>('start', { required: true })
const end = defineModel<string>('end', { required: true })
const model = defineModel<string | null>('model', { required: true })

const { t } = useI18n()
const modelOptions = computed(() => props.models.map(value => ({ value, label: value })))
</script>
