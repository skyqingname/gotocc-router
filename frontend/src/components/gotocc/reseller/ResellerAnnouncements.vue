<template>
  <div class="space-y-5">
    <section class="card p-5 sm:p-6">
      <div class="grid gap-6 md:grid-cols-2">
        <div class="flex items-start justify-between gap-5">
          <div><label for="reseller-announcements-enabled" class="text-sm font-semibold">{{ tr('向客户展示公告', 'Show customer announcements') }}</label><p class="mt-1.5 text-xs leading-5 text-gray-500">{{ tr('默认关闭。开启后，客户只看到本站已发布的公告。', 'Off by default. When enabled, customers see only announcements published by your site.') }}</p></div>
          <Toggle id="reseller-announcements-enabled" v-model="enabled" :aria-label="tr('向客户展示公告', 'Show customer announcements')" />
        </div>
        <div class="flex items-start justify-between gap-5 md:border-l md:border-gray-100 md:pl-6 dark:md:border-dark-700">
          <div><label for="reseller-announcement-sync" class="text-sm font-semibold">{{ tr('接收主站公告待审核', 'Receive main-site announcements for review') }}</label><p class="mt-1.5 text-xs leading-5 text-gray-500">{{ tr('默认开启。收到后须由你审核，才能向客户发布。', 'On by default. Your approval is required before any received announcement is published to customers.') }}</p></div>
          <Toggle id="reseller-announcement-sync" v-model="syncMain" :aria-label="tr('接收主站公告待审核', 'Receive main-site announcements for review')" />
        </div>
      </div>
      <div class="mt-5 flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700"><button type="button" class="btn btn-secondary btn-sm" :disabled="settingsSaving || !settingsChanged" @click="saveSettings">{{ settingsSaving ? tr('保存中…', 'Saving…') : tr('保存公告设置', 'Save settings') }}</button></div>
    </section>

    <p v-if="!profile.announcements_enabled" class="flex items-start gap-2 rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-xs leading-5 text-gray-600 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-300">
      <Icon name="bell" size="sm" class="mt-0.5 shrink-0" />
      {{ tr('客户公告目前已关闭。你可以写稿和审核，开启展示并保存设置后，客户才会看到已发布内容。', 'Customer announcements are currently hidden. You can write and review now; published content becomes visible after enabling display and saving the setting.') }}
    </p>

    <section class="card overflow-hidden">
      <header class="flex flex-wrap items-center justify-between gap-4 border-b border-gray-100 p-5 dark:border-dark-700">
        <div class="flex gap-1 rounded-lg bg-gray-100 p-1 text-sm dark:bg-dark-800" role="tablist" :aria-label="tr('公告来源', 'Announcement source')">
          <button type="button" role="tab" :aria-selected="tab === 'own'" :class="tabClass('own')" @click="tab = 'own'">{{ tr('本站公告', 'Site announcements') }}</button>
          <button type="button" role="tab" :aria-selected="tab === 'main'" :class="tabClass('main')" @click="tab = 'main'">{{ tr('主站待审核', 'Main-site review') }}<span v-if="pendingCount" class="ml-1.5 rounded bg-amber-100 px-1.5 py-0.5 text-[11px] font-semibold text-amber-800 dark:bg-amber-900/30 dark:text-amber-300">{{ pendingCount }}</span></button>
        </div>
        <div class="flex gap-2"><button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="load">{{ tr('刷新', 'Refresh') }}</button><button type="button" class="btn btn-primary btn-sm" @click="openEditor()">{{ tr('发新公告', 'New announcement') }}</button></div>
      </header>
      <p v-if="error" role="alert" class="m-5 text-sm text-red-600">{{ error }}</p>
      <p v-if="loading" class="p-10 text-center text-sm text-gray-500">{{ tr('正在读取公告…', 'Loading announcements…') }}</p>
      <template v-else-if="tab === 'own'">
        <div v-if="items.length" class="divide-y divide-gray-100 dark:divide-dark-700">
          <article v-for="item in items" :key="item.id" class="flex flex-col gap-4 p-5 sm:flex-row sm:items-start">
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2"><span :class="statusClass(item.status)" class="rounded px-2 py-0.5 text-xs font-medium">{{ statusLabel(item.status) }}</span><span v-if="item.source_announcement_id" class="text-xs text-gray-400">{{ tr('主站同步', 'Synced from main site') }}</span><span class="text-xs text-gray-400">{{ date(item.updated_at) }}</span></div>
              <button type="button" class="mt-2.5 block break-words text-left text-sm font-semibold hover:text-primary-600" @click="preview = item">{{ item.title }}</button>
              <p class="mt-1.5 line-clamp-2 whitespace-pre-wrap break-words text-sm leading-6 text-gray-500">{{ item.content }}</p>
            </div>
            <div class="flex shrink-0 items-center gap-3 text-xs sm:pt-1">
              <button type="button" class="font-medium text-primary-600" @click="preview = item">{{ tr('预览', 'Preview') }}</button>
              <button v-if="!item.source_announcement_id" type="button" class="text-gray-600 dark:text-dark-300" @click="openEditor(item)">{{ tr('编辑', 'Edit') }}</button>
              <button v-if="item.status === 'active'" type="button" class="text-gray-600 dark:text-dark-300" :disabled="busyID === item.id" @click="setStatus(item, 'archived')">{{ tr('下线', 'Unpublish') }}</button>
              <button v-else-if="!item.source_announcement_id" type="button" class="font-medium text-primary-600" :disabled="busyID === item.id" @click="setStatus(item, 'active')">{{ tr('发布', 'Publish') }}</button>
              <button v-else type="button" class="text-primary-600" @click="tab = 'main'">{{ tr('查看审核', 'View review') }}</button>
            </div>
          </article>
        </div>
        <div v-else class="px-6 py-14 text-center"><Icon name="bell" size="lg" class="mx-auto text-gray-300 dark:text-dark-500" /><h3 class="mt-4 text-sm font-medium">{{ tr('从一条属于你的公告开始', 'Start with your own announcement') }}</h3><p class="mt-2 text-sm text-gray-500">{{ tr('主站历史公告不会直接展示给你的客户。', 'Main-site announcements are never shown directly to your customers.') }}</p><button type="button" class="btn btn-secondary mt-5" @click="openEditor()">{{ tr('发新公告', 'New announcement') }}</button></div>
      </template>
      <template v-else>
        <p v-if="!profile.sync_main_announcements" class="px-6 py-12 text-center text-sm text-gray-500">{{ tr('主站公告同步已关闭。开启并保存后可查看待审核内容。', 'Main-site sync is off. Enable it and save to receive announcements for review.') }}</p>
        <div v-else-if="reviews.length" class="divide-y divide-gray-100 dark:divide-dark-700">
          <article v-for="item in reviews" :key="item.id" class="flex flex-col gap-4 p-5 sm:flex-row sm:items-center">
            <div class="min-w-0 flex-1"><div class="flex flex-wrap items-center gap-2"><span class="rounded px-2 py-0.5 text-xs font-medium" :class="reviewClass(item.review_status)">{{ reviewLabel(item.review_status) }}</span><span class="text-xs text-gray-400">{{ date(item.updated_at) }}</span></div><h3 class="mt-2.5 break-words text-sm font-semibold">{{ item.title }}</h3><p class="mt-1.5 line-clamp-2 whitespace-pre-wrap break-words text-sm leading-6 text-gray-500">{{ item.content }}</p></div>
            <button type="button" class="btn btn-secondary btn-sm shrink-0 self-start sm:self-auto" @click="reviewItem = item">{{ item.review_status === 'pending' ? tr('查看并审核', 'Review announcement') : tr('查看审核', 'View review') }}</button>
          </article>
        </div>
        <p v-else class="px-6 py-14 text-center text-sm text-gray-500">{{ tr('暂时没有可同步的主站公告。新公告发布后会出现在这里。', 'There are no main-site announcements to sync. New published announcements will appear here.') }}</p>
      </template>
    </section>

    <BaseDialog :show="editorOpen" :title="editingID ? tr('编辑公告', 'Edit announcement') : tr('发新公告', 'New announcement')" width="wide" @close="editorOpen = false">
      <form id="reseller-announcement-editor" class="space-y-5" @submit.prevent="saveAnnouncement('active')">
        <label class="block text-sm font-medium">{{ tr('标题', 'Title') }}<input v-model="draft.title" class="input mt-2" :maxlength="defaults.communications.announcement_title_max_length" required /></label>
        <label class="block text-sm font-medium">{{ tr('正文', 'Content') }}<textarea v-model="draft.content" class="input mt-2 min-h-56 resize-y font-mono text-sm leading-6" rows="10" required :placeholder="tr('支持 Markdown 排版', 'Markdown supported')" /></label>
        <label class="block text-sm font-medium">{{ tr('提醒方式', 'Notification') }}<select v-model="draft.notify_mode" class="input mt-2"><option value="silent">{{ tr('仅在公告列表显示', 'Announcement list only') }}</option><option value="popup">{{ tr('弹窗提醒客户', 'Show a popup to customers') }}</option></select></label>
        <p v-if="!profile.announcements_enabled" class="text-xs leading-5 text-gray-500">{{ tr('客户展示开关目前关闭，发布后仍需开启展示。', 'Customer display is off. Enable it to make published content visible.') }}</p>
        <p v-if="dialogError" role="alert" class="text-sm text-red-600">{{ dialogError }}</p>
      </form>
      <template #footer><button type="button" class="btn btn-secondary" :disabled="saving" @click="saveAnnouncement('draft')">{{ tr('保存草稿', 'Save draft') }}</button><button type="submit" form="reseller-announcement-editor" class="btn btn-primary" :disabled="saving">{{ saving ? tr('保存中…', 'Saving…') : tr('发布公告', 'Publish announcement') }}</button></template>
    </BaseDialog>

    <BaseDialog :show="reviewItem !== null" :title="tr('主站公告审核', 'Review main-site announcement')" width="wide" @close="reviewItem = null">
      <div v-if="reviewItem" class="space-y-5">
        <div><p class="text-xs text-gray-500">{{ tr('主站更新于', 'Updated on main site') }} {{ date(reviewItem.updated_at) }}</p><h3 class="mt-2 text-xl font-semibold leading-8">{{ reviewItem.title }}</h3></div>
        <div class="markdown-body prose prose-sm max-w-none border-y border-gray-100 py-5 dark:prose-invert dark:border-dark-700" v-html="reviewContent" />
        <p class="text-xs leading-5 text-gray-500">{{ tr('只发布你当前审核的版本。主站后续修改正文，需要你再次审核。', 'Only the version you review is published. Later main-site changes require your approval again.') }}</p>
        <p v-if="!profile.announcements_enabled" class="text-xs leading-5 text-amber-700 dark:text-amber-300">{{ tr('客户公告展示已关闭；通过后，开启展示才会对客户可见。', 'Customer display is off. Approved content becomes visible after you enable it.') }}</p>
        <p v-if="reviewError" role="alert" class="text-sm text-red-600">{{ reviewError }}</p>
      </div>
      <template #footer><button type="button" class="btn btn-secondary" :disabled="reviewing" @click="review('rejected')">{{ tr('拒绝', 'Reject') }}</button><button type="button" class="btn btn-primary" :disabled="reviewing" @click="review('approved')">{{ reviewing ? tr('处理中…', 'Saving…') : tr('审核通过并发布', 'Approve and publish') }}</button></template>
    </BaseDialog>
    <AnnouncementPopup :announcement="preview" preview @close="preview = null" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import BaseDialog from '@/components/common/BaseDialog.vue'
import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { resellerAPI, type ResellerProfile, type ResellerAnnouncement, type ResellerAnnouncementReview, type ResellerAnnouncementInput } from '@/api/reseller'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import defaults from '../../../../../reseller-defaults.json'
import '@/styles/announcement-markdown.css'

const props = defineProps<{ profile: ResellerProfile }>()
const emit = defineEmits<{ updated: [profile: ResellerProfile] }>()
const { locale } = useI18n()
const tr = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const app = useAppStore()
const tab = ref<'own' | 'main'>('own')
const enabled = ref(props.profile.announcements_enabled)
const syncMain = ref(props.profile.sync_main_announcements)
const settingsSaving = ref(false)
const settingsChanged = computed(() => enabled.value !== props.profile.announcements_enabled || syncMain.value !== props.profile.sync_main_announcements)
watch(() => props.profile, profile => { enabled.value = profile.announcements_enabled; syncMain.value = profile.sync_main_announcements })
const loading = ref(false)
const error = ref('')
const items = ref<ResellerAnnouncement[]>([])
const reviews = ref<ResellerAnnouncementReview[]>([])
const pendingCount = computed(() => reviews.value.filter(item => item.review_status === 'pending').length)
const preview = ref<ResellerAnnouncement | null>(null)
const busyID = ref<number | null>(null)
const editorOpen = ref(false)
const editingID = ref<number | null>(null)
const draft = reactive<ResellerAnnouncementInput>({ title: '', content: '', status: 'draft', notify_mode: 'silent' })
const saving = ref(false)
const dialogError = ref('')
const reviewItem = ref<ResellerAnnouncementReview | null>(null)
const reviewing = ref(false)
const reviewError = ref('')
watch(reviewItem, () => { reviewError.value = '' })
const reviewContent = computed(() => DOMPurify.sanitize(marked.parse(reviewItem.value?.content ?? '', { breaks: true, gfm: true }) as string))
const date = (value: string) => formatDateTime(value)
const tabClass = (value: typeof tab.value) => ['rounded-md px-3 py-2 transition-colors', tab.value === value ? 'bg-white font-medium text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-800 dark:hover:text-dark-200']
const statusLabel = (status: ResellerAnnouncement['status']) => ({ draft: tr('草稿', 'Draft'), active: tr('已发布', 'Published'), archived: tr('已下线', 'Unpublished') }[status])
const statusClass = (status: string) => status === 'active' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
const reviewLabel = (status: ResellerAnnouncementReview['review_status']) => ({ pending: tr('待审核', 'Pending review'), approved: tr('已通过', 'Approved'), rejected: tr('已拒绝', 'Rejected') }[status])
const reviewClass = (status: string) => status === 'pending' ? 'bg-amber-50 text-amber-800 dark:bg-amber-950 dark:text-amber-300' : statusClass(status === 'approved' ? 'active' : 'archived')

async function load() {
  loading.value = true
  error.value = ''
  try { [items.value, reviews.value] = await Promise.all([resellerAPI.announcements(), resellerAPI.mainAnnouncements()]) }
  catch (e) { error.value = extractApiErrorMessage(e, tr('读取公告失败', 'Could not load announcements')) }
  finally { loading.value = false }
}

async function saveSettings() {
  settingsSaving.value = true
  try {
    const profile = await resellerAPI.saveCommunicationSettings({ contact_enabled: props.profile.contact_enabled, contact_info: props.profile.contact_info, announcements_enabled: enabled.value, sync_main_announcements: syncMain.value })
    emit('updated', profile)
    await load()
    app.showSuccess(tr('公告设置已保存', 'Announcement settings saved'))
  } catch (e) { app.showError(extractApiErrorMessage(e, tr('保存设置失败', 'Could not save settings'))) }
  finally { settingsSaving.value = false }
}

function openEditor(item?: ResellerAnnouncement) {
  editingID.value = item?.id ?? null
  draft.title = item?.title ?? ''
  draft.content = item?.content ?? ''
  draft.notify_mode = item?.notify_mode ?? 'silent'
  dialogError.value = ''
  editorOpen.value = true
}

async function saveAnnouncement(status: ResellerAnnouncementInput['status']) {
  saving.value = true
  dialogError.value = ''
  try {
    const input = { title: draft.title, content: draft.content, notify_mode: draft.notify_mode, status }
    if (editingID.value) await resellerAPI.updateAnnouncement(editingID.value, input)
    else await resellerAPI.createAnnouncement(input)
    editorOpen.value = false
    tab.value = 'own'
    await load()
    app.showSuccess(status === 'active' ? tr('公告已发布', 'Announcement published') : tr('草稿已保存', 'Draft saved'))
  } catch (e) { dialogError.value = extractApiErrorMessage(e, tr('保存公告失败', 'Could not save announcement')) }
  finally { saving.value = false }
}

async function setStatus(item: ResellerAnnouncement, status: ResellerAnnouncementInput['status']) {
  busyID.value = item.id
  try { await resellerAPI.setAnnouncementStatus(item.id, status); await load() }
  catch (e) { app.showError(extractApiErrorMessage(e, tr('更新公告失败', 'Could not update announcement'))) }
  finally { busyID.value = null }
}

async function review(status: 'approved' | 'rejected') {
  if (!reviewItem.value) return
  reviewing.value = true
  reviewError.value = ''
  try {
    await resellerAPI.reviewMainAnnouncement(reviewItem.value.id, reviewItem.value.updated_at, status)
    reviewItem.value = null
    await load()
    app.showSuccess(status === 'approved' ? tr('审核通过，已同步到本站公告', 'Approved and added to your site announcements') : tr('已拒绝此公告', 'Announcement rejected'))
  } catch (e) { reviewError.value = extractApiErrorMessage(e, tr('审核失败，请刷新后重试', 'Review failed. Refresh and try again.')) }
  finally { reviewing.value = false }
}

onMounted(load)
</script>
