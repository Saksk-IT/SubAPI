<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-[1080px] flex-col gap-6">
      <section class="flex flex-col gap-4 rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800 sm:p-6 lg:flex-row lg:items-center lg:justify-between">
        <div class="min-w-0">
          <p class="text-xs font-black uppercase text-primary-600 dark:text-primary-400">
            {{ t('nav.activityManagement') }}
          </p>
          <h1 class="mt-2 text-2xl font-black text-gray-950 dark:text-white">
            {{ t('admin.dailyCheckIn.title') }}
          </h1>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-300">
            {{ t('admin.dailyCheckIn.description') }}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadConfig">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            {{ t('common.refresh') }}
          </button>
          <button type="button" class="btn btn-primary" :disabled="saving || loading" @click="saveConfig">
            <span v-if="saving" class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
            <Icon v-else name="check" size="md" />
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </section>

      <section v-if="loading" class="card flex min-h-64 items-center justify-center">
        <LoadingSpinner size="lg" />
      </section>

      <template v-else>
        <section class="card p-5 sm:p-6" data-testid="daily-check-in-config">
          <div class="flex items-start justify-between gap-5">
            <div>
              <h2 class="text-lg font-black text-gray-950 dark:text-white">
                {{ t('admin.dailyCheckIn.activitySwitch') }}
              </h2>
              <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">
                {{ t('admin.dailyCheckIn.activitySwitchHint') }}
              </p>
            </div>
            <Toggle
              v-model="form.enabled"
              data-testid="daily-check-in-switch"
              :aria-label="t('admin.dailyCheckIn.activitySwitch')"
            />
          </div>

          <div class="mt-6 border-t border-gray-200 pt-6 dark:border-dark-700">
            <label class="input-label" for="daily-check-in-reward">
              {{ t('admin.dailyCheckIn.rewardAmount') }}
            </label>
            <div class="mt-2 flex max-w-md items-center gap-3">
              <input
                id="daily-check-in-reward"
                v-model.number="form.reward_amount"
                class="input"
                data-testid="daily-check-in-reward"
                type="number"
                min="0.00000001"
                max="1000000"
                step="0.01"
              />
              <span class="shrink-0 text-sm font-semibold text-gray-500 dark:text-gray-400">USD</span>
            </div>
            <p class="mt-3 max-w-2xl text-sm leading-6 text-gray-500 dark:text-gray-400">
              {{ t('admin.dailyCheckIn.rewardAmountHint') }}
            </p>
          </div>
        </section>

        <section class="rounded-2xl border border-emerald-200 bg-emerald-50/70 p-5 dark:border-emerald-500/20 dark:bg-emerald-500/10 sm:p-6">
          <div class="flex items-start gap-4">
            <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-emerald-600 text-white">
              <Icon name="calendar" size="md" />
            </span>
            <div>
              <h2 class="font-black text-gray-950 dark:text-white">
                {{ t('admin.dailyCheckIn.ruleTitle') }}
              </h2>
              <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-300">
                {{ t('admin.dailyCheckIn.ruleDescription', { timezone: timezone }) }}
              </p>
            </div>
          </div>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminActivitiesAPI } from '@/api/admin/activities'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Toggle from '@/components/common/Toggle.vue'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(true)
const saving = ref(false)
const timezone = ref('Asia/Shanghai')
const form = reactive({ enabled: false, reward_amount: 1 })

async function loadConfig() {
  loading.value = true
  try {
    const response = await adminActivitiesAPI.getDailyCheckIn()
    form.enabled = response.data.enabled
    form.reward_amount = Number(response.data.reward_amount) || 0
    timezone.value = response.data.timezone || 'Asia/Shanghai'
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.dailyCheckIn.loadFailed')))
  } finally {
    loading.value = false
  }
}

function validReward(): boolean {
  const amount = Number(form.reward_amount)
  return Number.isFinite(amount) && amount > 0 && amount <= 1_000_000
}

async function saveConfig() {
  if (saving.value) return
  if (!validReward()) {
    appStore.showError(t('admin.dailyCheckIn.rewardInvalid'))
    return
  }
  saving.value = true
  try {
    const response = await adminActivitiesAPI.updateDailyCheckIn({
      enabled: form.enabled,
      reward_amount: Number(form.reward_amount),
    })
    form.enabled = response.data.enabled
    form.reward_amount = Number(response.data.reward_amount)
    timezone.value = response.data.timezone || timezone.value
    appStore.showSuccess(t('admin.dailyCheckIn.saveSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.dailyCheckIn.saveFailed')))
  } finally {
    saving.value = false
  }
}

onMounted(loadConfig)
</script>
