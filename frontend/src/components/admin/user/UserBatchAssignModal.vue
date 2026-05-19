<template>
  <BaseDialog
    :show="show"
    :title="t('admin.users.batchAssign.title')"
    width="wide"
    @close="handleClose"
  >
    <form id="batch-assign-form" class="space-y-5" @submit.prevent="handleSubmit">
      <div class="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800/70 dark:bg-amber-950/30">
        <div class="flex gap-3">
          <Icon name="exclamationTriangle" size="md" class="mt-0.5 flex-shrink-0 text-amber-600 dark:text-amber-400" />
          <div class="space-y-1">
            <p class="text-sm font-medium text-amber-900 dark:text-amber-100">
              {{ t('admin.users.batchAssign.allUsersTitle') }}
            </p>
            <p class="text-sm text-amber-800 dark:text-amber-200">
              {{ t('admin.users.batchAssign.allUsersHint') }}
            </p>
          </div>
        </div>
      </div>

      <div>
        <label class="input-label">{{ t('admin.users.batchAssign.actionType') }}</label>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <button
            type="button"
            :class="actionButtonClass('balance')"
            @click="form.action = 'balance'"
          >
            <Icon name="dollar" size="md" class="flex-shrink-0" />
            <span class="min-w-0 text-left">
              <span class="block font-medium">{{ t('admin.users.batchAssign.balanceAction') }}</span>
              <span class="block text-xs opacity-75">{{ t('admin.users.batchAssign.balanceActionHint') }}</span>
            </span>
          </button>
          <button
            type="button"
            :class="actionButtonClass('subscription')"
            @click="form.action = 'subscription'"
          >
            <Icon name="gift" size="md" class="flex-shrink-0" />
            <span class="min-w-0 text-left">
              <span class="block font-medium">{{ t('admin.users.batchAssign.subscriptionAction') }}</span>
              <span class="block text-xs opacity-75">{{ t('admin.users.batchAssign.subscriptionActionHint') }}</span>
            </span>
          </button>
        </div>
      </div>

      <section v-if="form.action === 'balance'" class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.users.batchAssign.balanceMode') }}</label>
          <div class="grid grid-cols-2 gap-2">
            <button
              type="button"
              :class="modeButtonClass('add')"
              @click="form.balanceOperation = 'add'"
            >
              {{ t('admin.users.batchAssign.addBalance') }}
            </button>
            <button
              type="button"
              :class="modeButtonClass('subtract')"
              @click="form.balanceOperation = 'subtract'"
            >
              {{ t('admin.users.batchAssign.subtractBalance') }}
            </button>
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.users.batchAssign.amount') }}</label>
          <div class="relative">
            <span class="absolute left-3 top-1/2 -translate-y-1/2 text-sm font-medium text-gray-500">$</span>
            <input
              v-model.number="form.amount"
              type="number"
              min="0"
              step="any"
              required
              class="input pl-8"
              :placeholder="t('admin.users.batchAssign.amountPlaceholder')"
            />
          </div>
          <p class="input-hint">{{ t('admin.users.batchAssign.amountHint') }}</p>
        </div>

        <div>
          <label class="input-label">{{ t('admin.users.notes') }}</label>
          <textarea
            v-model.trim="form.notes"
            rows="3"
            class="input"
            :placeholder="t('admin.users.batchAssign.balanceNotesPlaceholder')"
          ></textarea>
        </div>
      </section>

      <section v-else class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.users.batchAssign.subscriptionGroup') }}</label>
          <Select
            v-model="form.groupId"
            :options="subscriptionGroupOptions"
            :placeholder="t('admin.users.batchAssign.selectSubscriptionGroup')"
            searchable
            :empty-text="t('admin.users.batchAssign.noSubscriptionGroups')"
          >
            <template #selected="{ option }">
              <GroupBadge
                v-if="option"
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
              />
              <span v-else class="text-gray-400">{{ t('admin.users.batchAssign.selectSubscriptionGroup') }}</span>
            </template>
            <template #option="{ option, selected }">
              <GroupOptionItem
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :description="(option as unknown as GroupOption).description"
                :selected="selected"
              />
            </template>
          </Select>
          <p class="input-hint">{{ t('admin.users.batchAssign.subscriptionGroupHint') }}</p>
        </div>

        <div>
          <label class="input-label">{{ t('admin.users.batchAssign.validityDays') }}</label>
          <input
            v-model.number="form.validityDays"
            type="number"
            min="1"
            max="36500"
            step="1"
            required
            class="input"
          />
          <p class="input-hint">{{ t('admin.users.batchAssign.validityDaysHint') }}</p>
        </div>

        <div>
          <label class="input-label">{{ t('admin.users.notes') }}</label>
          <textarea
            v-model.trim="form.notes"
            rows="3"
            class="input"
            :placeholder="t('admin.users.batchAssign.subscriptionNotesPlaceholder')"
          ></textarea>
        </div>
      </section>

      <div
        v-if="lastResult"
        class="grid grid-cols-2 gap-3 rounded-lg border border-gray-200 p-4 dark:border-dark-600 sm:grid-cols-3"
      >
        <div>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.users.batchAssign.resultTarget') }}</p>
          <p class="text-lg font-semibold text-gray-900 dark:text-white">{{ lastResult.target_count }}</p>
        </div>
        <div>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.users.batchAssign.resultSuccess') }}</p>
          <p class="text-lg font-semibold text-emerald-600 dark:text-emerald-400">{{ lastResult.success_count }}</p>
        </div>
        <div>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.users.batchAssign.resultFailed') }}</p>
          <p class="text-lg font-semibold text-red-600 dark:text-red-400">{{ lastResult.failed_count }}</p>
        </div>
        <div v-if="form.action === 'balance'">
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.users.batchAssign.resultBalance') }}</p>
          <p class="text-lg font-semibold text-gray-900 dark:text-white">{{ lastResult.balance_affected_count }}</p>
        </div>
        <div v-if="form.action === 'subscription'">
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.users.batchAssign.resultAssigned') }}</p>
          <p class="text-lg font-semibold text-gray-900 dark:text-white">{{ lastResult.subscription_assigned }}</p>
        </div>
        <div v-if="form.action === 'subscription'">
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.users.batchAssign.resultExtended') }}</p>
          <p class="text-lg font-semibold text-gray-900 dark:text-white">{{ lastResult.subscription_extended }}</p>
        </div>
      </div>

      <div
        v-if="lastResult?.errors?.length"
        class="max-h-28 overflow-y-auto rounded-lg border border-red-200 bg-red-50 p-3 text-xs text-red-700 dark:border-red-800 dark:bg-red-950/30 dark:text-red-300"
      >
        <p v-for="error in lastResult.errors.slice(0, 5)" :key="error">{{ error }}</p>
      </div>
    </form>

    <template #footer>
      <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        <button type="button" class="btn btn-secondary" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="batch-assign-form"
          class="btn btn-primary"
          :disabled="submitting || !canSubmit"
        >
          {{ submitting ? t('admin.users.batchAssign.submitting') : t('admin.users.batchAssign.confirm') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { AdminGroup, SubscriptionType } from '@/types'
import type { BatchAssignUsersResult } from '@/api/admin/users'
import BaseDialog from '@/components/common/BaseDialog.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

type BatchAction = 'balance' | 'subscription'
type BalanceOperation = 'add' | 'subtract'

interface GroupOption extends Record<string, unknown> {
  value: number
  label: string
  description: string | null
  platform: AdminGroup['platform']
  subscriptionType: SubscriptionType
  rate: number
}

const props = defineProps<{
  show: boolean
  groups: AdminGroup[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'success'): void
  (e: 'load-groups'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const submitting = ref(false)
const lastResult = ref<BatchAssignUsersResult | null>(null)
const form = reactive({
  action: 'balance' as BatchAction,
  balanceOperation: 'add' as BalanceOperation,
  amount: 0,
  groupId: null as number | null,
  validityDays: 30,
  notes: ''
})

const subscriptionGroupOptions = computed<GroupOption[]>(() =>
  props.groups
    .filter((group) => group.subscription_type === 'subscription' && group.status === 'active')
    .map((group) => ({
      value: group.id,
      label: group.name,
      description: group.description,
      platform: group.platform,
      subscriptionType: group.subscription_type,
      rate: group.rate_multiplier
    }))
)

const canSubmit = computed(() => {
  if (form.action === 'balance') {
    return form.amount > 0
  }
  return !!form.groupId && form.validityDays > 0 && form.validityDays <= 36500
})

watch(
  () => props.show,
  (visible) => {
    if (!visible) return
    lastResult.value = null
    form.action = 'balance'
    form.balanceOperation = 'add'
    form.amount = 0
    form.groupId = null
    form.validityDays = 30
    form.notes = ''
    emit('load-groups')
  }
)

const actionButtonClass = (action: BatchAction) => [
  'flex min-h-[76px] items-center gap-3 rounded-lg border px-4 py-3 text-sm transition-colors',
  form.action === action
    ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-950/40 dark:text-primary-300'
    : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:border-dark-500'
]

const modeButtonClass = (operation: BalanceOperation) => [
  'rounded-lg border px-3 py-2 text-sm font-medium transition-colors',
  form.balanceOperation === operation
    ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-950/40 dark:text-primary-300'
    : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:border-dark-500'
]

const handleClose = () => {
  if (!submitting.value) {
    emit('close')
  }
}

const handleSubmit = async () => {
  if (!canSubmit.value) {
    appStore.showError(t('admin.users.batchAssign.invalidInput'))
    return
  }
  submitting.value = true
  lastResult.value = null
  try {
    const result = await adminAPI.users.batchAssign({
      all: true,
      ...(form.action === 'balance'
        ? {
            balance: {
              operation: form.balanceOperation,
              amount: form.amount,
              notes: form.notes
            }
          }
        : {
            subscription: {
              group_id: form.groupId!,
              validity_days: form.validityDays,
              notes: form.notes
            }
          })
    })
    lastResult.value = result
    appStore.showSuccess(t('admin.users.batchAssign.success', {
      success: result.success_count,
      failed: result.failed_count
    }))
    emit('success')
  } catch (error: any) {
    console.error('Failed to batch assign users:', error)
    appStore.showError(error.response?.data?.detail || t('admin.users.batchAssign.failed'))
  } finally {
    submitting.value = false
  }
}
</script>
