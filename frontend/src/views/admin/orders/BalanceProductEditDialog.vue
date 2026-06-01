<template>
  <BaseDialog :show="show" :title="product ? t('payment.admin.editBalanceProduct') : t('payment.admin.createBalanceProduct')" width="wide" @close="emit('close')">
    <form id="balance-product-form" class="space-y-4" @submit.prevent="handleSave">
      <div>
        <label class="input-label">{{ t('payment.admin.productName') }} <span class="text-red-500">*</span></label>
        <input v-model="form.name" type="text" class="input" required />
      </div>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="input-label">{{ t('payment.admin.payPrice') }} <span class="text-red-500">*</span></label>
          <input v-model.number="form.price" type="number" min="0.01" step="0.01" class="input" required />
        </div>
        <div>
          <label class="input-label">{{ t('payment.admin.creditAmount') }} <span class="text-red-500">*</span></label>
          <input v-model.number="form.amount" type="number" min="0.01" step="0.01" class="input" required />
        </div>
      </div>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="input-label">{{ t('payment.admin.originalPrice') }}</label>
          <input v-model.number="form.original_price" type="number" min="0" step="0.01" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('payment.admin.paymentProductName') }}</label>
          <input v-model="form.product_name" type="text" class="input" />
        </div>
      </div>
      <div>
        <label class="input-label">{{ t('payment.admin.purchaseLimit') }}</label>
        <input v-model.number="form.purchase_limit" type="number" min="0" step="1" class="input" />
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('payment.admin.purchaseLimitHint') }}</p>
      </div>
      <div>
        <label class="input-label">{{ t('payment.admin.productDescription') }}</label>
        <textarea v-model="form.description" rows="2" class="input"></textarea>
      </div>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="input-label">{{ t('payment.admin.productTags') }}</label>
          <textarea v-model="form.tags" rows="3" class="input" :placeholder="t('payment.admin.productTagsPlaceholder')"></textarea>
        </div>
        <div>
          <label class="input-label">{{ t('payment.admin.features') }}</label>
          <textarea v-model="form.features" rows="3" class="input" :placeholder="t('payment.admin.featuresPlaceholder')"></textarea>
        </div>
      </div>
      <div class="flex items-center gap-3">
        <label class="text-sm text-gray-700 dark:text-gray-300">{{ t('payment.admin.forSale') }}</label>
        <button
          type="button"
          :class="[
            'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
            form.for_sale ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
          ]"
          @click="form.for_sale = !form.for_sale"
        >
          <span :class="[
            'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
            form.for_sale ? 'translate-x-5' : 'translate-x-0'
          ]" />
        </button>
      </div>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button type="submit" form="balance-product-form" :disabled="saving" class="btn btn-primary">{{ saving ? t('common.saving') : t('common.save') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminPaymentAPI } from '@/api/admin/payment'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { BalanceProduct } from '@/types/payment'
import BaseDialog from '@/components/common/BaseDialog.vue'

const props = defineProps<{
  show: boolean
  product: BalanceProduct | null
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const { t } = useI18n()
const appStore = useAppStore()
const saving = ref(false)
const form = reactive({
  name: '',
  description: '',
  price: 0,
  amount: 0,
  original_price: 0,
  tags: '',
  features: '',
  product_name: '',
  for_sale: true,
  purchase_limit: 0,
})

function textList(value: string | string[] | undefined): string {
  if (Array.isArray(value)) return value.join('\n')
  return value || ''
}

watch(() => props.show, (visible) => {
  if (!visible) return
  if (props.product) {
    Object.assign(form, {
      name: props.product.name,
      description: props.product.description || '',
      price: props.product.price || 0,
      amount: props.product.amount || 0,
      original_price: props.product.original_price || 0,
      tags: textList(props.product.tags),
      features: textList(props.product.features),
      product_name: props.product.product_name || '',
      for_sale: props.product.for_sale,
      purchase_limit: props.product.purchase_limit || 0,
    })
  } else {
    Object.assign(form, {
      name: '',
      description: '',
      price: 0,
      amount: 0,
      original_price: 0,
      tags: '',
      features: '',
      product_name: '',
      for_sale: true,
      purchase_limit: 0,
    })
  }
})

function buildPayload() {
  return {
    name: form.name,
    description: form.description,
    price: Number(form.price) || 0,
    amount: Number(form.amount) || 0,
    original_price: Number(form.original_price) || 0,
    tags: form.tags,
    features: form.features,
    product_name: form.product_name,
    for_sale: form.for_sale,
    purchase_limit: Math.max(0, Math.floor(Number(form.purchase_limit) || 0)),
  }
}

async function handleSave() {
  if (!form.name.trim()) {
    appStore.showError(t('payment.admin.productNameRequired'))
    return
  }
  if ((Number(form.price) || 0) <= 0 || (Number(form.amount) || 0) <= 0) {
    appStore.showError(t('payment.admin.productAmountRequired'))
    return
  }
  saving.value = true
  try {
    const payload = buildPayload()
    if (props.product) await adminPaymentAPI.updateBalanceProduct(props.product.id, payload)
    else await adminPaymentAPI.createBalanceProduct(payload)
    appStore.showSuccess(t('common.saved'))
    emit('close')
    emit('saved')
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    saving.value = false
  }
}
</script>
