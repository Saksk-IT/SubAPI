<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-[1440px] flex-col gap-6">
      <section class="flex flex-col gap-4 rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800 sm:p-6 lg:flex-row lg:items-center lg:justify-between">
        <div class="min-w-0">
          <p class="text-xs font-black uppercase text-primary-600 dark:text-primary-400">
            {{ t('nav.activityManagement') }}
          </p>
          <h1 class="mt-2 text-2xl font-black text-gray-950 dark:text-white">
            {{ t('admin.firstRecharge.title') }}
          </h1>
          <p class="mt-2 max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-300">
            {{ t('admin.firstRecharge.description') }}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loading"
            @click="loadConfig(true)"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            {{ t('common.refresh') }}
          </button>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="saving || loading"
            @click="saveConfig"
          >
            <span v-if="saving" class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></span>
            <Icon v-else name="check" size="md" />
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </section>

      <div v-if="loading" class="card flex items-center justify-center py-16">
        <LoadingSpinner size="lg" />
      </div>

      <template v-else>
        <section class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
          <div class="space-y-6">
            <section class="card p-5 sm:p-6">
              <div class="grid gap-5 lg:grid-cols-2">
                <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
                  <div class="flex items-center justify-between gap-4">
                    <div>
                      <h2 class="text-lg font-black text-gray-950 dark:text-white">
                        {{ t('admin.firstRecharge.activitySwitch') }}
                      </h2>
                      <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">
                        {{ t('admin.firstRecharge.activitySwitchHint') }}
                      </p>
                    </div>
                    <Toggle v-model="form.enabled" />
                  </div>
                  <p v-if="form.eligible_since" class="mt-3 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.firstRecharge.eligibleSince', { time: formatDateTime(form.eligible_since) }) }}
                  </p>
                </div>

                <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
                  <label class="input-label">{{ t('admin.firstRecharge.scope') }}</label>
                  <Select
                    v-model="form.eligibility_scope"
                    :options="scopeOptions"
                    class="mt-2"
                    @change="handleScopeChange"
                  />
                  <p class="mt-3 text-sm leading-6 text-gray-500 dark:text-gray-400">
                    {{ scopeHint }}
                  </p>
                </div>
              </div>
            </section>

            <section class="card p-5 sm:p-6">
              <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h2 class="text-lg font-black text-gray-950 dark:text-white">
                    {{ t('admin.firstRecharge.offers') }}
                  </h2>
                  <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">
                    {{ t('admin.firstRecharge.offersHint') }}
                  </p>
                </div>
                <button
                  type="button"
                  class="btn btn-secondary"
                  :disabled="form.offers.length >= maxOffers"
                  @click="addOffer"
                >
                  <Icon name="plus" size="md" />
                  {{ t('admin.firstRecharge.addOffer') }}
                </button>
              </div>

              <div class="mt-5 space-y-4">
                <article
                  v-for="(offer, index) in form.offers"
                  :key="offer.local_id"
                  class="rounded-xl border border-gray-200 p-4 dark:border-dark-700"
                >
                  <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
                    <div class="flex items-center gap-2">
                      <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary-50 text-sm font-black text-primary-700 dark:bg-primary-500/15 dark:text-primary-200">
                        {{ index + 1 }}
                      </span>
                      <span class="text-sm font-semibold text-gray-600 dark:text-gray-300">
                        {{ offer.id ? t('admin.firstRecharge.existingOffer') : t('admin.firstRecharge.newOffer') }}
                      </span>
                    </div>
                    <div class="flex items-center gap-3">
                      <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
                        <Toggle v-model="offer.enabled" />
                        {{ t('common.enabled') }}
                      </label>
                      <button
                        type="button"
                        class="rounded-lg p-2 text-gray-500 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-500/10 dark:hover:text-red-300"
                        :title="t('common.delete')"
                        @click="removeOffer(index)"
                      >
                        <Icon name="trash" size="md" />
                      </button>
                    </div>
                  </div>

                  <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                    <div>
                      <label class="input-label">{{ t('admin.firstRecharge.offerName') }}</label>
                      <input v-model="offer.name" type="text" class="input" maxlength="80" />
                    </div>
                    <div>
                      <label class="input-label">{{ t('admin.firstRecharge.price') }}</label>
                      <input v-model.number="offer.price" type="number" min="0.01" step="0.01" class="input" />
                    </div>
                    <div>
                      <label class="input-label">{{ t('admin.firstRecharge.amount') }}</label>
                      <input v-model.number="offer.amount" type="number" min="0.01" step="0.01" class="input" />
                    </div>
                    <div>
                      <label class="input-label">{{ t('admin.firstRecharge.sortOrder') }}</label>
                      <input v-model.number="offer.sort_order" type="number" step="1" class="input" />
                    </div>
                  </div>
                  <div class="mt-4">
                    <label class="input-label">{{ t('admin.firstRecharge.offerDescription') }}</label>
                    <textarea v-model="offer.description" rows="2" class="input" maxlength="240"></textarea>
                  </div>
                </article>

                <div
                  v-if="form.offers.length === 0"
                  class="rounded-xl border border-dashed border-gray-300 px-4 py-10 text-center dark:border-dark-600"
                >
                  <Icon name="gift" size="xl" class="mx-auto text-gray-400" />
                  <p class="mt-3 text-sm text-gray-500 dark:text-gray-400">
                    {{ t('admin.firstRecharge.emptyOffers') }}
                  </p>
                </div>
              </div>
            </section>
          </div>

          <section class="card p-5 sm:p-6">
            <div class="flex items-start justify-between gap-3">
              <div>
                <h2 class="text-lg font-black text-gray-950 dark:text-white">
                  {{ t('admin.firstRecharge.specifiedUsers') }}
                </h2>
                <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">
                  {{ t('admin.firstRecharge.specifiedUsersHint') }}
                </p>
              </div>
              <span class="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-black text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                {{ specifiedUsersTotal }}
              </span>
            </div>

            <div class="mt-5 space-y-3">
              <label class="input-label">{{ t('admin.firstRecharge.searchUser') }}</label>
              <div class="flex gap-2">
                <input
                  v-model="lookupQuery"
                  type="text"
                  class="input"
                  :placeholder="t('admin.firstRecharge.searchUserPlaceholder')"
                  @keyup.enter="lookupUsers"
                />
                <button
                  type="button"
                  class="btn btn-secondary shrink-0"
                  :disabled="lookupLoading"
                  @click="lookupUsers"
                >
                  <Icon name="search" size="md" />
                </button>
              </div>
              <div v-if="lookupResults.length > 0" class="rounded-xl border border-gray-200 dark:border-dark-700">
                <button
                  v-for="user in lookupResults"
                  :key="user.id"
                  type="button"
                  class="flex w-full items-center justify-between gap-3 border-b border-gray-100 px-3 py-2 text-left last:border-b-0 hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-700/60"
                  @click="addSpecifiedUser(user.id)"
                >
                  <span class="min-w-0">
                    <span class="block truncate text-sm font-semibold text-gray-900 dark:text-white">
                      {{ user.username || user.email }}
                    </span>
                    <span class="block truncate text-xs text-gray-500 dark:text-gray-400">
                      #{{ user.id }} · {{ user.email }}
                    </span>
                  </span>
                  <Icon name="plus" size="sm" class="shrink-0 text-primary-500" />
                </button>
              </div>
            </div>

            <div class="mt-6 space-y-3">
              <div class="flex items-center gap-2">
                <input
                  v-model="specifiedSearch"
                  type="text"
                  class="input"
                  :placeholder="t('admin.firstRecharge.filterSpecifiedUsers')"
                  @keyup.enter="loadSpecifiedUsers(1)"
                />
                <button type="button" class="btn btn-secondary shrink-0" @click="loadSpecifiedUsers(1)">
                  <Icon name="refresh" size="md" />
                </button>
              </div>

              <div v-if="specifiedUsers.length > 0" class="space-y-2">
                <article
                  v-for="user in specifiedUsers"
                  :key="user.user_id"
                  class="flex items-center justify-between gap-3 rounded-xl border border-gray-200 p-3 dark:border-dark-700"
                >
                  <div class="min-w-0">
                    <p class="truncate text-sm font-semibold text-gray-950 dark:text-white">
                      {{ user.username || user.email }}
                    </p>
                    <p class="truncate text-xs text-gray-500 dark:text-gray-400">
                      #{{ user.user_id }} · {{ user.email }}
                    </p>
                    <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">
                      {{ t('admin.firstRecharge.addedAt', { time: formatDateTime(user.created_at) }) }}
                    </p>
                  </div>
                  <button
                    type="button"
                    class="rounded-lg p-2 text-gray-500 transition hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-500/10 dark:hover:text-red-300"
                    :title="t('common.delete')"
                    @click="removeSpecifiedUser(user.user_id)"
                  >
                    <Icon name="trash" size="md" />
                  </button>
                </article>
              </div>
              <p v-else class="rounded-xl bg-gray-50 px-4 py-8 text-center text-sm text-gray-500 dark:bg-dark-900/60 dark:text-gray-400">
                {{ t('admin.firstRecharge.emptySpecifiedUsers') }}
              </p>

              <Pagination
                v-if="specifiedUsersTotal > specifiedPageSize"
                :page="specifiedPage"
                :total="specifiedUsersTotal"
                :page-size="specifiedPageSize"
                @update:page="loadSpecifiedUsers"
                @update:pageSize="handleSpecifiedPageSizeChange"
              />
            </div>
          </section>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminActivitiesAPI, type FirstRechargeUserLookupItem } from '@/api/admin/activities'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import type {
  FirstRechargeEligibilityScope,
  FirstRechargeOffer,
  FirstRechargeOfferInput,
  FirstRechargeSpecifiedUser,
} from '@/types/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'

type OfferForm = FirstRechargeOfferInput & {
  local_id: string
}

const { t } = useI18n()
const appStore = useAppStore()

const maxOffers = 20
const loading = ref(true)
const saving = ref(false)
const lookupLoading = ref(false)
const lookupQuery = ref('')
const lookupResults = ref<FirstRechargeUserLookupItem[]>([])
const specifiedSearch = ref('')
const specifiedUsers = ref<FirstRechargeSpecifiedUser[]>([])
const specifiedUsersTotal = ref(0)
const specifiedPage = ref(1)
const specifiedPageSize = ref(10)

const form = reactive<{
  enabled: boolean
  eligibility_scope: FirstRechargeEligibilityScope
  eligible_since: string
  offers: OfferForm[]
}>({
  enabled: false,
  eligibility_scope: 'new_users_after_enabled',
  eligible_since: '',
  offers: [],
})

const scopeOptions = computed(() => [
  { value: 'new_users_after_enabled', label: t('admin.firstRecharge.scopeNewUsers') },
  { value: 'all_users', label: t('admin.firstRecharge.scopeAllUsers') },
  { value: 'specified_users', label: t('admin.firstRecharge.scopeSpecifiedUsers') },
])

const scopeHint = computed(() => {
  if (form.eligibility_scope === 'all_users') return t('admin.firstRecharge.scopeAllUsersHint')
  if (form.eligibility_scope === 'specified_users') return t('admin.firstRecharge.scopeSpecifiedUsersHint')
  return t('admin.firstRecharge.scopeNewUsersHint')
})

function toOfferForm(offer: FirstRechargeOffer): OfferForm {
  return {
    id: offer.id,
    name: offer.name,
    description: offer.description,
    price: Number(offer.price) || 0,
    amount: Number(offer.amount) || 0,
    enabled: offer.enabled,
    sort_order: Number(offer.sort_order) || 0,
    local_id: `offer-${offer.id}-${Date.now()}`,
  }
}

function newOffer(): OfferForm {
  const nextOrder = form.offers.length > 0
    ? Math.max(...form.offers.map((offer) => Number(offer.sort_order) || 0)) + 10
    : 10
  return {
    name: t('admin.firstRecharge.defaultOfferName'),
    description: '',
    price: 0,
    amount: 0,
    enabled: true,
    sort_order: nextOrder,
    local_id: `new-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
  }
}

async function loadConfig(force = false) {
  if (force) loading.value = true
  try {
    const [configResponse] = await Promise.all([
      adminActivitiesAPI.getFirstRecharge(),
      loadSpecifiedUsers(specifiedPage.value),
    ])
    const payload = configResponse.data
    form.enabled = payload.config.enabled
    form.eligibility_scope = payload.config.eligibility_scope
    form.eligible_since = payload.config.eligible_since || ''
    form.offers = [...payload.offers]
      .sort((a, b) => {
        if (a.sort_order === b.sort_order) return a.id - b.id
        return a.sort_order - b.sort_order
      })
      .map(toOfferForm)
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.firstRecharge.loadFailed')))
  } finally {
    loading.value = false
  }
}

function buildOfferPayload(): FirstRechargeOfferInput[] {
  return form.offers.map((offer) => ({
    id: offer.id,
    name: offer.name.trim(),
    description: offer.description.trim(),
    price: Number(offer.price) || 0,
    amount: Number(offer.amount) || 0,
    enabled: offer.enabled,
    sort_order: Math.trunc(Number(offer.sort_order) || 0),
  }))
}

function validateForm(): boolean {
  if (form.offers.length > maxOffers) {
    appStore.showError(t('admin.firstRecharge.maxOffersError', { count: maxOffers }))
    return false
  }
  const offers = buildOfferPayload()
  if (form.enabled && !offers.some((offer) => offer.enabled)) {
    appStore.showError(t('admin.firstRecharge.enabledOfferRequired'))
    return false
  }
  const invalid = offers.find((offer) => !offer.name || offer.price <= 0 || offer.amount <= 0)
  if (invalid) {
    appStore.showError(t('admin.firstRecharge.offerInvalid'))
    return false
  }
  return true
}

async function saveConfig() {
  if (saving.value || !validateForm()) return
  saving.value = true
  try {
    const response = await adminActivitiesAPI.updateFirstRecharge({
      enabled: form.enabled,
      eligibility_scope: form.eligibility_scope,
      offers: buildOfferPayload(),
    })
    form.eligible_since = response.data.config.eligible_since || ''
    form.offers = response.data.offers.map(toOfferForm)
    appStore.showSuccess(t('admin.firstRecharge.saveSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.firstRecharge.saveFailed')))
  } finally {
    saving.value = false
  }
}

function addOffer() {
  if (form.offers.length >= maxOffers) {
    appStore.showError(t('admin.firstRecharge.maxOffersError', { count: maxOffers }))
    return
  }
  form.offers = [...form.offers, newOffer()]
}

function removeOffer(index: number) {
  form.offers = form.offers.filter((_, currentIndex) => currentIndex !== index)
}

function handleScopeChange() {
  if (form.eligibility_scope === 'specified_users') {
    loadSpecifiedUsers(1).catch(() => {})
  }
}

async function lookupUsers() {
  const keyword = lookupQuery.value.trim()
  if (!keyword) {
    lookupResults.value = []
    return
  }
  lookupLoading.value = true
  try {
    const response = await adminActivitiesAPI.lookupFirstRechargeUsers(keyword)
    lookupResults.value = response.data
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.firstRecharge.lookupFailed')))
  } finally {
    lookupLoading.value = false
  }
}

async function loadSpecifiedUsers(page = specifiedPage.value) {
  specifiedPage.value = page
  try {
    const response = await adminActivitiesAPI.listFirstRechargeUsers({
      page: specifiedPage.value,
      page_size: specifiedPageSize.value,
      search: specifiedSearch.value.trim() || undefined,
    })
    specifiedUsers.value = response.data.items || []
    specifiedUsersTotal.value = response.data.total || 0
    specifiedPage.value = response.data.page || page
    specifiedPageSize.value = response.data.page_size || specifiedPageSize.value
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.firstRecharge.loadUsersFailed')))
  }
}

async function addSpecifiedUser(userId: number) {
  try {
    await adminActivitiesAPI.addFirstRechargeUser(userId)
    appStore.showSuccess(t('admin.firstRecharge.addUserSuccess'))
    lookupResults.value = lookupResults.value.filter((item) => item.id !== userId)
    await loadSpecifiedUsers(1)
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.firstRecharge.addUserFailed')))
  }
}

async function removeSpecifiedUser(userId: number) {
  try {
    await adminActivitiesAPI.removeFirstRechargeUser(userId)
    appStore.showSuccess(t('admin.firstRecharge.removeUserSuccess'))
    await loadSpecifiedUsers(specifiedPage.value)
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.firstRecharge.removeUserFailed')))
  }
}

function handleSpecifiedPageSizeChange(pageSize: number) {
  specifiedPageSize.value = pageSize
  loadSpecifiedUsers(1).catch(() => {})
}

onMounted(() => {
  loadConfig(true).catch(() => {})
})
</script>
