<template>
  <section class="card overflow-hidden">
    <div class="border-b border-gray-100 px-6 py-5 dark:border-dark-700">
      <h2 class="text-base font-semibold">{{ tr('客户联系方式', 'Customer support contact') }}</h2>
      <p class="mt-1.5 text-sm text-gray-500">{{ tr('显示在所属客户的个人下拉菜单中，由你独立维护。', 'Manage the support contact shown in your customers’ account menu.') }}</p>
    </div>
    <form class="grid gap-8 p-6 lg:grid-cols-[minmax(0,1fr)_320px]" @submit.prevent="save">
      <div class="space-y-5">
        <div class="flex items-start justify-between gap-6">
          <div>
            <label for="reseller-contact-enabled" class="text-sm font-medium">{{ tr('展示联系方式', 'Show contact information') }}</label>
            <p class="mt-1 text-xs leading-5 text-gray-500">{{ tr('默认关闭。开启后只展示你填写的内容。', 'Off by default. When enabled, customers see only the contact you provide.') }}</p>
          </div>
          <Toggle id="reseller-contact-enabled" v-model="enabled" :aria-label="tr('展示联系方式', 'Show contact information')" />
        </div>
        <label class="block text-sm font-medium" for="reseller-contact-info">{{ tr('联系方式', 'Contact information') }}</label>
        <textarea id="reseller-contact-info" v-model="contact" rows="4" class="input resize-y" :required="enabled" :placeholder="tr('填写你的微信、邮箱或其他联系渠道', 'Your email, support handle or another contact channel')" />
        <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
        <button class="btn btn-primary" type="submit" :disabled="saving || !changed">{{ saving ? tr('保存中…', 'Saving…') : tr('保存联系方式', 'Save contact') }}</button>
      </div>
      <aside class="rounded-xl border border-gray-200 bg-gray-50/70 p-5 dark:border-dark-600 dark:bg-dark-800">
        <p class="text-xs font-medium tracking-wide text-gray-500">{{ tr('客户菜单预览', 'Customer menu preview') }}</p>
        <div v-if="enabled && contact.trim()" class="mt-5 flex items-start gap-3 rounded-lg bg-white p-4 dark:bg-dark-900">
          <Icon name="chat" size="md" class="shrink-0 text-gray-400" />
          <div class="min-w-0"><p class="text-xs text-gray-500">{{ tr('联系客服', 'Contact support') }}</p><p class="mt-1 whitespace-pre-wrap break-words text-sm font-medium">{{ contact.trim() }}</p></div>
        </div>
        <p v-else class="mt-5 py-5 text-sm leading-6 text-gray-500">{{ tr('客户菜单将隐藏联系方式。', 'Contact information will be hidden from the customer menu.') }}</p>
        <p class="mt-4 text-xs text-gray-400">{{ tr('保存后生效', 'Changes take effect after saving') }}</p>
      </aside>
    </form>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { resellerAPI, type ResellerProfile } from '@/api/reseller'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{ profile: ResellerProfile }>()
const emit = defineEmits<{ updated: [profile: ResellerProfile] }>()
const { locale } = useI18n()
const tr = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const app = useAppStore()
const enabled = ref(props.profile.contact_enabled)
const contact = ref(props.profile.contact_info)
const saving = ref(false)
const error = ref('')
const changed = computed(() => enabled.value !== props.profile.contact_enabled || contact.value !== props.profile.contact_info)
watch(() => props.profile, profile => { enabled.value = profile.contact_enabled; contact.value = profile.contact_info })

async function save() {
  saving.value = true
  error.value = ''
  try {
    const profile = await resellerAPI.saveCommunicationSettings({
      contact_enabled: enabled.value,
      contact_info: contact.value,
      announcements_enabled: props.profile.announcements_enabled,
      sync_main_announcements: props.profile.sync_main_announcements
    })
    emit('updated', profile)
    app.showSuccess(tr('联系方式已保存', 'Contact saved'))
  } catch (e) { error.value = extractApiErrorMessage(e, tr('保存失败', 'Could not save contact')) }
  finally { saving.value = false }
}
</script>
