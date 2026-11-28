<template>
  <dl class="key-stats">
    <div v-for="stat in stats" :key="stat.key" class="key-stat" :data-stat="stat.key">
      <dt class="key-stat-label">{{ stat.label }}</dt>
      <dd class="key-stat-value gc-num">
        <span v-if="stat.loading" class="gc-skeleton inline-block h-5 w-20 align-middle"></span>
        <span v-else :key="stat.value" class="gc-reveal inline-block">{{ stat.value }}</span>
      </dd>
    </div>
  </dl>
</template>

<script setup lang="ts">
export interface KeyStat {
  key: string
  label: string
  value: string
  loading: boolean
}

defineProps<{ stats: KeyStat[] }>()
</script>

<style scoped>
.key-stats { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; }
.key-stat {
  min-width: 0; padding: 10px 14px;
  border: 1px solid var(--surface-edge); border-radius: var(--surface-radius-card);
  background: var(--surface-soft);
}
.key-stat-label { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; color: var(--surface-muted); }
.key-stat-value { margin-top: 2px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 18px; font-weight: 650; letter-spacing: -.01em; color: var(--surface-text); }
@media (max-width: 767px) {
  .key-stats { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; }
  .key-stat { padding: 8px 12px; }
  .key-stat-value { font-size: 16px; }
}
</style>
