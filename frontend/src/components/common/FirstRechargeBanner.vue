<template>
  <section
    v-if="showBanner"
    class="border-b border-amber-200 bg-amber-50/95 px-4 py-3 dark:border-amber-500/25 dark:bg-amber-500/10"
  >
    <div class="mx-auto flex w-full max-w-[1720px] flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
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
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useFirstRechargeStore } from '@/stores/firstRecharge'
import { formatPaymentAmount } from '@/components/payment/currency'
import Icon from '@/components/icons/Icon.vue'

const { t, locale } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const firstRechargeStore = useFirstRechargeStore()

const showBanner = computed(() => authStore.isAuthenticated && firstRechargeStore.available)
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
  const offer = bestOffer.value
  if (!offer) return t('firstRecharge.banner.subtitle')
  return t('firstRecharge.banner.offerSummary', {
    price: formatPaymentAmount(offer.price, 'CNY', localeCode.value),
    amount: formatPaymentAmount(offer.amount, 'USD', localeCode.value),
  })
})

function fetchStatus(force = false) {
  if (!authStore.isAuthenticated) return
  firstRechargeStore.fetchStatus(force).catch(() => {})
}

function goToFirstRecharge() {
  router.push({
    path: '/purchase',
    query: { tab: 'recharge', first_recharge: '1' },
  })
}

onMounted(() => fetchStatus())

watch(() => authStore.user?.id, (userId, oldUserId) => {
  if (!userId) {
    firstRechargeStore.clear()
    return
  }
  if (userId !== oldUserId) {
    fetchStatus(true)
  }
})
</script>
