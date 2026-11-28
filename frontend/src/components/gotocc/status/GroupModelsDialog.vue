<template>
  <BaseDialog :show="group !== null" :title="group ? t('serviceStatus.modelsDialog.title', { group: group.name }) : ''" width="wide" close-on-click-outside @close="emit('close')">
    <div class="gm-body" data-test="status-models-dialog">
      <div v-if="loading" class="gm-grid">
        <span v-for="index in 9" :key="index" class="gc-skeleton h-10"></span>
      </div>
      <p v-else-if="failed" class="gm-empty">{{ t('serviceStatus.modelsDialog.loadFailed') }}</p>
      <p v-else-if="!models.length" class="gm-empty">{{ t('serviceStatus.modelsDialog.empty') }}</p>
      <template v-else>
        <p class="gm-count">{{ t('serviceStatus.modelsDialog.count', { n: models.length }) }}</p>
        <section v-for="section in sections" :key="section.platform" class="gm-section">
          <h4 v-if="sections.length > 1" class="gm-platform"><PlatformIcon :platform="section.platform" size="sm" />{{ platformLabel(section.platform) }}</h4>
          <div class="gm-grid">
            <span v-for="model in section.models" :key="model.name" class="gm-model">
              <ModelIcon :model="model.name" size="18px" />
              <span class="gm-name">{{ model.name }}</span>
            </span>
          </div>
        </section>
      </template>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { serviceStatusAPI, type GroupModel, type GroupStatus } from '@/api/serviceStatus'
import { platformLabel } from '@/utils/platformColors'

// 分组可用模型：数据和可用渠道同一份目录，组合分组按平台分节。
const props = defineProps<{ group: GroupStatus | null }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const models = ref<GroupModel[]>([])
const loading = ref(false)
const failed = ref(false)

const sections = computed(() => {
  const byPlatform = new Map<string, GroupModel[]>()
  for (const model of models.value) byPlatform.set(model.platform, [...(byPlatform.get(model.platform) ?? []), model])
  return [...byPlatform].map(([platform, items]) => ({ platform: platform as GroupModel['platform'], models: items }))
})

watch(() => props.group?.id, async id => {
  models.value = []
  failed.value = false
  if (!id) return
  loading.value = true
  try {
    const result = await serviceStatusAPI.getGroupModels(id)
    if (props.group?.id === id) models.value = result
  } catch {
    if (props.group?.id === id) failed.value = true
  } finally {
    if (props.group?.id === id) loading.value = false
  }
})
</script>

<style scoped>
.gm-body { display: flex; flex-direction: column; gap: 14px; }
.gm-count { font-size: 12px; color: var(--surface-muted); }
.gm-section { display: flex; flex-direction: column; gap: 8px; }
.gm-platform { display: flex; align-items: center; gap: 6px; font-size: 13px; font-weight: 650; color: var(--surface-text); }
.gm-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(13rem, 1fr)); gap: 8px; }
.gm-model {
  display: flex; min-width: 0; align-items: center; gap: 8px; border-radius: 10px; padding: 9px 11px;
  background: var(--surface-soft);
  transition: background-color var(--motion-hover) var(--motion-ease), transform var(--motion-hover) var(--motion-ease), box-shadow var(--motion-hover) var(--motion-ease);
}
.gm-model:hover { background: var(--surface-hover); transform: translateY(var(--hover-lift)); box-shadow: inset 0 0 0 1px var(--card-hover-edge); }
.gm-name { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; color: var(--surface-text); }
.gm-empty { padding: 28px 0; text-align: center; font-size: 13px; color: var(--surface-muted); }
</style>
