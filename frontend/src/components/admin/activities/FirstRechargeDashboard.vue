<template>
  <section class="space-y-5">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h2 class="text-lg font-black text-gray-950 dark:text-white">
          {{ t('admin.firstRecharge.dashboard') }}
        </h2>
        <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">
          {{ t('admin.firstRecharge.dashboardHint') }}
        </p>
      </div>
      <button
        type="button"
        class="btn btn-secondary"
        :disabled="loading"
        @click="emit('refresh')"
      >
        <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
        {{ t('common.refresh') }}
      </button>
    </div>

    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <article
        v-for="card in statCards"
        :key="card.key"
        class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800"
      >
        <div class="flex items-center gap-3">
          <div :class="['flex h-10 w-10 shrink-0 items-center justify-center rounded-lg', card.iconClass]">
            <Icon :name="card.icon" size="md" :class="card.iconTextClass" />
          </div>
          <div class="min-w-0">
            <p class="text-xs font-semibold text-gray-500 dark:text-gray-400">{{ card.label }}</p>
            <p class="mt-1 truncate text-xl font-black text-gray-950 dark:text-white">{{ card.value }}</p>
          </div>
        </div>
      </article>
    </div>

    <section class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800 sm:p-6">
      <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h3 class="text-base font-black text-gray-950 dark:text-white">
            {{ t('admin.firstRecharge.offerStats') }}
          </h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.firstRecharge.offerStatsHint') }}
          </p>
        </div>
        <LoadingSpinner v-if="loading" size="sm" />
      </div>

      <div v-if="offerStats.length > 0" class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        <article
          v-for="stat in offerStats"
          :key="stat.offer_id"
          class="rounded-xl border border-gray-100 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900/50"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <p class="truncate text-sm font-bold text-gray-950 dark:text-white">
                {{ stat.name || t('admin.firstRecharge.unknownOffer', { id: stat.offer_id }) }}
              </p>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.firstRecharge.offerPriceAmount', { price: formatMoney(stat.price), amount: formatMoney(stat.amount) }) }}
              </p>
            </div>
            <span class="rounded-full bg-primary-50 px-2.5 py-1 text-xs font-black text-primary-700 dark:bg-primary-500/15 dark:text-primary-200">
              {{ stat.count }}
            </span>
          </div>
          <div class="mt-4 flex items-end justify-between gap-3">
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.firstRecharge.successRevenue') }}</span>
            <span class="text-lg font-black text-gray-950 dark:text-white">${{ formatMoney(stat.revenue) }}</span>
          </div>
        </article>
      </div>
      <p v-else class="mt-4 rounded-xl bg-gray-50 px-4 py-8 text-center text-sm text-gray-500 dark:bg-dark-900/60 dark:text-gray-400">
        {{ loading ? t('common.loading') : t('admin.firstRecharge.emptyOfferStats') }}
      </p>
    </section>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DashboardStats, PaymentOfferStat } from '@/types/payment'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

const props = defineProps<{
  stats: DashboardStats | null
  loading: boolean
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
}>()

const { t } = useI18n()

const offerStats = computed<PaymentOfferStat[]>(() => props.stats?.offer_stats || [])

const statCards = computed(() => [
  {
    key: 'today_amount',
    label: t('admin.firstRecharge.todayRevenue'),
    value: `$${formatMoney(props.stats?.today_amount || 0)}`,
    icon: 'dollar' as const,
    iconClass: 'bg-green-100 dark:bg-green-900/30',
    iconTextClass: 'text-green-600 dark:text-green-400',
  },
  {
    key: 'today_count',
    label: t('admin.firstRecharge.todayCount'),
    value: String(props.stats?.today_count || 0),
    icon: 'chart' as const,
    iconClass: 'bg-blue-100 dark:bg-blue-900/30',
    iconTextClass: 'text-blue-600 dark:text-blue-400',
  },
  {
    key: 'total_amount',
    label: t('admin.firstRecharge.totalRevenue'),
    value: `$${formatMoney(props.stats?.total_amount || 0)}`,
    icon: 'creditCard' as const,
    iconClass: 'bg-purple-100 dark:bg-purple-900/30',
    iconTextClass: 'text-purple-600 dark:text-purple-400',
  },
  {
    key: 'total_count',
    label: t('admin.firstRecharge.totalCount'),
    value: String(props.stats?.total_count || 0),
    icon: 'gift' as const,
    iconClass: 'bg-amber-100 dark:bg-amber-900/30',
    iconTextClass: 'text-amber-600 dark:text-amber-400',
  },
])

function formatMoney(value: number): string {
  return (Number(value) || 0).toFixed(2)
}
</script>
