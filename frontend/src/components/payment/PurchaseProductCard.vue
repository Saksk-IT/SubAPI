<template>
  <article class="flex h-full min-w-0 flex-col rounded-2xl border border-gray-200 bg-white p-4 shadow-sm transition hover:-translate-y-0.5 hover:shadow-lg sm:p-5 dark:border-dark-700 dark:bg-dark-800">
    <div class="mb-4 flex items-start justify-between gap-3">
      <div class="min-w-0">
        <div class="mb-2 flex flex-wrap gap-1.5">
          <span
            v-for="(tag, index) in normalizedTags"
            :key="tag"
            data-testid="product-tag"
            :class="productTagClass(index)"
          >
            {{ tag }}
          </span>
        </div>
        <h3 class="break-words text-2xl font-black leading-tight text-gray-950 [overflow-wrap:anywhere] sm:text-3xl dark:text-white">{{ product.name }}</h3>
      </div>
      <button
        v-if="product.description"
        type="button"
        class="shrink-0 rounded-full border border-gray-200 px-3 py-1 text-xs font-semibold text-gray-500 transition hover:border-gray-300 hover:text-gray-900 dark:border-dark-600 dark:text-gray-300 dark:hover:text-white"
        @click="showDetail = !showDetail"
      >
        {{ t('payment.product.detail') }}
      </button>
    </div>

    <div class="mb-4">
      <div v-if="heroMetrics.length > 0" class="rounded-2xl border border-emerald-100 bg-emerald-50/70 px-4 py-3 dark:border-emerald-500/20 dark:bg-emerald-500/10">
        <p class="text-xs font-black text-emerald-600 dark:text-emerald-300">{{ heroMetrics[0].label }}</p>
        <div class="mt-1 flex flex-wrap items-end gap-2">
          <span class="min-w-0 break-words text-4xl font-black leading-none tracking-normal text-gray-950 [overflow-wrap:anywhere] sm:text-5xl dark:text-white">
            {{ heroMetrics[0].value }}
          </span>
        </div>
        <div v-if="heroMetrics.length > 1" class="mt-3 grid gap-2" :class="heroMetrics.length >= 3 ? 'grid-cols-2' : 'grid-cols-1'">
          <div
            v-for="metric in heroMetrics.slice(1)"
            :key="metric.label"
            class="min-w-0 rounded-xl border border-emerald-200 bg-white/70 px-3 py-2 dark:border-emerald-500/25 dark:bg-dark-800/70"
          >
            <p class="text-[11px] font-black text-emerald-600 dark:text-emerald-300">{{ metric.label }}</p>
            <p class="mt-1 break-words text-xl font-black leading-tight text-gray-950 [overflow-wrap:anywhere] dark:text-white">{{ metric.value }}</p>
          </div>
        </div>
      </div>
      <div v-else class="flex flex-wrap items-end gap-2">
        <span v-if="product.original_price" class="break-words pb-1 text-sm text-gray-400 line-through [overflow-wrap:anywhere] dark:text-gray-500">
          {{ formatAmount(product.original_price) }}
        </span>
        <span class="min-w-0 break-words text-4xl font-black leading-none tracking-normal text-gray-950 [overflow-wrap:anywhere] sm:text-5xl dark:text-white">
          {{ formatAmount(product.price) }}
        </span>
        <span v-if="priceSuffix" class="break-words pb-1 text-sm font-medium text-gray-500 [overflow-wrap:anywhere] dark:text-gray-400">{{ priceSuffix }}</span>
      </div>
      <p v-if="product.description" class="mt-3 break-words text-sm font-semibold leading-relaxed text-gray-700 [overflow-wrap:anywhere] dark:text-gray-200">
        {{ product.description }}
      </p>
    </div>

    <div v-if="metrics.length > 0" class="grid grid-cols-2 gap-3" :class="{ 'sm:[grid-template-columns:repeat(3,minmax(0,1fr))]': metrics.length >= 3 }">
      <div
        v-for="metric in metrics"
        :key="metric.label"
        class="min-w-0 rounded-xl border p-3"
        :class="metric.tone === 'strong' ? 'border-emerald-200 bg-emerald-50/70 dark:border-emerald-500/25 dark:bg-emerald-500/10' : 'border-gray-200 dark:border-dark-600'"
      >
        <p class="text-xs text-gray-400 dark:text-gray-500">{{ metric.label }}</p>
        <p
          class="mt-1 break-words text-lg font-black leading-snug [overflow-wrap:anywhere]"
          :class="metric.tone === 'strong' ? 'text-emerald-700 dark:text-emerald-200' : 'text-gray-950 dark:text-white'"
        >{{ metric.value }}</p>
      </div>
    </div>

    <div v-if="visibleFeatures.length > 0 || (showDetail && product.detail)" class="mt-4 rounded-xl border border-gray-200 p-3 dark:border-dark-600">
      <p v-if="showDetail && product.detail" class="mb-3 text-sm leading-relaxed text-gray-600 dark:text-gray-300">
        {{ product.detail }}
      </p>
      <div v-if="visibleFeatures.length > 0" class="space-y-2">
        <p v-for="feature in visibleFeatures" :key="feature" class="border-l border-gray-300 pl-3 text-sm leading-6 text-gray-500 dark:border-dark-500 dark:text-gray-400">
          {{ feature }}
        </p>
      </div>
    </div>

    <div class="mt-auto pt-4">
      <div
        v-if="priceRows.length > 0"
        data-testid="product-price-summary"
        class="mb-3 rounded-2xl border border-gray-300 bg-white px-4 py-3 shadow-sm ring-1 ring-gray-100 dark:border-dark-500 dark:bg-dark-900/70 dark:ring-dark-700"
      >
        <div
          v-for="row in priceRows"
          :key="row.label"
          class="flex items-center justify-between gap-3 py-0.5"
        >
          <span
            class="shrink-0 text-sm"
            :class="row.tone === 'muted' ? 'font-semibold text-gray-500 dark:text-gray-400' : 'font-black text-gray-700 dark:text-gray-200'"
          >{{ row.label }}</span>
          <span
            data-testid="product-price-row-value"
            class="min-w-0 break-words text-right font-black [overflow-wrap:anywhere]"
            :class="row.tone === 'muted' ? 'text-sm text-gray-400 line-through dark:text-gray-500' : 'text-lg leading-none text-gray-950 sm:text-xl dark:text-white'"
          >
            {{ row.value }}
          </span>
        </div>
      </div>
      <p class="mb-2 text-xs font-semibold text-green-600 dark:text-green-400">{{ t('payment.product.autoApply') }}</p>
      <div v-if="methods.length > 0" class="space-y-2">
        <button
          v-for="method in sortedMethods"
          :key="method.type"
          type="button"
          class="flex w-full items-center justify-center gap-2 rounded-full px-4 py-3 text-sm font-bold transition active:scale-[0.99]"
          :class="methodButtonClass(method.type, method.available)"
          :disabled="!method.available || submitting"
          @click="emit('pay', method.type)"
        >
          <span v-if="submitting" class="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"></span>
          <span v-else data-testid="payment-method-icon-shell" class="flex h-7 w-7 items-center justify-center rounded-full bg-white shadow-sm">
            <img :src="methodIcon(method.type)" alt="" class="h-5 w-5 object-contain" />
          </span>
          <span>{{ t(`payment.methods.${method.type}`, method.type) }}</span>
        </button>
      </div>
      <p v-else class="rounded-xl bg-gray-50 px-3 py-3 text-center text-sm text-gray-500 dark:bg-dark-700 dark:text-gray-400">
        {{ t('payment.notAvailable') }}
      </p>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { METHOD_ORDER } from './providerConfig'
import { formatPaymentAmount } from './currency'
import alipayIcon from '@/assets/icons/alipay.svg'
import wxpayIcon from '@/assets/icons/wxpay.svg'
import stripeIcon from '@/assets/icons/stripe.svg'
import airwallexIcon from '@/assets/icons/airwallex.svg'
import type { PaymentMethodOption } from './PaymentMethodSelector.vue'
import type { PurchaseProductMetric, PurchaseProductViewModel } from './purchaseProductTypes'

const props = defineProps<{
  product: PurchaseProductViewModel
  metrics: PurchaseProductMetric[]
  heroMetrics?: PurchaseProductMetric[]
  priceRows?: PurchaseProductMetric[]
  methods: PaymentMethodOption[]
  currency: string
  locale?: string
  priceSuffix?: string
  submitting?: boolean
}>()

defineOptions({ name: 'PurchaseProductCard' })

const emit = defineEmits<{
  pay: [method: string]
}>()

const { t } = useI18n()
const showDetail = ref(false)

const normalizedTags = computed(() => props.product.tags.filter(Boolean).slice(0, 4))
const visibleFeatures = computed(() => props.product.features.filter(Boolean).slice(0, 5))
const heroMetrics = computed(() => (props.heroMetrics || []).filter(item => item.label && item.value).slice(0, 3))
const priceRows = computed(() => (props.priceRows || []).filter(item => item.label && item.value).slice(0, 3))
const sortedMethods = computed(() => {
  const order: readonly string[] = METHOD_ORDER
  return [...props.methods].sort((a, b) => {
    const ai = order.indexOf(a.type)
    const bi = order.indexOf(b.type)
    return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
  })
})

function formatAmount(value: number): string {
  return formatPaymentAmount(value, props.currency, props.locale)
}

function methodIcon(type: string): string {
  if (type.includes('alipay')) return alipayIcon
  if (type.includes('wxpay')) return wxpayIcon
  if (type === 'airwallex') return airwallexIcon
  return stripeIcon
}

function productTagClass(index: number): string {
  const base = 'rounded-full px-2.5 py-1 text-[11px] font-black leading-none shadow-sm ring-1'
  if (index === 0) return `${base} bg-gradient-to-r from-orange-500 to-rose-500 text-white ring-orange-200`
  if (index === 1) return `${base} bg-amber-100 text-amber-700 ring-amber-200 dark:bg-amber-300/15 dark:text-amber-200 dark:ring-amber-300/20`
  return `${base} bg-gray-100 text-gray-600 ring-gray-200 dark:bg-dark-700 dark:text-gray-200 dark:ring-dark-600`
}

function methodButtonClass(type: string, available: boolean): string {
  if (!available) return 'cursor-not-allowed border border-gray-200 bg-gray-50 text-gray-400 dark:border-dark-700 dark:bg-dark-800/50'
  if (type.includes('alipay')) return 'bg-[#02A9F1] text-white hover:bg-[#0297d8]'
  if (type.includes('wxpay')) return 'bg-[#09BB07] text-white hover:bg-[#08a806]'
  if (type === 'stripe') return 'bg-[#676BE5] text-white hover:bg-[#575bd8]'
  if (type === 'airwallex') return 'bg-[#FF6B3D] text-white hover:bg-[#f05d2f]'
  return 'bg-gray-950 text-white hover:bg-black dark:bg-white dark:text-gray-950 dark:hover:bg-gray-100'
}
</script>
