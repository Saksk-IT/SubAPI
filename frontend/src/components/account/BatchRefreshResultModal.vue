<template>
  <BaseDialog
    :show="show"
    :title="title"
    width="extra-wide"
    @close="emit('close')"
  >
    <div v-if="summary" class="space-y-4">
      <div class="grid gap-3 sm:grid-cols-3">
        <div class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium text-gray-400 dark:text-dark-500">
            {{ t('admin.accounts.batchRefreshResult.total') }}
          </div>
          <div class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">
            {{ summary.total }}
          </div>
        </div>
        <div class="rounded-lg border border-emerald-200 bg-emerald-50/70 p-3 dark:border-emerald-900/40 dark:bg-emerald-950/20">
          <div class="text-xs font-medium text-emerald-700 dark:text-emerald-300">
            {{ t('admin.accounts.batchRefreshResult.success') }}
          </div>
          <div class="mt-1 text-2xl font-semibold text-emerald-700 dark:text-emerald-300">
            {{ summary.success }}
          </div>
        </div>
        <div class="rounded-lg border border-rose-200 bg-rose-50/70 p-3 dark:border-rose-900/40 dark:bg-rose-950/20">
          <div class="text-xs font-medium text-rose-700 dark:text-rose-300">
            {{ t('admin.accounts.batchRefreshResult.failed') }}
          </div>
          <div class="mt-1 text-2xl font-semibold text-rose-700 dark:text-rose-300">
            {{ summary.failed }}
          </div>
        </div>
      </div>

      <div class="grid gap-4 xl:grid-cols-2">
        <section class="rounded-lg border border-emerald-200 bg-white dark:border-emerald-900/40 dark:bg-dark-800">
          <div class="flex items-center justify-between gap-3 border-b border-emerald-100 px-4 py-3 dark:border-emerald-900/30">
            <div class="flex items-center gap-2">
              <Icon name="checkCircle" size="sm" class="text-emerald-600 dark:text-emerald-400" />
              <h4 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('admin.accounts.batchRefreshResult.successAccounts') }}
              </h4>
            </div>
            <span class="text-xs text-emerald-600 dark:text-emerald-300">
              {{ successAccounts.length }}
            </span>
          </div>
          <div class="max-h-[38vh] space-y-2 overflow-auto p-4">
            <div
              v-if="successAccounts.length === 0"
              class="rounded-lg border border-dashed border-emerald-200 px-3 py-4 text-center text-sm text-gray-500 dark:border-emerald-900/40 dark:text-dark-400"
            >
              {{ t('admin.accounts.batchRefreshResult.noSuccessAccounts') }}
            </div>
            <article
              v-for="account in successAccounts"
              :key="`success-${account.account_id}`"
              class="rounded-lg border border-emerald-100 bg-emerald-50/40 px-3 py-2.5 dark:border-emerald-900/30 dark:bg-emerald-950/10"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <div class="truncate text-sm font-medium text-gray-900 dark:text-white">
                    {{ account.name }}
                  </div>
                  <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                    <span class="rounded-full bg-white px-2 py-0.5 dark:bg-dark-700">
                      {{ displayPlatformType(account) }}
                    </span>
                    <span v-if="account.warning" class="rounded-full bg-amber-100 px-2 py-0.5 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">
                      {{ t('admin.accounts.batchRefreshResult.warning') }}
                    </span>
                  </div>
                </div>
                <Icon name="checkCircle" size="sm" class="mt-0.5 text-emerald-500 dark:text-emerald-400" />
              </div>
              <div v-if="account.warning" class="mt-2 text-xs text-amber-700 dark:text-amber-300">
                {{ account.warning }}
              </div>
            </article>
          </div>
        </section>

        <section class="rounded-lg border border-rose-200 bg-white dark:border-rose-900/40 dark:bg-dark-800">
          <div class="flex items-center justify-between gap-3 border-b border-rose-100 px-4 py-3 dark:border-rose-900/30">
            <div class="flex items-center gap-2">
              <Icon name="xCircle" size="sm" class="text-rose-600 dark:text-rose-400" />
              <h4 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('admin.accounts.batchRefreshResult.failedAccounts') }}
              </h4>
            </div>
            <span class="text-xs text-rose-600 dark:text-rose-300">
              {{ failedAccounts.length }}
            </span>
          </div>
          <div class="max-h-[38vh] space-y-2 overflow-auto p-4">
            <div
              v-if="failedAccounts.length === 0"
              class="rounded-lg border border-dashed border-rose-200 px-3 py-4 text-center text-sm text-gray-500 dark:border-rose-900/40 dark:text-dark-400"
            >
              {{ t('admin.accounts.batchRefreshResult.noFailedAccounts') }}
            </div>
            <article
              v-for="account in failedAccounts"
              :key="`failed-${account.account_id}`"
              class="rounded-lg border border-rose-100 bg-rose-50/40 px-3 py-2.5 dark:border-rose-900/30 dark:bg-rose-950/10"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0 flex-1">
                  <div class="truncate text-sm font-medium text-gray-900 dark:text-white">
                    {{ account.name }}
                  </div>
                  <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                    <span class="rounded-full bg-white px-2 py-0.5 dark:bg-dark-700">
                      {{ displayPlatformType(account) }}
                    </span>
                    <span v-if="account.schedulingDisabled" class="rounded-full bg-amber-100 px-2 py-0.5 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">
                      {{ t('admin.accounts.batchRefreshResult.schedulingDisabled') }}
                    </span>
                  </div>
                </div>
                <Icon name="xCircle" size="sm" class="mt-0.5 text-rose-500 dark:text-rose-400" />
              </div>
              <div v-if="account.error" class="mt-2 text-xs text-rose-700 dark:text-rose-300">
                {{ account.error }}
              </div>
            </article>
          </div>
        </section>
      </div>
    </div>

    <template #footer>
      <div class="ml-auto flex flex-wrap gap-2">
        <button
          v-if="failedAccounts.length > 0"
          class="btn btn-danger"
          type="button"
          :disabled="deletingFailed || disablingFailed"
          @click="emit('delete-failed')"
        >
          <Icon :name="deletingFailed ? 'refresh' : 'trash'" size="sm" :class="deletingFailed ? 'animate-spin' : ''" />
          <span>{{ t('admin.accounts.batchRefreshResult.deleteFailed') }}</span>
        </button>
        <button
          v-if="failedAccounts.length > 0"
          class="btn btn-warning"
          type="button"
          :disabled="deletingFailed || disablingFailed || !hasFailedAccountsToDisable"
          @click="emit('disable-failed')"
        >
          <Icon :name="disablingFailed ? 'refresh' : 'ban'" size="sm" :class="disablingFailed ? 'animate-spin' : ''" />
          <span>{{ t('admin.accounts.batchRefreshResult.disableFailed') }}</span>
        </button>
        <button class="btn btn-secondary" type="button" @click="emit('close')">
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, toRefs } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

export interface BatchRefreshResultAccount {
  account_id: number
  name: string
  platform: string
  type: string
  success: boolean
  schedulingDisabled?: boolean
  error?: string
  warning?: string
}

export interface BatchRefreshResultSummary {
  total: number
  success: number
  failed: number
}

interface Props {
  show: boolean
  title: string
  summary: BatchRefreshResultSummary | null
  successAccounts: BatchRefreshResultAccount[]
  failedAccounts: BatchRefreshResultAccount[]
  deletingFailed?: boolean
  disablingFailed?: boolean
}

interface Emits {
  (e: 'close'): void
  (e: 'delete-failed'): void
  (e: 'disable-failed'): void
}

const props = withDefaults(defineProps<Props>(), {
  deletingFailed: false,
  disablingFailed: false
})

const emit = defineEmits<Emits>()
const { t } = useI18n()
const { summary, successAccounts, failedAccounts, show, title, deletingFailed, disablingFailed } = toRefs(props)

const hasFailedAccountsToDisable = computed(() => failedAccounts.value.some(account => !account.schedulingDisabled))

const displayPlatformType = (account: BatchRefreshResultAccount) => {
  const parts = [account.platform, account.type].filter(Boolean)
  return parts.length > 0 ? parts.join(' / ') : `#${account.account_id}`
}
</script>
