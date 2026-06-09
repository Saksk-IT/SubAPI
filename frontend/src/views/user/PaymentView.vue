<template>
  <AppLayout>
    <div class="mx-auto w-full max-w-[1720px] space-y-6">
      <div v-if="loading" class="flex items-center justify-center py-20">
        <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent"></div>
      </div>
      <template v-else>
        <!-- Payment in progress (shared by recharge and subscription) -->
        <template v-if="paymentPhase === 'paying'">
          <div class="mx-auto max-w-3xl">
            <PaymentStatusPanel
              :order-id="paymentState.orderId"
              :qr-code="paymentState.qrCode"
              :expires-at="paymentState.expiresAt"
              :payment-type="paymentState.paymentType"
              :pay-url="paymentState.payUrl"
              :order-type="paymentState.orderType"
              :currency="paymentState.currency || selectedCurrency"
              @done="onPaymentDone"
              @success="onPaymentSuccess"
              @settled="onPaymentSettled"
            />
          </div>
        </template>
        <!-- Tab content (select phase) -->
        <div v-else class="space-y-6">
          <div
            v-if="!selectedPlan"
            class="grid gap-5 xl:grid-cols-[minmax(0,1fr)_380px]"
          >
            <section class="card p-5 sm:p-6">
              <p class="text-xs font-semibold uppercase text-primary-600 dark:text-primary-400">
                {{ t('payment.purchaseGuide.eyebrow') }}
              </p>
              <div class="mt-2 max-w-5xl">
                <h1 class="text-2xl font-black leading-tight text-gray-950 dark:text-white sm:text-3xl">
                  {{ t('payment.purchaseGuide.title') }}
                </h1>
                <p class="mt-3 text-sm leading-6 text-gray-600 dark:text-gray-300 sm:text-base sm:leading-7">
                  {{ t('payment.purchaseGuide.subtitle') }}
                </p>
              </div>
              <div class="mt-6 grid gap-4 lg:grid-cols-3">
                <section class="min-w-0 border-l-2 border-sky-400 pl-4">
                  <h2 class="text-base font-bold text-gray-950 dark:text-white">
                    {{ t('payment.purchaseGuide.apiRechargeTitle') }}
                  </h2>
                  <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
                    {{ t('payment.purchaseGuide.apiRechargeDescription') }}
                  </p>
                </section>
                <section class="min-w-0 border-l-2 border-emerald-400 pl-4">
                  <h2 class="text-base font-bold text-gray-950 dark:text-white">
                    {{ t('payment.purchaseGuide.rechargeTitle') }}
                  </h2>
                  <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
                    {{ t('payment.purchaseGuide.rechargeDescription') }}
                  </p>
                </section>
                <section class="min-w-0 border-l-2 border-violet-400 pl-4">
                  <h2 class="text-base font-bold text-gray-950 dark:text-white">
                    {{ t('payment.purchaseGuide.subscriptionTitle') }}
                  </h2>
                  <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
                    {{ t('payment.purchaseGuide.subscriptionDescription') }}
                  </p>
                </section>
              </div>
              <div class="mt-6 border-t border-gray-100 pt-5 dark:border-dark-700">
                <h2 class="text-base font-bold text-gray-950 dark:text-white">
                  {{ t('payment.purchaseGuide.billingTitle') }}
                </h2>
                <div class="mt-3 grid gap-3 text-sm leading-6 text-gray-500 dark:text-gray-400 md:grid-cols-2">
                  <p>{{ t('payment.purchaseGuide.billingRecharge') }}</p>
                  <p>{{ t('payment.purchaseGuide.billingSubscription') }}</p>
                </div>
              </div>
            </section>

            <aside class="card p-5 sm:p-6">
              <p class="text-xs font-semibold uppercase text-primary-600 dark:text-primary-400">
                {{ t('payment.purchaseGuide.contactEyebrow') }}
              </p>
              <h2 class="mt-2 text-xl font-black leading-tight text-gray-950 dark:text-white">
                {{ t('payment.purchaseGuide.contactTitle') }}
              </h2>
              <p v-if="supportContactSubtitle" class="mt-3 text-sm leading-6 text-gray-600 dark:text-gray-300">
                {{ supportContactSubtitle }}
              </p>
              <div class="mt-5 space-y-4">
                <button
                  v-if="supportImageUrl"
                  type="button"
                  class="w-full rounded-xl border border-gray-100 bg-gray-50 p-3 transition hover:border-gray-200 hover:bg-white dark:border-dark-700 dark:bg-dark-900/60 dark:hover:bg-dark-900"
                  @click="previewImage = supportImageUrl"
                >
                  <img
                    :src="supportImageUrl"
                    :alt="t('payment.purchaseGuide.contactQr')"
                    class="mx-auto h-44 max-w-full object-contain"
                  />
                  <span class="mt-2 block text-xs font-medium text-gray-500 dark:text-gray-400">
                    {{ t('payment.purchaseGuide.contactQr') }}
                  </span>
                </button>
                <div v-if="supportContactInfo">
                  <p class="text-xs font-semibold text-gray-400 dark:text-gray-500">
                    {{ t('payment.purchaseGuide.contactInfo') }}
                  </p>
                  <p class="mt-2 whitespace-pre-line break-words text-sm font-medium leading-6 text-gray-800 [overflow-wrap:anywhere] dark:text-gray-100">
                    {{ supportContactInfo }}
                  </p>
                </div>
                <div v-if="supportHelpText">
                  <p class="text-xs font-semibold text-gray-400 dark:text-gray-500">
                    {{ t('payment.purchaseGuide.helpTitle') }}
                  </p>
                  <p class="mt-2 whitespace-pre-line break-words text-sm leading-6 text-gray-600 [overflow-wrap:anywhere] dark:text-gray-300">
                    {{ supportHelpText }}
                  </p>
                </div>
                <p
                  v-if="!supportImageUrl && !supportContactInfo && !supportHelpText"
                  class="rounded-xl bg-gray-50 px-4 py-3 text-sm leading-6 text-gray-500 dark:bg-dark-900/60 dark:text-gray-400"
                >
                  {{ t('payment.purchaseGuide.contactEmpty') }}
                </p>
              </div>
            </aside>
          </div>

          <section
            class="min-w-0 space-y-6"
          >
            <!-- Tab Switcher (hide during subscription confirm) -->
            <div
              v-if="tabs.length > 1 && !selectedPlan"
              class="grid grid-cols-2 gap-1.5 rounded-2xl border border-primary-200 bg-white p-1.5 shadow-lg shadow-primary-500/10 ring-1 ring-primary-100/80 dark:border-primary-500/30 dark:bg-dark-900/80 dark:shadow-primary-900/20 dark:ring-primary-500/20 sm:gap-2"
              role="tablist"
              :aria-label="t('nav.buySubscription')"
            >
              <button
                v-for="tab in tabs"
                :key="tab.key"
                type="button"
                role="tab"
                :aria-selected="activeTab === tab.key"
                class="relative min-h-12 rounded-xl px-4 py-3 text-sm font-black transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-offset-dark-900 sm:text-base"
                :class="activeTab === tab.key ? 'bg-gradient-to-r from-primary-500 to-sky-500 text-white shadow-md shadow-primary-500/30 ring-1 ring-white/30 dark:from-primary-500 dark:to-sky-400' : 'bg-white/70 text-gray-700 ring-1 ring-gray-200 hover:bg-primary-50 hover:text-primary-700 hover:ring-primary-200 dark:bg-dark-800/80 dark:text-gray-300 dark:ring-dark-600 dark:hover:bg-primary-500/10 dark:hover:text-primary-200 dark:hover:ring-primary-500/30'"
                @click="activeTab = tab.key"
              >
                {{ tab.label }}
              </button>
            </div>

            <!-- Top-up Tab -->
            <template v-if="activeTab === 'recharge'">
              <!-- Recharge Account Card -->
              <div class="card p-5">
                <p class="text-xs font-medium text-gray-400 dark:text-gray-500">{{ t('payment.rechargeAccount') }}</p>
                <p class="mt-1 text-base font-semibold text-gray-900 dark:text-white">{{ user?.username || '' }}</p>
                <p class="mt-0.5 text-sm font-medium text-green-600 dark:text-green-400">{{ t('payment.currentBalance') }}: {{ user?.balance?.toFixed(2) || '0.00' }}</p>
              </div>
              <section
                v-if="firstRechargeCards.length > 0"
                id="first-recharge"
                class="space-y-4 rounded-2xl border border-amber-200 bg-amber-50/70 p-4 dark:border-amber-500/25 dark:bg-amber-500/10 sm:p-5"
              >
                <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
                  <div>
                    <p class="text-xs font-black uppercase text-amber-600 dark:text-amber-300">
                      {{ t('firstRecharge.label') }}
                    </p>
                    <h2 class="mt-1 text-xl font-black text-gray-950 dark:text-white">
                      {{ t('payment.firstRecharge.title') }}
                    </h2>
                    <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-300">
                      {{ t('payment.firstRecharge.subtitle') }}
                    </p>
                  </div>
                  <span class="rounded-full bg-white px-3 py-1 text-xs font-black text-amber-700 shadow-sm ring-1 ring-amber-200 dark:bg-dark-900/70 dark:text-amber-200 dark:ring-amber-400/20">
                    {{ t('firstRecharge.noAffiliateRebate') }}
                  </span>
                </div>
                <div :class="productGridClass">
                  <PurchaseProductCard
                    v-for="item in firstRechargeCards"
                    :key="item.product.id"
                    :product="item.product"
                    :hero-metrics="item.heroMetrics"
                    :metrics="item.metrics"
                    :price-rows="item.priceRows"
                    :methods="item.methods"
                    :currency="paymentPriceCurrency"
                    :locale="localeCode"
                    :submitting="submittingProductKey === `first-recharge:${item.raw.id}`"
                    @pay="handleSubmitFirstRechargeOffer(item.raw, $event)"
                  />
                </div>
              </section>
              <div v-if="enabledMethods.length === 0" class="card py-16 text-center">
                <p class="text-gray-500 dark:text-gray-400">{{ t('payment.notAvailable') }}</p>
              </div>
              <template v-else>
                <div v-if="balanceProductCards.length > 0" :class="productGridClass">
                  <PurchaseProductCard
                    v-for="item in balanceProductCards"
                    :key="item.product.id"
                    :product="item.product"
                    :hero-metrics="item.heroMetrics"
                    :metrics="item.metrics"
                    :price-rows="item.priceRows"
                    :methods="item.methods"
                    :currency="paymentPriceCurrency"
                    :locale="localeCode"
                    :submitting="submittingProductKey === `balance:${item.product.id}`"
                    @pay="handleSubmitBalanceProduct(item.raw, $event)"
                  />
                </div>
                <div class="card p-5 sm:p-6">
                  <div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(280px,380px)]">
                    <div class="min-w-0 space-y-4">
                      <AmountInput
                        v-model="amount"
                        :amounts="[10, 20, 50, 100, 200, 500, 1000, 2000, 5000]"
                        :min="globalMinAmount"
                        :max="globalMaxAmount"
                        :input-prefix="amountInputPrefix"
                      />
                      <div
                        v-if="supportHelpText"
                        class="rounded-xl border border-gray-100 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/60"
                        data-testid="payment-recharge-help-text"
                      >
                        <p class="text-sm font-medium text-gray-700 dark:text-gray-300">
                          {{ t('payment.purchaseGuide.helpTitle') }}
                        </p>
                        <p class="mt-2 whitespace-pre-line break-words text-sm leading-6 text-gray-500 [overflow-wrap:anywhere] dark:text-gray-400">
                          {{ supportHelpText }}
                        </p>
                      </div>
                    </div>
                    <div class="min-w-0 space-y-4">
                      <PaymentMethodSelector
                        :methods="methodOptions"
                        :selected="selectedMethod"
                        @select="selectedMethod = $event"
                      />
                      <div v-if="validAmount > 0" class="space-y-2 border-t border-gray-200 pt-4 text-sm dark:border-dark-600">
                        <div class="flex justify-between gap-3">
                          <span class="text-gray-500 dark:text-gray-400">{{ t('payment.paymentAmount') }}</span>
                          <span class="break-words text-right text-gray-900 [overflow-wrap:anywhere] dark:text-white">{{ formatSelectedPaymentAmount(validAmount) }}</span>
                        </div>
                        <div v-if="feeRate > 0" class="flex justify-between gap-3">
                          <span class="text-gray-500 dark:text-gray-400">{{ t('payment.fee') }} ({{ feeRate }}%)</span>
                          <span class="break-words text-right text-gray-900 [overflow-wrap:anywhere] dark:text-white">{{ formatSelectedPaymentAmount(feeAmount) }}</span>
                        </div>
                        <div v-if="feeRate > 0" class="flex justify-between gap-3 border-t border-gray-200 pt-2 dark:border-dark-600">
                          <span class="font-medium text-gray-700 dark:text-gray-300">{{ t('payment.actualPay') }}</span>
                          <span class="break-words text-right text-lg font-bold text-primary-600 [overflow-wrap:anywhere] dark:text-primary-400">{{ formatSelectedPaymentAmount(totalAmount) }}</span>
                        </div>
                        <div v-if="balanceRechargeMultiplier !== 1" class="flex justify-between gap-3" :class="{ 'border-t border-gray-200 pt-2 dark:border-dark-600': feeRate <= 0 }">
                          <span class="text-gray-500 dark:text-gray-400">{{ t('payment.creditedBalance') }}</span>
                          <span class="break-words text-right text-gray-900 [overflow-wrap:anywhere] dark:text-white">{{ formatQuotaAmount(creditedAmount) }}</span>
                        </div>
                        <p v-if="balanceRechargeMultiplier !== 1" class="border-t border-gray-200 pt-2 text-xs text-gray-500 dark:border-dark-600 dark:text-gray-400">
                          {{ t('payment.rechargeRatePreview', { usd: balanceRechargeMultiplier.toFixed(2) }) }}
                        </p>
                        <p v-if="amountError" class="text-xs text-amber-600 dark:text-amber-300">{{ amountError }}</p>
                      </div>
                      <p v-else-if="amountError" class="text-xs text-amber-600 dark:text-amber-300">{{ amountError }}</p>
                      <button :class="['btn w-full py-3 text-base font-medium', paymentButtonClass]" :disabled="!canSubmit || submitting" @click="handleSubmitRecharge">
                        <span v-if="submitting && submittingProductKey === ''" class="flex items-center justify-center gap-2">
                          <span class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
                          {{ t('common.processing') }}
                        </span>
                        <span v-else>{{ t('payment.createOrder') }} {{ formatSelectedPaymentAmount(totalAmount) }}</span>
                      </button>
                    </div>
                  </div>
                </div>
              </template>
            </template>
            <!-- Subscribe Tab -->
            <template v-else-if="activeTab === 'subscription'">
              <!-- Subscription confirm (inline, replaces plan list) -->
              <template v-if="selectedPlan">
                <div class="card p-5">
                  <!-- Header: platform badge + plan name -->
                  <div class="mb-3 flex flex-wrap items-center gap-2">
                    <span :class="['rounded-md border px-2 py-0.5 text-xs font-medium', planBadgeClass]">
                      {{ formatPlatformLabel(selectedPlan.group_platform) }}
                    </span>
                    <h3 class="text-lg font-bold text-gray-900 dark:text-white">{{ selectedPlan.name }}</h3>
                  </div>
                  <!-- Price -->
                  <div class="flex flex-wrap items-baseline gap-2">
                    <span v-if="selectedPlan.original_price" class="text-sm text-gray-400 line-through dark:text-gray-500">
                      {{ formatSelectedPaymentAmount(selectedPlan.original_price) }}
                    </span>
                    <span :class="['min-w-0 break-words text-3xl font-bold [overflow-wrap:anywhere]', planTextClass]">{{ formatSelectedPaymentAmount(selectedPlan.price) }}</span>
                    <span class="text-sm text-gray-500 dark:text-gray-400">/ {{ planValiditySuffix }}</span>
                  </div>
                  <!-- Description -->
                  <p v-if="selectedPlan.description" class="mt-2 text-sm leading-relaxed text-gray-500 dark:text-gray-400">
                    {{ selectedPlan.description }}
                  </p>
                  <!-- Rate + Limits grid -->
                  <div class="mt-3 grid grid-cols-2 gap-3">
                    <div>
                      <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.rate') }}</span>
                      <div class="flex items-baseline">
                        <span :class="['text-lg font-bold', planTextClass]">×{{ selectedPlan.rate_multiplier ?? 1 }}</span>
                      </div>
                    </div>
                    <div v-if="selectedDailyLimit != null">
                      <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.dailyLimit') }}</span>
                      <div class="break-words text-lg font-semibold text-gray-800 [overflow-wrap:anywhere] dark:text-gray-200">${{ selectedDailyLimit }}</div>
                    </div>
                    <div v-if="selectedWeeklyLimit != null">
                      <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.weeklyLimit') }}</span>
                      <div class="break-words text-lg font-semibold text-gray-800 [overflow-wrap:anywhere] dark:text-gray-200">${{ selectedWeeklyLimit }}</div>
                    </div>
                    <div v-if="selectedMonthlyLimit != null">
                      <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.monthlyLimit') }}</span>
                      <div class="break-words text-lg font-semibold text-gray-800 [overflow-wrap:anywhere] dark:text-gray-200">${{ selectedMonthlyLimit }}</div>
                    </div>
                    <div v-if="selectedDailyLimit == null && selectedWeeklyLimit == null && selectedMonthlyLimit == null">
                      <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.quota') }}</span>
                      <div class="text-lg font-semibold text-gray-800 dark:text-gray-200">{{ t('payment.planCard.unlimited') }}</div>
                    </div>
                  </div>
                </div>
                <div v-if="enabledMethods.length >= 1" class="card p-6">
                  <PaymentMethodSelector
                    :methods="subMethodOptions"
                    :selected="selectedMethod"
                    @select="selectedMethod = $event"
                  />
                </div>
                <div v-if="feeRate > 0 && selectedPlan.price > 0" class="card p-6">
                  <div class="space-y-2 text-sm">
                    <div class="flex justify-between gap-3">
                      <span class="text-gray-500 dark:text-gray-400">{{ t('payment.amountLabel') }}</span>
                      <span class="break-words text-right text-gray-900 [overflow-wrap:anywhere] dark:text-white">{{ formatSelectedPaymentAmount(selectedPlan.price) }}</span>
                    </div>
                    <div class="flex justify-between gap-3">
                      <span class="text-gray-500 dark:text-gray-400">{{ t('payment.fee') }} ({{ feeRate }}%)</span>
                      <span class="break-words text-right text-gray-900 [overflow-wrap:anywhere] dark:text-white">{{ formatSelectedPaymentAmount(subFeeAmount) }}</span>
                    </div>
                    <div class="flex justify-between gap-3 border-t border-gray-200 pt-2 dark:border-dark-600">
                      <span class="font-medium text-gray-700 dark:text-gray-300">{{ t('payment.actualPay') }}</span>
                      <span class="break-words text-right text-lg font-bold text-primary-600 [overflow-wrap:anywhere] dark:text-primary-400">{{ formatSelectedPaymentAmount(subTotalAmount) }}</span>
                    </div>
                  </div>
                </div>
                <button :class="['btn w-full py-3 text-base font-medium', paymentButtonClass]" :disabled="!canSubmitSubscription || submitting" @click="confirmSubscribe">
                  <span v-if="submitting" class="flex items-center justify-center gap-2">
                    <span class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
                    {{ t('common.processing') }}
                  </span>
                  <span v-else>{{ t('payment.createOrder') }} {{ formatSelectedPaymentAmount(feeRate > 0 ? subTotalAmount : selectedPlan.price) }}</span>
                </button>
                <button class="btn btn-secondary w-full" @click="selectedPlan = null">{{ t('common.cancel') }}</button>
              </template>
              <!-- Plan list -->
              <template v-else>
                <div v-if="checkout.plans.length === 0" class="card py-16 text-center">
                  <Icon name="gift" size="xl" class="mx-auto mb-3 text-gray-300 dark:text-dark-600" />
                  <p class="text-gray-500 dark:text-gray-400">{{ t('payment.noPlans') }}</p>
                </div>
                <div v-else class="space-y-6">
                  <section
                    v-for="section in subscriptionProductSections"
                    :key="section.platform"
                    class="space-y-3"
                    data-testid="subscription-platform-section"
                  >
                    <div class="flex items-center justify-between gap-3">
                      <div class="min-w-0">
                        <p class="text-xs font-black uppercase text-gray-400 dark:text-gray-500">
                          {{ t('payment.product.platformCategory') }}
                        </p>
                        <h2 class="mt-1 flex min-w-0 items-center gap-2 text-xl font-black text-gray-950 dark:text-white">
                          <span :class="['h-2.5 w-2.5 shrink-0 rounded-full', platformAccentBarClass(section.platform)]"></span>
                          <span class="truncate">{{ section.label }}</span>
                        </h2>
                      </div>
                      <span class="shrink-0 rounded-full border border-gray-200 px-2.5 py-1 text-xs font-black text-gray-500 dark:border-dark-600 dark:text-gray-300">
                        {{ t('payment.product.platformPlanCount', { count: section.items.length }) }}
                      </span>
                    </div>
                    <div :class="productGridClass">
                      <PurchaseProductCard
                        v-for="item in section.items"
                        :key="item.product.id"
                        :product="item.product"
                        :hero-metrics="item.heroMetrics"
                        :metrics="item.metrics"
                        :price-rows="item.priceRows"
                        :methods="item.methods"
                        :currency="paymentPriceCurrency"
                        :locale="localeCode"
                        :price-suffix="item.priceSuffix"
                        :submitting="submittingProductKey === `subscription:${item.product.id}`"
                        @pay="handleSubmitSubscriptionPlan(item.raw, $event)"
                      />
                    </div>
                  </section>
                </div>
                <!-- Active subscriptions (compact, below plan list) -->
                <div v-if="activeSubscriptions.length > 0">
                  <p class="mb-2 text-xs font-medium text-gray-400 dark:text-gray-500">{{ t('payment.activeSubscription') }}</p>
                  <div class="space-y-2">
                    <div v-for="sub in activeSubscriptions" :key="sub.id"
                      class="flex items-center gap-3 rounded-xl border border-gray-100 bg-white px-3 py-2 dark:border-dark-700 dark:bg-dark-800">
                      <div :class="['h-6 w-1 shrink-0 rounded-full', platformAccentBarClass(sub.group?.platform || '')]" />
                      <div class="min-w-0 flex-1">
                        <div class="flex items-center gap-1.5">
                          <span class="truncate text-xs font-semibold text-gray-900 dark:text-white">{{ sub.group?.name || t('payment.groupFallback', { id: sub.group_id }) }}</span>
                          <span :class="['shrink-0 rounded-full px-1.5 py-0.5 text-[9px] font-medium', platformBadgeLightClass(sub.group?.platform || '')]">{{ formatPlatformLabel(sub.group?.platform) }}</span>
                        </div>
                        <div class="flex flex-wrap gap-x-3 text-[11px] text-gray-400 dark:text-gray-500">
                          <span>{{ t('payment.planCard.rate') }}: ×{{ sub.group?.rate_multiplier ?? 1 }}</span>
                          <span v-if="sub.group?.daily_limit_usd == null && sub.group?.weekly_limit_usd == null && sub.group?.monthly_limit_usd == null">{{ t('payment.planCard.quota') }}: {{ t('payment.planCard.unlimited') }}</span>
                          <span v-if="sub.expires_at">{{ t('userSubscriptions.daysRemaining', { days: getDaysRemaining(sub.expires_at) }) }}</span>
                          <span v-else>{{ t('userSubscriptions.noExpiration') }}</span>
                        </div>
                      </div>
                      <span class="badge badge-success shrink-0 text-[10px]">{{ t('userSubscriptions.status.active') }}</span>
                    </div>
                  </div>
                </div>
              </template>
            </template>
          </section>
        </div>
      </template>
    </div>
    <!-- Renewal Plan Selection Modal -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="showRenewalModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4" @click.self="closeRenewalModal">
          <div class="relative w-full max-w-lg rounded-2xl border border-gray-200 bg-white p-6 shadow-2xl dark:border-dark-700 dark:bg-dark-900">
            <!-- Close button -->
            <button class="absolute right-4 top-4 rounded-lg p-1 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700 dark:hover:text-gray-200" @click="closeRenewalModal">
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
            </button>
            <h3 class="mb-4 text-lg font-semibold text-gray-900 dark:text-white">{{ t('payment.selectPlan') }}</h3>
            <div class="space-y-4">
              <SubscriptionPlanCard v-for="plan in renewalPlans" :key="plan.id" :plan="plan" :active-subscriptions="activeSubscriptions" @select="selectPlanFromModal" />
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
    <!-- Image Preview Overlay -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="previewImage" class="fixed inset-0 z-[60] flex items-center justify-center bg-black/70 backdrop-blur-sm" @click="previewImage = ''">
          <img :src="previewImage" alt="" class="max-h-[85vh] max-w-[90vw] rounded-xl object-contain shadow-2xl" />
        </div>
      </Transition>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { usePaymentStore } from '@/stores/payment'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { useFirstRechargeStore } from '@/stores/firstRecharge'
import { useAppStore } from '@/stores'
import { paymentAPI } from '@/api/payment'
import { extractApiErrorMessage, extractI18nErrorMessage } from '@/utils/apiError'
import { isMobileDevice } from '@/utils/device'
import type { BalanceProduct, FirstRechargeOffer, SubscriptionPlan, CheckoutInfoResponse, CreateOrderResult, OrderType } from '@/types/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import AmountInput from '@/components/payment/AmountInput.vue'
import PaymentMethodSelector from '@/components/payment/PaymentMethodSelector.vue'
import { METHOD_ORDER, getPaymentPopupFeatures } from '@/components/payment/providerConfig'
import {
  PAYMENT_RECOVERY_STORAGE_KEY,
  buildCreateOrderPayload,
  clearPaymentRecoverySnapshot,
  decidePaymentLaunch,
  getVisibleMethods,
  normalizeVisibleMethod,
  readPaymentRecoverySnapshot,
  type PaymentRecoverySnapshot,
  writePaymentRecoverySnapshot,
} from '@/components/payment/paymentFlow'
import { platformAccentBarClass, platformBadgeLightClass, platformBadgeClass, platformTextClass, platformLabel } from '@/utils/platformColors'
import SubscriptionPlanCard from '@/components/payment/SubscriptionPlanCard.vue'
import PaymentStatusPanel from '@/components/payment/PaymentStatusPanel.vue'
import PurchaseProductCard from '@/components/payment/PurchaseProductCard.vue'
import type { PurchaseProductMetric, PurchaseProductViewModel } from '@/components/payment/purchaseProductTypes'
import Icon from '@/components/icons/Icon.vue'
import { formatPaymentAmount, normalizePaymentCurrency } from '@/components/payment/currency'
import type { PaymentMethodOption } from '@/components/payment/PaymentMethodSelector.vue'
import { calculateSubscriptionTotalQuotaUSD, formatSubscriptionValidityUnit, normalizePositiveQuota } from '@/utils/subscriptionQuota'
import { buildPaymentErrorToastMessage, describePaymentScenarioError } from './paymentUx'
import { hasWechatResumeQuery, parseWechatResumeRoute, stripWechatResumeQuery } from './paymentWechatResume'

const i18n = useI18n()
const { t } = i18n
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const paymentStore = usePaymentStore()
const subscriptionStore = useSubscriptionStore()
const firstRechargeStore = useFirstRechargeStore()
const appStore = useAppStore()

const user = computed(() => authStore.user)
const activeSubscriptions = computed(() => subscriptionStore.activeSubscriptions)

function getDaysRemaining(expiresAt: string): number {
  const diff = new Date(expiresAt).getTime() - Date.now()
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)))
}

const loading = ref(true)
const submitting = ref(false)
const errorMessage = ref('')
const errorHintMessage = ref('')
const activeTab = ref<'recharge' | 'subscription'>('recharge')
const amount = ref<number | null>(null)
const selectedMethod = ref('')
const selectedPlan = ref<SubscriptionPlan | null>(null)
const previewImage = ref('')
const submittingProductKey = ref('')

const paymentPhase = ref<'select' | 'paying'>('select')

interface CreateOrderOptions {
  openid?: string
  wechatResumeToken?: string
  paymentType?: string
  balanceProductId?: number
  firstRechargeOfferId?: number
  isResume?: boolean
  mobileQrFallbackAttempted?: boolean
}

interface WeixinJSBridgeLike {
  invoke(
    action: string,
    payload: Record<string, unknown>,
    callback: (result: Record<string, unknown>) => void,
  ): void
}

function emptyPaymentState(): PaymentRecoverySnapshot {
  return {
    orderId: 0,
    amount: 0,
    qrCode: '',
    expiresAt: '',
    paymentType: '',
    payUrl: '',
    outTradeNo: '',
    clientSecret: '',
    intentId: '',
    currency: '',
    countryCode: '',
    paymentEnv: '',
    payAmount: 0,
    orderType: '',
    paymentMode: '',
    resumeToken: '',
    createdAt: 0,
  }
}

function getWeixinJSBridge(): WeixinJSBridgeLike | undefined {
  return (window as Window & { WeixinJSBridge?: WeixinJSBridgeLike }).WeixinJSBridge
}

function waitForWeixinJSBridge(timeoutMs = 4000): Promise<WeixinJSBridgeLike | null> {
  const existing = getWeixinJSBridge()
  if (existing) return Promise.resolve(existing)

  return new Promise((resolve) => {
    let settled = false
    const finish = (bridge: WeixinJSBridgeLike | null) => {
      if (settled) return
      settled = true
      document.removeEventListener('WeixinJSBridgeReady', handleReady)
      document.removeEventListener('onWeixinJSBridgeReady', handleReady)
      window.clearTimeout(timer)
      resolve(bridge)
    }
    const handleReady = () => finish(getWeixinJSBridge() ?? null)
    const timer = window.setTimeout(() => finish(getWeixinJSBridge() ?? null), timeoutMs)
    document.addEventListener('WeixinJSBridgeReady', handleReady, false)
    document.addEventListener('onWeixinJSBridgeReady', handleReady, false)
  })
}

async function invokeWechatJsapiPayment(payload: Record<string, unknown>): Promise<Record<string, unknown>> {
  const bridge = await waitForWeixinJSBridge()
  if (!bridge) {
    throw new Error('WECHAT_JSAPI_UNAVAILABLE')
  }
  return new Promise((resolve) => {
    bridge.invoke('getBrandWCPayRequest', payload, (result) => resolve(result || {}))
  })
}

const paymentState = ref<PaymentRecoverySnapshot>(emptyPaymentState())

function persistRecoverySnapshot(snapshot: PaymentRecoverySnapshot) {
  if (typeof window === 'undefined' || !snapshot.orderId) return
  writePaymentRecoverySnapshot(window.localStorage, snapshot, PAYMENT_RECOVERY_STORAGE_KEY)
}

function removeRecoverySnapshot() {
  if (typeof window === 'undefined') return
  clearPaymentRecoverySnapshot(window.localStorage, PAYMENT_RECOVERY_STORAGE_KEY)
}

function resetPayment() {
  paymentPhase.value = 'select'
  paymentState.value = emptyPaymentState()
  removeRecoverySnapshot()
}

async function redirectToPaymentResult(state: PaymentRecoverySnapshot): Promise<void> {
  const query: Record<string, string | undefined> = {}
  if (state.orderId > 0) {
    query.order_id = String(state.orderId)
  }
  if (state.outTradeNo) {
    query.out_trade_no = state.outTradeNo
  }
  if (state.resumeToken) {
    query.resume_token = state.resumeToken
  }
  await router.push({
    path: '/payment/result',
    query,
  })
}

function buildWechatOAuthAuthorizeUrl(
  authorizeUrl: string,
  context: {
    paymentType: string
    orderType: OrderType
    planId?: number
    balanceProductId?: number
    firstRechargeOfferId?: number
    orderAmount: number
  },
): string {
  const normalizedUrl = authorizeUrl.trim()
  if (!normalizedUrl || typeof window === 'undefined') {
    return normalizedUrl
  }

  try {
    const targetUrl = new URL(normalizedUrl, window.location.origin)
    const redirectPath = targetUrl.searchParams.get('redirect') || '/purchase'
    const redirectUrl = new URL(redirectPath, window.location.origin)
    const paymentType = normalizeVisibleMethod(context.paymentType) || context.paymentType.trim() || 'wxpay'

    redirectUrl.searchParams.set('payment_type', paymentType)
    redirectUrl.searchParams.set('order_type', context.orderType)

    if (context.planId) {
      redirectUrl.searchParams.set('plan_id', String(context.planId))
    } else {
      redirectUrl.searchParams.delete('plan_id')
    }
    if (context.balanceProductId) {
      redirectUrl.searchParams.set('balance_product_id', String(context.balanceProductId))
    } else {
      redirectUrl.searchParams.delete('balance_product_id')
    }
    if (context.firstRechargeOfferId) {
      redirectUrl.searchParams.set('first_recharge_offer_id', String(context.firstRechargeOfferId))
    } else {
      redirectUrl.searchParams.delete('first_recharge_offer_id')
    }

    if (context.orderAmount > 0) {
      redirectUrl.searchParams.set('amount', String(context.orderAmount))
    } else {
      redirectUrl.searchParams.delete('amount')
    }

    targetUrl.searchParams.set('redirect', `${redirectUrl.pathname}${redirectUrl.search}`)
    return targetUrl.toString()
  } catch {
    return normalizedUrl
  }
}

function onPaymentDone() {
  const wasSubscription = paymentState.value.orderType === 'subscription'
  resetPayment()
  selectedPlan.value = null
  if (wasSubscription) {
    subscriptionStore.fetchActiveSubscriptions(true).catch(() => {})
  }
  firstRechargeStore.fetchStatus(true).catch(() => {})
}

function onPaymentSuccess() {
  removeRecoverySnapshot()
  authStore.refreshUser()
  if (paymentState.value.orderType === 'subscription') {
    subscriptionStore.fetchActiveSubscriptions(true).catch(() => {})
  }
  firstRechargeStore.fetchStatus(true).catch(() => {})
}

function onPaymentSettled() {
  removeRecoverySnapshot()
}

// All checkout data from single API call
const checkout = ref<CheckoutInfoResponse>({
  methods: {}, global_min: 0, global_max: 0,
  balance_products: [], plans: [], balance_disabled: false, balance_recharge_multiplier: 1, recharge_fee_rate: 0, help_text: '', help_image_url: '', stripe_publishable_key: '',
})

const tabs = computed(() => {
  const result: { key: 'recharge' | 'subscription'; label: string }[] = []
  if (!checkout.value.balance_disabled) result.push({ key: 'recharge', label: t('payment.tabTopUp') })
  result.push({ key: 'subscription', label: t('payment.tabSubscribe') })
  return result
})

const visibleMethods = computed(() => getVisibleMethods(checkout.value.methods))
const enabledMethods = computed(() => Object.keys(visibleMethods.value))
const validAmount = computed(() => amount.value ?? 0)
const balanceRechargeMultiplier = computed(() => {
  const multiplier = checkout.value.balance_recharge_multiplier
  return multiplier > 0 ? multiplier : 1
})
const creditedAmount = computed(() => Math.round((validAmount.value * balanceRechargeMultiplier.value) * 100) / 100)
const supportContactInfo = computed(() => appStore.contactInfo.trim())
const supportHelpText = computed(() => checkout.value.help_text.trim())
const supportImageUrl = computed(() => checkout.value.help_image_url.trim())
const supportContactSubtitle = computed(() => t('payment.purchaseGuide.contactSubtitle').trim())

const productGridClass = 'grid gap-4 sm:grid-cols-2 lg:grid-cols-5'

// Check if an amount fits a method's [min, max]. 0 = no limit.
function amountFitsMethod(amt: number, methodType: string): boolean {
  if (amt <= 0) return true
  const ml = visibleMethods.value[methodType]
  if (!ml) return false
  if (ml.single_min > 0 && amt < ml.single_min) return false
  if (ml.single_max > 0 && amt > ml.single_max) return false
  return true
}

const globalMinAmount = computed(() => {
  const limits = Object.values(visibleMethods.value)
  if (limits.length === 0) return 0
  if (limits.some(limit => limit.single_min <= 0)) return 0
  return Math.min(...limits.map(limit => limit.single_min))
})
const globalMaxAmount = computed(() => {
  const limits = Object.values(visibleMethods.value)
  if (limits.length === 0) return 0
  if (limits.some(limit => limit.single_max <= 0)) return 0
  return Math.max(...limits.map(limit => limit.single_max))
})

// Selected method's limits (for validation and error messages)
const selectedLimit = computed(() => visibleMethods.value[selectedMethod.value])
const selectedCurrency = computed(() => normalizePaymentCurrency(selectedLimit.value?.currency))
const paymentPriceCurrency = 'CNY'
const quotaDisplayCurrency = 'USD'
const localeCode = computed(() => {
  const raw = i18n.locale as unknown
  if (typeof raw === 'string') return raw
  if (raw && typeof raw === 'object' && 'value' in raw) {
    return String((raw as { value?: string }).value || '')
  }
  return undefined
})

function formatSelectedPaymentAmount(value: number): string {
  return formatPaymentAmount(value, selectedCurrency.value, localeCode.value)
}

function formatQuotaAmount(value: number): string {
  const amount = Number.isFinite(value) ? value : 0
  try {
    return new Intl.NumberFormat(localeCode.value || undefined, {
      style: 'currency',
      currency: quotaDisplayCurrency,
      currencyDisplay: 'narrowSymbol',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(amount)
  } catch {
    return `${quotaDisplayCurrency} ${amount.toFixed(0)}`
  }
}

const amountInputPrefix = computed(() => {
  try {
    const part = new Intl.NumberFormat(localeCode.value || undefined, {
      style: 'currency',
      currency: selectedCurrency.value,
      currencyDisplay: 'narrowSymbol',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).formatToParts(0).find(item => item.type === 'currency')
    return part?.value || selectedCurrency.value
  } catch {
    return selectedCurrency.value
  }
})

type PurchaseCardItem<T> = {
  raw: T
  product: PurchaseProductViewModel
  heroMetrics?: PurchaseProductMetric[]
  metrics: PurchaseProductMetric[]
  priceRows?: PurchaseProductMetric[]
  methods: PaymentMethodOption[]
  priceSuffix?: string
}

type SubscriptionProductSection = {
  platform: string
  label: string
  items: PurchaseCardItem<SubscriptionPlan>[]
}

function normalizePlatformKey(platform: string | null | undefined): string {
  return String(platform || '').trim().toLowerCase() || 'unknown'
}

function formatPlatformLabel(platform: string | null | undefined): string {
  const key = normalizePlatformKey(platform)
  if (key === 'unknown') return t('payment.product.unknownPlatform')
  return platformLabel(key) || t('payment.product.unknownPlatform')
}

function normalizeTextList(value: string[] | string | undefined): string[] {
  if (Array.isArray(value)) return value.map(item => String(item).trim()).filter(Boolean)
  if (!value) return []
  return String(value).split('\n').map(item => item.trim()).filter(Boolean)
}

function amountMethodOptions(value: number): PaymentMethodOption[] {
  return enabledMethods.value.map((type) => {
    const ml = visibleMethods.value[type]
    return {
      type,
      fee_rate: ml?.fee_rate ?? 0,
      available: ml?.available !== false && amountFitsMethod(value, type),
    }
  })
}

function formatQuota(value: number | null | undefined): string {
  if (value == null || value <= 0) return t('payment.planCard.unlimited')
  return formatQuotaAmount(Number(value))
}

function formatProductPriceAmount(value: number): string {
  return formatPaymentAmount(value, paymentPriceCurrency, localeCode.value)
}

function buildProductPriceRows(price: number, originalPrice?: number | null, suffix = ''): PurchaseProductMetric[] {
  const rows: PurchaseProductMetric[] = []
  if (originalPrice != null && originalPrice > 0) {
    rows.push({
      label: t('payment.product.originalPrice'),
      value: formatProductPriceAmount(originalPrice),
      tone: 'muted',
    })
  }
  rows.push({
    label: t('payment.product.payPrice'),
    value: `${formatProductPriceAmount(price)}${suffix}`,
    tone: 'strong',
  })
  return rows
}

function formatExchangeRate(amount: number, price: number): string {
  if (amount <= 0 || price <= 0) return t('payment.planCard.unlimited')
  return `1¥:${Number((amount / price).toFixed(2))}$`
}

const feeRate = computed(() => checkout.value?.recharge_fee_rate ?? 0)

const methodOptions = computed<PaymentMethodOption[]>(() =>
  enabledMethods.value.map((type) => {
    const ml = visibleMethods.value[type]
    return {
      type,
      fee_rate: ml?.fee_rate ?? 0,
      available: ml?.available !== false && amountFitsMethod(validAmount.value, type),
    }
  }),
)

const feeAmount = computed(() =>
  feeRate.value > 0 && validAmount.value > 0
    ? Math.ceil(((validAmount.value * feeRate.value) / 100) * 100) / 100
    : 0,
)
const totalAmount = computed(() =>
  feeRate.value > 0 && validAmount.value > 0
    ? Math.round((validAmount.value + feeAmount.value) * 100) / 100
    : validAmount.value,
)

const amountError = computed(() => {
  if (validAmount.value <= 0) return ''
  if (!enabledMethods.value.some((method) => amountFitsMethod(validAmount.value, method))) {
    return t('payment.amountNoMethod')
  }
  const limit = selectedLimit.value
  if (limit) {
    if (limit.single_min > 0 && validAmount.value < limit.single_min) {
      return t('payment.amountTooLow', { min: formatSelectedPaymentAmount(limit.single_min) })
    }
    if (limit.single_max > 0 && validAmount.value > limit.single_max) {
      return t('payment.amountTooHigh', { max: formatSelectedPaymentAmount(limit.single_max) })
    }
  }
  return ''
})

const canSubmit = computed(() =>
  validAmount.value > 0
    && amountFitsMethod(validAmount.value, selectedMethod.value)
    && selectedLimit.value?.available !== false
)

function getPlanTotalQuota(plan: SubscriptionPlan): number | null {
  return normalizePositiveQuota(plan.total_quota) ?? calculateSubscriptionTotalQuotaUSD(plan, plan)
}

const balanceProductCards = computed<PurchaseCardItem<BalanceProduct>[]>(() =>
  (checkout.value.balance_products || []).map((product) => {
    const price = Number(product.price) || 0
    const amount = Number(product.amount) || 0
    const purchaseLimit = Math.max(0, Math.floor(Number(product.purchase_limit) || 0))
    const metrics: PurchaseProductMetric[] = [
      { label: t('payment.product.exchangeRate'), value: formatExchangeRate(amount, price) },
      { label: t('payment.product.validity'), value: t('payment.product.permanent') },
    ]
    if (purchaseLimit > 0) {
      metrics.push({ label: t('payment.product.purchaseLimit'), value: t('payment.product.purchaseLimitValue', { count: purchaseLimit }) })
    }
    return {
      raw: product,
      product: {
        id: product.id,
        name: product.name,
        description: product.description || '',
        price,
        original_price: product.original_price,
        tags: normalizeTextList(product.tags),
        features: normalizeTextList(product.features),
      },
      heroMetrics: [
        {
          label: t('payment.product.balanceAmount'),
          value: formatQuotaAmount(amount),
          tone: 'strong',
        },
      ],
      metrics,
      priceRows: buildProductPriceRows(price, product.original_price),
      methods: amountMethodOptions(price),
    }
  }),
)

const firstRechargeCards = computed<PurchaseCardItem<FirstRechargeOffer>[]>(() => {
  const status = firstRechargeStore.status
  if (!status?.enabled || !status.eligible || status.completed) return []
  return (status.offers || []).map((offer) => {
    const price = Number(offer.price) || 0
    const amount = Number(offer.amount) || 0
    return {
      raw: offer,
      product: {
        id: offer.id,
        name: offer.name || t('firstRecharge.label'),
        description: offer.description || t('payment.firstRecharge.cardDescription'),
        price,
        tags: [t('firstRecharge.label')],
        features: [t('payment.firstRecharge.featureNoRebate'), t('payment.firstRecharge.featureOnce')],
      },
      heroMetrics: [
        {
          label: t('payment.product.balanceAmount'),
          value: formatQuotaAmount(amount),
          tone: 'strong',
        },
      ],
      metrics: [
        { label: t('payment.product.exchangeRate'), value: formatExchangeRate(amount, price) },
        { label: t('payment.product.validity'), value: t('payment.product.permanent') },
      ],
      priceRows: buildProductPriceRows(price, undefined),
      methods: amountMethodOptions(price),
    }
  })
})

function loadFirstRechargeStatus(force = false) {
  if (!authStore.isAuthenticated) return Promise.resolve(null)
  return firstRechargeStore.fetchStatus(force)
}

const subscriptionProductCards = computed<PurchaseCardItem<SubscriptionPlan>[]>(() =>
  checkout.value.plans.map((plan) => {
    const totalQuota = getPlanTotalQuota(plan)
    const metrics: PurchaseProductMetric[] = [
      { label: t('payment.product.totalQuota'), value: formatQuota(totalQuota) },
    ]
    const dailyQuota = normalizePositiveQuota(plan.daily_quota) ?? normalizePositiveQuota(plan.daily_limit_usd)
    const weeklyQuota = normalizePositiveQuota(plan.weekly_limit_usd)
    const monthlyQuota = normalizePositiveQuota(plan.monthly_limit_usd)
    const quotaHeroMetrics: PurchaseProductMetric[] = []
    if (dailyQuota != null) {
      quotaHeroMetrics.push({ label: t('payment.product.dailyQuota'), value: formatQuota(dailyQuota), tone: 'strong' })
    }
    if (weeklyQuota != null) {
      quotaHeroMetrics.push({ label: t('payment.product.weeklyQuota'), value: formatQuota(weeklyQuota), tone: 'strong' })
    }
    if (monthlyQuota != null) {
      quotaHeroMetrics.push({ label: t('payment.product.monthlyQuota'), value: formatQuota(monthlyQuota), tone: 'strong' })
    }
    if (quotaHeroMetrics.length === 0) {
      quotaHeroMetrics.push({ label: t('payment.product.availableQuota'), value: t('payment.planCard.unlimited'), tone: 'strong' })
    }
    metrics.push({ label: t('payment.product.validity'), value: formatPlanValidity(plan) })
    const validitySuffix = ` / ${formatPlanValidity(plan)}`
    return {
      raw: plan,
      product: {
        id: plan.id,
        name: plan.name,
        description: plan.description || '',
        detail: plan.display_notes || '',
        price: Number(plan.price) || 0,
        original_price: plan.original_price,
        tags: normalizeTextList(plan.tags),
        features: normalizeTextList(plan.features),
      },
      heroMetrics: quotaHeroMetrics,
      metrics,
      priceRows: buildProductPriceRows(Number(plan.price) || 0, plan.original_price, validitySuffix),
      methods: amountMethodOptions(Number(plan.price) || 0),
    }
  }),
)

const subscriptionProductSections = computed<SubscriptionProductSection[]>(() => {
  const cardsByPlatform = new Map<string, PurchaseCardItem<SubscriptionPlan>[]>()
  const firstSeenPlatforms: string[] = []
  const sortedCards = [...subscriptionProductCards.value].sort((a, b) =>
    (a.raw.sort_order ?? 0) - (b.raw.sort_order ?? 0) || a.raw.id - b.raw.id,
  )
  sortedCards.forEach((item) => {
    const platform = normalizePlatformKey(item.raw.group_platform)
    if (!cardsByPlatform.has(platform)) {
      cardsByPlatform.set(platform, [])
      firstSeenPlatforms.push(platform)
    }
    cardsByPlatform.get(platform)!.push(item)
  })
  return firstSeenPlatforms
    .map(platform => ({
      platform,
      label: formatPlatformLabel(platform),
      items: cardsByPlatform.get(platform) || [],
    }))
    .filter(section => section.items.length > 0)
})

// Subscription-specific: method options based on plan price
const subMethodOptions = computed<PaymentMethodOption[]>(() => {
  const planPrice = selectedPlan.value?.price ?? 0
  return enabledMethods.value.map((type) => {
    const ml = visibleMethods.value[type]
    return {
      type,
      fee_rate: ml?.fee_rate ?? 0,
      available: ml?.available !== false && amountFitsMethod(planPrice, type),
    }
  })
})

const subFeeAmount = computed(() => {
  const price = selectedPlan.value?.price ?? 0
  if (feeRate.value <= 0 || price <= 0) return 0
  return Math.ceil(((price * feeRate.value) / 100) * 100) / 100
})

const subTotalAmount = computed(() => {
  const price = selectedPlan.value?.price ?? 0
  if (feeRate.value <= 0 || price <= 0) return price
  return Math.round((price + subFeeAmount.value) * 100) / 100
})

const canSubmitSubscription = computed(() =>
  selectedPlan.value !== null
    && amountFitsMethod(selectedPlan.value.price, selectedMethod.value)
    && selectedLimit.value?.available !== false
)

// Auto-switch to first available method when current selection can't handle the amount
watch(() => [validAmount.value, selectedMethod.value] as const, ([amt, method]) => {
  if (amt <= 0 || amountFitsMethod(amt, method)) return
  const available = enabledMethods.value.find((m) => amountFitsMethod(amt, m))
  if (available) selectedMethod.value = available
})

// Payment button class: follows selected payment method color
const paymentButtonClass = computed(() => {
  const m = selectedMethod.value
  if (!m) return 'btn-primary'
  if (m.includes('alipay')) return 'btn-alipay'
  if (m.includes('wxpay')) return 'btn-wxpay'
  if (m === 'stripe') return 'btn-stripe'
  if (m === 'airwallex') return 'btn-airwallex'
  return 'btn-primary'
})

// Subscription confirm: platform accent colors (clean card, no gradient)
const planBadgeClass = computed(() => platformBadgeClass(selectedPlan.value?.group_platform || ''))
const planTextClass = computed(() => platformTextClass(selectedPlan.value?.group_platform || ''))

// Renewal modal state
const showRenewalModal = ref(false)
const renewGroupId = ref<number | null>(null)
const renewalPlans = computed(() => {
  if (renewGroupId.value == null) return []
  return checkout.value.plans.filter(p => p.group_id === renewGroupId.value)
})

const planValiditySuffix = computed(() => {
  if (!selectedPlan.value) return ''
  return formatPlanValidity(selectedPlan.value)
})

const selectedDailyLimit = computed(() => normalizePositiveQuota(selectedPlan.value?.daily_limit_usd))
const selectedWeeklyLimit = computed(() => normalizePositiveQuota(selectedPlan.value?.weekly_limit_usd))
const selectedMonthlyLimit = computed(() => normalizePositiveQuota(selectedPlan.value?.monthly_limit_usd))

function formatPlanValidity(plan: SubscriptionPlan): string {
  return formatSubscriptionValidityUnit(plan.validity_days, plan.validity_unit, {
    days: t('payment.days'),
    weeks: t('payment.admin.weeks'),
    months: t('payment.months'),
  })
}

function selectPlanFromModal(plan: SubscriptionPlan) {
  showRenewalModal.value = false
  renewGroupId.value = null
  selectedPlan.value = plan
  errorMessage.value = ''
}

function closeRenewalModal() {
  showRenewalModal.value = false
  renewGroupId.value = null
}

async function handleSubmitRecharge() {
  if (!canSubmit.value || submitting.value) return
  await createOrder(validAmount.value, 'balance')
}

async function handleSubmitBalanceProduct(product: BalanceProduct, paymentType: string) {
  if (submitting.value) return
  submittingProductKey.value = `balance:${product.id}`
  await createOrder(Number(product.price) || 0, 'balance', undefined, { paymentType, balanceProductId: product.id })
}

async function handleSubmitFirstRechargeOffer(offer: FirstRechargeOffer, paymentType: string) {
  if (submitting.value) return
  submittingProductKey.value = `first-recharge:${offer.id}`
  await createOrder(Number(offer.price) || 0, 'balance', undefined, {
    paymentType,
    firstRechargeOfferId: offer.id,
  })
}

async function confirmSubscribe() {
  if (!selectedPlan.value || submitting.value) return
  await createOrder(selectedPlan.value.price, 'subscription', selectedPlan.value.id)
}

async function handleSubmitSubscriptionPlan(plan: SubscriptionPlan, paymentType: string) {
  if (submitting.value) return
  submittingProductKey.value = `subscription:${plan.id}`
  await createOrder(Number(plan.price) || 0, 'subscription', plan.id, { paymentType })
}

async function createOrder(orderAmount: number, orderType: OrderType, planId?: number, options: CreateOrderOptions = {}) {
  submitting.value = true
  errorMessage.value = ''
  errorHintMessage.value = ''
  const requestType = normalizeVisibleMethod(options.paymentType || selectedMethod.value) || options.paymentType || selectedMethod.value
  try {
    const payload = buildCreateOrderPayload({
      amount: orderAmount,
      paymentType: requestType,
      orderType,
      planId,
      balanceProductId: options.balanceProductId,
      origin: typeof window !== 'undefined' ? window.location.origin : '',
      isMobile: isMobileDevice(),
      isWechatBrowser: typeof window !== 'undefined' && /MicroMessenger/i.test(window.navigator.userAgent),
      forceQRCode: !!(checkout.value.alipay_force_qrcode && normalizeVisibleMethod(requestType) === 'alipay'),
    })
    if (options.openid) {
      payload.openid = options.openid
    }
    if (options.wechatResumeToken) {
      payload.wechat_resume_token = options.wechatResumeToken
    }
    if (options.firstRechargeOfferId) {
      payload.first_recharge_offer_id = options.firstRechargeOfferId
    }

    const result = await paymentStore.createOrder(payload) as CreateOrderResult & { resume_token?: string }
    const openWindow = (url: string) => {
      const win = window.open(url, 'paymentPopup', getPaymentPopupFeatures())
      if (!win || win.closed) {
        window.location.href = url
      }
    }
    const visibleMethod = normalizeVisibleMethod(requestType) || requestType
    // When user clicks the dedicated Stripe button, leave method blank so the
    // landing page renders Stripe's full Payment Element (card/link/alipay/wxpay).
    const stripeMethod = visibleMethod === 'stripe'
      ? ''
      : visibleMethod === 'wxpay' ? 'wechat_pay' : 'alipay'
    const stripeRouteUrl = result.client_secret && visibleMethod !== 'airwallex'
      ? router.resolve({
        path: '/payment/stripe',
        query: {
          order_id: String(result.order_id),
          client_secret: result.client_secret,
          method: stripeMethod || undefined,
          resume_token: result.resume_token || undefined,
        },
      }).href
      : ''
    const airwallexRouteUrl = result.client_secret && result.intent_id
      ? router.resolve({
        path: '/payment/airwallex',
        query: {
          order_id: String(result.order_id),
          out_trade_no: result.out_trade_no || undefined,
          resume_token: result.resume_token || undefined,
        },
      }).href
      : ''
    const decision = decidePaymentLaunch(result, {
      visibleMethod,
      orderType,
      isMobile: isMobileDevice(),
      isWechatBrowser: typeof window !== 'undefined' && /MicroMessenger/i.test(window.navigator.userAgent),
      forceQRCode: !!(checkout.value.alipay_force_qrcode && visibleMethod === 'alipay'),
      stripePopupUrl: stripeRouteUrl,
      stripeRouteUrl,
      airwallexRouteUrl,
    })

    if (decision.kind === 'wechat_oauth' && decision.oauth?.authorize_url) {
      window.location.href = buildWechatOAuthAuthorizeUrl(decision.oauth.authorize_url, {
        paymentType: visibleMethod,
        orderType,
        planId,
        balanceProductId: options.balanceProductId,
        firstRechargeOfferId: options.firstRechargeOfferId,
        orderAmount,
      })
      return
    }

    if (decision.kind === 'unhandled') {
      applyScenarioError({ reason: 'UNHANDLED_PAYMENT_SCENARIO' }, visibleMethod)
      return
    }

    paymentState.value = decision.paymentState
    paymentPhase.value = 'paying'
    persistRecoverySnapshot(decision.recovery)

    if (decision.kind === 'stripe_popup') {
      openWindow(decision.paymentState.payUrl)
      return
    }
    if (decision.kind === 'stripe_route') {
      window.location.href = decision.paymentState.payUrl
      return
    }
    if (decision.kind === 'airwallex_route') {
      window.location.href = decision.paymentState.payUrl
      return
    }
    if (decision.kind === 'wechat_jsapi' && decision.jsapi) {
      try {
        const jsapiResult = await invokeWechatJsapiPayment(decision.jsapi as Record<string, unknown>)
        const errMsg = String(jsapiResult.err_msg || '').toLowerCase()
        if (errMsg.includes('cancel')) {
          appStore.showInfo(t('payment.qr.cancelled'))
          resetPayment()
        } else if (errMsg && !errMsg.includes('ok')) {
          resetPayment()
          const fallbackApplied = await attemptMobileQrFallback(
            { reason: 'WECHAT_JSAPI_FAILED', message: errMsg },
            {
              orderAmount,
              orderType,
              planId,
              balanceProductId: options.balanceProductId,
              firstRechargeOfferId: options.firstRechargeOfferId,
              paymentType: visibleMethod,
              attempted: options.mobileQrFallbackAttempted === true,
            },
          )
          if (!fallbackApplied) {
            applyScenarioError({ reason: 'WECHAT_JSAPI_FAILED', message: errMsg }, visibleMethod)
          }
        } else {
          const resultState = { ...decision.paymentState }
          resetPayment()
          await redirectToPaymentResult(resultState)
        }
      } catch (err: unknown) {
        resetPayment()
        const fallbackApplied = await attemptMobileQrFallback(err, {
          orderAmount,
          orderType,
          planId,
          balanceProductId: options.balanceProductId,
          firstRechargeOfferId: options.firstRechargeOfferId,
          paymentType: visibleMethod,
          attempted: options.mobileQrFallbackAttempted === true,
        })
        if (!fallbackApplied) {
          throw err
        }
      }
      return
    }
    if (decision.kind === 'redirect_waiting' && decision.paymentState.payUrl) {
      if (isMobileDevice()) {
        window.location.href = decision.paymentState.payUrl
        return
      }
      openWindow(decision.paymentState.payUrl)
    }
  } catch (err: unknown) {
    const apiErr = err as Record<string, unknown>
    if (apiErr.reason === 'TOO_MANY_PENDING') {
      const metadata = apiErr.metadata as Record<string, unknown> | undefined
      errorMessage.value = t('payment.errors.tooManyPending', { max: metadata?.max || '' })
      errorHintMessage.value = ''
    } else if (apiErr.reason === 'CANCEL_RATE_LIMITED') {
      errorMessage.value = t('payment.errors.cancelRateLimited')
      errorHintMessage.value = ''
    } else if (await attemptMobileQrFallback(err, {
      orderAmount,
      orderType,
      planId,
      balanceProductId: options.balanceProductId,
      firstRechargeOfferId: options.firstRechargeOfferId,
      paymentType: requestType,
      attempted: options.mobileQrFallbackAttempted === true,
    })) {
      return
    } else {
      const handled = applyScenarioError(
        err,
        normalizeVisibleMethod(options.paymentType || selectedMethod.value) || selectedMethod.value,
      )
      if (!handled) {
        errorMessage.value = extractI18nErrorMessage(err, t, 'payment.errors', extractApiErrorMessage(err, t('payment.result.failed')))
        errorHintMessage.value = ''
      }
      if (handled) {
        return
      }
    }
    appStore.showError(buildPaymentErrorToastMessage(errorMessage.value, errorHintMessage.value))
  } finally {
    submitting.value = false
    submittingProductKey.value = ''
  }
}

interface MobileQrFallbackContext {
  orderAmount: number
  orderType: OrderType
  planId?: number
  balanceProductId?: number
  firstRechargeOfferId?: number
  paymentType: string
  attempted: boolean
}

function shouldFallbackToDesktopQr(err: unknown, paymentMethod: string, attempted: boolean): boolean {
  if (attempted || !isMobileDevice()) {
    return false
  }

  const normalizedMethod = normalizeVisibleMethod(paymentMethod) || paymentMethod
  const reason = typeof err === 'object' && err && 'reason' in err && typeof err.reason === 'string'
    ? err.reason
    : ''
  const message = err instanceof Error
    ? err.message
    : (typeof err === 'object' && err && 'message' in err && typeof err.message === 'string'
      ? err.message
      : '')
  const normalizedMessage = message.toLowerCase()

  if (normalizedMethod === 'wxpay') {
    return reason === 'WECHAT_H5_NOT_AUTHORIZED'
      || reason === 'WECHAT_PAYMENT_MP_NOT_CONFIGURED'
      || reason === 'WECHAT_JSAPI_FAILED'
      || reason === 'PAYMENT_GATEWAY_ERROR'
      || reason === 'UNHANDLED_PAYMENT_SCENARIO'
      || normalizedMessage.includes('weixinjsbridge is unavailable')
      || normalizedMessage.includes('wechat_jsapi_unavailable')
  }

  if (normalizedMethod === 'alipay') {
    return reason === 'PAYMENT_GATEWAY_ERROR' || reason === 'UNHANDLED_PAYMENT_SCENARIO'
  }

  return false
}

async function attemptMobileQrFallback(err: unknown, context: MobileQrFallbackContext): Promise<boolean> {
  if (!shouldFallbackToDesktopQr(err, context.paymentType, context.attempted)) {
    return false
  }

  try {
    const visibleMethod = normalizeVisibleMethod(context.paymentType) || context.paymentType
    const payload = buildCreateOrderPayload({
      amount: context.orderAmount,
      paymentType: visibleMethod,
      orderType: context.orderType,
      planId: context.planId,
      balanceProductId: context.balanceProductId,
      origin: typeof window !== 'undefined' ? window.location.origin : '',
      isMobile: false,
      isWechatBrowser: false,
    })
    if (context.firstRechargeOfferId) {
      payload.first_recharge_offer_id = context.firstRechargeOfferId
    }
    const result = await paymentStore.createOrder(payload) as CreateOrderResult & { resume_token?: string }
    const stripeMethod = visibleMethod === 'wxpay' ? 'wechat_pay' : 'alipay'
    const stripeRouteUrl = result.client_secret
      ? router.resolve({
        path: '/payment/stripe',
        query: {
          order_id: String(result.order_id),
          client_secret: result.client_secret,
          method: stripeMethod,
          resume_token: result.resume_token || undefined,
        },
      }).href
      : ''
    const decision = decidePaymentLaunch(result, {
      visibleMethod,
      orderType: context.orderType,
      isMobile: false,
      isWechatBrowser: false,
      stripePopupUrl: stripeRouteUrl,
      stripeRouteUrl,
    })

    if (decision.kind !== 'qr_waiting' || !decision.paymentState.qrCode) {
      return false
    }

    errorMessage.value = ''
    errorHintMessage.value = ''
    paymentState.value = decision.paymentState
    paymentPhase.value = 'paying'
    persistRecoverySnapshot(decision.recovery)
    appStore.showWarning(t('payment.errors.mobilePaymentFallbackToQr'))
    return true
  } catch {
    return false
  }
}

function applyScenarioError(err: unknown, paymentMethod: string): boolean {
  const descriptor = describePaymentScenarioError(err, {
    paymentMethod,
    isMobile: isMobileDevice(),
    isWechatBrowser: typeof window !== 'undefined' && /MicroMessenger/i.test(window.navigator.userAgent),
  })
  if (!descriptor) {
    errorMessage.value = ''
    errorHintMessage.value = ''
    return false
  }
  errorMessage.value = t(descriptor.messageKey)
  errorHintMessage.value = descriptor.hintKey ? t(descriptor.hintKey) : ''
  appStore.showError(buildPaymentErrorToastMessage(errorMessage.value, errorHintMessage.value))
  return true
}

async function resumeWechatPaymentFromQuery() {
  const resume = parseWechatResumeRoute(route.query, checkout.value.plans, validAmount.value)
  if (!resume) {
    return
  }

  selectedMethod.value = resume.paymentType
  if (resume.orderType === 'balance' && resume.orderAmount > 0) {
    amount.value = resume.orderAmount
  }
  if (resume.orderType === 'subscription' && resume.planId) {
    selectedPlan.value = checkout.value.plans.find(plan => plan.id === resume.planId) ?? null
  }

  await router.replace({ path: route.path, query: stripWechatResumeQuery(route.query) })

  if (resume.wechatResumeToken) {
    await createOrder(0, resume.orderType, resume.planId, {
      wechatResumeToken: resume.wechatResumeToken,
      paymentType: resume.paymentType,
      isResume: true,
      balanceProductId: resume.balanceProductId,
      firstRechargeOfferId: resume.firstRechargeOfferId,
    })
    return
  }

  if (resume.orderAmount > 0 && resume.openid) {
    await createOrder(resume.orderAmount, resume.orderType, resume.planId, {
      openid: resume.openid,
      paymentType: resume.paymentType,
      isResume: true,
      balanceProductId: resume.balanceProductId,
      firstRechargeOfferId: resume.firstRechargeOfferId,
    })
  }
}

onMounted(async () => {
  try {
    await loadFirstRechargeStatus()
    const res = await paymentAPI.getCheckoutInfo()
    checkout.value = res.data
    if (enabledMethods.value.length) {
      const order: readonly string[] = METHOD_ORDER
      const sorted = [...enabledMethods.value].sort((a, b) => {
        const ai = order.indexOf(a)
        const bi = order.indexOf(b)
        return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
      })
      selectedMethod.value = sorted[0]
    }
    if (typeof window !== 'undefined') {
      if (hasWechatResumeQuery(route.query)) {
        removeRecoverySnapshot()
      }
      const routeResumeToken = typeof route.query.resume_token === 'string'
        ? route.query.resume_token
        : typeof route.query.wechat_resume_token === 'string'
          ? route.query.wechat_resume_token
          : undefined
      const restored = readPaymentRecoverySnapshot(
        window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY),
        { resumeToken: routeResumeToken },
      )
      if (restored) {
        paymentState.value = restored
        paymentPhase.value = 'paying'
        const restoredMethod = normalizeVisibleMethod(restored.paymentType)
        if (restoredMethod) {
          selectedMethod.value = restoredMethod
        }
      } else {
        removeRecoverySnapshot()
      }
    }
    await resumeWechatPaymentFromQuery()
    if (checkout.value.balance_disabled) {
      activeTab.value = 'subscription'
    }
    if (route.query.tab === 'recharge' || route.query.first_recharge === '1') {
      activeTab.value = 'recharge'
    }
    // Handle renewal navigation: ?tab=subscription&group=123
    if (route.query.tab === 'subscription') {
      activeTab.value = 'subscription'
      if (route.query.group) {
        const groupId = Number(route.query.group)
        const groupPlans = checkout.value.plans.filter(p => p.group_id === groupId)
        if (groupPlans.length === 1) {
          selectedPlan.value = groupPlans[0]
        } else if (groupPlans.length > 1) {
          renewGroupId.value = groupId
          showRenewalModal.value = true
        }
      }
    }
  } catch (err: unknown) { appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error'))) }
  finally { loading.value = false }
  // Fetch active subscriptions (uses cache, non-blocking)
  subscriptionStore.fetchActiveSubscriptions().catch(() => {})
})
</script>
