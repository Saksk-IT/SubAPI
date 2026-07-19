<template>
  <section
    v-if="showBanner"
    class="border-b border-gray-200 dark:border-dark-700"
    data-testid="activity-banner"
  >
    <div
      class="grid w-full"
      :class="showBothBanners ? 'divide-y divide-gray-200 dark:divide-dark-700 lg:grid-cols-2 lg:divide-x lg:divide-y-0' : ''"
    >
      <div
        v-if="showFirstRecharge"
        class="flex min-w-0 flex-col gap-3 bg-amber-50/95 px-4 py-3 sm:flex-row sm:items-center sm:justify-between dark:bg-amber-500/10"
        data-testid="first-recharge-banner"
      >
        <button
          type="button"
          class="group flex min-w-0 flex-1 items-start gap-3 text-left"
          @click="goToFirstRecharge"
        >
          <span class="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-amber-500 text-white shadow-sm">
            <Icon name="gift" size="md" />
          </span>
          <span class="min-w-0">
            <span class="block text-sm font-black text-gray-950 dark:text-white">
              {{ t('firstRecharge.banner.title') }}
            </span>
            <span class="mt-0.5 block break-words text-sm leading-5 text-gray-700 [overflow-wrap:anywhere] dark:text-gray-200">
              {{ offerSummary }}
            </span>
          </span>
        </button>
        <button
          type="button"
          class="btn btn-primary w-full shrink-0 sm:w-auto"
          @click="goToFirstRecharge"
        >
          {{ t('firstRecharge.banner.cta') }}
        </button>
      </div>

      <div
        v-if="showDailyCheckIn"
        class="flex min-w-0 flex-col gap-3 bg-emerald-50/95 px-4 py-3 sm:flex-row sm:items-center sm:justify-between dark:bg-emerald-500/10"
        data-testid="daily-check-in-banner"
      >
        <button
          type="button"
          class="group flex min-w-0 flex-1 items-start gap-3 text-left"
          @click="goToDailyCheckIn"
        >
          <span class="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-emerald-600 text-white shadow-sm">
            <Icon :name="dailyCheckInStatus?.checked_in_today ? 'checkCircle' : 'calendar'" size="md" />
          </span>
          <span class="min-w-0">
            <span class="block text-sm font-black text-gray-950 dark:text-white">
              {{ t('activities.dailyCheckIn.title') }}
            </span>
            <span class="mt-0.5 block break-words text-sm leading-5 text-gray-700 [overflow-wrap:anywhere] dark:text-gray-200">
              {{ dailyCheckInSummary }}
            </span>
          </span>
        </button>
        <button
          type="button"
          class="btn btn-primary w-full shrink-0 sm:w-auto"
          data-testid="daily-check-in-banner-cta"
          @click="goToDailyCheckIn"
        >
          {{ dailyCheckInStatus?.checked_in_today
            ? t('activities.dailyCheckIn.viewDetails')
            : t('activities.dailyCheckIn.bannerCta') }}
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useActivityStore } from '@/stores/activities'
import { useFirstRechargeStore } from '@/stores/firstRecharge'
import { formatPaymentAmount } from '@/components/payment/currency'
import Icon from '@/components/icons/Icon.vue'

const { t, locale } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const activityStore = useActivityStore()
const firstRechargeStore = useFirstRechargeStore()

const dailyCheckInStatus = computed(() =>
  activityStore.activities.find((activity) => activity.type === 'daily_check_in')?.daily_check_in || null
)
const showFirstRecharge = computed(() => authStore.isAuthenticated && firstRechargeStore.available)
const showDailyCheckIn = computed(() => authStore.isAuthenticated && dailyCheckInStatus.value?.enabled === true)
const showBothBanners = computed(() => showFirstRecharge.value && showDailyCheckIn.value)
const showBanner = computed(() => showFirstRecharge.value || showDailyCheckIn.value)
const bestOffer = computed(() => {
  const offers = firstRechargeStore.status?.offers || []
  return [...offers].sort((a, b) => {
    if (a.price === b.price) return b.amount - a.amount
    return a.price - b.price
  })[0]
})

const localeCode = computed(() => {
  const raw = locale as unknown
  if (typeof raw === 'string') return raw
  if (raw && typeof raw === 'object' && 'value' in raw) {
    return String((raw as { value?: string }).value || '')
  }
  return undefined
})

const offerSummary = computed(() => {
  if (firstRechargeStore.status?.purchase_mode === 'product_link') {
    return t('firstRecharge.banner.productLinkSummary')
  }
  const offer = bestOffer.value
  if (!offer) return t('firstRecharge.banner.subtitle')
  return t('firstRecharge.banner.offerSummary', {
    price: formatPaymentAmount(offer.price, 'CNY', localeCode.value),
    amount: formatPaymentAmount(offer.amount, 'USD', localeCode.value),
  })
})

const dailyCheckInSummary = computed(() => {
  const status = dailyCheckInStatus.value
  if (!status) return ''
  if (status.checked_in_today) {
    return t('activities.dailyCheckIn.bannerCompletedSummary')
  }
  return t('activities.dailyCheckIn.bannerSummary', {
    amount: formatPaymentAmount(status.reward_amount, 'USD', localeCode.value),
  })
})

function fetchStatuses(force = false) {
  if (!authStore.isAuthenticated) return
  firstRechargeStore.fetchStatus(force).catch(() => {})
  activityStore.fetchActivities(force).catch(() => {})
}

function goToFirstRecharge() {
  const status = firstRechargeStore.status
  if (status?.purchase_mode === 'product_link' && status.product_url) {
    window.location.assign(status.product_url)
    return
  }
  router.push({
    path: '/purchase',
    query: { tab: 'recharge', first_recharge: '1' },
  })
}

function goToDailyCheckIn() {
  router.push('/activities')
}

onMounted(() => fetchStatuses())

watch(() => authStore.user?.id, (userId, oldUserId) => {
  if (!userId) {
    firstRechargeStore.clear()
    activityStore.reset()
    return
  }
  if (userId !== oldUserId) {
    fetchStatuses(true)
  }
})
</script>
