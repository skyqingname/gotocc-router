<template>
  <div v-if="contact" class="card overflow-hidden">
    <div class="flex min-h-[61px] items-center justify-between gap-3 border-b border-gray-100 px-5 py-3 dark:border-dark-700">
      <h2 class="flex min-w-0 items-center gap-2 text-base font-semibold text-gray-900 dark:text-white">
        <Icon name="chat" size="md" class="shrink-0 text-primary-600 dark:text-primary-400" />
        <span class="truncate">{{ t('gotocc.contactCard.title') }}</span>
      </h2>
      <button
        v-if="!entries"
        type="button"
        class="inline-flex h-8 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-xs font-medium transition-colors surface-muted hover:surface-hover hover:text-gray-900 dark:hover:text-white"
        @click="copyToClipboard(contact)"
      >
        <Icon name="copy" size="sm" />
        {{ t('common.copy') }}
      </button>
    </div>

    <!-- 每行都是「标签：值」、链接或单个值时逐条展示：链接在新标签页打开，其他值点一下复制 -->
    <div v-if="entries" class="space-y-2 p-4">
      <template v-for="(entry, index) in entries" :key="index">
        <a
          v-if="entry.url"
          :href="entry.url"
          target="_blank"
          rel="noopener noreferrer"
          class="text-link-area group flex w-full items-center gap-3 rounded-xl p-3 transition-colors surface-inset hover:surface-hover"
          :title="entry.value"
        >
          <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary-100 text-primary-600 dark:bg-primary-500/15 dark:text-primary-300">
            <Icon name="link" size="md" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="text-link block truncate text-sm font-medium">{{ entry.label || displayUrl(entry.url) }}</span>
            <span v-if="entry.label" class="block truncate text-xs text-gray-500 dark:text-dark-400">{{ displayUrl(entry.url) }}</span>
          </span>
          <Icon name="externalLink" size="sm" class="shrink-0 text-gray-400 transition-colors group-hover:text-primary-500 dark:text-dark-500 dark:group-hover:text-primary-400" />
        </a>
        <button
          v-else
          type="button"
          class="group flex w-full items-center gap-3 rounded-xl p-3 text-left transition-colors surface-inset hover:surface-hover"
          :title="t('common.copy')"
          @click="copyToClipboard(entry.value)"
        >
          <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary-100 text-primary-600 dark:bg-primary-500/15 dark:text-primary-300">
            <Icon name="chat" size="md" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block truncate text-sm font-medium text-gray-900 dark:text-white">{{ entry.label || entry.value }}</span>
            <span v-if="entry.label" class="block truncate text-xs tabular-nums text-gray-500 dark:text-dark-400">{{ entry.value }}</span>
          </span>
          <Icon name="copy" size="sm" class="shrink-0 text-gray-400 transition-colors group-hover:text-primary-500 dark:text-dark-500 dark:group-hover:text-primary-400" />
        </button>
      </template>
    </div>

    <!-- 一整段说明原样展示，不硬拆，标题栏右侧可整段复制 -->
    <p v-else class="whitespace-pre-wrap break-words p-5 text-sm leading-6 text-gray-700 [overflow-wrap:anywhere] dark:text-dark-200">{{ contact }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useUserView } from '@/composables/useUserView'
import { useAppStore } from '@/stores/app'

// 仪表盘「联系我们」卡，结构参照 TokenRouter（LGPL-3.0）UserDashboardContact.vue 与 utils/contactInfo.ts。
// 联系方式与顶栏用户菜单同源：站长客户读所属站长填写的内容，其他用户读站点设置；没有联系方式时整张卡不显示。
// 现有内容一行一个渠道（如「微信：xxx」、单独一行邮箱）：每行都能认出来时逐条展示，否则按一整段展示。

interface ContactEntry {
  label: string
  value: string
  url: string
}

const LINK = /^https?:\/\/\S+$/
// 「标签：值」：标签里没有冒号，值是不含空白的一段；全角半角冒号都认。
const LABELED = /^([^:：]+?)\s*[:：]\s*(\S+)$/
const SINGLE_VALUE = /^\S+$/

const { t } = useI18n()
const appStore = useAppStore()
const view = useUserView()
const { copyToClipboard } = useClipboard()

const contact = computed(() => {
  const user = view.user!
  return (user.reseller_customer ? user.reseller_customer.contact_info : appStore.contactInfo).trim()
})

const parseLine = (line: string): ContactEntry | null => {
  if (LINK.test(line)) return { label: '', value: line, url: line }
  const labeled = line.match(LABELED)
  if (labeled) return { label: labeled[1], value: labeled[2], url: LINK.test(labeled[2]) ? labeled[2] : '' }
  return SINGLE_VALUE.test(line) ? { label: '', value: line, url: '' } : null
}

// 任何一行认不出来就整段展示，返回 null。
const entries = computed(() => {
  const parsed = contact.value.split('\n').map((line) => line.trim()).filter(Boolean).map(parseLine)
  return parsed.every((entry) => entry !== null) ? (parsed as ContactEntry[]) : null
})

// 显示时去掉协议和末尾斜杠，打开的仍是原地址。
const displayUrl = (url: string) => url.replace(/^https?:\/\//, '').replace(/\/$/, '')
</script>
