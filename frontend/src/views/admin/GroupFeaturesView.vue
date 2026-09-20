<template>
  <AppLayout>
    <main class="mx-auto max-w-5xl space-y-5 p-6">
      <header class="flex flex-wrap items-center justify-between gap-3">
        <h1 class="text-xl font-semibold">{{ t('title') }}</h1>
        <RouterLink class="btn btn-secondary" to="/admin/groups">{{ t('back') }}</RouterLink>
      </header>
      <p class="text-sm text-gray-500">{{ t('explanation') }}</p>
      <div class="flex flex-wrap items-end gap-3">
        <label class="flex-1 space-y-1">
          <span class="text-sm">{{ t('group') }}</span>
          <select v-model.number="selectedID" class="input w-full" :disabled="busy">
            <option :value="0" disabled>{{ t('choose') }}</option>
            <option v-for="group in groups" :key="group.id" :value="group.id">#{{ group.id }} · {{ group.name }}</option>
          </select>
        </label>
        <button type="button" class="btn btn-secondary" :disabled="busy || !selectedID" @click="loadSelected">{{ t('load') }}</button>
      </div>
      <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
      <p v-if="notice" role="status" class="text-sm text-emerald-600">{{ notice }}</p>
      <div v-if="config && activeGroup && current" class="space-y-4" :aria-busy="busy">
        <p class="text-sm">{{ activeGroup.name }} · {{ t('base') }} {{ activeGroup.rate_multiplier }} · {{ t('version') }} {{ current.version }}</p>
        <p v-if="current.source === 'legacy'" class="text-sm text-amber-700">{{ t('legacy') }}</p>
        <fieldset :disabled="busy" class="space-y-3">
          <RateScheduleEditor
            v-model="config"
            :server-timezone="current.server_timezone"
            :base-multiplier="activeGroup.rate_multiplier"
            @validity="valid = $event"
          />
          <div class="flex items-center gap-3">
            <button type="button" data-test="save" class="btn btn-primary" :disabled="busy || !valid || !dirty" @click="save">{{ busy ? t('working') : t('save') }}</button>
            <span v-if="dirty" class="text-sm text-amber-700">{{ t('unsaved') }}</span>
          </div>
        </fieldset>
        <p class="text-xs text-gray-500">{{ t('scope') }}</p>
      </div>
    </main>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { onBeforeRouteLeave, RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { isAxiosError } from 'axios'
import AppLayout from '@/components/layout/AppLayout.vue'
import RateScheduleEditor from '@/components/groups/RateScheduleEditor.vue'
import { getAllIncludingInactive } from '@/api/admin/groups'
import { getGroupRateSchedule, saveGroupRateSchedule, type GroupRateScheduleView } from '@/api/admin/groupFeatures'
import type { RateScheduleConfig } from '@/utils/rate-schedule'
import type { AdminGroup } from '@/types'

const en = {
  title: 'Group time-window rates', back: 'Back to groups', group: 'Group', choose: 'Choose a group',
  explanation: 'Configure independently saved daily rate windows for an existing group. Base rates and account bindings are not changed by this page.',
  load: 'Load configuration', base: 'Base multiplier', version: 'Version',
  legacy: 'This is the existing single-window configuration. Saving creates a versioned schedule that replaces, rather than stacks with, that window.',
  save: 'Save schedule', working: 'Working…', unsaved: 'Unsaved changes',
  scope: 'Applies to text/token pricing. Independent image and video prices are unchanged. Existing WebSocket sessions retain the configuration loaded when connected.',
  discard: 'Discard unsaved schedule changes?', saved: 'Schedule saved. New HTTP requests use this version.',
  conflict: 'Another administrator changed this configuration. Your edits are retained; reload and reconcile before saving again.',
  failed: 'The operation failed. No success is assumed; reload to confirm the saved version.'
}
const zh: typeof en = {
  title: '分组时段倍率', back: '返回分组管理', group: '分组', choose: '请选择分组',
  explanation: '为已有分组单独保存每日时段规则；本页不会修改基础倍率或账号绑定。',
  load: '加载配置', base: '基础倍率', version: '版本',
  legacy: '当前显示原有单时段配置。保存后启用有版本记录的新规则，替代原规则，不重复叠乘。',
  save: '保存时段配置', working: '处理中…', unsaved: '存在未保存修改',
  scope: '用于文本和 token 计费，不改变独立图片、视频价格。已连接的 WebSocket 会话保留建立连接时加载的配置。',
  discard: '放弃尚未保存的时段修改？', saved: '配置已保存，新的 HTTP 请求使用该版本。',
  conflict: '其他管理员已更新配置。你的修改仍保留，请重新加载并核对后再保存。',
  failed: '操作未成功确认。请重新加载核对已保存版本，不要据此认定保存成功。'
}
const { t } = useI18n({ useScope: 'local', inheritLocale: true, messages: { en, zh } })
const route = useRoute()
const groups = ref<AdminGroup[]>([])
const selectedID = ref(0)
const activeID = ref(0)
const current = ref<GroupRateScheduleView | null>(null)
const config = ref<RateScheduleConfig | null>(null)
const baseline = ref('')
const busy = ref(false)
const valid = ref(false)
const error = ref('')
const notice = ref('')
const dirty = computed(() => config.value !== null && JSON.stringify(config.value) !== baseline.value)
const activeGroup = computed(() => groups.value.find(group => group.id === activeID.value))

function accept(view: GroupRateScheduleView): void {
  current.value = view
  activeID.value = view.group_id
  config.value = structuredClone(view.config)
  baseline.value = JSON.stringify(config.value)
}

async function loadSelected(): Promise<void> {
  if (busy.value || !selectedID.value) return
  if (dirty.value && !window.confirm(t('discard'))) return
  busy.value = true
  error.value = notice.value = ''
  try { accept(await getGroupRateSchedule(selectedID.value)) }
  catch { error.value = t('failed') }
  finally { busy.value = false }
}

async function save(): Promise<void> {
  if (busy.value || !valid.value || !config.value || !current.value || !activeID.value) return
  const id = activeID.value
  const version = current.value.version
  const payload = JSON.parse(JSON.stringify(config.value)) as RateScheduleConfig
  busy.value = true
  error.value = notice.value = ''
  try {
    accept(await saveGroupRateSchedule(id, version, payload))
    notice.value = t('saved')
  } catch (failure) {
    error.value = isAxiosError(failure) && failure.response?.status === 409 ? t('conflict') : t('failed')
  } finally { busy.value = false }
}

onBeforeRouteLeave(() => !dirty.value || window.confirm(t('discard')))
onMounted(async () => {
  busy.value = true
  try {
    groups.value = await getAllIncludingInactive()
    const requested = Number(route.query.group)
    selectedID.value = groups.value.some(group => group.id === requested) ? requested : (groups.value[0]?.id ?? 0)
  } catch { error.value = t('failed') }
  finally { busy.value = false }
  if (selectedID.value) await loadSelected()
})
</script>
