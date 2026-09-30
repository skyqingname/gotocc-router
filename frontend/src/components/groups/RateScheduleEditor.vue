<template>
  <section class="space-y-3 rounded-lg border border-gray-200 p-4 dark:border-gray-700">
    <label class="flex items-center gap-2 font-medium">
      <input
        data-test="schedule-enabled"
        type="checkbox"
        :checked="modelValue.enabled"
        @change="setEnabled"
      />
      {{ t('enabled') }}
    </label>
    <p class="text-sm text-gray-500">{{ t('semantics') }}</p>
    <label class="block space-y-1 text-sm">
      <span>{{ t('timezone') }}</span>
      <input
        data-test="timezone"
        type="text"
        class="input w-full"
        :value="modelValue.timezone"
        :placeholder="serverTimezone"
        @input="setTimezone"
      />
    </label>
    <p class="text-xs text-gray-500">{{ t('inheritTimezone', { timezone: serverTimezone }) }}</p>
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left">
            <th class="p-1">{{ t('active') }}</th>
            <th class="p-1">{{ t('start') }}</th>
            <th class="p-1">{{ t('end') }}</th>
            <th class="p-1">{{ t('factor') }}</th>
            <th class="p-1"><span class="sr-only">{{ t('remove') }}</span></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(rule, index) in modelValue.rules" :key="rule.id" data-test="rule">
            <td class="p-1">
              <input type="checkbox" :aria-label="t('active')" :checked="rule.enabled" @change="setRuleEnabled(index, $event)" />
            </td>
            <td class="p-1">
              <input class="input w-24" type="text" inputmode="text" placeholder="00:00" :aria-label="t('start')" :value="rule.start" @input="setClock(index, 'start', $event)" />
            </td>
            <td class="p-1">
              <input class="input w-24" type="text" inputmode="text" placeholder="24:00" :aria-label="t('end')" :value="rule.end" @input="setClock(index, 'end', $event)" />
            </td>
            <td class="p-1">
              <input class="input w-28" type="number" min="0" step="any" :aria-label="t('factor')" :value="rule.multiplier" @input="setFactor(index, $event)" />
            </td>
            <td class="p-1">
              <button data-test="remove-rule" type="button" class="btn btn-secondary" @click="removeRule(index)">{{ t('remove') }}</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <button
      data-test="add-rule"
      type="button"
      class="btn btn-secondary"
      :disabled="modelValue.rules.length >= MAX_RATE_SCHEDULE_RULES"
      @click="addRule"
    >{{ t('add') }}</button>
    <p class="text-xs text-gray-500">{{ t('boundaries') }}</p>
    <p v-if="error" data-test="error" role="alert" class="text-sm text-red-600">
      {{ t(`errors.${error.code}`) }} <span class="text-xs">({{ error.field }})</span>
    </p>
    <p v-else-if="preview" data-test="preview" aria-live="polite" class="text-sm">
      {{ t('preview', { base: preview.base_multiplier, factor: preview.schedule_multiplier, effective: formatMultiplier(preview.effective_multiplier) }) }}
      <span class="text-gray-500"> · {{ preview.timezone }}</span>
    </p>
    <p class="text-xs text-gray-500">{{ t('authority') }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  compileRateSchedule,
  MAX_RATE_SCHEDULE_RULES,
  RateScheduleError,
  type RateScheduleConfig,
  type RateScheduleRule
} from '@/utils/rate-schedule'

const props = defineProps<{
  modelValue: RateScheduleConfig
  serverTimezone: string
  baseMultiplier: number
  previewAt?: Date
}>()
const emit = defineEmits<{
  (event: 'update:modelValue', value: RateScheduleConfig): void
  (event: 'validity', valid: boolean): void
}>()

const en = {
  enabled: 'Enable text time-window rates',
  semantics: 'Additional factor: effective multiplier = resolved base multiplier × window factor. Images and videos retain their existing rates. Disabling preserves the saved rules.',
  timezone: 'IANA timezone',
  inheritTimezone: 'Leave blank to inherit the server timezone: {timezone}.',
  active: 'Active', start: 'Start', end: 'End', factor: 'Additional factor',
  add: 'Add window', remove: 'Remove',
  boundaries: 'Start is inclusive; end is exclusive. Overnight windows are allowed. Use 00:00–24:00 for all day. Enabled windows must not overlap.',
  preview: 'Current preview: {base} × {factor} = {effective}',
  authority: 'Preview only. The server must validate, save, and snapshot the rate before billing.',
  errors: {
    invalid_timezone: 'Enter a valid IANA timezone or configure an explicit server timezone.',
    invalid_enabled: 'The enable flag must be a boolean.',
    invalid_rules: 'Rules must be a list.',
    too_many_rules: 'At most 64 windows are allowed.',
    invalid_id: 'A rule has an invalid identifier.',
    duplicate_id: 'Rule identifiers must be unique.',
    invalid_time: 'Use HH:MM. 24:00 is allowed only as an end time.',
    empty_window: 'Start and end must differ.',
    invalid_multiplier: 'The factor must be finite and non-negative; zero is allowed.',
    overlapping_windows: 'Enabled windows overlap. Change or disable one of them.',
    invalid_base: 'The base multiplier must be finite and non-negative.',
    invalid_result: 'The effective multiplier is outside the supported numeric range.'
  }
}
const zh: typeof en = {
  enabled: '启用文本时段倍率',
  semantics: '时段系数与基础倍率相乘：文本实际倍率 = 已确定的基础倍率 × 时段系数。图片和视频保持原有倍率，关闭开关会保留规则。',
  timezone: 'IANA 时区',
  inheritTimezone: '留空则继承服务器时区：{timezone}。',
  active: '启用', start: '开始时间', end: '结束时间', factor: '时段系数',
  add: '添加时段', remove: '删除',
  boundaries: '包含开始时间，不包含结束时间；支持跨午夜。全天请填写 00:00–24:00。已启用时段不可重叠。',
  preview: '当前预览：{base} × {factor} = {effective}',
  authority: '此处仅为预览；计费前必须由服务端校验、保存并固定实际倍率。',
  errors: {
    invalid_timezone: '请填写有效的 IANA 时区，或明确配置服务器时区。',
    invalid_enabled: '启用状态必须为布尔值。',
    invalid_rules: '时段规则必须为列表。',
    too_many_rules: '最多支持 64 条时段规则。',
    invalid_id: '规则标识无效。',
    duplicate_id: '规则标识不可重复。',
    invalid_time: '请使用 HH:MM；24:00 仅可用于结束时间。',
    empty_window: '开始和结束时间不能相同。',
    invalid_multiplier: '系数必须是有限的非负数，允许填写 0。',
    overlapping_windows: '已启用的时段有重叠，请调整或停用其中一条。',
    invalid_base: '基础倍率必须是有限的非负数。',
    invalid_result: '实际倍率超出支持的数值范围。'
  }
}
const { t, locale } = useI18n({ useScope: 'local', inheritLocale: true, messages: { en, zh } })
const now = ref(new Date())
let timer: ReturnType<typeof setInterval> | undefined
onMounted(() => { timer = setInterval(() => { now.value = new Date() }, 1000) })
onUnmounted(() => { if (timer !== undefined) clearInterval(timer) })

const compiled = computed(() => {
  try {
    return { schedule: compileRateSchedule(props.modelValue, props.serverTimezone), error: null }
  } catch (error) {
    return { schedule: null, error: asScheduleError(error) }
  }
})
const state = computed(() => {
  if (!compiled.value.schedule) return { preview: null, error: compiled.value.error }
  try {
    return {
      preview: compiled.value.schedule.resolve(props.previewAt ?? now.value, props.baseMultiplier),
      error: null
    }
  } catch (error) {
    return { preview: null, error: asScheduleError(error) }
  }
})
const error = computed(() => state.value.error)
const preview = computed(() => state.value.preview)
watch(() => error.value === null, valid => emit('validity', valid), { immediate: true })

function asScheduleError(error: unknown): RateScheduleError {
  return error instanceof RateScheduleError ? error : new RateScheduleError('invalid_rules', 'rules')
}
function patchConfig(patch: Partial<RateScheduleConfig>): void {
  emit('update:modelValue', { ...props.modelValue, ...patch })
}
function patchRule(index: number, patch: Partial<RateScheduleRule>): void {
  patchConfig({ rules: props.modelValue.rules.map((rule, position) => position === index ? { ...rule, ...patch } : { ...rule }) })
}
function setEnabled(event: Event): void { patchConfig({ enabled: (event.target as HTMLInputElement).checked }) }
function setTimezone(event: Event): void { patchConfig({ timezone: (event.target as HTMLInputElement).value }) }
function setRuleEnabled(index: number, event: Event): void { patchRule(index, { enabled: (event.target as HTMLInputElement).checked }) }
function setClock(index: number, field: 'start' | 'end', event: Event): void { patchRule(index, { [field]: (event.target as HTMLInputElement).value }) }
function setFactor(index: number, event: Event): void { patchRule(index, { multiplier: (event.target as HTMLInputElement).valueAsNumber }) }
function removeRule(index: number): void { patchConfig({ rules: props.modelValue.rules.filter((_, position) => position !== index) }) }
function addRule(): void {
  if (props.modelValue.rules.length >= MAX_RATE_SCHEDULE_RULES) return
  const ids = new Set(props.modelValue.rules.map(rule => rule.id))
  let suffix = 1
  while (ids.has(`rule_${suffix}`)) suffix++
  patchConfig({ rules: [...props.modelValue.rules, { id: `rule_${suffix}`, enabled: true, start: '', end: '', multiplier: 1 }] })
}
function formatMultiplier(value: number): string { return new Intl.NumberFormat(locale.value, { maximumSignificantDigits: 12 }).format(value) }
</script>
