<template>
  <div class="min-w-0 space-y-2">
    <ol class="divide-y divide-gray-100 dark:divide-dark-700">
      <li v-for="(id, index) in modelValue" :key="id" class="flex min-w-0 items-center gap-2 py-1">
        <span class="w-6 shrink-0 text-right text-xs tabular-nums text-gray-500">{{ index + 1 }}</span>
        <span class="min-w-0 flex-1 truncate text-sm" :title="groupName(id)">{{ groupName(id) }}</span>
        <div class="flex shrink-0 gap-1">
          <button type="button" data-action="up" class="btn btn-secondary h-11 w-11 p-0" :disabled="disabled || index === 0" :title="t('admin.smartRoutingPolicy.moveUp')" :aria-label="t('admin.smartRoutingPolicy.moveUp')" @click="move(index, -1)"><Icon name="arrowUp" size="sm" /></button>
          <button type="button" data-action="down" class="btn btn-secondary h-11 w-11 p-0" :disabled="disabled || index === modelValue.length - 1" :title="t('admin.smartRoutingPolicy.moveDown')" :aria-label="t('admin.smartRoutingPolicy.moveDown')" @click="move(index, 1)"><Icon name="arrowDown" size="sm" /></button>
          <button type="button" data-action="remove" class="btn btn-secondary h-11 w-11 p-0" :disabled="disabled" :title="t('common.remove')" :aria-label="t('common.remove')" @click="remove(index)"><Icon name="x" size="sm" /></button>
        </div>
      </li>
    </ol>
    <div class="flex min-w-0 items-center gap-2">
      <Select v-model="selected" class="min-w-0 flex-1" :options="options" :disabled="disabled || modelValue.length >= 256" :placeholder="t('admin.smartRoutingPolicy.selectGroup')" :aria-label="t('admin.smartRoutingPolicy.selectGroup')" />
      <button type="button" class="btn btn-secondary h-11 w-11 shrink-0 p-0" :disabled="disabled || selected == null || modelValue.length >= 256" :title="t('common.add')" :aria-label="t('common.add')" @click="add"><Icon name="plus" size="sm" /></button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AdminGroup } from '@/types'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{ modelValue: number[]; groups: Pick<AdminGroup, 'id' | 'name' | 'platform'>[]; disabled?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: number[]] }>()
const { t } = useI18n()
const selected = ref<number | string | null>(null)
const options = computed(() => props.groups.filter(group => !props.modelValue.includes(group.id)).map(group => ({ value: group.id, label: `${group.name} (${group.platform}) #${group.id}` })))
const groupName = (id: number) => {
  const group = props.groups.find(item => item.id === id)
  return group ? `${group.name} #${id}` : `#${id}`
}
function move(index: number, delta: number) {
  const target = index + delta
  if (props.disabled || target < 0 || target >= props.modelValue.length) return
  const value = [...props.modelValue]
  ;[value[index], value[target]] = [value[target], value[index]]
  emit('update:modelValue', value)
}
function remove(index: number) {
  emit('update:modelValue', props.modelValue.filter((_, position) => position !== index))
}
function add() {
  const id = Number(selected.value)
  if (!props.groups.some(group => group.id === id) || props.modelValue.includes(id)) return
  emit('update:modelValue', [...props.modelValue, id])
  selected.value = null
}
</script>
