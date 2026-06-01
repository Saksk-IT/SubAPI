<template>
  <section class="space-y-4 rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800 sm:p-6">
    <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
      <div>
        <h2 class="text-lg font-black text-gray-950 dark:text-white">
          {{ t('admin.firstRecharge.orders') }}
        </h2>
        <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">
          {{ t('admin.firstRecharge.ordersHint') }}
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <input
          :value="keyword"
          type="text"
          class="input w-full sm:w-64"
          :placeholder="t('payment.admin.searchOrders')"
          @input="emit('update:keyword', ($event.target as HTMLInputElement).value)"
        />
        <Select
          :model-value="status"
          :options="statusFilterOptions"
          class="w-36"
          @change="handleStatusChange"
        />
        <Select
          :model-value="paymentType"
          :options="paymentTypeFilterOptions"
          class="w-40"
          @change="handlePaymentTypeChange"
        />
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="loading"
          :title="t('common.refresh')"
          @click="emit('refresh')"
        >
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
        </button>
      </div>
    </div>

    <OrderTable :orders="orders" :loading="loading" show-user>
      <template #actions="{ row }">
        <button
          type="button"
          class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-600"
          @click="emit('detail', row)"
        >
          <Icon name="eye" size="sm" />
          {{ t('common.view') }}
        </button>
      </template>
    </OrderTable>

    <Pagination
      v-if="total > 0"
      :page="page"
      :total="total"
      :page-size="pageSize"
      @update:page="emit('update:page', $event)"
      @update:pageSize="emit('update:pageSize', $event)"
    />
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PaymentOrder } from '@/types/payment'
import Icon from '@/components/icons/Icon.vue'
import OrderTable from '@/components/payment/OrderTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'

defineProps<{
  orders: PaymentOrder[]
  loading: boolean
  keyword: string
  status: string
  paymentType: string
  page: number
  pageSize: number
  total: number
}>()

const emit = defineEmits<{
  (e: 'update:keyword', value: string): void
  (e: 'update:status', value: string): void
  (e: 'update:paymentType', value: string): void
  (e: 'update:page', page: number): void
  (e: 'update:pageSize', pageSize: number): void
  (e: 'filter'): void
  (e: 'refresh'): void
  (e: 'detail', order: PaymentOrder): void
}>()

const { t } = useI18n()

const statusFilterOptions = computed(() => [
  { value: '', label: t('payment.admin.allStatuses') },
  { value: 'PENDING', label: t('payment.status.pending') },
  { value: 'PAID', label: t('payment.status.paid') },
  { value: 'RECHARGING', label: t('payment.status.recharging') },
  { value: 'COMPLETED', label: t('payment.status.completed') },
  { value: 'EXPIRED', label: t('payment.status.expired') },
  { value: 'CANCELLED', label: t('payment.status.cancelled') },
  { value: 'FAILED', label: t('payment.status.failed') },
  { value: 'REFUNDED', label: t('payment.status.refunded') },
  { value: 'REFUND_REQUESTED', label: t('payment.status.refund_requested') },
  { value: 'REFUND_FAILED', label: t('payment.status.refund_failed') },
])

const paymentTypeFilterOptions = computed(() => [
  { value: '', label: t('payment.admin.allPaymentTypes') },
  { value: 'alipay', label: t('payment.methods.alipay') },
  { value: 'wxpay', label: t('payment.methods.wxpay') },
  { value: 'stripe', label: t('payment.methods.stripe') },
  { value: 'airwallex', label: t('payment.methods.airwallex') },
])

function handleStatusChange(value: string | number | boolean | null) {
  emit('update:status', String(value || ''))
  emit('filter')
}

function handlePaymentTypeChange(value: string | number | boolean | null) {
  emit('update:paymentType', String(value || ''))
  emit('filter')
}
</script>
