<template>
  <AppLayout appearance="user" workspace>
    <TablePageLayout>
      <template #actions>
        <div class="keys-overview">
          <div class="keys-overview-top">
            <ApiEndpointBar :settings="publicSettings" class="min-w-0 flex-1" />
            <div class="keys-actions">
              <ScopeDropdown v-if="teamFeatureEnabled" v-model="scope" data-tour="keys-scope-switch" @change="onScopeChange" />
              <button
                type="button"
                class="btn btn-secondary px-2 md:px-3"
                :disabled="clientConfigKeys.length === 0"
                :title="t('keys.oneClickAccess')"
                data-test="one-click-access-global"
                @click="openOneClickAccess"
              >
                <Icon name="terminal" size="md" class="md:mr-1.5" />
                <span class="hidden md:inline">{{ t('keys.oneClickAccess') }}</span>
              </button>
              <button
                type="button"
                class="btn btn-secondary"
                :disabled="loading"
                :title="t('common.refresh')"
                @click="loadApiKeys"
              >
                <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
              </button>
              <button v-support-readonly type="button" class="btn btn-secondary" data-testid="create-aggregate-key" @click="startAggregateCreate">
                {{ t('keys.createAggregate') }}
              </button>
              <button v-support-readonly type="button" class="btn btn-primary" data-tour="keys-create-btn" @click="showCreateModal = true">
                <Icon name="plus" size="md" />
                {{ t('keys.createKey') }}
              </button>
            </div>
          </div>
          <KeyStatsBar :stats="overviewStats" />
        </div>
      </template>

      <template #filters>
        <div class="keys-filters">
          <div class="keys-tabs-scroll">
            <div v-segmented class="segmented inline-flex" role="tablist" :aria-label="t('keyConsole.list.platformTabs')">
              <button
                v-for="tab in platformTabs"
                :key="tab.id"
                type="button"
                role="tab"
                :aria-selected="tab.id === platformTab"
                class="segmented-item keys-tab"
                :class="tab.id === platformTab && 'segmented-item-active'"
                :data-tab="tab.id"
                @click="selectPlatformTab(tab.id)"
              >
                <PlatformIcon v-if="tab.platform" :platform="tab.platform" size="xs" />
                <span>{{ tab.label }}</span>
                <span class="keys-tab-count gc-num">{{ tab.count }}</span>
              </button>
            </div>
          </div>
          <div class="filter-bar">
            <div class="filter-field filter-field--search">
              <SearchInput v-model="filterSearch" :placeholder="t('keys.searchPlaceholder')" @search="onFilterChange" />
            </div>
            <div class="filter-field">
              <Select :model-value="filterStatus" :options="statusFilterOptions" @update:model-value="onStatusFilterChange" />
            </div>
            <div v-if="selectedIds.length" class="flex flex-wrap items-center gap-3 text-sm">
              <span class="surface-muted">{{ t('keys.bulkEdit.selectedCount', { count: selectedIds.length }) }}</span>
              <button class="btn btn-primary btn-sm" :disabled="loading" data-test="bulk-edit-keys" @click="showBulkEditModal = true">
                {{ t('keys.bulkEdit.title') }}
              </button>
              <button class="btn btn-secondary btn-sm" @click="selectedIds = []">
                {{ t('keys.bulkEdit.clearSelection') }}
              </button>
            </div>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          class="workspace-table-viewport"
          :columns="columns"
          :data="pageKeys"
          :loading="loading"
          selectable
          row-key="id"
          :selected-keys="selectedIds"
          :selection-label="(key: ApiKey) => t('keys.bulkEdit.selectKey', { name: key.name })"
          clickable-rows
          :active-row-key="detailKeyId"
          :server-side-sort="true"
          default-sort-key="name"
          default-sort-order="asc"
          @update:selected-keys="handleSelectionChange"
          @row-click="openDetail"
          @sort="handleSort"
        >
          <template #cell-name="{ row }">
            <div class="keys-name">
              <span class="keys-status-dot" :data-status="row.status" :title="t('keys.status.' + row.status)"></span>
              <span class="font-medium text-gray-900 dark:text-white">{{ row.name }}</span>
              <code class="keys-tail">{{ keyTail(row.key) }}</code>
              <Icon
                v-if="row.ip_whitelist?.length > 0 || row.ip_blacklist?.length > 0"
                name="shield"
                size="sm"
                class="text-blue-500"
                :title="t('keys.ipRestrictionEnabled')"
              />
            </div>
          </template>

          <template #cell-group="{ row }">
            <span v-if="isAutoRouting(row)" data-testid="api-key-auto-routing-badge" class="inline-flex items-center rounded bg-violet-100 px-2 py-1 text-xs font-medium text-violet-700 dark:bg-violet-900/30 dark:text-violet-300">
              {{ t('keys.routingMode.auto') }}
            </span>
            <GroupBadge
              v-else-if="row.group"
              :name="row.group.name"
              :show-rate="row.group.platform !== 'composite'"
              :platform="row.group.platform"
              :subscription-type="row.group.subscription_type"
              :rate-multiplier="row.group.rate_multiplier"
              :user-rate-multiplier="userGroupRates[row.group.id]"
              :peak-rate-enabled="row.group.peak_rate_enabled"
              :peak-start="row.group.peak_start"
              :peak-end="row.group.peak_end"
              :peak-rate-multiplier="row.group.peak_rate_multiplier"
            />
            <span v-else class="text-sm surface-muted">{{
              row.group_binding_mode === 'all' ? t('keys.aggregateAll') : row.group_binding_mode === 'custom' ? t('keys.aggregateCustomCount', { count: row.aggregate_group_ids?.length || 0 }) : t('keys.noGroup')
            }}</span>
          </template>

          <template #cell-last_used_at="{ value }">
            <span class="text-sm surface-muted gc-num">{{ value ? formatDateTime(value) : t('keyConsole.detail.neverUsed') }}</span>
          </template>

          <template #cell-today="{ row }">
            <div class="keys-today">
              <span v-if="usageLoading && !usageStats[row.id]" data-test="usage-pending" class="gc-skeleton inline-block h-3.5 w-16"></span>
              <span v-else class="font-medium text-gray-900 dark:text-white gc-num">{{ withCreditUnit((usageStats[row.id]?.today_actual_cost ?? 0).toFixed(4)) }}</span>
              <Icon name="chevronRight" size="sm" class="keys-row-chevron" />
            </div>
          </template>

          <template #empty>
            <EmptyState v-support-readonly
              :title="apiKeys.length ? t('keyConsole.list.noMatch') : t('keys.noKeysYet')"
              :description="apiKeys.length ? t('keyConsole.list.noMatchHint') : t('keys.createFirstKey')"
              :action-text="t('keys.createKey')"
              @action="showCreateModal = true"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          :page="page"
          :total="filteredKeys.length"
          :page-size="pageSize"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <!-- Create/Edit Modal; a successful create turns into the success page in place. -->
    <BaseDialog
      :show="showCreateModal || showEditModal"
      :title="createdKey ? t('keyConsole.created.title') : showEditModal ? t('keys.editKey') : t('keys.createKey')"
      width="normal"
      @close="!submitting && closeModals()"
    >
      <div class="gc-collapse" :data-open="!createdKey" :inert="!!createdKey || undefined">
        <div>
          <form v-support-readonly id="key-form" @submit.prevent="handleSubmit" class="space-y-5">
            <div>
              <label class="input-label">{{ t('keys.nameLabel') }}</label>
              <input
                v-model="formData.name"
                type="text"
                required
                class="input"
                :placeholder="t('keys.namePlaceholder')"
                data-tour="key-form-name"
              />
            </div>

            <fieldset v-if="!showEditModal && formData.group_binding_mode === 'single'" data-tour="key-form-provider">
              <legend class="input-label">{{ t('keys.providerLabel') }}</legend>
              <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
                <label
                  v-for="provider in createProviderOptions"
                  :key="provider.value"
                  class="relative min-w-0"
                  :class="provider.count === 0 ? 'cursor-not-allowed' : 'cursor-pointer'"
                >
                  <input
                    type="radio"
                    name="key-provider"
                    :value="provider.value"
                    :checked="createProvider === provider.value"
                    :disabled="provider.count === 0"
                    class="peer sr-only"
                    @change="selectCreateProvider(provider.value)"
                  />
                  <span
                    class="flex h-full flex-col items-center gap-2 rounded-lg border border-gray-200 surface-inset px-2 py-3 text-center transition-colors peer-checked:border-primary-500 peer-checked:bg-primary-50/60 peer-checked:ring-1 peer-checked:ring-primary-500 peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-primary-500 peer-disabled:opacity-40 dark:border-dark-600 dark:peer-checked:bg-primary-500/10"
                    :class="provider.count > 0 && 'hover:border-primary-300 dark:hover:border-primary-700'"
                  >
                    <span class="flex h-8 items-center justify-center gap-1.5" aria-hidden="true">
                      <span
                        v-for="platform in KEY_GROUP_PROVIDER_ICONS[provider.value]"
                        :key="platform"
                        class="flex h-8 w-8 items-center justify-center rounded-lg"
                        :class="platformBadgeLightClass(platform)"
                      >
                        <PlatformIcon :platform="platform" size="lg" />
                      </span>
                    </span>
                    <span class="text-sm font-semibold text-gray-800 dark:text-gray-100">{{ provider.label }}</span>
                  </span>
                  <span
                    v-if="createProvider === provider.value"
                    class="absolute right-1.5 top-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-primary-500 text-white"
                    aria-hidden="true"
                  >
                    <Icon name="check" size="xs" :stroke-width="3" />
                  </span>
                </label>
              </div>
              <p class="mt-2 text-xs leading-5 surface-muted" aria-live="polite">
                {{ groups.length === 0 ? t('common.noGroupsAvailable') : t(`keys.providerHints.${createProvider}`) }}
              </p>
            </fieldset>

            <div v-if="formData.group_binding_mode === 'single'">
              <label class="input-label">{{ t('keys.routingMode.label') }}</label>
              <div
                class="grid grid-cols-2 gap-1 rounded-lg bg-gray-100 p-1 dark:bg-dark-700"
                role="radiogroup"
                :aria-label="t('keys.routingMode.label')"
              >
                <button
                  type="button"
                  role="radio"
                  data-test="key-routing-fixed"
                  :aria-checked="formData.routing_mode === 'fixed'"
                  :class="[
                    'rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    formData.routing_mode === 'fixed'
                      ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                      : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'
                  ]"
                  @click="setFormRoutingMode('fixed')"
                >
                  {{ t('keys.routingMode.fixed') }}
                </button>
                <button
                  type="button"
                  role="radio"
                  data-test="key-routing-auto"
                  :aria-checked="formData.routing_mode === 'auto'"
                  :class="[
                    'rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    formData.routing_mode === 'auto'
                      ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                      : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'
                  ]"
                  @click="setFormRoutingMode('auto')"
                >
                  {{ t('keys.routingMode.auto') }}
                </button>
              </div>
              <p class="input-hint">
                {{ formData.routing_mode === 'auto' ? t('keys.routingMode.autoHint') : t('keys.routingMode.fixedHint') }}
              </p>
            </div>

            <RoutingPriorityPanel
              v-if="(showCreateModal || showEditModal) && formData.group_binding_mode === 'single' && formData.routing_mode === 'auto'"
              :key="showEditModal ? selectedKey?.id : 'create'"
              ref="routingPriorityPanel"
              :key-id="showEditModal ? selectedKey?.id : undefined"
              :disabled="submitting"
              :scope="scope"
            />

            <div v-if="formData.group_binding_mode !== 'single'" class="space-y-3" data-testid="aggregate-key-fields">
              <div>
                <label class="input-label" for="key-aggregate-mode">{{ t('keys.aggregateScope') }}</label>
                <Select id="key-aggregate-mode" v-model="formData.group_binding_mode" :options="aggregateModeOptions" :searchable="false" />
              </div>
              <p class="text-sm surface-muted">{{ t('keys.aggregateHint') }}</p>
              <CheckboxDropdown v-if="formData.group_binding_mode === 'custom'" v-model="formData.aggregate_group_ids" :options="aggregateGroupOptions" :label="t('keys.selectAggregateGroups')" searchable :search-placeholder="t('keys.searchGroup')">
                <template #selected="{ options }">
                  <span class="flex min-w-0 flex-wrap gap-1.5">
                    <template v-for="option in options" :key="option.value">
                      <GroupBadge
                        v-if="option.group"
                        data-testid="aggregate-selected-group"
                        class="min-w-0 max-w-full"
                        :name="option.group.name"
                        :platform="option.group.platform"
                        :subscription-type="option.group.subscription_type"
                        :rate-multiplier="option.group.rate_multiplier"
                        :user-rate-multiplier="userGroupRates[option.value]"
                        :truncate="false"
                      />
                      <span v-else class="surface-muted">{{ option.label }}</span>
                    </template>
                  </span>
                </template>
                <template #option="{ option }">
                  <GroupBadge
                    v-if="option.group"
                    data-testid="aggregate-group-option"
                    class="min-w-0 max-w-full"
                    :name="option.group.name"
                    :platform="option.group.platform"
                    :subscription-type="option.group.subscription_type"
                    :rate-multiplier="option.group.rate_multiplier"
                    :user-rate-multiplier="userGroupRates[option.value]"
                    :truncate="false"
                  />
                  <span v-else class="surface-muted">{{ option.label }}</span>
                </template>
              </CheckboxDropdown>
              <p v-if="formData.group_binding_mode === 'custom' && aggregateGroupOptions.length === 0" class="text-sm surface-muted">{{ t('common.noGroupsAvailable') }}</p>
            </div>

            <div v-else-if="formData.routing_mode === 'fixed'">
              <label class="input-label" for="key-form-group">{{ t('keys.groupLabel') }}</label>
              <Select
                :key="showEditModal ? 'edit' : createProvider"
                id="key-form-group"
                :aria-label="t('keys.groupLabel')"
                v-model="formData.group_id"
                :options="formGroupOptions"
                :placeholder="t('keys.selectGroup')"
                :empty-text="t('common.noGroupsAvailable')"
                :searchable="true"
                :search-placeholder="t('keys.searchGroup')"
                data-tour="key-form-group"
              >
                <template #selected="{ option }">
                  <GroupBadge
                    v-if="option"
                    :name="(option as unknown as GroupOption).label"
                    :show-rate="false"
                    :platform="(option as unknown as GroupOption).platform"
                    :subscription-type="(option as unknown as GroupOption).subscriptionType"
                    :rate-multiplier="(option as unknown as GroupOption).rate"
                    :user-rate-multiplier="(option as unknown as GroupOption).userRate"
                    :peak-rate-enabled="(option as unknown as GroupOption).peakRateEnabled"
                    :peak-start="(option as unknown as GroupOption).peakStart"
                    :peak-end="(option as unknown as GroupOption).peakEnd"
                    :peak-rate-multiplier="(option as unknown as GroupOption).peakRateMultiplier"
                  />
                  <span v-else class="surface-muted">{{ t('keys.selectGroup') }}</span>
                </template>
                <template #option="{ option, selected }">
                  <GroupOptionItem
                    :name="(option as unknown as GroupOption).label"
                    :platform="(option as unknown as GroupOption).platform"
                    :subscription-type="(option as unknown as GroupOption).subscriptionType"
                    :rate-multiplier="(option as unknown as GroupOption).rate"
                    :user-rate-multiplier="(option as unknown as GroupOption).userRate"
                    :peak-rate-enabled="(option as unknown as GroupOption).peakRateEnabled"
                    :peak-start="(option as unknown as GroupOption).peakStart"
                    :peak-end="(option as unknown as GroupOption).peakEnd"
                    :peak-rate-multiplier="(option as unknown as GroupOption).peakRateMultiplier"
                    :description="(option as unknown as GroupOption).description"
                    :selected="selected"
                  >
                    <template #rates><GroupRateSummary :group="quoteFor((option as unknown as GroupOption).value)" :loading="quotesLoading" :failed="quotesFailed" compact /></template>
                  </GroupOptionItem>
                </template>
              </Select>
              <GroupRateSummary v-if="formData.group_id" class="mt-3 rounded-lg surface-inset p-3" :group="quoteFor(formData.group_id)" :loading="quotesLoading" :failed="quotesFailed" @retry="refreshQuotes" />
            </div>

            <NativeAccessFields v-if="formData.group_binding_mode === 'single' && groups.find(group => group.id === formData.group_id)?.platform === 'composite'" ref="nativeAccessFields" v-model:conversion-enabled="formData.conversion_enabled" v-model:client="formData.default_client" v-model:endpoint="formData.default_endpoint" :group-id="formData.group_id" :base-url="endpointBaseUrl" />

            <!-- Custom Key Section (only for create) -->
            <div v-if="!showEditModal" class="space-y-3">
              <div class="flex items-center justify-between">
                <label class="input-label mb-0">{{ t('keys.customKeyLabel') }}</label>
                <Toggle
                  v-model="formData.use_custom_key"
                />
              </div>
              <div v-if="formData.use_custom_key">
                <input
                  v-model="formData.custom_key"
                  type="text"
                  class="input font-mono"
                  :placeholder="t('keys.customKeyPlaceholder')"
                  :class="{ 'border-red-500 dark:border-red-500': customKeyError }"
                />
                <p v-if="customKeyError" class="mt-1 text-sm text-red-500">{{ customKeyError }}</p>
                <p v-else class="input-hint">{{ t('keys.customKeyHint') }}</p>
              </div>
            </div>

            <div v-if="showEditModal">
              <label class="input-label">{{ t('keys.statusLabel') }}</label>
              <Select
                v-model="formData.status"
                :options="statusOptions"
                :placeholder="t('keys.selectStatus')"
              />
            </div>

            <!-- IP Restriction Section -->
            <div class="space-y-3">
              <div class="flex items-center justify-between">
                <label class="input-label mb-0">{{ t('keys.ipRestriction') }}</label>
                <Toggle
                  v-model="formData.enable_ip_restriction"
                />
              </div>

              <div v-if="formData.enable_ip_restriction" class="space-y-4 pt-2">
                <div>
                  <label class="input-label">{{ t('keys.ipWhitelist') }}</label>
                  <textarea
                    v-model="formData.ip_whitelist"
                    rows="3"
                    class="input font-mono text-sm"
                    :placeholder="t('keys.ipWhitelistPlaceholder')"
                  />
                  <p class="input-hint">{{ t('keys.ipWhitelistHint') }}</p>
                </div>

                <div>
                  <label class="input-label">{{ t('keys.ipBlacklist') }}</label>
                  <textarea
                    v-model="formData.ip_blacklist"
                    rows="3"
                    class="input font-mono text-sm"
                    :placeholder="t('keys.ipBlacklistPlaceholder')"
                  />
                  <p class="input-hint">{{ t('keys.ipBlacklistHint') }}</p>
                </div>
              </div>
            </div>

            <!-- Quota Limit Section -->
            <div class="space-y-3">
              <label class="input-label">{{ t('keys.quotaLimit') }}</label>
              <!-- Switch commented out - always show input, 0 = unlimited
              <div class="flex items-center justify-between">
                <label class="input-label mb-0">{{ t('keys.quotaLimit') }}</label>
                <Toggle
                  v-model="formData.enable_quota"
                />
              </div>
              -->

              <div class="space-y-4">
                <div>
                  <div class="relative">
                    <span class="absolute left-3 top-1/2 -translate-y-1/2 surface-muted">{{ creditUnit() }}</span>
                    <input
                      v-model.number="formData.quota"
                      type="number"
                      step="0.01"
                      min="0"
                      class="input pr-20"
                      :placeholder="t('keys.quotaAmountPlaceholder')"
                    />
                  </div>
                  <p class="input-hint">{{ t('keys.quotaAmountHint') }}</p>
                </div>

                <!-- Quota used display (only in edit mode) -->
                <div v-if="showEditModal && selectedKey && selectedKey.quota > 0">
                  <label class="input-label">{{ t('keys.quotaUsed') }}</label>
                  <div class="flex items-center gap-2">
                    <div class="flex-1 rounded-lg surface-inset px-3 py-2">
                      <span class="font-medium text-gray-900 dark:text-white">
                        {{ withCreditUnit(selectedKey.quota_used?.toFixed(4) || '0.0000') }}
                      </span>
                      <span class="mx-2 surface-muted">/</span>
                      <span class="surface-muted">
                        {{ withCreditUnit(selectedKey.quota?.toFixed(2) || '0.00') }}
                      </span>
                    </div>
                    <button v-support-readonly
                      type="button"
                      @click="confirmResetQuota"
                      class="btn btn-secondary text-sm"
                      :title="t('keys.resetQuotaUsed')"
                    >
                      {{ t('keys.reset') }}
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <!-- Rate Limit Section -->
            <div class="space-y-3">
              <div class="flex items-center justify-between">
                <label class="input-label mb-0">{{ t('keys.rateLimitSection') }}</label>
                <Toggle
                  v-model="formData.enable_rate_limit"
                />
              </div>

              <div v-if="formData.enable_rate_limit" class="space-y-4 pt-2">
                <p class="input-hint -mt-2">{{ t('keys.rateLimitHint') }}</p>
                <!-- 5-Hour Limit -->
                <div>
                  <label class="input-label">{{ t('keys.rateLimit5h') }}</label>
                  <div class="relative">
                    <span class="absolute left-3 top-1/2 -translate-y-1/2 surface-muted">{{ creditUnit() }}</span>
                    <input
                      v-model.number="formData.rate_limit_5h"
                      type="number"
                      step="0.01"
                      min="0"
                      class="input pr-20"
                      :placeholder="'0'"
                    />
                  </div>
                  <!-- Usage info (edit mode only) -->
                  <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_5h > 0" class="mt-2">
                    <div class="flex items-center gap-2">
                      <div class="flex-1 rounded-lg surface-inset px-3 py-2 text-sm">
                        <span :class="[
                          'font-medium',
                          selectedKey.usage_5h >= selectedKey.rate_limit_5h ? 'text-red-500' :
                          selectedKey.usage_5h >= selectedKey.rate_limit_5h * 0.8 ? 'text-yellow-500' :
                          'text-gray-900 dark:text-white'
                        ]">
                          {{ withCreditUnit(selectedKey.usage_5h?.toFixed(4) || '0.0000') }}
                        </span>
                        <span class="mx-2 surface-muted">/</span>
                        <span class="surface-muted">
                          {{ withCreditUnit(selectedKey.rate_limit_5h?.toFixed(2) || '0.00') }}
                        </span>
                      </div>
                    </div>
                    <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                      <div
                        :class="[
                          'console-data-mark h-full rounded-full transition-all',
                          selectedKey.usage_5h >= selectedKey.rate_limit_5h ? 'bg-red-500' :
                          selectedKey.usage_5h >= selectedKey.rate_limit_5h * 0.8 ? 'bg-yellow-500' :
                          'bg-green-500'
                        ]"
                        :style="{ width: Math.min((selectedKey.usage_5h / selectedKey.rate_limit_5h) * 100, 100) + '%' }"
                      />
                    </div>
                  </div>
                </div>

                <!-- Daily Limit -->
                <div>
                  <label class="input-label">{{ t('keys.rateLimit1d') }}</label>
                  <div class="relative">
                    <span class="absolute left-3 top-1/2 -translate-y-1/2 surface-muted">{{ creditUnit() }}</span>
                    <input
                      v-model.number="formData.rate_limit_1d"
                      type="number"
                      step="0.01"
                      min="0"
                      class="input pr-20"
                      :placeholder="'0'"
                    />
                  </div>
                  <!-- Usage info (edit mode only) -->
                  <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_1d > 0" class="mt-2">
                    <div class="flex items-center gap-2">
                      <div class="flex-1 rounded-lg surface-inset px-3 py-2 text-sm">
                        <span :class="[
                          'font-medium',
                          selectedKey.usage_1d >= selectedKey.rate_limit_1d ? 'text-red-500' :
                          selectedKey.usage_1d >= selectedKey.rate_limit_1d * 0.8 ? 'text-yellow-500' :
                          'text-gray-900 dark:text-white'
                        ]">
                          {{ withCreditUnit(selectedKey.usage_1d?.toFixed(4) || '0.0000') }}
                        </span>
                        <span class="mx-2 surface-muted">/</span>
                        <span class="surface-muted">
                          {{ withCreditUnit(selectedKey.rate_limit_1d?.toFixed(2) || '0.00') }}
                        </span>
                      </div>
                    </div>
                    <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                      <div
                        :class="[
                          'console-data-mark h-full rounded-full transition-all',
                          selectedKey.usage_1d >= selectedKey.rate_limit_1d ? 'bg-red-500' :
                          selectedKey.usage_1d >= selectedKey.rate_limit_1d * 0.8 ? 'bg-yellow-500' :
                          'bg-green-500'
                        ]"
                        :style="{ width: Math.min((selectedKey.usage_1d / selectedKey.rate_limit_1d) * 100, 100) + '%' }"
                      />
                    </div>
                  </div>
                </div>

                <!-- 7-Day Limit -->
                <div>
                  <label class="input-label">{{ t('keys.rateLimit7d') }}</label>
                  <div class="relative">
                    <span class="absolute left-3 top-1/2 -translate-y-1/2 surface-muted">{{ creditUnit() }}</span>
                    <input
                      v-model.number="formData.rate_limit_7d"
                      type="number"
                      step="0.01"
                      min="0"
                      class="input pr-20"
                      :placeholder="'0'"
                    />
                  </div>
                  <!-- Usage info (edit mode only) -->
                  <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_7d > 0" class="mt-2">
                    <div class="flex items-center gap-2">
                      <div class="flex-1 rounded-lg surface-inset px-3 py-2 text-sm">
                        <span :class="[
                          'font-medium',
                          selectedKey.usage_7d >= selectedKey.rate_limit_7d ? 'text-red-500' :
                          selectedKey.usage_7d >= selectedKey.rate_limit_7d * 0.8 ? 'text-yellow-500' :
                          'text-gray-900 dark:text-white'
                        ]">
                          {{ withCreditUnit(selectedKey.usage_7d?.toFixed(4) || '0.0000') }}
                        </span>
                        <span class="mx-2 surface-muted">/</span>
                        <span class="surface-muted">
                          {{ withCreditUnit(selectedKey.rate_limit_7d?.toFixed(2) || '0.00') }}
                        </span>
                      </div>
                    </div>
                    <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                      <div
                        :class="[
                          'console-data-mark h-full rounded-full transition-all',
                          selectedKey.usage_7d >= selectedKey.rate_limit_7d ? 'bg-red-500' :
                          selectedKey.usage_7d >= selectedKey.rate_limit_7d * 0.8 ? 'bg-yellow-500' :
                          'bg-green-500'
                        ]"
                        :style="{ width: Math.min((selectedKey.usage_7d / selectedKey.rate_limit_7d) * 100, 100) + '%' }"
                      />
                    </div>
                  </div>
                </div>

                <!-- Reset Rate Limit button (edit mode only) -->
                <div v-if="showEditModal && selectedKey && (selectedKey.rate_limit_5h > 0 || selectedKey.rate_limit_1d > 0 || selectedKey.rate_limit_7d > 0)">
                  <button v-support-readonly
                    type="button"
                    @click="confirmResetRateLimit"
                    class="btn btn-secondary text-sm"
                  >
                    {{ t('keys.resetRateLimitUsage') }}
                  </button>
                </div>
              </div>
            </div>

            <!-- Expiration Section -->
            <div class="space-y-3">
              <div class="flex items-center justify-between">
                <label class="input-label mb-0">{{ t('keys.expiration') }}</label>
                <Toggle
                  v-model="formData.enable_expiration"
                />
              </div>

              <div v-if="formData.enable_expiration" class="space-y-4 pt-2">
                <!-- Quick select buttons (for both create and edit mode) -->
                <div class="flex flex-wrap gap-2">
                  <button
                    v-for="days in ['7', '30', '90']"
                    :key="days"
                    type="button"
                    @click="setExpirationDays(parseInt(days))"
                    :class="[
                      'rounded-lg px-3 py-1.5 text-sm transition-colors',
                      formData.expiration_preset === days
                        ? 'bg-primary-100/[0.15] text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                        : 'surface-inset text-gray-600 hover:surface-hover dark:surface-inset dark:text-gray-400 dark:hover:surface-hover'
                    ]"
                  >
                    {{ showEditModal ? t('keys.extendDays', { days }) : t('keys.expiresInDays', { days }) }}
                  </button>
                  <button
                    type="button"
                    @click="formData.expiration_preset = 'custom'"
                    :class="[
                      'rounded-lg px-3 py-1.5 text-sm transition-colors',
                      formData.expiration_preset === 'custom'
                        ? 'bg-primary-100/[0.15] text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                        : 'surface-inset text-gray-600 hover:surface-hover dark:surface-inset dark:text-gray-400 dark:hover:surface-hover'
                    ]"
                  >
                    {{ t('keys.customDate') }}
                  </button>
                </div>

                <!-- Date picker (always show for precise adjustment) -->
                <div>
                  <label class="input-label">{{ t('keys.expirationDate') }}</label>
                  <DateTimePicker :aria-label="t('keys.expirationDate')"
                    v-model="formData.expiration_date"
                    type="datetime-local"
                    class="input"
                  />
                  <p class="input-hint">{{ t('keys.expirationDateHint') }}</p>
                </div>

                <!-- Current expiration display (only in edit mode) -->
                <div v-if="showEditModal && selectedKey?.expires_at" class="text-sm">
                  <span class="surface-muted">{{ t('keys.currentExpiration') }}: </span>
                  <span class="font-medium text-gray-900 dark:text-white">
                    {{ formatDateTime(selectedKey.expires_at) }}
                  </span>
                </div>
              </div>
            </div>
          </form>
        </div>
      </div>
      <div class="gc-collapse" :data-open="!!createdKey">
        <div>
          <KeyCreatedPanel v-if="createdKey" :api-key="createdKey" :base-url="endpointBaseUrl" />
        </div>
      </div>
      <template #footer>
        <div v-if="createdKey" :key="'created'" class="gc-reveal flex flex-wrap justify-end gap-3">
          <button v-if="showCcs" type="button" class="btn btn-secondary" data-test="key-created-ccs" @click="openCcSwitch(createdKey)">
            <Icon name="upload" size="sm" />
            {{ t('keyConsole.ccs.oneClick') }}
          </button>
          <button type="button" class="btn btn-primary" data-test="key-created-done" @click="closeModals">
            {{ t('keyConsole.created.done') }}
          </button>
        </div>
        <div v-else class="flex justify-end gap-3">
          <button @click="closeModals" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button v-support-readonly
            form="key-form"
            type="submit"
            :disabled="submitting"
            class="btn btn-primary"
            data-tour="key-form-submit"
          >
            <svg
              v-if="submitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{
              submitting
                ? t('keys.saving')
                : showEditModal
                  ? t('common.update')
                  : t('common.create')
            }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <KeyDetailDialog
      :show="detailKey !== null"
      :api-key="detailKey"
      :usage="detailKey ? usageStats[detailKey.id] : undefined"
      :usage-loading="usageLoading"
      :user-rate="detailKey?.group ? userGroupRates[detailKey.group.id] ?? null : null"
      :can-configure="detailKey ? canUseClientConfig(detailKey) : false"
      :show-ccs="showCcs"
      :toggling="toggling"
      @close="detailKeyId = null"
      @copy="detailKey && copyKey(detailKey)"
      @ccswitch="detailKey && openCcSwitch(detailKey)"
      @use="detailKey && openUseKeyModal(detailKey)"
      @test="detailKey && openTestModal(detailKey)"
      @edit="detailKey && editKey(detailKey)"
      @toggle="detailKey && toggleKeyStatus(detailKey)"
      @logs="detailKey && openLogs(detailKey)"
      @delete="detailKey && confirmDelete(detailKey)"
    />

    <CcSwitchImportDialog
      :show="ccsKey !== null"
      :api-key="ccsKey"
      :base-url="endpointBaseUrl"
      :homepage="publicSettings?.api_base_url || windowOrigin"
      @close="ccsKey = null"
    />

    <BulkEditKeysModal
      :show="showBulkEditModal"
      :selected-keys="selectedApiKeys"
      :groups="groups"
      :quoted-groups="quotedGroups"
      :quotes-loading="quotesLoading"
      :quotes-failed="quotesFailed"
      @retry-quotes="refreshQuotes"
      @close="showBulkEditModal = false"
      @updated="handleBulkUpdated"
    />

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('keys.deleteKey')"
      :message="t('keys.deleteConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="handleDelete"
      @cancel="showDeleteDialog = false"
    />

    <!-- Reset Quota Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetQuotaDialog"
      :title="t('keys.resetQuotaTitle')"
      :message="t('keys.resetQuotaConfirmMessage', { name: selectedKey?.name, used: selectedKey?.quota_used?.toFixed(4) })"
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetQuotaUsed"
      @cancel="showResetQuotaDialog = false"
    />

    <!-- Reset Rate Limit Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetRateLimitDialog"
      :title="t('keys.resetRateLimitTitle')"
      :message="t('keys.resetRateLimitConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetRateLimitUsage"
      @cancel="showResetRateLimitDialog = false"
    />

    <!-- Global one-click access always selects from plaintext personal keys. -->
    <BaseDialog
      :show="showOneClickKeySelect"
      :title="t('keys.oneClickSelect.title')"
      width="normal"
      @close="closeOneClickKeySelect"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-400">
          {{ t('keys.oneClickSelect.description') }}
        </p>
        <div class="max-h-80 space-y-2 overflow-y-auto pr-1">
          <button
            v-for="key in clientConfigKeys"
            :key="key.id"
            type="button"
            class="flex w-full items-center justify-between rounded-lg border border-gray-200 px-4 py-3 text-left transition-colors hover:border-primary-500 hover:bg-primary-50 dark:border-dark-600 dark:hover:border-primary-500 dark:hover:bg-primary-900/20"
            data-test="one-click-key-option"
            @click="selectOneClickKey(key)"
          >
            <div class="min-w-0">
              <div class="truncate font-medium text-gray-900 dark:text-white">
                {{ key.name }}
              </div>
              <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                <code class="code">{{ maskApiKey(key.key) }}</code>
                <span v-if="isAutoRouting(key)" class="text-violet-700 dark:text-violet-300">{{ t('keys.routingMode.auto') }}</span>
                <span v-else-if="key.group?.name">{{ key.group.name }}</span>
                <span v-else>{{ t('keys.noGroup') }}</span>
              </div>
            </div>
            <Icon name="chevronRight" size="sm" class="ml-3 flex-shrink-0 text-gray-400" />
          </button>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end">
          <button type="button" class="btn btn-secondary" @click="closeOneClickKeySelect">
            {{ t('common.cancel') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Use Key Modal -->
    <UseKeyModal
      :show="showUseKeyModal"
      :aggregate-key-id="useKey?.group_binding_mode === 'all' || useKey?.group_binding_mode === 'custom' ? useKey.id : undefined"
      :api-key="useKey?.key || ''"
      :api-key-id="useKey?.id"
      :base-url="endpointBaseUrl"
      :platform="useKey?.group?.platform || null"
      :group-id="useKey?.group_id || null"
      :conversion-enabled="useKey?.conversion_enabled || false"
      :default-client="useKey?.default_client || ''"
      :default-endpoint="useKey?.default_endpoint || ''"
      :claude-code-only="useKey?.group?.claude_code_only || false"
      :routing-mode="useKey?.routing_mode || 'fixed'"
      :allow-messages-dispatch="useKey?.group?.allow_messages_dispatch || false"
      @close="closeUseKeyModal"
    />

    <!-- Test Key Modal -->
    <KeyTestModal
      :show="showTestModal"
      :api-key="selectedTestKey"
      :base-url="publicSettings?.api_base_url || ''"
      @close="closeTestModal"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { withCreditUnit, creditUnit } from '@/utils/credits'
import DateTimePicker from '@/components/common/DateTimePicker.vue'
import { useLocaleRefresh } from '@/utils/localeRefresh'
import { supportReadonly as vSupportReadonly } from '@/directives/supportReadonly'
import Toggle from '@/components/common/Toggle.vue'
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useOnboardingStore } from '@/stores/onboarding'
import { useClipboard } from '@/composables/useClipboard'
import { clearAutoRoutingCapabilities } from '@/composables/useAsyncImageAccess'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useApiEndpoint } from '@/composables/useApiEndpoint'
import RoutingPriorityPanel from '@/components/keys/RoutingPriorityPanel.vue'
import { keysAPI, authAPI, usageAPI, userGroupsAPI } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import BulkEditKeysModal from '@/components/keys/BulkEditKeysModal.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select from '@/components/common/Select.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import NativeAccessFields from '@/components/keys/NativeAccessFields.vue'
import UseKeyModal from '@/components/keys/UseKeyModal.vue'
import KeyTestModal from '@/components/keys/KeyTestModal.vue'
import CheckboxDropdown from '@/components/common/CheckboxDropdown.vue'
import ApiEndpointBar from '@/components/keys/ApiEndpointBar.vue'
import KeyStatsBar, { type KeyStat } from '@/components/keys/KeyStatsBar.vue'
import KeyDetailDialog from '@/components/keys/KeyDetailDialog.vue'
import KeyCreatedPanel from '@/components/keys/KeyCreatedPanel.vue'
import CcSwitchImportDialog from '@/components/keys/CcSwitchImportDialog.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import GroupRateSummary from '@/components/keys/GroupRateSummary.vue'
import { vSegmented } from '@/components/gotocc/vSegmented'
import { useGroupRateQuotes } from '@/composables/useGroupRateQuotes'
import { useQuoteRefresh } from '@/composables/useQuoteRefresh'
import type { ApiKey, ApiKeyRoutingMode, Group, PublicSettings, SubscriptionType, GroupPlatform, UpdateApiKeyRequest } from '@/types'
import ScopeDropdown, { type DataScope } from '@/components/team/ScopeDropdown.vue'
import type { Column } from '@/components/common/types'
import type { BatchApiKeyUsageStats, UserDashboardStats } from '@/api/usage'
import { formatDateTime } from '@/utils/format'
import { maskApiKey } from '@/utils/maskApiKey'
import { extractI18nErrorMessage } from '@/utils/apiError'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { platformBadgeLightClass, platformLabel } from '@/utils/platformColors'
import { KEY_GROUP_PROVIDERS, KEY_GROUP_PROVIDER_ICONS, getKeyGroupProvider, type KeyGroupProvider } from '@/utils/keyGroupProviders'

const routingPriorityPanel = ref<InstanceType<typeof RoutingPriorityPanel> | null>(null)
const createdKeyForRetry = ref<ApiKey | null>(null)

const { t } = useI18n()

// Helper to format date for datetime-local input
const formatDateTimeLocal = (isoDate: string): string => {
  const date = new Date(isoDate)
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

interface GroupOption {
  value: number
  label: string
  description: string | null
  rate: number
  userRate: number | null
  peakRateEnabled: boolean
  peakStart: string
  peakEnd: string
  peakRateMultiplier: number
  subscriptionType: SubscriptionType
  platform: GroupPlatform
}

const appStore = useAppStore()
const route = useRoute()
const router = useRouter()
const scope = ref<DataScope>(route.query.scope === 'team' ? 'team' : 'personal')
const onboardingStore = useOnboardingStore()
const { copyToClipboard: clipboardCopy } = useClipboard()
const { groups: quotedGroups, loading: quotesLoading, failed: quotesFailed, refresh: refreshQuotes } = useGroupRateQuotes(() => scope.value)
const quoteFor = (id: number | null) => quotedGroups.value.find(group => group.id === id)
useQuoteRefresh(refreshQuotes)

const publicSettings = ref<PublicSettings | null>(null)
const teamFeatureEnabled = computed(() => publicSettings.value?.team_enabled !== false)
const showCcs = computed(() => !publicSettings.value?.hide_ccs_import_button)
const windowOrigin = window.location.origin
// Every Base URL shown or copied on this page follows the entry chosen in the endpoint bar.
const { baseUrl: endpointBaseUrl } = useApiEndpoint(publicSettings)

// ==================== Key list ====================
// The whole scope is loaded once (the API allows 1000 per page) so platform tabs,
// search, status and paging switch instantly on the client.
const KEY_LIST_LIMIT = 1000

const apiKeys = ref<ApiKey[]>([])
const groups = ref<Group[]>([])
const loading = ref(false)
const submitting = ref(false)
const toggling = ref(false)
const usageStats = ref<Record<string, BatchApiKeyUsageStats>>({})
const usageLoading = ref(false)
const dashboardStats = ref<UserDashboardStats | null>(null)
const userGroupRates = ref<Record<number, number>>({})
let abortController: AbortController | null = null

const filterSearch = ref('')
const filterStatus = ref('')
const platformTab = ref('all')
const page = ref(1)
const pageSize = ref(getPersistedPageSize())
const sortState = ref({ key: 'name', order: 'asc' as 'asc' | 'desc' })

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('common.name'), sortable: true },
  { key: 'group', label: t('keys.group'), sortable: false },
  { key: 'last_used_at', label: t('keys.lastUsedAt'), sortable: true },
  { key: 'today', label: t('keyConsole.stats.today'), sortable: true }
])

const isAggregate = (key: ApiKey) => key.group_binding_mode === 'all' || key.group_binding_mode === 'custom'
const isAutoRouting = (key: Pick<ApiKey, 'routing_mode'> | null | undefined) => key?.routing_mode === 'auto'
const keyCategory = (key: ApiKey): string =>
  isAggregate(key) ? 'aggregate' : isAutoRouting(key) ? 'auto' : key.group?.platform ?? 'none'
const keyTail = (key: string) => `…${key.slice(-4)}`

const platformTabs = computed(() => {
  const counts = new Map<string, number>()
  for (const key of apiKeys.value) counts.set(keyCategory(key), (counts.get(keyCategory(key)) ?? 0) + 1)
  const label = (id: string) =>
    id === 'aggregate' ? t('keyConsole.list.aggregate')
      : id === 'auto' ? t('keys.routingMode.auto')
        : id === 'none' ? t('keys.noGroup')
          : platformLabel(id)
  const special = new Set(['aggregate', 'auto', 'none'])
  return [
    { id: 'all', label: t('keyConsole.list.all'), count: apiKeys.value.length, platform: undefined },
    ...[...counts.entries()]
      .sort((a, b) => b[1] - a[1] || label(a[0]).localeCompare(label(b[0])))
      .map(([id, count]) => ({ id, label: label(id), count, platform: special.has(id) ? undefined : id as GroupPlatform }))
  ]
})

const filteredKeys = computed(() => {
  const query = filterSearch.value.trim().toLowerCase()
  const list = apiKeys.value.filter(key =>
    (platformTab.value === 'all' || keyCategory(key) === platformTab.value) &&
    (!filterStatus.value || key.status === filterStatus.value) &&
    (!query || key.name.toLowerCase().includes(query) || key.key.toLowerCase().includes(query) || (key.group?.name.toLowerCase().includes(query) ?? false))
  )
  const direction = sortState.value.order === 'asc' ? 1 : -1
  const sortValue = (key: ApiKey): string | number =>
    sortState.value.key === 'today' ? usageStats.value[key.id]?.today_actual_cost ?? 0
      : sortState.value.key === 'last_used_at' ? (key.last_used_at ? new Date(key.last_used_at).getTime() : 0)
        : key.name.toLowerCase()
  return [...list].sort((a, b) => {
    const left = sortValue(a)
    const right = sortValue(b)
    return (left < right ? -1 : left > right ? 1 : 0) * direction
  })
})

const pageKeys = computed(() => filteredKeys.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))

const statusFilterOptions = computed(() => [
  { value: '', label: t('keys.allStatus') },
  { value: 'active', label: t('keys.status.active') },
  { value: 'inactive', label: t('keys.status.inactive') },
  { value: 'disabled', label: t('keys.status.disabled') },
  { value: 'quota_exhausted', label: t('keys.status.quota_exhausted') },
  { value: 'expired', label: t('keys.status.expired') }
])

const onFilterChange = () => {
  selectedIds.value = []
  page.value = 1
}

const onStatusFilterChange = (value: string | number | boolean | null) => {
  filterStatus.value = value as string
  onFilterChange()
}

const selectPlatformTab = (id: string) => {
  platformTab.value = id
  onFilterChange()
}

watch(filterSearch, onFilterChange)

const handlePageChange = (value: number) => {
  selectedIds.value = []
  page.value = value
}

const handlePageSizeChange = (value: number) => {
  selectedIds.value = []
  pageSize.value = value
  page.value = 1
}

const handleSort = (key: string, order: 'asc' | 'desc') => {
  selectedIds.value = []
  sortState.value = { key, order }
  page.value = 1
}

// ==================== Overview stats ====================
const overviewStats = computed<KeyStat[]>(() => {
  const personal = scope.value === 'personal'
  const keyStats = Object.values(usageStats.value)
  const sum = (pick: (stat: BatchApiKeyUsageStats) => number) => keyStats.reduce((total, stat) => total + pick(stat), 0)
  const lastUsed = apiKeys.value.reduce<string | null>((latest, key) =>
    key.last_used_at && (!latest || key.last_used_at > latest) ? key.last_used_at : latest, null)
  const statsLoading = personal ? dashboardStats.value === null : usageLoading.value
  return [
    {
      key: 'today',
      label: t('keyConsole.stats.today'),
      value: withCreditUnit((personal ? dashboardStats.value?.today_actual_cost ?? 0 : sum(stat => stat.today_actual_cost)).toFixed(2)),
      loading: statsLoading
    },
    {
      key: 'total',
      // The batch per-key endpoint covers 30 days; only the personal dashboard has an all-time total.
      label: personal ? t('keyConsole.stats.total') : t('keyConsole.stats.last30'),
      value: withCreditUnit((personal ? dashboardStats.value?.total_actual_cost ?? 0 : sum(stat => stat.total_actual_cost)).toFixed(2)),
      loading: statsLoading
    },
    {
      key: 'active',
      label: t('keyConsole.stats.active'),
      value: `${apiKeys.value.filter(key => key.status === 'active').length} / ${apiKeys.value.length}`,
      loading: loading.value && apiKeys.value.length === 0
    },
    {
      key: 'lastUsed',
      label: t('keyConsole.stats.lastUsed'),
      value: lastUsed ? formatDateTime(lastUsed) : t('keyConsole.detail.neverUsed'),
      loading: loading.value && apiKeys.value.length === 0
    }
  ]
})

// ==================== Selection & bulk edit ====================
const selectedIds = ref<number[]>([])
const showBulkEditModal = ref(false)
const selectedApiKeys = computed(() => apiKeys.value.filter((key) => selectedIds.value.includes(key.id)))

const handleSelectionChange = (ids: Array<string | number>) => {
  const visibleIds = new Set(pageKeys.value.map((key) => key.id))
  selectedIds.value = [...new Set(ids.map(Number))].filter((id) => visibleIds.has(id))
}

const handleBulkUpdated = (succeededIds: number[]) => {
  const succeeded = new Set(succeededIds)
  selectedIds.value = selectedIds.value.filter((id) => !succeeded.has(id))
  loadApiKeys()
}

// ==================== Dialog state ====================
const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteDialog = ref(false)
const showResetQuotaDialog = ref(false)
const showResetRateLimitDialog = ref(false)
const showUseKeyModal = ref(false)
const showOneClickKeySelect = ref(false)
const showTestModal = ref(false)
const selectedKey = ref<ApiKey | null>(null)
const useKey = ref<ApiKey | null>(null)
const selectedTestKey = ref<ApiKey | null>(null)
const detailKeyId = ref<number | null>(null)
const detailKey = computed(() => apiKeys.value.find(key => key.id === detailKeyId.value) ?? null)
const ccsKey = ref<ApiKey | null>(null)
const createdKey = ref<ApiKey | null>(null)
const editingAggregateGroupIDs = ref<number[]>([])
let keyEditRequestID = 0

const canUseClientConfig = (key: ApiKey) => {
  const isTeamKey = key.scope === 'team' || key.team_id != null
  return scope.value === 'team' ? isTeamKey : !isTeamKey
}

const clientConfigKeys = computed(() => apiKeys.value.filter(canUseClientConfig))

const openDetail = (key: ApiKey) => {
  detailKeyId.value = key.id
}

const copyKey = (key: ApiKey) => {
  void clipboardCopy(key.key, t('keys.copied'))
}

const openCcSwitch = (key: ApiKey) => {
  detailKeyId.value = null
  closeModals()
  ccsKey.value = key
}

const openLogs = (key: ApiKey) => {
  void router.push({ path: '/usage', query: { api_key_id: String(key.id) } })
}

const nativeAccessFields = ref<InstanceType<typeof NativeAccessFields> | null>(null)
const emptyForm = () => ({
  group_binding_mode: 'single' as 'single' | 'all' | 'custom',
  aggregate_group_ids: [] as number[],
  name: '',
  conversion_enabled: false,
  default_client: '',
  default_endpoint: '',
  group_id: null as number | null,
  routing_mode: 'fixed' as ApiKeyRoutingMode,
  status: 'active' as 'active' | 'inactive',
  use_custom_key: false,
  custom_key: '',
  enable_ip_restriction: false,
  ip_whitelist: '',
  ip_blacklist: '',
  // Quota settings (empty = unlimited)
  enable_quota: false,
  quota: null as number | null,
  // Rate limit settings
  enable_rate_limit: false,
  rate_limit_5h: null as number | null,
  rate_limit_1d: null as number | null,
  rate_limit_7d: null as number | null,
  enable_expiration: false,
  expiration_preset: '30' as '7' | '30' | '90' | 'custom',
  expiration_date: ''
})
const formData = ref(emptyForm())

// 自定义Key验证
const customKeyError = computed(() => {
  if (!formData.value.use_custom_key || !formData.value.custom_key) {
    return ''
  }
  const key = formData.value.custom_key
  if (key.length < 16) {
    return t('keys.customKeyTooShort')
  }
  // 检查字符：只允许字母、数字、下划线、连字符
  if (!/^[a-zA-Z0-9_-]+$/.test(key)) {
    return t('keys.customKeyInvalidChars')
  }
  return ''
})

const statusOptions = computed(() => [
  { value: 'active', label: t('common.active') },
  { value: 'inactive', label: t('common.inactive') }
])

const shouldSubmitEditStatus = (key: ApiKey, status: 'active' | 'inactive') => {
  if (key.status === 'disabled' || key.status === 'quota_exhausted' || key.status === 'expired') {
    return status === 'active'
  }
  return true
}

// Convert groups to Select options format with rate multiplier and subscription type
const groupOptions = computed(() =>
  groups.value.map((group) => ({
    value: group.id,
    label: group.name,
    description: group.description,
    rate: group.rate_multiplier,
    userRate: userGroupRates.value[group.id] ?? null,
    peakRateEnabled: group.peak_rate_enabled,
    peakStart: group.peak_start,
    peakEnd: group.peak_end,
    peakRateMultiplier: group.peak_rate_multiplier,
    subscriptionType: group.subscription_type,
    platform: group.platform
  }))
)

const aggregateModeOptions = computed(() => [
  { value: 'all', label: t('keys.aggregateAll') },
  { value: 'custom', label: t('keys.aggregateCustom') }
])
const aggregateGroupOptions = computed(() => {
  const options: { value: number; label: string; group?: Group }[] = groups.value
    .filter(group => group.platform !== 'composite' && group.subscription_type === 'standard')
    .map(group => ({ value: group.id, label: `${group.name} · ×${userGroupRates.value[group.id] ?? group.rate_multiplier}`, group }))
  for (const id of formData.value.aggregate_group_ids) {
    if (!options.some(option => option.value === id)) options.push({ value: id, label: t('keys.aggregateUnavailable', { id }) })
  }
  return options
})
const startAggregateCreate = () => {
  closeModals()
  formData.value.group_binding_mode = 'all'
  showCreateModal.value = true
}

const createProvider = ref<KeyGroupProvider>('anthropic')
const createProviderOptions = computed(() => KEY_GROUP_PROVIDERS.map((value) => ({
  value,
  label: t(`keys.providers.${value}`),
  count: groups.value.filter((group) => getKeyGroupProvider(group.platform) === value).length
})))

const formGroupOptions = computed(() => showEditModal.value
  ? groupOptions.value
  : groupOptions.value.filter((group) => getKeyGroupProvider(group.platform) === createProvider.value)
)

const selectCreateProvider = (provider: KeyGroupProvider) => {
  if (createProvider.value === provider) return
  createProvider.value = provider
  formData.value.group_id = null
}

// Also handles groups arriving after the create dialog has already opened.
watch([showCreateModal, createProviderOptions], ([isOpen, providers], [wasOpen]) => {
  if (!isOpen) return
  if (!wasOpen || !providers.some((provider) => provider.value === createProvider.value && provider.count > 0)) {
    selectCreateProvider(providers.find((provider) => provider.count > 0)?.value ?? 'anthropic')
  }
  if (!formGroupOptions.value.some((group) => group.value === formData.value.group_id)) {
    formData.value.group_id = null
  }
})

const setFormRoutingMode = (routingMode: ApiKeyRoutingMode) => {
  if (submitting.value) return
  formData.value.routing_mode = routingMode
  if (routingMode === 'auto') {
    formData.value.group_id = null
  }
}

const isAbortError = (error: unknown) => {
  if (!error || typeof error !== 'object') return false
  const { name, code } = error as { name?: string; code?: string }
  return name === 'AbortError' || code === 'ERR_CANCELED'
}

// ==================== Loading ====================
const loadApiKeys = async () => {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  const { signal } = controller
  loading.value = true
  try {
    const response = await keysAPI.list(1, KEY_LIST_LIMIT, { scope: scope.value, sort_by: 'name', sort_order: 'asc' }, { signal })
    if (signal.aborted) return
    apiKeys.value = response.items
    handleSelectionChange(selectedIds.value)
    if (!platformTabs.value.some(tab => tab.id === platformTab.value)) platformTab.value = 'all'
    const pages = Math.max(Math.ceil(filteredKeys.value.length / pageSize.value), 1)
    if (page.value > pages) page.value = pages

    // 用量统计要聚合 30 天的 usage_logs，可能比列表慢很多：不阻塞列表渲染，单独加载
    usageLoading.value = response.items.length > 0
    if (response.items.length > 0) {
      void loadUsageStats(response.items.map((k) => k.id), signal)
    }
  } catch (error) {
    if (isAbortError(error)) {
      return
    }
    appStore.showError(t('keys.failedToLoad'))
  } finally {
    if (abortController === controller) {
      loading.value = false
    }
  }
}

let groupsRequest = 0
const loadUsageStats = async (keyIds: number[], signal: AbortSignal) => {
  try {
    const usageResponse = await usageAPI.getDashboardApiKeysUsage(keyIds, { signal })
    if (signal.aborted) return
    usageStats.value = usageResponse.stats
  } catch (e) {
    if (!isAbortError(e)) {
      console.error('Failed to load usage stats:', e)
    }
  } finally {
    if (!signal.aborted) {
      usageLoading.value = false
    }
  }
}

const loadDashboardStats = async () => {
  dashboardStats.value = await usageAPI.getDashboardStats()
}

const loadGroups = async () => {
  const request = ++groupsRequest
  try {
    const result = await userGroupsAPI.getAvailable(scope.value)
    if (request === groupsRequest) groups.value = result
  } catch (error) {
    console.error('Failed to load groups:', error)
  }
}

const loadUserGroupRates = async () => {
  try {
    userGroupRates.value = await userGroupsAPI.getUserGroupRates(scope.value)
  } catch (error) {
    console.error('Failed to load user group rates:', error)
  }
}

const loadPublicSettings = async () => {
  try {
    publicSettings.value = await authAPI.getPublicSettings()
    if (!teamFeatureEnabled.value && scope.value === 'team') scope.value = 'personal'
  } catch (error) {
    console.error('Failed to load public settings:', error)
  }
}

// ==================== Client config, test ====================
const openUseKeyModal = (key: ApiKey) => {
  if (!canUseClientConfig(key)) return
  detailKeyId.value = null
  useKey.value = key
  showUseKeyModal.value = true
}

const openOneClickAccess = () => {
  if (clientConfigKeys.value.length === 0) return
  if (clientConfigKeys.value.length === 1) {
    openUseKeyModal(clientConfigKeys.value[0])
    return
  }
  showOneClickKeySelect.value = true
}

const closeOneClickKeySelect = () => {
  showOneClickKeySelect.value = false
}

const selectOneClickKey = (key: ApiKey) => {
  closeOneClickKeySelect()
  openUseKeyModal(key)
}

const closeUseKeyModal = () => {
  showUseKeyModal.value = false
  useKey.value = null
}

const openTestModal = (key: ApiKey) => {
  detailKeyId.value = null
  selectedTestKey.value = key
  showTestModal.value = true
}

const closeTestModal = () => {
  showTestModal.value = false
  selectedTestKey.value = null
}

// ==================== Edit, status, delete ====================
const editKey = async (key: ApiKey) => {
  detailKeyId.value = null
  const requestID = ++keyEditRequestID
  selectedKey.value = key
  // The list hides groups the user can no longer use, but the stored aggregate
  // scope must survive an unrelated edit; read it from the key detail.
  let storedAggregateGroupIDs = [...(key.aggregate_group_ids || [])]
  if (key.group_binding_mode === 'custom') {
    try {
      const detail = await keysAPI.getById(key.id)
      storedAggregateGroupIDs = [...(detail.aggregate_group_ids || storedAggregateGroupIDs)]
    } catch (error) {
      if (requestID !== keyEditRequestID) return
      console.error('Error loading API key detail:', error)
      appStore.showError(t('keys.failedToLoad'))
      selectedKey.value = null
      return
    }
  }
  if (requestID !== keyEditRequestID) return
  editingAggregateGroupIDs.value = [...storedAggregateGroupIDs]
  const hasIPRestriction = (key.ip_whitelist?.length > 0) || (key.ip_blacklist?.length > 0)
  const hasExpiration = !!key.expires_at
  formData.value = {
    group_binding_mode: key.group_binding_mode || 'single',
    aggregate_group_ids: storedAggregateGroupIDs,
    name: key.name,
    conversion_enabled: key.conversion_enabled || false,
    default_client: key.default_client || '',
    default_endpoint: key.default_endpoint || '',
    group_id: key.group_id,
    routing_mode: isAutoRouting(key) ? 'auto' : 'fixed',
    status: key.status === 'active' ? 'active' : 'inactive',
    use_custom_key: false,
    custom_key: '',
    enable_ip_restriction: hasIPRestriction,
    ip_whitelist: (key.ip_whitelist || []).join('\n'),
    ip_blacklist: (key.ip_blacklist || []).join('\n'),
    enable_quota: key.quota > 0,
    quota: key.quota > 0 ? key.quota : null,
    enable_rate_limit: (key.rate_limit_5h > 0) || (key.rate_limit_1d > 0) || (key.rate_limit_7d > 0),
    rate_limit_5h: key.rate_limit_5h || null,
    rate_limit_1d: key.rate_limit_1d || null,
    rate_limit_7d: key.rate_limit_7d || null,
    enable_expiration: hasExpiration,
    expiration_preset: 'custom',
    expiration_date: key.expires_at ? formatDateTimeLocal(key.expires_at) : ''
  }
  showEditModal.value = true
}

const toggleKeyStatus = async (key: ApiKey) => {
  const newStatus = key.status === 'active' ? 'inactive' : 'active'
  toggling.value = true
  try {
    await keysAPI.toggleStatus(key.id, newStatus)
    appStore.showSuccess(
      newStatus === 'active' ? t('keys.keyEnabledSuccess') : t('keys.keyDisabledSuccess')
    )
    await loadApiKeys()
  } catch (error) {
    appStore.showError(t('keys.failedToUpdateStatus'))
  } finally {
    toggling.value = false
  }
}

const confirmDelete = (key: ApiKey) => {
  selectedKey.value = key
  showDeleteDialog.value = true
}

const handleSubmit = async () => {
  if (nativeAccessFields.value && !nativeAccessFields.value.validate()) return
  if (submitting.value) return
  if (formData.value.routing_mode === 'auto' && routingPriorityPanel.value?.validate?.() === false) return
  // Fixed routing keeps the existing group requirement. Auto routing uses the
  // explicit null/group contract and lets the server resolve a group per request.
  if (formData.value.group_binding_mode === 'single' && formData.value.routing_mode === 'fixed' && formData.value.group_id === null) {
    appStore.showError(t('keys.groupRequired'))
    return
  }

  if (formData.value.group_binding_mode === 'custom' && formData.value.aggregate_group_ids.length === 0) {
    appStore.showError(t('keys.aggregateGroupsRequired'))
    return
  }

  // Validate custom key if enabled
  if (!showEditModal.value && formData.value.use_custom_key) {
    if (!formData.value.custom_key) {
      appStore.showError(t('keys.customKeyRequired'))
      return
    }
    if (customKeyError.value) {
      appStore.showError(customKeyError.value)
      return
    }
  }

  // Parse IP lists only if IP restriction is enabled
  const parseIPList = (text: string): string[] =>
    text.split('\n').map(ip => ip.trim()).filter(ip => ip.length > 0)
  const ipWhitelist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_whitelist) : []
  const ipBlacklist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_blacklist) : []

  // Calculate quota value (null/empty/0 = unlimited, stored as 0)
  const quota = formData.value.quota && formData.value.quota > 0 ? formData.value.quota : 0

  // Calculate expiration
  let expiresInDays: number | undefined
  let expiresAt: string | null | undefined
  if (formData.value.enable_expiration && formData.value.expiration_date) {
    if (!showEditModal.value) {
      // Create mode: calculate days from date
      const expDate = new Date(formData.value.expiration_date)
      const now = new Date()
      const diffDays = Math.ceil((expDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
      expiresInDays = diffDays > 0 ? diffDays : 1
    } else {
      // Edit mode: use custom date directly
      expiresAt = new Date(formData.value.expiration_date).toISOString()
    }
  } else if (showEditModal.value) {
    // Edit mode: if expiration disabled or date cleared, send empty string to clear
    expiresAt = ''
  }

  // Calculate rate limit values (send 0 when toggle is off)
  const rateLimitData = formData.value.enable_rate_limit ? {
    rate_limit_5h: formData.value.rate_limit_5h && formData.value.rate_limit_5h > 0 ? formData.value.rate_limit_5h : 0,
    rate_limit_1d: formData.value.rate_limit_1d && formData.value.rate_limit_1d > 0 ? formData.value.rate_limit_1d : 0,
    rate_limit_7d: formData.value.rate_limit_7d && formData.value.rate_limit_7d > 0 ? formData.value.rate_limit_7d : 0,
  } : { rate_limit_5h: 0, rate_limit_1d: 0, rate_limit_7d: 0 }

  submitting.value = true
  let keyFieldsSaved = false
  try {
    const groupID = formData.value.routing_mode === 'auto' ? null : formData.value.group_id
    const targetKey = showEditModal.value ? selectedKey.value : createdKeyForRetry.value
    if (targetKey) {
      const mode = formData.value.group_binding_mode
      const originalIDs = showEditModal.value ? editingAggregateGroupIDs.value : (targetKey.aggregate_group_ids || [])
      const currentIDs = mode === 'custom' ? formData.value.aggregate_group_ids : []
      const bindingChanged = mode !== (targetKey.group_binding_mode || 'single') ||
        (mode === 'custom' && (currentIDs.length !== originalIDs.length || currentIDs.some(id => !originalIDs.includes(id))))
      const updates: UpdateApiKeyRequest = {
        ...(mode !== 'single' && bindingChanged ? { group_binding_mode: mode, aggregate_group_ids: currentIDs } : {}),
        name: formData.value.name,
        group_id: groupID,
        routing_mode: formData.value.routing_mode,
        conversion_enabled: groups.value.find(group => group.id === formData.value.group_id)?.platform === 'composite' && formData.value.conversion_enabled,
        default_client: formData.value.default_client,
        default_endpoint: formData.value.default_endpoint,
        ip_whitelist: ipWhitelist,
        ip_blacklist: ipBlacklist,
        quota: quota,
        expires_at: expiresAt,
        rate_limit_5h: rateLimitData.rate_limit_5h,
        rate_limit_1d: rateLimitData.rate_limit_1d,
        rate_limit_7d: rateLimitData.rate_limit_7d,
      }
      if (shouldSubmitEditStatus(targetKey, formData.value.status)) {
        updates.status = formData.value.status
      }
      await keysAPI.update(targetKey.id, updates)
      keyFieldsSaved = true
      if (formData.value.routing_mode === 'auto') await routingPriorityPanel.value?.save?.(targetKey.id)
      clearAutoRoutingCapabilities()
      appStore.showSuccess(t('keys.keyUpdatedSuccess'))
      closeModals()
    } else {
      const customKey = formData.value.use_custom_key ? formData.value.custom_key : undefined
      const created = await keysAPI.create(
        formData.value.name,
        groupID,
        customKey,
        ipWhitelist,
        ipBlacklist,
        quota,
        expiresInDays,
        rateLimitData,
        scope.value,
        formData.value.routing_mode,
        { conversion_enabled: groups.value.find(group => group.id === formData.value.group_id)?.platform === 'composite' && formData.value.conversion_enabled, default_client: formData.value.default_client, default_endpoint: formData.value.default_endpoint },
        formData.value.group_binding_mode !== 'single' ? { group_binding_mode: formData.value.group_binding_mode, aggregate_group_ids: formData.value.group_binding_mode === 'custom' ? formData.value.aggregate_group_ids : [] } : undefined
      )
      clearAutoRoutingCapabilities()
      keyFieldsSaved = true
      createdKeyForRetry.value = created
      if (formData.value.routing_mode === 'auto') await routingPriorityPanel.value?.save?.(created.id)
      // Only advance tour if active, on submit step, and creation succeeded
      if (onboardingStore.isCurrentStep('[data-tour="key-form-submit"]')) {
        onboardingStore.nextStep(500)
      }
      // The dialog turns into the success page in place.
      createdKey.value = { ...created, group: created.group ?? groups.value.find(group => group.id === created.group_id) }
    }
    loadApiKeys()
  } catch (error: unknown) {
    if (keyFieldsSaved) void loadApiKeys()
    appStore.showError(extractI18nErrorMessage(error, t, 'keys.errors', t('keys.failedToSave')))
    // Don't advance tour on error
  } finally {
    submitting.value = false
  }
}

const onScopeChange = async () => {
  closeOneClickKeySelect()
  closeUseKeyModal()
  detailKeyId.value = null
  ccsKey.value = null
  page.value = 1
  platformTab.value = 'all'
  await router.replace({ query: { ...route.query, scope: scope.value } })
  await Promise.all([loadApiKeys(), loadGroups(), loadUserGroupRates(), refreshQuotes()])
}

/**
 * 处理删除 API Key 的操作
 * 优化：错误处理改进，优先显示后端返回的具体错误消息（如权限不足等），
 * 若后端未返回消息则显示默认的国际化文本
 */
const handleDelete = async () => {
  if (!selectedKey.value) return

  try {
    await keysAPI.delete(selectedKey.value.id)
    appStore.showSuccess(t('keys.keyDeletedSuccess'))
    showDeleteDialog.value = false
    if (detailKeyId.value === selectedKey.value.id) detailKeyId.value = null
    loadApiKeys()
  } catch (error: any) {
    // 优先使用后端返回的错误消息，提供更具体的错误信息给用户
    const errorMsg = error?.message || t('keys.failedToDelete')
    appStore.showError(errorMsg)
  }
}

const closeModals = () => {
  createdKeyForRetry.value = null
  createdKey.value = null
  keyEditRequestID++
  showCreateModal.value = false
  showEditModal.value = false
  selectedKey.value = null
  formData.value = emptyForm()
}

// Show reset quota confirmation dialog
const confirmResetQuota = () => {
  showResetQuotaDialog.value = true
}

// Set expiration date based on quick select days
const setExpirationDays = (days: number) => {
  formData.value.expiration_preset = days.toString() as '7' | '30' | '90'
  const expDate = new Date()
  expDate.setDate(expDate.getDate() + days)
  formData.value.expiration_date = formatDateTimeLocal(expDate.toISOString())
}

// Reset quota used for an API key
const resetQuotaUsed = async () => {
  const key = selectedKey.value
  if (!key) return
  showResetQuotaDialog.value = false
  try {
    const updatedKey = await keysAPI.update(key.id, { reset_quota: true })
    appStore.showSuccess(t('keys.quotaResetSuccess'))
    key.quota_used = updatedKey.quota_used
    if (key.status !== updatedKey.status) {
      key.status = updatedKey.status
      if (selectedKey.value?.id === key.id) {
        formData.value.status = updatedKey.status === 'active' ? 'active' : 'inactive'
      }
    }
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToResetQuota')
    appStore.showError(errorMsg)
  }
}

// Show reset rate limit confirmation dialog (from edit modal)
const confirmResetRateLimit = () => {
  showResetRateLimitDialog.value = true
}

// Reset rate limit usage for an API key
const resetRateLimitUsage = async () => {
  if (!selectedKey.value) return
  showResetRateLimitDialog.value = false
  try {
    await keysAPI.update(selectedKey.value.id, { reset_rate_limit_usage: true })
    appStore.showSuccess(t('keys.rateLimitResetSuccess'))
    // Refresh key data
    await loadApiKeys()
    // Update the editing key with fresh data
    const refreshedKey = apiKeys.value.find(k => k.id === selectedKey.value!.id)
    if (refreshedKey) {
      selectedKey.value = refreshedKey
    }
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToResetRateLimit')
    appStore.showError(errorMsg)
  }
}

const loadScopeStats = () => {
  if (scope.value === 'personal') {
    dashboardStats.value = null
    void loadDashboardStats()
  }
}

watch(scope, loadScopeStats)

onMounted(async () => {
  void refreshQuotes()
  loadScopeStats()
  await loadPublicSettings()
  await Promise.all([loadApiKeys(), loadGroups(), loadUserGroupRates()])
})

useLocaleRefresh(() => Promise.all([loadApiKeys(), loadGroups(), refreshQuotes()]))
</script>

<style scoped>
.keys-overview { display: flex; flex-direction: column; gap: 14px; }
.keys-overview-top { display: flex; flex-wrap: wrap; align-items: flex-end; gap: 12px 20px; }
.keys-actions { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 8px; }
.keys-filters { display: flex; flex-direction: column; gap: 12px; }
.keys-tabs-scroll { max-width: 100%; overflow-x: auto; scrollbar-width: none; }
.keys-tab { display: inline-flex; height: 32px; align-items: center; gap: 6px; padding-inline: 12px; font-size: 13px; font-weight: 500; white-space: nowrap; }
.keys-tab-count { min-width: 18px; padding: 0 5px; border-radius: 9999px; background: var(--surface-soft); font-size: 11px; line-height: 18px; text-align: center; color: var(--surface-muted); }
.keys-name { display: flex; align-items: center; gap: 8px; }
.keys-tail { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; color: var(--surface-muted); }
.keys-status-dot { height: 8px; width: 8px; flex: none; border-radius: 9999px; background: theme('colors.gray.400'); }
.keys-status-dot[data-status='active'] { background: theme('colors.emerald.500'); box-shadow: 0 0 0 3px rgb(16 185 129 / 16%); }
.keys-status-dot[data-status='quota_exhausted'] { background: theme('colors.amber.500'); }
.keys-status-dot[data-status='expired'] { background: theme('colors.red.500'); }
.keys-today { display: flex; align-items: center; gap: 10px; }
.keys-row-chevron { margin-left: auto; color: var(--surface-muted); opacity: .45; transition: opacity var(--motion-fast) var(--motion-ease), transform var(--motion-fast) var(--motion-ease); }
:deep(.table-data-row:hover) .keys-row-chevron { opacity: 1; transform: translateX(2px); }
@media (max-width: 640px) {
  .keys-actions { justify-content: flex-start; }
  .keys-actions :deep(.btn) { flex: 1 1 auto; }
}
</style>
