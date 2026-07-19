<template>
  <AppLayout :hide-first-recharge-banner="true">
    <div class="mx-auto flex w-full max-w-[1240px] flex-col gap-6">
      <section class="relative overflow-hidden rounded-2xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-800 sm:p-8">
        <div class="pointer-events-none absolute -right-16 -top-20 h-64 w-64 rounded-full bg-primary-200/30 blur-3xl dark:bg-primary-500/10"></div>
        <div class="relative flex items-start gap-4">
          <span class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-primary-600 text-white shadow-lg shadow-primary-500/25">
            <Icon name="sparkles" size="lg" />
          </span>
          <div>
            <h1 class="text-2xl font-black text-gray-950 dark:text-white sm:text-3xl">
              {{ t('activities.title') }}
            </h1>
            <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-gray-300 sm:text-base">
              {{ t('activities.description') }}
            </p>
          </div>
        </div>
      </section>

      <section v-if="loading && !loaded" class="card flex min-h-64 items-center justify-center">
        <LoadingSpinner size="lg" />
      </section>

      <section
        v-else-if="activities.length === 0"
        class="card flex min-h-72 flex-col items-center justify-center p-8 text-center"
        data-testid="activities-empty"
      >
        <span class="flex h-16 w-16 items-center justify-center rounded-2xl bg-gray-100 text-gray-400 dark:bg-dark-700 dark:text-gray-500">
          <Icon name="gift" size="xl" />
        </span>
        <h2 class="mt-5 text-lg font-black text-gray-950 dark:text-white">
          {{ t('activities.emptyTitle') }}
        </h2>
        <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
          {{ t('activities.emptyDescription') }}
        </p>
      </section>

      <section v-else class="space-y-6" data-testid="activities-list">
        <article
          v-for="activity in activities"
          :id="`activity-${activity.id}`"
          :key="activity.id"
          class="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800"
          :data-testid="`activity-card-${activity.id}`"
        >
          <template v-if="activity.type === 'daily_check_in' && activity.daily_check_in">
            <div class="border-b border-emerald-100 bg-emerald-50/70 p-6 dark:border-emerald-500/20 dark:bg-emerald-500/10 sm:p-8">
              <div class="flex flex-col gap-5 sm:flex-row sm:items-start sm:justify-between">
                <div class="flex items-start gap-4">
                  <span class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-emerald-600 text-white shadow-lg shadow-emerald-500/25">
                    <Icon name="calendar" size="lg" />
                  </span>
                  <div>
                    <h2 class="text-xl font-black text-gray-950 dark:text-white sm:text-2xl">
                      {{ t('activities.dailyCheckIn.title') }}
                    </h2>
                    <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-gray-300">
                      {{ t('activities.dailyCheckIn.description', {
                        amount: formatPrice(activity.daily_check_in.reward_amount, 'USD'),
                      }) }}
                    </p>
                  </div>
                </div>
                <button
                  type="button"
                  class="btn btn-primary shrink-0"
                  data-testid="daily-check-in-button"
                  :disabled="activity.daily_check_in.checked_in_today || checkingInId === activity.id"
                  @click="checkIn(activity)"
                >
                  <span
                    v-if="checkingInId === activity.id"
                    class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"
                  ></span>
                  <Icon v-else :name="activity.daily_check_in.checked_in_today ? 'checkCircle' : 'gift'" size="sm" />
                  {{ activity.daily_check_in.checked_in_today
                    ? t('activities.dailyCheckIn.checkedIn')
                    : t('activities.dailyCheckIn.checkIn') }}
                </button>
              </div>
            </div>

            <div class="grid gap-4 p-6 sm:grid-cols-2 sm:p-8">
              <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
                <p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('activities.dailyCheckIn.todayReward') }}
                </p>
                <p class="mt-2 text-xl font-black text-emerald-600 dark:text-emerald-400">
                  {{ formatPrice(activity.daily_check_in.reward_amount, 'USD') }}
                </p>
              </div>
              <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
                <p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('activities.dailyCheckIn.totalCheckIns') }}
                </p>
                <p class="mt-2 text-xl font-black text-gray-950 dark:text-white">
                  {{ t('activities.dailyCheckIn.totalCheckInsValue', {
                    count: activity.daily_check_in.total_check_ins,
                  }) }}
                </p>
              </div>
            </div>
          </template>

          <template v-if="activity.type === 'first_recharge' && activity.first_recharge">
            <div class="border-b border-amber-100 bg-amber-50/70 p-6 dark:border-amber-500/20 dark:bg-amber-500/10 sm:p-8">
              <div class="flex flex-col gap-5 sm:flex-row sm:items-start sm:justify-between">
                <div class="flex items-start gap-4">
                  <span class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-amber-500 text-white shadow-lg shadow-amber-500/25">
                    <Icon name="gift" size="lg" />
                  </span>
                  <div>
                    <h2 class="text-xl font-black text-gray-950 dark:text-white sm:text-2xl">
                      {{ t('activities.firstRecharge.title') }}
                    </h2>
                    <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-gray-300">
                      {{ activity.first_recharge.purchase_mode === 'product_link'
                        ? t('activities.firstRecharge.productLinkDescription')
                        : t('activities.firstRecharge.internalPaymentDescription') }}
                    </p>
                  </div>
                </div>
                <button type="button" class="btn btn-primary shrink-0" @click="participate(activity)">
                  {{ activity.first_recharge.purchase_mode === 'product_link'
                    ? t('activities.firstRecharge.openProduct')
                    : t('activities.firstRecharge.participate') }}
                  <Icon
                    :name="activity.first_recharge.purchase_mode === 'product_link' ? 'externalLink' : 'arrowRight'"
                    size="sm"
                  />
                </button>
              </div>
            </div>

            <div v-if="activity.first_recharge.purchase_mode === 'internal_payment'" class="p-6 sm:p-8">
              <div>
                <h3 class="text-base font-black text-gray-950 dark:text-white">
                  {{ t('activities.firstRecharge.offersTitle') }}
                </h3>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t('activities.firstRecharge.offersDescription') }}
                </p>
              </div>
              <div class="mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                <button
                  v-for="offer in activity.first_recharge.offers"
                  :key="offer.id"
                  type="button"
                  class="group rounded-2xl border border-gray-200 p-5 text-left transition hover:border-primary-300 hover:bg-primary-50/40 dark:border-dark-700 dark:hover:border-primary-500/40 dark:hover:bg-primary-500/5"
                  @click="participate(activity)"
                >
                  <span class="flex items-start justify-between gap-4">
                    <span>
                      <span class="block font-black text-gray-950 dark:text-white">{{ offer.name }}</span>
                      <span v-if="offer.description" class="mt-1 block text-sm leading-5 text-gray-500 dark:text-gray-400">
                        {{ offer.description }}
                      </span>
                    </span>
                    <Icon name="arrowRight" size="sm" class="mt-1 shrink-0 text-gray-400 transition-transform group-hover:translate-x-1 group-hover:text-primary-500" />
                  </span>
                  <span class="mt-5 flex items-end justify-between gap-4 border-t border-gray-100 pt-4 dark:border-dark-700">
                    <span>
                      <span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('activities.firstRecharge.pay') }}</span>
                      <span class="mt-1 block text-lg font-black text-gray-950 dark:text-white">{{ formatPrice(offer.price, 'CNY') }}</span>
                    </span>
                    <span class="text-right">
                      <span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('activities.firstRecharge.credit') }}</span>
                      <span class="mt-1 block text-lg font-black text-primary-600 dark:text-primary-400">{{ formatPrice(offer.amount, 'USD') }}</span>
                    </span>
                  </span>
                </button>
              </div>
            </div>

            <div v-else class="flex items-center gap-3 p-6 text-sm leading-6 text-gray-600 dark:text-gray-300 sm:px-8">
              <Icon name="link" size="md" class="shrink-0 text-primary-500" />
              {{ t('activities.firstRecharge.productLinkHint') }}
            </div>
          </template>
        </article>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useActivityStore } from '@/stores/activities'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { formatPaymentAmount } from '@/components/payment/currency'
import { extractApiErrorCode, extractI18nErrorMessage } from '@/utils/apiError'
import { activitiesAPI } from '@/api/activities'
import type { UserActivity } from '@/types/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

const { t, locale } = useI18n()
const router = useRouter()
const activityStore = useActivityStore()
const appStore = useAppStore()
const authStore = useAuthStore()
const { activities, loading, loaded } = storeToRefs(activityStore)
const checkingInId = ref('')

function formatPrice(value: number, currency: string): string {
  return formatPaymentAmount(value, currency, String(locale.value || ''))
}

function participate(activity: UserActivity) {
  const status = activity.first_recharge
  if (!status) return
  if (status.purchase_mode === 'product_link' && status.product_url) {
    window.location.assign(status.product_url)
    return
  }
  router.push({
    path: '/purchase',
    query: { tab: 'recharge', first_recharge: '1' },
  })
}

async function checkIn(activity: UserActivity) {
  const status = activity.daily_check_in
  if (!status || status.checked_in_today || checkingInId.value) return
  checkingInId.value = activity.id
  try {
    const response = await activitiesAPI.checkIn()
    status.checked_in_today = true
    status.total_check_ins += 1
    status.last_checked_in_at = response.data.checked_in_at
    authStore.refreshUser().catch(() => undefined)
    appStore.showSuccess(t('activities.dailyCheckIn.success', {
      amount: formatPrice(response.data.reward_amount, 'USD'),
    }))
  } catch (error: unknown) {
    if (extractApiErrorCode(error) === 'DAILY_CHECK_IN_ALREADY_DONE') {
      status.checked_in_today = true
    }
    appStore.showError(extractI18nErrorMessage(
      error,
      t,
      'activities.errors',
      t('activities.dailyCheckIn.failed'),
    ))
  } finally {
    checkingInId.value = ''
  }
}

onMounted(async () => {
  await activityStore.fetchActivities(true)
  try {
    await activityStore.markAllViewed()
  } catch {
    appStore.showError(t('activities.markViewedFailed'))
  }
})
</script>
