<template>
  <BaseDialog :show="show" :title="t('payment.admin.bulkEditPlansTitle')" width="wide" @close="emit('close')">
    <form id="plan-bulk-edit-form" class="space-y-4" @submit.prevent="handleSubmit">
      <div class="rounded-lg bg-primary-50 px-3 py-2 text-sm font-medium text-primary-900 dark:bg-primary-900/20 dark:text-primary-100">
        {{ t('payment.admin.bulkEditPlansSelected', { count: planIds.length }) }}
      </div>

      <div class="space-y-3">
        <label class="flex items-start gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <input v-model="enabled.price_multiplier" type="checkbox" class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
          <div class="min-w-0 flex-1">
            <span class="input-label mb-1 block">{{ t('payment.admin.planPriceMultiplier') }}</span>
            <input v-model.number="form.price_multiplier" type="number" min="0" step="any" class="input" :disabled="!enabled.price_multiplier" />
          </div>
        </label>

        <label class="flex items-start gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <input v-model="enabled.description" type="checkbox" class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
          <div class="min-w-0 flex-1">
            <span class="input-label mb-1 block">{{ t('payment.admin.planDescription') }}</span>
            <textarea v-model="form.description" rows="2" class="input" :disabled="!enabled.description"></textarea>
          </div>
        </label>

        <label class="flex items-start gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <input v-model="enabled.features" type="checkbox" class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
          <div class="min-w-0 flex-1">
            <span class="input-label mb-1 block">{{ t('payment.admin.features') }}</span>
            <textarea v-model="form.features" rows="3" class="input" :placeholder="t('payment.admin.featuresPlaceholder')" :disabled="!enabled.features"></textarea>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('payment.admin.featuresHint') }}</p>
          </div>
        </label>

        <label class="flex items-start gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <input v-model="enabled.tags" type="checkbox" class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
          <div class="min-w-0 flex-1">
            <span class="input-label mb-1 block">{{ t('payment.admin.productTags') }}</span>
            <textarea v-model="form.tags" rows="3" class="input" :placeholder="t('payment.admin.productTagsPlaceholder')" :disabled="!enabled.tags"></textarea>
          </div>
        </label>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button type="submit" form="plan-bulk-edit-form" :disabled="saving || planIds.length === 0" class="btn btn-primary">
          {{ saving ? t('common.saving') : t('common.save') }}
        </button>
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
import BaseDialog from '@/components/common/BaseDialog.vue'

const props = defineProps<{
  show: boolean
  planIds: number[]
}>()

const emit = defineEmits<{
  close: []
  updated: []
}>()

const { t } = useI18n()
const appStore = useAppStore()
const saving = ref(false)
const enabled = reactive({
  price_multiplier: false,
  description: false,
  features: false,
  tags: false,
})
const form = reactive({
  price_multiplier: 1,
  description: '',
  features: '',
  tags: '',
})

watch(() => props.show, (visible) => {
  if (!visible) return
  Object.assign(enabled, {
    price_multiplier: false,
    description: false,
    features: false,
    tags: false,
  })
  Object.assign(form, {
    price_multiplier: 1,
    description: '',
    features: '',
    tags: '',
  })
})

function buildFields() {
  const fields: Record<string, unknown> = {}
  if (enabled.price_multiplier) fields.price_multiplier = Number(form.price_multiplier)
  if (enabled.description) fields.description = form.description
  if (enabled.features) fields.features = form.features
  if (enabled.tags) fields.tags = form.tags
  return fields
}

async function handleSubmit() {
  const fields = buildFields()
  if (Object.keys(fields).length === 0) {
    appStore.showError(t('payment.admin.bulkEditPlansNoFields'))
    return
  }
  if (enabled.price_multiplier && (!Number.isFinite(Number(form.price_multiplier)) || Number(form.price_multiplier) <= 0)) {
    appStore.showError(t('payment.admin.planPriceMultiplierRequired'))
    return
  }
  saving.value = true
  try {
    const res = await adminPaymentAPI.bulkUpdatePlans({
      plan_ids: props.planIds,
      fields,
    })
    appStore.showSuccess(t('payment.admin.bulkEditPlansSuccess', { count: res.data.updated }))
    emit('updated')
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('payment.admin.batchUpdateFailed')))
  } finally {
    saving.value = false
  }
}
</script>
