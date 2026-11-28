<template>
  <AppLayout>
    <div class="hub">
      <div v-if="loading" class="flex flex-col gap-4">
        <div class="gc-skeleton h-6 w-72"></div>
        <div class="grid gap-4 lg:grid-cols-2"><div class="gc-skeleton h-64 w-full"></div><div class="gc-skeleton h-64 w-full"></div></div>
      </div>
      <div v-else-if="error && !overview && !affiliate" class="card p-6"><p role="alert" class="text-red-600">{{ error }}</p><button class="btn btn-secondary mt-4" @click="load">{{ tr('重试', 'Retry') }}</button></div>
      <template v-else>
        <div v-if="balanceEmpty" role="alert" class="hub-alert">
          <Icon name="exclamationTriangle" size="md" class="shrink-0" />
          <span class="min-w-0 flex-1">{{ tr('你的平台余额已用完：客户的请求会因站长余额不足而失败。', 'Your platform balance is used up: customer requests will fail until you top up.') }}</span>
          <RouterLink to="/wallet" class="btn btn-primary btn-sm">{{ tr('去充值', 'Top up') }}</RouterLink>
        </div>
        <!-- 两种赚钱方式，各用各的邀请链接 -->
        <p class="hub-lead">
          <template v-if="resellerOn && affiliateOn">{{ tr('这里有两种赚钱方式，各用各的邀请链接，别发错：', 'Two ways to earn here, each with its own invitation link. Keep them apart:') }}</template>
          <template v-else-if="resellerOn">{{ tr('把客户站邀请链接发给客户，客户向你付款、你给他加额度，按你定的价格卖。', 'Share your customer-site link. Customers pay you, you add their credits and sell at your own price.') }}</template>
          <template v-else>{{ tr('把主站邀请链接发给朋友，对方充值后你按比例拿返利，可转入余额。', 'Share your main-site link. When friends top up, you earn a rebate you can move into your balance.') }}</template>
        </p>

        <div class="hub-ways" :class="{ 'hub-ways-single': !(resellerOn && affiliateOn) }">
          <!-- 邀请返利：主站链接，对方在主站自己充值 -->
          <section v-if="affiliateOn" class="card hub-way" data-test="hub-way-affiliate">
            <header class="hub-way-head">
              <span class="hub-way-icon hub-way-icon-rebate"><Icon name="userPlus" size="md" /></span>
              <div class="min-w-0">
                <h2 class="hub-way-title">{{ tr('邀请返利', 'Referral rebates') }}<span class="hub-way-tag" :data-tip="tr('这条链接打开的是主站注册页', 'This link opens the main-site sign-up page')">{{ tr('主站链接', 'Main site') }}</span></h2>
                <p class="hub-way-lead">{{ tr(`把主站邀请链接发给朋友，对方在主站自己充值，你按 ${rebateRates} 拿返利，可转入余额。`, `Share your main-site link with friends. When they top up on the main site you earn ${rebateRates}, which you can move into your balance.`) }}</p>
              </div>
            </header>
            <p v-if="affiliateError" role="alert" class="text-sm text-red-600">{{ affiliateError }}</p>
            <template v-else-if="affiliate">
              <div class="hub-link">
                <span class="hub-link-label">{{ tr('主站邀请链接', 'Main-site invitation link') }}</span>
                <div class="hub-link-box">
                  <code class="hub-link-code" :data-tip="tr('朋友打开后在主站注册，之后他的充值会给你返利', 'Friends sign up on the main site; their later top-ups earn you a rebate')">{{ affiliateLink }}</code>
                  <button type="button" class="btn btn-secondary btn-sm shrink-0" :data-tip="tr('复制主站邀请链接', 'Copy the main-site link')" @click="copyAffiliateLink"><Icon name="copy" size="sm" />{{ tr('复制', 'Copy') }}</button>
                </div>
              </div>
              <HubSteps :steps="affiliateSteps" :open="guideOpen.affiliate" @toggle="guideOpen.affiliate = !guideOpen.affiliate" />
              <button type="button" class="hub-way-more" @click="tab = 'affiliate'">{{ tr('邀请记录、返利和转入余额', 'Invitees, rebates and balance transfer') }}<Icon name="chevronRight" size="sm" class="row-arrow" /></button>
            </template>
          </section>

          <!-- 发展客户：客户站链接，客户向你付款 -->
          <section v-if="resellerOn && overview" class="card hub-way" data-test="hub-way-customers">
            <header class="hub-way-head">
              <span class="hub-way-icon hub-way-icon-customer"><Icon name="users" size="md" /></span>
              <div class="min-w-0">
                <h2 class="hub-way-title">{{ tr('发展客户', 'Grow your customers') }}<span class="hub-way-tag" :data-tip="tr('这条链接打开的是客户站注册页，注册的人归你管理', 'This link opens the customer-site sign-up page; whoever registers becomes your customer')">{{ tr('客户站链接', 'Customer site') }}</span></h2>
                <p class="hub-way-lead">{{ tr('客户通过你的专属链接在客户站注册，向你付款、你给他加额度；客户按你定的价格消耗额度，平台按你的成本从你的平台余额扣。', 'Customers sign up on the customer site through your link and pay you; you add their credits. They spend credits at your price, and the platform charges your cost to your balance.') }}</p>
              </div>
            </header>
            <div class="hub-link">
              <span class="hub-link-label">{{ tr('客户站邀请链接', 'Customer-site invitation link') }}<span v-if="customerSite" class="hub-link-site" :data-tip="tr('客户登录、调用都用这个地址', 'Customers sign in and call the API here')">{{ customerSite }}</span></span>
              <div class="hub-link-box">
                <code class="hub-link-code" :data-tip="tr('客户打开后在客户站注册，自动成为你的客户', 'Customers who sign up here become yours automatically')">{{ inviteLink }}</code>
                <button type="button" class="btn btn-secondary btn-sm shrink-0" :disabled="!inviteLink" :data-tip="tr('复制客户站邀请链接', 'Copy the customer-site link')" @click="copyInvite"><Icon name="copy" size="sm" />{{ tr('复制', 'Copy') }}</button>
              </div>
            </div>
            <HubSteps :steps="customerSteps" :open="guideOpen.customers" @toggle="guideOpen.customers = !guideOpen.customers" />
          </section>
        </div>

        <!-- 概览：返利和客户经营分两组 -->
        <section class="hub-overview" :aria-label="tr('概览', 'Overview')">
          <div v-if="affiliateOn && affiliate" class="hub-group">
            <h3 class="hub-group-title">{{ tr('邀请返利', 'Referral rebates') }}</h3>
            <div class="hub-tiles hub-tiles-3">
              <div v-for="tile in affiliateTiles" :key="tile.key" class="card gc-stat hub-tile" :data-tip="tile.tip">
                <span class="hub-tile-label">{{ tile.label }}</span>
                <strong class="hub-tile-value gc-num" :class="tile.tone">{{ tile.value }}</strong>
                <span class="hub-tile-hint">{{ tile.hint }}</span>
              </div>
            </div>
          </div>
          <div v-if="resellerOn && overview" class="hub-group">
            <h3 class="hub-group-title">{{ tr('客户经营 · 本月', 'Customers · this month') }}</h3>
            <div class="hub-tiles hub-tiles-6">
              <div v-for="tile in customerTiles" :key="tile.key" class="card gc-stat hub-tile" :class="{ 'hub-tile-danger': tile.key === 'balance' && balanceEmpty }" :data-tip="tile.tip">
                <span class="hub-tile-label">{{ tile.label }}</span>
                <strong class="hub-tile-value gc-num" :class="tile.tone">{{ tile.value }}</strong>
                <span class="hub-tile-hint">{{ tile.hint }}</span>
              </div>
            </div>
          </div>
        </section>

        <WorkspaceTabs v-if="tabs.length > 1" v-model="tab" class="hub-tabs" :tabs="tabs" :label="tr('站长中心', 'Reseller center')" />
        <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>

        <div :key="tab" class="gc-reveal">
          <section v-if="tab === 'customers'" class="card hub-panel">
            <header class="hub-panel-head">
              <div class="min-w-0"><h2 class="hub-panel-title">{{ tr('客户管理', 'Customers') }}</h2><p class="hub-panel-lead">{{ tr('客户付款后点「额度」给他加额度；在「管理」里改资料、权限和 Key。', 'After a customer pays, use Credits to add credits; Manage edits profile, access and keys.') }}</p></div>
              <div class="flex flex-wrap items-center gap-2">
                <form class="flex gap-2" @submit.prevent="customerPage = 1; loadCustomers()"><input v-model="search" class="input w-56" :placeholder="tr('搜索邮箱、用户名或备注', 'Search customers')" /><button class="btn btn-secondary shrink-0">{{ tr('搜索', 'Search') }}</button></form>
                <button class="btn btn-primary" :data-tip="tr('直接替客户建账户，按你设的初始额度发放', 'Create an account for a customer with your initial credits')" @click="openCustomer(null)"><Icon name="plus" size="sm" />{{ tr('创建客户', 'Create customer') }}</button>
              </div>
            </header>
            <div v-if="!customers?.items.length" class="hub-empty">
              <span class="hub-empty-icon"><Icon name="users" size="lg" /></span>
              <p class="text-sm font-medium">{{ search ? tr('没有找到匹配的客户', 'No matching customers') : tr('还没有客户', 'No customers yet') }}</p>
              <p class="text-xs surface-muted">{{ tr('复制上方客户站邀请链接发给客户，或直接创建客户。', 'Share the customer-site link above, or create a customer directly.') }}</p>
            </div>
            <div v-else class="record-table-scroll">
              <table class="record-table w-full min-w-[720px] text-left text-sm">
                <thead><tr class="surface-muted">
                  <th class="px-4 py-2 font-medium" :data-tip="tr('客户的用户名和邮箱；标签是账户状态', 'Customer name and email; the badge is the account status')">{{ tr('客户', 'Customer') }}</th>
                  <th class="px-4 py-2 font-medium" :data-tip="tr('还能用的额度；冻结是进行中的任务暂扣的', 'Credits left; reserved is held by running tasks')">{{ tr('可用 / 冻结额度', 'Available / reserved') }}</th>
                  <th class="px-4 py-2 font-medium" :data-tip="tr('客户累计按你的价格用掉的额度，以及平台按你的成本扣你的金额', 'Credits used at your price, and what the platform charged you at cost')">{{ tr('已用额度 / 你的成本', 'Credits used / your cost') }}</th>
                  <th class="px-4 py-2 font-medium" :data-tip="tr('只有你能看到的备注', 'Private notes only you can see')">{{ tr('备注', 'Notes') }}</th>
                  <th class="px-4 py-2 font-medium" :data-tip="tr('加减额度、管理账户、单独定价', 'Adjust credits, manage the account, set pricing')">{{ tr('操作', 'Actions') }}</th>
                </tr></thead>
                <tbody><tr v-for="customer in customers.items" :key="customer.user_id">
                  <td class="px-4 py-3"><div class="flex items-center gap-2"><p class="font-medium">{{ customer.username || customer.email }}</p><span class="hub-badge" :class="customer.status === 'active' ? 'hub-badge-ok' : 'hub-badge-off'" :data-tip="customer.status === 'active' ? tr('启用：可以正常登录和调用', 'Active: can sign in and call the API') : tr('停用：登录和调用都会被拒绝', 'Disabled: sign-in and API calls are refused')">{{ customer.status === 'active' ? tr('启用', 'Active') : tr('停用', 'Disabled') }}</span></div><p class="mt-1 text-xs surface-muted">{{ customer.email }}</p></td>
                  <td class="px-4 py-3 gc-num"><p class="font-medium" :data-tip="exactCredit(customer.credit_balance)">{{ money(customer.credit_balance) }}</p><p class="mt-1 text-xs surface-muted" :data-tip="exactCredit(customer.frozen_credit)">{{ tr('冻结', 'Reserved') }} {{ money(customer.frozen_credit) }}</p></td>
                  <td class="px-4 py-3 gc-num"><p :data-tip="exactCredit(customer.charged)">{{ money(customer.charged) }}</p><p class="mt-1 text-xs surface-muted" :data-tip="exactCredit(customer.cost)">{{ tr('成本', 'Cost') }} {{ money(customer.cost) }}</p></td>
                  <td class="max-w-48 break-words px-4 py-3 text-xs surface-muted">{{ customer.notes || '—' }}</td>
                  <td class="px-4 py-3"><div class="flex gap-1 whitespace-nowrap"><button class="table-action" :data-tip="tr('客户付款后给他加额度，或减少额度', 'Add credits after payment, or deduct')" @click="creditCustomer = customer">{{ tr('额度', 'Credits') }}</button><button class="table-action row-secondary" :data-tip="tr('改资料、密码、权限和 Key', 'Edit profile, password, access and keys')" @click="openCustomer(customer)">{{ tr('管理', 'Manage') }}</button><button class="table-action row-secondary" :data-tip="tr('只给这个客户单独定价', 'Set prices for this customer only')" @click="openPricing(customer)">{{ tr('定价', 'Pricing') }}</button></div></td>
                </tr></tbody>
              </table>
            </div>
            <Pagination v-if="customers && customers.total > customers.page_size" embedded class="hub-pagination" :total="customers.total" :page="customerPage" :page-size="customers.page_size" :show-page-size-selector="false" @update:page="customerPage = $event; loadCustomers()" />
          </section>

          <section v-else-if="tab === 'pricing'" class="hub-pricing">
            <div class="card hub-panel">
              <header class="hub-panel-head">
                <div class="min-w-0"><h2 class="hub-panel-title">{{ tr('客户定价', 'Customer pricing') }}</h2><p class="hub-panel-lead">{{ tr('客户价 = 你的分组价 × 倍率。可以按分组单独设，也可以在客户管理里给单个客户另定。', 'Customer price = your group price × multiplier. Set it per group here, or per customer from Customers.') }}</p></div>
                <button class="btn btn-primary" @click="openPricing(null)">{{ tr('编辑默认定价', 'Edit default pricing') }}</button>
              </header>
              <div class="hub-price-body">
                <div class="hub-price-rate" :data-tip="tr('没有单独设置的分组和客户都按这个倍率', 'Groups and customers without their own setting use this multiplier')">
                  <span class="hub-tile-label">{{ tr('默认倍率', 'Default multiplier') }}</span>
                  <strong class="hub-price-value gc-num">{{ defaultMultiplier }}×</strong>
                </div>
                <div class="hub-price-example">
                  <p class="text-sm font-medium">{{ tr('举个例子', 'For example') }}</p>
                  <p class="mt-1 text-sm surface-muted">{{ tr(`一条请求按你的分组价要 ${withCreditUnit('1.00')}，倍率 ${defaultMultiplier}× 时，客户的额度扣 ${withCreditUnit(defaultMultiplier.toFixed(2))}，平台从你的余额扣 ${withCreditUnit('1.00')}。`, `A request that costs ${withCreditUnit('1.00')} at your group price takes ${withCreditUnit(defaultMultiplier.toFixed(2))} of customer credits at ${defaultMultiplier}×, and the platform charges you ${withCreditUnit('1.00')}.`) }}</p>
                  <ul class="mt-3 space-y-1 text-xs surface-muted">
                    <li>{{ tr('1× 不加价；改价只影响之后的请求。', '1× adds no markup; new prices apply to later requests.') }}</li>
                    <li>{{ tr('已创建的异步任务保留原价格；失败释放的任务不记差价。', 'Existing async tasks keep their quoted price; released failed tasks earn no margin.') }}</li>
                    <li>{{ tr('赠送给客户的额度，客户用掉时成本也由你承担。', 'You also pay the cost when customers spend gifted credits.') }}</li>
                  </ul>
                </div>
              </div>
            </div>
            <form class="card hub-panel" @submit.prevent="saveInitialCredit">
              <header class="hub-panel-head">
                <div class="min-w-0"><h2 class="hub-panel-title">{{ tr('新客户赠送', 'New-customer gift') }}</h2><p class="hub-panel-lead">{{ tr('通过邀请链接注册或由你创建的新客户，开户时自动送这些额度；只影响之后的新客户。', 'Credits given automatically to new customers who sign up through your link or are created by you. Only affects future customers.') }}</p></div>
              </header>
              <div class="flex flex-wrap items-end gap-3 px-5 pb-5">
                <label class="block text-xs font-medium surface-muted">{{ tr('初始额度', 'Initial credits') }}<input v-model.number="initialCredit" type="number" min="0" step="any" required :aria-label="tr('新客户初始额度', 'New customer initial credits')" class="input mt-1 w-40" /></label>
                <button type="submit" class="btn btn-secondary" :disabled="initialSaving || initialCredit === overview?.profile.initial_credit">{{ initialSaving ? tr('保存中…', 'Saving…') : tr('保存', 'Save') }}</button>
                <span class="text-xs surface-muted">{{ tr('赠送的额度客户用掉时，成本也由你承担。', 'You pay the cost when customers spend gifted credits.') }}</span>
              </div>
            </form>
          </section>

          <section v-else-if="tab === 'earnings'" class="card hub-panel">
            <header class="hub-panel-head">
              <div class="min-w-0"><h2 class="hub-panel-title">{{ tr('消费与成本记录', 'Usage and service costs') }}</h2><p class="hub-panel-lead">{{ tr('客户用掉的额度和平台成本分别记录；线下实际收了多少钱由你自己核对。', 'Customer credits and platform costs are recorded separately. Reconcile your own offline payments.') }}</p></div>
              <button class="btn btn-secondary btn-sm" @click="loadEarnings"><Icon name="refresh" size="sm" />{{ tr('刷新', 'Refresh') }}</button>
            </header>
            <div v-if="!earnings?.items.length" class="hub-empty">
              <span class="hub-empty-icon"><Icon name="chart" size="lg" /></span>
              <p class="text-sm font-medium">{{ tr('暂无消费记录', 'No usage yet') }}</p>
              <p class="text-xs surface-muted">{{ tr('客户开始调用后，每一笔都会记在这里。', 'Each customer request will be recorded here.') }}</p>
            </div>
            <div v-else class="record-table-scroll">
              <table class="record-table w-full min-w-[720px] text-left text-sm">
                <thead><tr class="surface-muted">
                  <th class="px-4 py-2 font-medium" :data-tip="tr('请求时间和发起的客户', 'When and which customer')">{{ tr('时间 / 客户', 'Time / customer') }}</th>
                  <th class="px-4 py-2 font-medium" :data-tip="tr('这次请求走的分组和模型', 'Group and model used')">{{ tr('分组 / 模型', 'Group / model') }}</th>
                  <th class="px-4 py-2 font-medium" :data-tip="tr('按你的价格扣掉的客户额度', 'Customer credits charged at your price')">{{ tr('客户额度', 'Customer credits') }}</th>
                  <th class="px-4 py-2 font-medium" :data-tip="tr('平台按你的成本从你的余额扣的金额', 'What the platform charged your balance at cost')">{{ tr('平台成本', 'Platform cost') }}</th>
                  <th class="px-4 py-2 font-medium" :data-tip="tr('站长余额结算是现在的方式；历史直接扣费是旧版本留下的记录', 'Reseller funded is the current mode; historical direct charge comes from older versions')">{{ tr('结算方式', 'Settlement') }}</th>
                </tr></thead>
                <tbody><tr v-for="entry in earnings.items" :key="entry.id">
                  <td class="px-4 py-3"><p>{{ entry.username || `#${entry.customer_id}` }}</p><p class="mt-1 whitespace-nowrap text-xs surface-muted" :data-tip="timeTip(entry.created_at)">{{ date(entry.created_at) }}</p></td>
                  <td class="px-4 py-3"><p>{{ entry.group_name }}</p><p class="mt-1 text-xs surface-muted">{{ entry.model }}</p></td>
                  <td class="px-4 py-3 gc-num" :data-tip="exactCredit(entry.charged)">{{ money(entry.charged) }}</td>
                  <td class="px-4 py-3 gc-num" :data-tip="exactCredit(entry.cost)">{{ money(entry.cost) }}</td>
                  <td class="px-4 py-3 text-xs surface-muted">{{ entry.settlement_type === 'managed_credit' ? tr('站长余额结算', 'Reseller funded') : tr('历史直接扣费', 'Historical direct charge') }}</td>
                </tr></tbody>
              </table>
            </div>
            <Pagination v-if="earnings && earnings.total > earnings.page_size" embedded class="hub-pagination" :total="earnings.total" :page="earningsPage" :page-size="earnings.page_size" :show-page-size-selector="false" @update:page="earningsPage = $event; loadEarnings()" />
          </section>

          <CustomerContactSettings v-else-if="tab === 'contact' && overview" :profile="overview.profile" @updated="updateProfile" />
          <ResellerAnnouncements v-else-if="tab === 'announcements' && overview" :profile="overview.profile" @updated="updateProfile" />
          <AffiliateView v-else-if="tab === 'affiliate'" embedded @changed="loadAffiliate" />
        </div>
      </template>
    </div>

    <BaseDialog :show="pricingOpen" :title="priceCustomer ? `${tr('客户定价', 'Customer pricing')} · ${priceCustomer.username || priceCustomer.email}` : tr('默认客户定价', 'Default pricing')" width="wide" @close="pricingOpen = false">
      <form id="reseller-prices" class="space-y-5" @submit.prevent="savePricing">
        <p class="text-sm leading-6 surface-muted">{{ tr('倍率相对于你的分组价格。留空的分组继承整体倍率；客户整体倍率留空时，继承你的默认客户定价。', 'Multipliers are relative to your group prices. Empty group rows inherit the overall multiplier. An empty customer overall multiplier inherits your defaults.') }}</p>
        <label class="block text-sm font-medium">{{ tr('整体倍率', 'Overall multiplier') }}<input v-model="overallInput" type="number" :min="defaults.minimum_multiplier" step="any" :required="!priceCustomer" class="input mt-2 w-48" :placeholder="priceCustomer ? tr('继承默认定价', 'Inherit defaults') : ''" /></label>
        <div class="record-table-scroll"><table class="record-table w-full text-sm"><thead><tr class="text-left surface-muted"><th class="px-4 py-2 font-medium">{{ tr('分组', 'Group') }}</th><th class="px-4 py-2 font-medium" :data-tip="tr('你自己用这个分组时的倍率', 'Your own rate on this group')">{{ tr('你的基础倍率', 'Your base rate') }}</th><th class="px-4 py-2 font-medium" :data-tip="tr('在你的价格上再乘的倍数', 'Multiplier on top of your price')">{{ tr('客户加价倍率', 'Customer multiplier') }}</th><th class="px-4 py-2 font-medium" :data-tip="tr('客户最终按标准价的几倍扣额度', 'Final rate customers pay against standard prices')">{{ tr('客户基础倍率', 'Customer base rate') }}</th></tr></thead><tbody><tr v-for="group in pricing?.groups" :key="group.id"><td class="px-4 py-3"><p class="font-medium">{{ group.name }}</p><p class="text-xs surface-muted">{{ group.platform }}</p></td><td class="px-4 py-3 gc-num">{{ baseRate(group) }}×</td><td class="px-4 py-3"><input v-model="groupInputs[group.id]" type="number" :min="defaults.minimum_multiplier" step="any" class="input w-36" :aria-label="`${group.name} ${tr('倍率', 'multiplier')}`" :placeholder="`${tr('继承', 'Inherit')} ${inheritedMultiplier(group.id)}×`" /></td><td class="px-4 py-3 font-medium gc-num text-primary-600">{{ Number((baseRate(group) * effectiveMultiplier(group.id)).toFixed(8)) }}×</td></tr></tbody></table></div>
        <p class="text-xs leading-5 surface-muted">{{ tr('视频平台预览视频基础倍率，其他平台预览文本基础倍率。图片和视频的独立费率也应用客户加价；文本时段系数继续按原规则计算。', 'Video groups preview video rates; other groups preview text rates. Independent media rates receive the same markup. Text time-window factors retain their existing behavior.') }}</p>
        <p v-if="dialogError" role="alert" class="text-sm text-red-600">{{ dialogError }}</p>
      </form>
      <template #footer><button class="btn btn-secondary" @click="pricingOpen = false">{{ tr('取消', 'Cancel') }}</button><button type="submit" form="reseller-prices" class="btn btn-primary" :disabled="saving">{{ saving ? tr('保存中…', 'Saving…') : tr('保存定价', 'Save pricing') }}</button></template>
    </BaseDialog>

    <CustomerWorkspace v-if="overview" :show="workspaceOpen" :customer="workspaceCustomer" :groups="pricing?.groups ?? []" :limits="overview.limits" @close="workspaceOpen = false" @saved="refreshCustomerData" />
    <CustomerCreditDialog :customer="creditCustomer" @close="creditCustomer = null" @saved="refreshCustomerData" />
  </AppLayout>
</template>
<script setup lang="ts">
import { withCreditUnit } from '@/utils/credits'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import WorkspaceTabs from '@/components/common/WorkspaceTabs.vue'
import Icon from '@/components/icons/Icon.vue'
import CustomerWorkspace from '@/components/gotocc/reseller/CustomerWorkspace.vue'
import CustomerCreditDialog from '@/components/gotocc/reseller/CustomerCreditDialog.vue'
import CustomerContactSettings from '@/components/gotocc/reseller/CustomerContactSettings.vue'
import ResellerAnnouncements from '@/components/gotocc/reseller/ResellerAnnouncements.vue'
import HubSteps, { type HubStep } from '@/components/gotocc/reseller/HubSteps.vue'
import AffiliateView from './AffiliateView.vue'
import { resellerAPI, type ResellerOverview, type ResellerProfile, type ResellerPage, type ResellerCustomer, type ResellerEarning, type ResellerPrices, type ResellerGroup } from '@/api/reseller'
import userAPI from '@/api/user'
import type { AgentProfile, UserAffiliateDetail } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import { exactCredit, timeTip } from '@/utils/consoleTips'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useResellerAccess } from '@/composables/useResellerAccess'
import { useClipboard } from '@/composables/useClipboard'
import defaults from '../../../../reseller-defaults.json'
import consoleGuide from '../../../../console-guide.json'

// 站长中心：发展客户（客户站链接，客户向你付款、你加额度）与邀请返利（主站链接，对方自己充值、你拿返利）两部分。
// 有站长资格才显示发展客户的部分，开着邀请返利才显示返利的部分；两者都有时并排。选项卡记在地址栏 ?tab=。
const { locale } = useI18n(); const tr = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const app = useAppStore(); const auth = useAuthStore(); const route = useRoute(); const router = useRouter()
const resellerAccess = useResellerAccess(); const { copyToClipboard } = useClipboard()
const loading = ref(true); const saving = ref(false); const error = ref(''); const dialogError = ref('')
const initialCredit = ref(defaults.default_initial_credit); const initialSaving = ref(false)
const overview = ref<ResellerOverview>(); const customers = ref<ResellerPage<ResellerCustomer>>(); const earnings = ref<ResellerPage<ResellerEarning>>(); const pricing = ref<ResellerPrices>()
const affiliate = ref<UserAffiliateDetail>(); const agent = ref<AgentProfile>(); const affiliateError = ref(''); const applyingAgent = ref(false)
const search = ref(''); const customerPage = ref(1); const earningsPage = ref(1)
const pricingOpen = ref(false); const priceCustomer = ref<ResellerCustomer|null>(null); const overallInput = ref(''); const groupInputs = ref<Record<number,string>>({})
const workspaceOpen = ref(false); const workspaceCustomer = ref<ResellerCustomer|null>(null); const creditCustomer = ref<ResellerCustomer|null>(null)

const resellerOn = computed(() => resellerAccess.enabled.value)
const affiliateOn = computed(() => isFeatureFlagEnabled(FeatureFlags.affiliate))

type HubTab = 'customers' | 'pricing' | 'earnings' | 'contact' | 'announcements' | 'affiliate'
const tabs = computed(() => [
  ...(resellerOn.value ? [
    { value: 'customers' as HubTab, label: tr('客户管理', 'Customers'), tip: tr('给客户加额度、管理账户和 Key', 'Credits, accounts and keys of your customers') },
    { value: 'pricing' as HubTab, label: tr('定价与赠送', 'Pricing & gifts'), tip: tr('客户价格倍率和新客户赠送额度', 'Customer multipliers and the new-customer gift') },
    { value: 'earnings' as HubTab, label: tr('消费与成本', 'Usage & costs'), tip: tr('每一笔客户消费和你的平台成本', 'Each customer charge and your platform cost') },
    { value: 'contact' as HubTab, label: tr('客户展示', 'Customer display'), tip: tr('客户在菜单里看到的你的联系方式', 'Your contact shown in customers’ menu') },
    { value: 'announcements' as HubTab, label: tr('公告管理', 'Announcements'), tip: tr('给你的客户发公告、审核主站公告', 'Publish to your customers and review main-site notices') },
  ] : []),
  ...(affiliateOn.value ? [{ value: 'affiliate' as HubTab, label: tr('邀请返利', 'Referral rebates'), tip: tr('邀请记录、返利比例和转入余额', 'Invitees, rebate rates and balance transfer') }] : []),
])
const tab = computed<HubTab>({
  get: () => tabs.value.find(item => item.value === route.query.tab)?.value ?? tabs.value[0]!.value,
  set: value => { void router.replace({ path: route.path, query: { tab: value } }) },
})

const money = (v: number) => withCreditUnit(v.toFixed(2))
const date = (v: string) => new Date(v).toLocaleString(locale.value)
const baseRate = (g: ResellerGroup) => g.platform === 'video' && g.video_independent ? g.video_multiplier : g.base_multiplier
const inviteLink = computed(() => overview.value?.invitation_url ?? '')
const customerSite = computed(() => inviteLink.value ? new URL(inviteLink.value).host : '')
const affiliateLink = computed(() => affiliate.value ? `${window.location.origin}/register?invitation_code=${encodeURIComponent(affiliate.value.aff_code)}` : '')
const rebateRates = computed(() => affiliate.value?.rebate_rates_percent.map(rate => `${Number(rate.toFixed(2))}%`).join(' / ') ?? '')
const defaultMultiplier = computed(() => pricing.value?.default_multiplier ?? overview.value?.profile.default_multiplier ?? defaults.default_multiplier)
const balanceEmpty = computed(() => !!overview.value && overview.value.summary.customer_count > 0 && overview.value.limits.balance <= 0)

// 「复制过链接」只记在本浏览器，用来给最短路径打勾。
const storageKey = (kind: string) => `${consoleGuide.reseller_guide.storage_prefix}${kind}_${auth.user!.id}`
const copied = reactive({ customer: false, affiliate: false })
function readCopied() { copied.customer = localStorage.getItem(storageKey('customer_link')) === '1'; copied.affiliate = localStorage.getItem(storageKey('affiliate_link')) === '1' }
function markCopied(kind: 'customer' | 'affiliate') { localStorage.setItem(storageKey(`${kind}_link`), '1'); copied[kind] = true }

const pricingChanged = computed(() => !!pricing.value && (pricing.value.default_multiplier !== defaults.default_multiplier || pricing.value.prices.some(p => p.customer_id === null)))
const hasCustomers = computed(() => (overview.value?.summary.customer_count ?? 0) > 0)
const isAgent = computed(() => agent.value?.status === 'approved')

const affiliateSteps = computed<HubStep[]>(() => [
  { key: 'agent', done: isAgent.value, title: tr('开通代理', 'Become an agent'), body: tr('一键开通，不用审核；开通后的充值才计返利。', 'One click, no review; rebates count from then on.'), action: isAgent.value ? undefined : { label: applyingAgent.value ? tr('开通中…', 'Enabling…') : tr('一键开通', 'Enable'), run: applyAgent, primary: true } },
  { key: 'link', done: copied.affiliate, title: tr('复制主站邀请链接发给朋友', 'Copy the main-site link to friends'), body: tr('朋友在主站注册并充值后，返利记到你这里。', 'Rebates arrive once friends sign up and top up on the main site.'), action: { label: tr('复制链接', 'Copy link'), run: copyAffiliateLink } },
])
const customerSteps = computed<HubStep[]>(() => [
  { key: 'pricing', done: pricingChanged.value, title: tr('定价', 'Set your price'), body: tr(`客户价 = 你的分组价 × 倍率，现在是 ${defaultMultiplier.value}×。`, `Customer price = your group price × multiplier, now ${defaultMultiplier.value}×.`), action: { label: tr('去定价', 'Set pricing'), run: () => { tab.value = 'pricing' } } },
  { key: 'link', done: copied.customer, title: tr('复制客户站链接发给客户', 'Copy the customer-site link to customers'), body: tr('客户用这条链接注册，自动成为你的客户。', 'Customers who sign up with it become yours.'), action: { label: tr('复制链接', 'Copy link'), run: copyInvite } },
  { key: 'credit', done: hasCustomers.value, title: tr('客户付款后给他加额度', 'Add credits after payment'), body: tr('在「客户管理」里点「额度」，或直接替客户创建账户。', 'Use Credits in Customers, or create the account yourself.'), action: { label: tr('创建客户', 'Create customer'), run: () => openCustomer(null), primary: true } },
])
// 各自的步骤都做完后收成一行，可以再展开。
const guideOpen = reactive({ affiliate: true, customers: true })
watch(() => affiliateSteps.value.every(step => step.done), done => { guideOpen.affiliate = !done }, { immediate: true })
watch(() => customerSteps.value.every(step => step.done), done => { guideOpen.customers = !done }, { immediate: true })

const affiliateTiles = computed(() => {
  const detail = affiliate.value!
  return [
    { key: 'available', label: tr('可转返利', 'Transferable rebate'), value: money(detail.aff_quota), tone: detail.aff_quota > 0 ? 'hub-tone-good' : '', hint: tr('可以转入余额', 'Ready to move to balance'), tip: `${tr('已过冻结期、还没转入余额的返利', 'Rebates past the freeze period and not yet moved to your balance')}\n${exactCredit(detail.aff_quota)}` },
    { key: 'total', label: tr('累计返利', 'Total rebates'), value: money(detail.aff_history_quota), tone: '', hint: detail.aff_frozen_quota > 0 ? tr(`冻结中 ${money(detail.aff_frozen_quota)}`, `${money(detail.aff_frozen_quota)} frozen`) : tr('开通以来的全部返利', 'All rebates since you started'), tip: `${tr('开通以来得到的全部返利，含已转入和冻结中的', 'All rebates earned, including transferred and frozen')}\n${exactCredit(detail.aff_history_quota)}` },
    { key: 'invited', label: tr('邀请人数', 'Invitees'), value: detail.aff_count.toLocaleString(), tone: '', hint: tr('通过你的主站链接注册', 'Signed up via your main-site link'), tip: tr('用你的邀请码在主站注册的人数', 'People who signed up on the main site with your code') },
  ]
})
const customerTiles = computed(() => {
  const view = overview.value!
  const summary = view.summary
  const margin = summary.month_charged - summary.month_cost
  return [
    { key: 'balance', label: tr('你的平台余额', 'Your platform balance'), value: money(view.limits.balance), tone: balanceEmpty.value ? 'hub-tone-bad' : '', hint: tr('客户的请求按你的成本从这里扣', 'Customer requests are charged here at your cost'), tip: `${tr('承担你自己和所有客户的实际服务成本；用完后客户的请求会失败', 'Pays the real cost of you and all customers; customer requests fail when it runs out')}\n${exactCredit(view.limits.balance)}` },
    { key: 'customers', label: tr('客户数', 'Customers'), value: summary.customer_count.toLocaleString(), tone: '', hint: tr(`本月新增 ${summary.month_new_customers}`, `${summary.month_new_customers} new this month`), tip: tr('归你管理的客户；本月新增按客户加入的时间算', 'Customers you manage; new this month counts by join date') },
    { key: 'credits', label: tr('客户可用额度', 'Customer credits left'), value: money(summary.credit_balance), tone: '', hint: tr('发额度不扣你的余额', 'Issuing credits doesn’t debit you'), tip: `${tr('所有客户还能用的额度合计；客户用掉时才按成本扣你的余额', 'Credits all customers can still spend; you are charged at cost only when they use them')}\n${exactCredit(summary.credit_balance)}` },
    { key: 'charged', label: tr('本月客户消费', 'Customer spend'), value: money(summary.month_charged), tone: '', hint: tr('按你的价格扣的客户额度', 'Credits charged at your price'), tip: `${tr('本月 1 日 0 点起，客户按你的价格用掉的额度', 'Credits customers used at your price since the 1st')}\n${exactCredit(summary.month_charged)}` },
    { key: 'cost', label: tr('本月平台成本', 'Platform cost'), value: money(summary.month_cost), tone: '', hint: tr('平台从你的余额扣的', 'Charged to your balance'), tip: `${tr('本月 1 日 0 点起，平台按你的成本从你的余额扣的金额', 'What the platform charged your balance at cost since the 1st')}\n${exactCredit(summary.month_cost)}` },
    { key: 'margin', label: tr('本月额度差价', 'Credit margin'), value: money(margin), tone: margin > 0 ? 'hub-tone-good' : '', hint: tr('客户消费 − 平台成本', 'Customer spend − platform cost'), tip: `${tr('只是客户额度和平台成本的差；线下实际收了多少钱由你自己核对，不等于利润', 'Only the gap between customer credits and platform cost. Reconcile your actual offline payments yourself; this is not profit')}\n${exactCredit(margin)}` },
  ]
})

function openCustomer(customer: ResellerCustomer|null) { workspaceCustomer.value = customer; workspaceOpen.value = true }
async function refreshCustomerData() { await Promise.all([loadCustomers(), loadEarnings()]); overview.value = await resellerAPI.overview() }
function updateProfile(profile: ResellerProfile) { if (overview.value) overview.value.profile = profile }

async function loadReseller() { overview.value = await resellerAPI.overview(); initialCredit.value = overview.value.profile.initial_credit; await Promise.all([loadCustomers(), loadEarnings(), loadPrices()]) }
async function loadAffiliate() {
  affiliateError.value = ''
  try { [affiliate.value, agent.value] = await Promise.all([userAPI.getAffiliateDetail(), userAPI.getAgentProfile()]) }
  catch (e) { affiliateError.value = extractApiErrorMessage(e, tr('读取邀请返利失败', 'Could not load referral rebates')) }
}
async function load() {
  loading.value = true; error.value = ''
  try {
    readCopied()
    await resellerAccess.load(true)
    await Promise.all([resellerOn.value ? loadReseller() : undefined, affiliateOn.value ? loadAffiliate() : undefined])
  } catch (e) { error.value = extractApiErrorMessage(e, tr('读取站长中心失败', 'Could not load reseller center')) }
  finally { loading.value = false }
}
async function loadCustomers() { try { customers.value = await resellerAPI.customers(customerPage.value, search.value) } catch (e) { error.value = extractApiErrorMessage(e, tr('读取客户失败', 'Could not load customers')) } }
async function loadEarnings() { try { earnings.value = await resellerAPI.earnings(earningsPage.value) } catch (e) { error.value = extractApiErrorMessage(e, tr('读取消费记录失败', 'Could not load usage')) } }
async function loadPrices() { pricing.value = await resellerAPI.prices() }
async function saveInitialCredit() {
  initialSaving.value = true
  try {
    const profile = await resellerAPI.saveInitialCredit(initialCredit.value)
    if (overview.value) overview.value.profile = profile
    initialCredit.value = profile.initial_credit
    app.showSuccess(tr('初始额度已保存', 'Initial credits saved'))
  } catch (e) { app.showError(extractApiErrorMessage(e, tr('保存初始额度失败', 'Could not save initial credits'))) }
  finally { initialSaving.value = false }
}
async function copyInvite() { if (await copyToClipboard(inviteLink.value, tr('客户站邀请链接已复制', 'Customer-site link copied'))) markCopied('customer') }
async function copyAffiliateLink() { if (await copyToClipboard(affiliateLink.value, tr('主站邀请链接已复制', 'Main-site link copied'))) markCopied('affiliate') }
async function applyAgent() {
  applyingAgent.value = true
  try { agent.value = await userAPI.applyForAgent(); await loadAffiliate(); app.showSuccess(tr('已开通代理，之后的充值开始计返利', 'You are now an agent; later top-ups earn rebates')) }
  catch (e) { app.showError(extractApiErrorMessage(e, tr('开通代理失败', 'Could not enable agent'))) }
  finally { applyingAgent.value = false }
}
async function openPricing(customer: ResellerCustomer|null) {
  error.value = ''; dialogError.value = ''
  try { await loadPrices(); priceCustomer.value = customer; const rows = pricing.value!.prices.filter(p => p.customer_id === (customer?.user_id ?? null)); overallInput.value = customer ? String(rows.find(p => p.group_id === null)?.multiplier ?? '') : String(pricing.value!.default_multiplier); groupInputs.value = Object.fromEntries(pricing.value!.groups.map(g => [g.id, String(rows.find(p => p.group_id === g.id)?.multiplier ?? '')])); pricingOpen.value = true } catch (e) { error.value = extractApiErrorMessage(e, tr('读取定价失败', 'Could not load pricing')) }
}
function inheritedMultiplier(groupID: number) { if (overallInput.value !== '') return Number(overallInput.value); return pricing.value?.prices.find(p => p.customer_id === null && p.group_id === groupID)?.multiplier ?? pricing.value?.default_multiplier ?? defaults.default_multiplier }
function effectiveMultiplier(groupID: number) { return groupInputs.value[groupID] === '' ? inheritedMultiplier(groupID) : Number(groupInputs.value[groupID]) }
async function savePricing() {
  saving.value = true; dialogError.value = ''
  try { await resellerAPI.savePrices(priceCustomer.value?.user_id ?? null, overallInput.value === '' ? null : Number(overallInput.value), (pricing.value?.groups ?? []).map(g => ({ customer_id: priceCustomer.value?.user_id ?? null, group_id: g.id, multiplier: groupInputs.value[g.id] === '' ? null : Number(groupInputs.value[g.id]) }))); pricingOpen.value = false; await loadPrices(); overview.value = await resellerAPI.overview(); app.showSuccess(tr('客户定价已保存，后续请求使用新价格', 'Pricing saved for subsequent requests')) } catch (e) { dialogError.value = extractApiErrorMessage(e, tr('保存定价失败', 'Could not save pricing')) } finally { saving.value = false }
}
watch(tab, value => { if (value === 'earnings' && resellerOn.value) void loadEarnings() })
onMounted(load)
</script>

<style scoped>
.hub { display: flex; flex-direction: column; gap: 16px; min-width: 0; }
.hub-lead { font-size: 13px; line-height: 1.7; color: var(--surface-muted); padding-inline: 2px; }
.hub-ways { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; align-items: start; }
.hub-ways-single { grid-template-columns: minmax(0, 1fr); }
.hub-way { position: relative; display: flex; flex-direction: column; gap: 14px; padding: 18px 20px; overflow: hidden; border-radius: var(--surface-radius); }
.hub-way::before { content: ''; position: absolute; inset: 0; pointer-events: none; border-radius: inherit; background: radial-gradient(22rem 9rem at 0% 0%, var(--hub-way-tint), transparent 70%); }
.hub-way > * { position: relative; }
.hub-way:has(.hub-way-icon-rebate) { --hub-way-tint: rgb(16 185 129 / 12%); }
.hub-way:has(.hub-way-icon-customer) { --hub-way-tint: rgb(99 102 241 / 13%); }
.hub-way-head { display: flex; align-items: flex-start; gap: 12px; }
.hub-way-icon { display: inline-flex; flex: none; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: var(--surface-radius-control); }
.hub-way-icon-rebate { background: rgb(16 185 129 / 14%); color: theme('colors.emerald.600'); }
.hub-way-icon-customer { background: rgb(99 102 241 / 14%); color: theme('colors.primary.600'); }
.dark .hub-way-icon-rebate { color: theme('colors.emerald.300'); }
.dark .hub-way-icon-customer { color: theme('colors.primary.300'); }
.hub-way-title { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; font-size: 16px; font-weight: 700; color: var(--surface-text); }
.hub-way-tag { border: 1px solid var(--surface-edge); border-radius: var(--surface-radius-tag); padding: 1px 7px; font-size: 11px; font-weight: 500; color: var(--surface-muted); }
.hub-way-lead { margin-top: 4px; font-size: 13px; line-height: 1.65; color: var(--surface-muted); }
.hub-link { display: flex; flex-direction: column; gap: 6px; }
.hub-link-label { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; font-size: 12px; font-weight: 600; color: var(--surface-text); }
.hub-link-site { border-radius: var(--surface-radius-tag); padding: 1px 7px; font-size: 11px; font-weight: 500; color: var(--console-accent-readable); background: var(--surface-soft); }
.hub-link-box { display: flex; align-items: center; gap: 8px; border: 1px solid var(--surface-edge); border-radius: var(--surface-radius-control); padding: 6px 6px 6px 12px; background: var(--surface-soft); }
.hub-link-code { min-width: 0; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; color: var(--surface-text); }
.hub-way-more { display: inline-flex; align-self: flex-start; align-items: center; gap: 4px; font-size: 12px; font-weight: 600; color: var(--console-accent-readable); }
.hub-way-more:hover { text-decoration: underline; }

.hub-overview { display: flex; flex-direction: column; gap: 14px; }
.hub-group { display: flex; flex-direction: column; gap: 8px; }
.hub-group-title { padding-inline: 2px; font-size: 12px; font-weight: 600; letter-spacing: .04em; color: var(--surface-muted); }
.hub-tiles { display: grid; gap: 12px; }
.hub-tiles-3 { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.hub-tiles-6 { grid-template-columns: repeat(6, minmax(0, 1fr)); }
.hub-tile { display: flex; min-width: 0; flex-direction: column; gap: 4px; padding: 14px 16px; border-radius: var(--surface-radius-card); }
.hub-tile-label { font-size: 12px; font-weight: 500; color: var(--surface-muted); }
.hub-tile-value { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 21px; line-height: 1.3; font-weight: 700; letter-spacing: -.01em; color: var(--surface-text); }
.hub-tile-hint { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 11px; color: var(--surface-muted); }
.hub-tone-good { color: theme('colors.emerald.600'); }
.hub-tone-bad { color: theme('colors.red.600'); }
.dark .hub-tone-good { color: theme('colors.emerald.400'); }
.dark .hub-tone-bad { color: theme('colors.red.400'); }
.hub-tile-danger { border-color: rgb(239 68 68 / 45%) !important; }
.hub-alert { display: flex; flex-wrap: wrap; align-items: center; gap: 8px 12px; border: 1px solid rgb(239 68 68 / 35%); border-radius: var(--surface-radius-card); padding: 10px 12px 10px 14px; font-size: 13px; color: theme('colors.red.700'); background: linear-gradient(90deg, rgb(239 68 68 / 12%), rgb(239 68 68 / 4%)), var(--surface-solid); }
.dark .hub-alert { color: theme('colors.red.300'); }

.hub-tabs { margin-top: 4px; }
.hub-panel { overflow: hidden; border-radius: var(--surface-radius); }
.hub-panel-head { display: flex; flex-wrap: wrap; align-items: flex-start; justify-content: space-between; gap: 12px 16px; padding: 18px 20px 12px; }
.hub-panel-title { font-size: 15px; font-weight: 600; color: var(--surface-text); }
.hub-panel-lead { margin-top: 3px; max-width: 40rem; font-size: 12px; line-height: 1.6; color: var(--surface-muted); }
.hub-pagination { padding: 8px 16px 14px; }
.hub-empty { display: flex; flex-direction: column; align-items: center; gap: 6px; padding: 36px 20px 44px; text-align: center; }
.hub-empty-icon { display: inline-flex; align-items: center; justify-content: center; width: 48px; height: 48px; margin-bottom: 6px; border-radius: var(--surface-radius-card); background: var(--surface-soft); color: var(--console-accent-readable); }
.hub-badge { border-radius: var(--surface-radius-tag); padding: 1px 7px; font-size: 11px; font-weight: 500; }
.hub-badge-ok { background: rgb(16 185 129 / 12%); color: theme('colors.emerald.700'); }
.hub-badge-off { background: var(--surface-soft); color: var(--surface-muted); }
.dark .hub-badge-ok { color: theme('colors.emerald.300'); }

.hub-pricing { display: flex; flex-direction: column; gap: 16px; }
.hub-price-body { display: grid; grid-template-columns: 14rem minmax(0, 1fr); gap: 16px; padding: 0 20px 20px; }
.hub-price-rate { display: flex; flex-direction: column; gap: 6px; border-radius: var(--surface-radius-card); padding: 16px; background: var(--surface-soft); }
.hub-price-value { font-size: 32px; line-height: 1.2; font-weight: 700; color: var(--surface-text); }
.hub-price-example { border: 1px dashed var(--surface-edge); border-radius: var(--surface-radius-card); padding: 14px 16px; }

@media (max-width: 1279px) {
  .hub-tiles-6 { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}
@media (max-width: 1023px) {
  .hub-ways { grid-template-columns: minmax(0, 1fr); }
}
@media (max-width: 640px) {
  .hub-tiles-3, .hub-tiles-6 { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .hub-tiles-3 > :last-child { grid-column: span 2; }
  .hub-price-body { grid-template-columns: minmax(0, 1fr); }
  .hub-way { padding: 16px; }
}
</style>
