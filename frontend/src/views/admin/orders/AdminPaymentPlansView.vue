<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="flex rounded-xl bg-gray-100 p-1 dark:bg-dark-800">
        <button
          class="flex-1 rounded-lg px-4 py-2.5 text-sm font-medium transition-all"
          :class="activeProductTab === 'balance' ? 'bg-white text-gray-900 shadow dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'"
          @click="activeProductTab = 'balance'"
        >
          {{ t('payment.admin.balanceProducts') }}
        </button>
        <button
          class="flex-1 rounded-lg px-4 py-2.5 text-sm font-medium transition-all"
          :class="activeProductTab === 'subscription' ? 'bg-white text-gray-900 shadow dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'"
          @click="activeProductTab = 'subscription'"
        >
          {{ t('payment.admin.subscriptionProducts') }}
        </button>
      </div>

      <!-- Actions -->
      <div class="flex items-center justify-end gap-2">
        <button
          v-if="activeProductTab === 'balance' && selectedBalanceProductIds.length > 0"
          @click="showBulkBalanceProductDialog = true"
          class="btn btn-primary"
        >
          <Icon name="edit" size="sm" />
          <span>{{ t('payment.admin.bulkEditBalanceProducts') }} ({{ selectedBalanceProductIds.length }})</span>
        </button>
        <button
          v-if="activeProductTab === 'subscription' && selectedPlanIds.length > 0"
          @click="showBulkPlanDialog = true"
          class="btn btn-primary"
        >
          <Icon name="edit" size="sm" />
          <span>{{ t('payment.admin.bulkEditPlans') }} ({{ selectedPlanIds.length }})</span>
        </button>
        <button @click="openSortDialog" :disabled="activeLoading" class="btn btn-secondary" :title="t('payment.admin.sortProducts')">
          <Icon name="sort" size="md" />
          <span>{{ t('payment.admin.sortProducts') }}</span>
        </button>
        <button @click="activeProductTab === 'balance' ? loadBalanceProducts() : loadPlans()" :disabled="activeLoading" class="btn btn-secondary" :title="t('common.refresh')">
          <Icon name="refresh" size="md" :class="activeLoading ? 'animate-spin' : ''" />
        </button>
        <button v-if="activeProductTab === 'balance'" @click="openBalanceProductEdit(null)" class="btn btn-primary">{{ t('payment.admin.createBalanceProduct') }}</button>
        <button v-else @click="openPlanEdit(null)" class="btn btn-primary">{{ t('payment.admin.createPlan') }}</button>
      </div>

      <DataTable
        v-if="activeProductTab === 'balance'"
        :columns="balanceProductColumns"
        :data="balanceProducts"
        :loading="balanceProductsLoading"
        :sticky-first-column="false"
      >
        <template #header-select>
          <input
            type="checkbox"
            class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            :checked="allBalanceProductsSelected"
            @change="toggleAllBalanceProducts($event)"
          />
        </template>
        <template #cell-select="{ row }">
          <input
            type="checkbox"
            class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            :checked="isBalanceProductSelected(row.id)"
            @change="toggleBalanceProductSelection(row.id)"
          />
        </template>
        <template #cell-price="{ value, row }">
          <div class="text-sm">
            <span class="font-medium text-gray-900 dark:text-white">¥{{ (value ?? 0).toFixed(2) }}</span>
            <span v-if="row.original_price" class="ml-1 text-xs text-gray-400 line-through">¥{{ row.original_price.toFixed(2) }}</span>
          </div>
        </template>
        <template #cell-amount="{ value }">
          <span class="text-sm font-medium text-gray-900 dark:text-white">${{ (value ?? 0).toFixed(2) }}</span>
        </template>
        <template #cell-purchase_limit="{ value }">
          <span class="text-sm text-gray-700 dark:text-gray-300">{{ formatPurchaseLimit(value) }}</span>
        </template>
        <template #cell-for_sale="{ value, row }">
          <button
            type="button"
            :class="[
              'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
              value ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
            ]"
            @click="toggleBalanceProductForSale(row)"
          >
            <span :class="[
              'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
              value ? 'translate-x-4' : 'translate-x-0'
            ]" />
          </button>
        </template>
        <template #cell-actions="{ row }">
          <div class="flex items-center gap-2">
            <button @click="openBalanceProductEdit(row)" class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400">
              <Icon name="edit" size="sm" />
              <span class="text-xs">{{ t('common.edit') }}</span>
            </button>
            <button @click="confirmDeleteBalanceProduct(row)" class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">
              <Icon name="trash" size="sm" />
              <span class="text-xs">{{ t('common.delete') }}</span>
            </button>
          </div>
        </template>
      </DataTable>

      <template v-else>
        <!-- Plans Table -->
        <DataTable
          :columns="planColumns"
          :data="plans"
          :loading="plansLoading"
          :sticky-first-column="false"
        >
          <template #header-select>
            <input
              type="checkbox"
              class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="allPlansSelected"
              @change="toggleAllPlans($event)"
            />
          </template>
          <template #cell-select="{ row }">
            <input
              type="checkbox"
              class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="isPlanSelected(row.id)"
              @change="togglePlanSelection(row.id)"
            />
          </template>
          <template #cell-name="{ value, row }">
            <span class="block min-w-0 whitespace-normal break-words text-sm font-medium leading-5" :class="getPlanNameClass(row.group_id)">{{ value }}</span>
          </template>
          <template #cell-price_multiplier="{ value }">
            <span class="text-sm font-medium text-gray-900 dark:text-white">{{ Number(value || 0).toFixed(4) }}x</span>
          </template>
          <template #cell-group_id="{ value }">
            <span v-if="isGroupMissing(value)" class="text-sm">
              <span class="text-gray-400">#{{ value }}</span>
              <span class="ml-1 badge badge-danger">{{ t('payment.admin.groupMissing') }}</span>
            </span>
            <GroupBadge
              v-else-if="getGroup(value)"
              :name="getGroup(value)!.name"
              :platform="getGroup(value)!.platform"
              :rate-multiplier="getGroup(value)!.rate_multiplier"
            />
            <span v-else class="text-sm text-gray-400">-</span>
          </template>
          <template #cell-price="{ value, row }">
            <div class="text-sm">
              <span class="font-medium text-gray-900 dark:text-white">¥{{ (value ?? 0).toFixed(2) }}</span>
              <span v-if="row.original_price" class="ml-1 text-xs text-gray-400 line-through">¥{{ row.original_price.toFixed(2) }}</span>
            </div>
          </template>
          <template #cell-validity_days="{ value, row }">
            <span class="text-sm">{{ value }} {{ t('payment.admin.' + (row.validity_unit || 'days')) }}</span>
          </template>
          <template #cell-for_sale="{ value, row }">
            <button
              type="button"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
                value ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
              ]"
              @click="toggleForSale(row)"
            >
              <span :class="[
                'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                value ? 'translate-x-4' : 'translate-x-0'
              ]" />
            </button>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center gap-2">
              <button @click="openPlanEdit(row)" class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400">
                <Icon name="edit" size="sm" />
                <span class="text-xs">{{ t('common.edit') }}</span>
              </button>
              <button @click="confirmDeletePlan(row)" class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400">
                <Icon name="trash" size="sm" />
                <span class="text-xs">{{ t('common.delete') }}</span>
              </button>
            </div>
          </template>
        </DataTable>
      </template>
    </div>

    <!-- Plan Edit Dialog -->
    <PlanEditDialog :show="showPlanDialog" :plan="editingPlan" :groups="groups" @close="showPlanDialog = false" @saved="loadPlans" />
    <BalanceProductEditDialog :show="showBalanceProductDialog" :product="editingBalanceProduct" @close="showBalanceProductDialog = false" @saved="loadBalanceProducts" />
    <BalanceProductBulkEditDialog :show="showBulkBalanceProductDialog" :product-ids="selectedBalanceProductIds" @close="showBulkBalanceProductDialog = false" @updated="handleBulkBalanceProductUpdated" />
    <PlanBulkEditDialog :show="showBulkPlanDialog" :plan-ids="selectedPlanIds" @close="showBulkPlanDialog = false" @updated="handleBulkPlanUpdated" />

    <BaseDialog
      :show="showSortDialog"
      :title="t('payment.admin.sortProducts')"
      width="normal"
      @close="closeSortDialog"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ sortDialogHint }}
        </p>
        <div class="max-h-[65vh] overflow-y-auto pr-1">
          <VueDraggable
            v-if="sortingProductTab === 'balance'"
            v-model="sortableBalanceProducts"
            :animation="200"
            class="space-y-2"
            handle=".product-sort-handle"
          >
            <div
              v-for="product in sortableBalanceProducts"
              :key="product.id"
              class="flex cursor-grab items-center gap-3 rounded-lg border border-gray-200 bg-white p-3 transition-shadow hover:shadow-md active:cursor-grabbing dark:border-dark-600 dark:bg-dark-700"
            >
              <div class="product-sort-handle text-gray-400">
                <Icon name="menu" size="md" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="truncate font-medium text-gray-900 dark:text-white">
                  {{ product.name }}
                </div>
                <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                  <span class="rounded-full bg-emerald-100 px-2 py-0.5 font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400">
                    ${{ (product.amount ?? 0).toFixed(2) }}
                  </span>
                  <span>¥{{ (product.price ?? 0).toFixed(2) }}</span>
                </div>
              </div>
              <div class="text-sm text-gray-400">#{{ product.id }}</div>
            </div>
          </VueDraggable>

          <VueDraggable
            v-else
            v-model="sortablePlanPlatformGroups"
            :animation="200"
            class="space-y-3"
            handle=".platform-sort-handle"
          >
            <div
              v-for="platformGroup in sortablePlanPlatformGroups"
              :key="platformGroup.platform"
              class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-700"
            >
              <div class="mb-3 flex cursor-grab items-center gap-3 active:cursor-grabbing">
                <div class="platform-sort-handle text-gray-400">
                  <Icon name="menu" size="md" />
                </div>
                <div class="min-w-0 flex-1">
                  <div class="truncate text-sm font-black text-gray-900 dark:text-white">
                    {{ platformGroup.label }}
                  </div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">
                    {{ t('payment.admin.platformPlanCount', { count: platformGroup.plans.length }) }}
                  </div>
                </div>
                <span class="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-500 dark:bg-dark-800 dark:text-gray-300">
                  {{ t('payment.admin.platformSortOrder') }}
                </span>
              </div>
              <VueDraggable
                v-model="platformGroup.plans"
                :animation="200"
                class="space-y-2"
                handle=".product-sort-handle"
              >
                <div
                  v-for="plan in platformGroup.plans"
                  :key="plan.id"
                  class="flex cursor-grab items-center gap-3 rounded-lg border border-gray-200 bg-gray-50 p-3 transition-shadow hover:shadow-md active:cursor-grabbing dark:border-dark-600 dark:bg-dark-800"
                >
                  <div class="product-sort-handle text-gray-400">
                    <Icon name="menu" size="md" />
                  </div>
                  <div class="min-w-0 flex-1">
                    <div class="truncate font-medium" :class="getPlanNameClass(plan.group_id)">
                      {{ plan.name }}
                    </div>
                    <div class="mt-1 flex flex-wrap items-center gap-2">
                      <GroupBadge
                        v-if="getGroup(plan.group_id)"
                        :name="getGroup(plan.group_id)!.name"
                        :platform="getGroup(plan.group_id)!.platform"
                        :rate-multiplier="getGroup(plan.group_id)!.rate_multiplier"
                      />
                      <span v-else class="text-xs text-gray-400">#{{ plan.group_id }}</span>
                      <span class="text-xs text-gray-500 dark:text-gray-400">¥{{ (plan.price ?? 0).toFixed(2) }}</span>
                    </div>
                  </div>
                  <div class="text-sm text-gray-400">#{{ plan.id }}</div>
                </div>
              </VueDraggable>
            </div>
          </VueDraggable>
        </div>
      </div>

      <template #footer>
        <div class="flex justify-end gap-3 pt-4">
          <button
            type="button"
            class="btn btn-secondary"
            @click="closeSortDialog"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            class="btn btn-primary"
            :disabled="sortSubmitting || sortableProductCount === 0"
            @click="saveSortOrder"
          >
            <Icon v-if="sortSubmitting" name="refresh" size="sm" class="animate-spin" />
            <span>{{ sortSubmitting ? t('common.saving') : t('common.save') }}</span>
          </button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog :show="showDeletePlanDialog" :title="t('payment.admin.deletePlan')" :message="t('payment.admin.deletePlanConfirm')" :confirm-text="t('common.delete')" danger @confirm="handleDeletePlan" @cancel="showDeletePlanDialog = false" />
    <ConfirmDialog :show="showDeleteBalanceProductDialog" :title="t('payment.admin.deleteBalanceProduct')" :message="t('payment.admin.deleteBalanceProductConfirm')" :confirm-text="t('common.delete')" danger @confirm="handleDeleteBalanceProduct" @cancel="showDeleteBalanceProductDialog = false" />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { VueDraggable } from 'vue-draggable-plus'
import { useAppStore } from '@/stores/app'
import { adminPaymentAPI } from '@/api/admin/payment'
import { extractI18nErrorMessage } from '@/utils/apiError'
import adminAPI from '@/api/admin'
import type { BalanceProduct, SubscriptionPlan } from '@/types/payment'
import type { AdminGroup } from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import PlanEditDialog from './PlanEditDialog.vue'
import PlanBulkEditDialog from './PlanBulkEditDialog.vue'
import BalanceProductEditDialog from './BalanceProductEditDialog.vue'
import BalanceProductBulkEditDialog from './BalanceProductBulkEditDialog.vue'
import { platformLabel, platformTextClass } from '@/utils/platformColors'

const { t } = useI18n()
const appStore = useAppStore()
type ProductTab = 'balance' | 'subscription'
type PlanPlatformSortGroup = {
  platform: string
  label: string
  plans: SubscriptionPlan[]
}

const activeProductTab = ref<ProductTab>('balance')
const activeLoading = computed(() => activeProductTab.value === 'balance' ? balanceProductsLoading.value : plansLoading.value)

// ==================== Groups ====================

const groups = ref<AdminGroup[]>([])

async function loadGroups() {
  try {
    groups.value = await adminAPI.groups.getAll()
  } catch { /* ignore */ }
}

function getGroup(id: number): AdminGroup | undefined {
  return groups.value.find(g => g.id === id)
}

function isGroupMissing(id: number): boolean {
  return id > 0 && !groups.value.find(g => g.id === id)
}

function getPlanNameClass(groupId: number): string {
  const group = getGroup(groupId)
  return group ? platformTextClass(group.platform) : 'text-gray-900 dark:text-white'
}


// ==================== Plans ====================

const plansLoading = ref(false)
const plans = ref<SubscriptionPlan[]>([])
const showPlanDialog = ref(false)
const showDeletePlanDialog = ref(false)
const showBulkPlanDialog = ref(false)
const editingPlan = ref<SubscriptionPlan | null>(null)
const deletingPlanId = ref<number | null>(null)
const selectedPlanIds = ref<number[]>([])

const balanceProductsLoading = ref(false)
const balanceProducts = ref<BalanceProduct[]>([])
const showBalanceProductDialog = ref(false)
const showDeleteBalanceProductDialog = ref(false)
const showBulkBalanceProductDialog = ref(false)
const editingBalanceProduct = ref<BalanceProduct | null>(null)
const deletingBalanceProductId = ref<number | null>(null)
const selectedBalanceProductIds = ref<number[]>([])
const showSortDialog = ref(false)
const sortSubmitting = ref(false)
const sortingProductTab = ref<ProductTab>('balance')
const sortableBalanceProducts = ref<BalanceProduct[]>([])
const sortablePlanPlatformGroups = ref<PlanPlatformSortGroup[]>([])

const sortDialogHint = computed(() =>
  sortingProductTab.value === 'balance'
    ? t('payment.admin.balanceSortOrderHint')
    : t('payment.admin.planSortOrderHint'),
)
const sortableProductCount = computed(() =>
  sortingProductTab.value === 'balance'
    ? sortableBalanceProducts.value.length
    : sortablePlanPlatformGroups.value.reduce((count, group) => count + group.plans.length, 0),
)

const balanceProductColumns = computed((): Column[] => [
  { key: 'select', label: '', class: 'w-10 min-w-10 max-w-10' },
  { key: 'name', label: t('payment.admin.productName'), class: 'w-56 min-w-56 max-w-56 whitespace-normal' },
  { key: 'id', label: 'ID', class: 'w-16 min-w-16 max-w-16' },
  { key: 'price', label: t('payment.admin.payPrice') },
  { key: 'amount', label: t('payment.admin.creditAmount') },
  { key: 'sales_count', label: t('payment.admin.salesCount') },
  { key: 'purchase_limit', label: t('payment.admin.purchaseLimit') },
  { key: 'for_sale', label: t('payment.admin.forSale') },
  { key: 'actions', label: t('common.actions') },
])

const planColumns = computed((): Column[] => [
  { key: 'select', label: '', class: 'w-10 min-w-10 max-w-10' },
  { key: 'name', label: t('payment.admin.planName'), class: 'w-64 min-w-64 max-w-64 whitespace-normal' },
  { key: 'id', label: 'ID', class: 'w-16 min-w-16 max-w-16' },
  { key: 'group_id', label: t('payment.admin.group') },
  { key: 'price_multiplier', label: t('payment.admin.planPriceMultiplier') },
  { key: 'price', label: t('payment.admin.price') },
  { key: 'validity_days', label: t('payment.admin.validityDays') },
  { key: 'sales_count', label: t('payment.admin.salesCount') },
  { key: 'for_sale', label: t('payment.admin.forSale') },
  { key: 'actions', label: t('common.actions') },
])

function formatPurchaseLimit(value: unknown): string {
  const limit = Number(value) || 0
  return limit > 0 ? String(limit) : t('payment.admin.unlimited')
}

async function loadPlans() {
  plansLoading.value = true
  try {
    const res = await adminPaymentAPI.getPlans()
    // Backend returns features as newline-separated string; parse to array
    plans.value = (res.data || []).map((p: Omit<SubscriptionPlan, 'features'> & { features: string | string[] }) => ({
      ...p,
      features: typeof p.features === 'string'
        ? p.features.split('\n').map((f: string) => f.trim()).filter(Boolean)
        : (p.features || []),
    }))
    const visibleIDs = new Set(plans.value.map(plan => plan.id))
    selectedPlanIds.value = selectedPlanIds.value.filter(id => visibleIDs.has(id))
  }
  catch (err: unknown) { appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error'))) }
  finally { plansLoading.value = false }
}

async function loadBalanceProducts() {
  balanceProductsLoading.value = true
  try {
    const res = await adminPaymentAPI.getBalanceProducts()
    balanceProducts.value = (res.data || []).map((p) => ({
      ...p,
      tags: typeof p.tags === 'string' ? p.tags.split('\n').map(item => item.trim()).filter(Boolean) : (p.tags || []),
      features: typeof p.features === 'string' ? p.features.split('\n').map(item => item.trim()).filter(Boolean) : (p.features || []),
    }))
    const visibleIDs = new Set(balanceProducts.value.map(product => product.id))
    selectedBalanceProductIds.value = selectedBalanceProductIds.value.filter(id => visibleIDs.has(id))
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    balanceProductsLoading.value = false
  }
}

function bySortOrder<T extends { id: number; sort_order: number }>(items: T[]): T[] {
  return [...items].sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
}

function normalizePlatformKey(platform: string | null | undefined): string {
  return String(platform || '').trim().toLowerCase() || 'unknown'
}

function formatPlatformLabel(platform: string | null | undefined): string {
  const key = normalizePlatformKey(platform)
  if (key === 'unknown') return t('payment.admin.unknownPlatform')
  return platformLabel(key) || t('payment.admin.unknownPlatform')
}

function getPlanPlatform(plan: SubscriptionPlan): string {
  const group = getGroup(plan.group_id)
  return normalizePlatformKey(group?.platform || plan.group_platform)
}

function buildPlanPlatformGroups(planItems: SubscriptionPlan[]): PlanPlatformSortGroup[] {
  const plansByPlatform = new Map<string, SubscriptionPlan[]>()
  const firstSeenPlatforms: string[] = []
  bySortOrder(planItems).forEach((plan) => {
    const platform = getPlanPlatform(plan)
    if (!plansByPlatform.has(platform)) {
      plansByPlatform.set(platform, [])
      firstSeenPlatforms.push(platform)
    }
    plansByPlatform.get(platform)!.push(plan)
  })
  return firstSeenPlatforms.map(platform => ({
    platform,
    label: formatPlatformLabel(platform),
    plans: [...(plansByPlatform.get(platform) || [])],
  }))
}

const allPlansSelected = computed(() => plans.value.length > 0 && plans.value.every(plan => selectedPlanIds.value.includes(plan.id)))
const allBalanceProductsSelected = computed(() =>
  balanceProducts.value.length > 0 && balanceProducts.value.every(product => selectedBalanceProductIds.value.includes(product.id)),
)

function isPlanSelected(id: number): boolean {
  return selectedPlanIds.value.includes(id)
}

function isBalanceProductSelected(id: number): boolean {
  return selectedBalanceProductIds.value.includes(id)
}

function togglePlanSelection(id: number) {
  selectedPlanIds.value = isPlanSelected(id)
    ? selectedPlanIds.value.filter(selectedID => selectedID !== id)
    : [...selectedPlanIds.value, id]
}

function toggleBalanceProductSelection(id: number) {
  selectedBalanceProductIds.value = isBalanceProductSelected(id)
    ? selectedBalanceProductIds.value.filter(selectedID => selectedID !== id)
    : [...selectedBalanceProductIds.value, id]
}

function toggleAllPlans(event: Event) {
  const checked = (event.target as HTMLInputElement).checked
  selectedPlanIds.value = checked ? plans.value.map(plan => plan.id) : []
}

function toggleAllBalanceProducts(event: Event) {
  const checked = (event.target as HTMLInputElement).checked
  selectedBalanceProductIds.value = checked ? balanceProducts.value.map(product => product.id) : []
}

async function handleBulkPlanUpdated() {
  showBulkPlanDialog.value = false
  selectedPlanIds.value = []
  await loadPlans()
}

async function handleBulkBalanceProductUpdated() {
  showBulkBalanceProductDialog.value = false
  selectedBalanceProductIds.value = []
  await loadBalanceProducts()
}

function openSortDialog() {
  sortingProductTab.value = activeProductTab.value
  sortableBalanceProducts.value = bySortOrder(balanceProducts.value)
  if (activeProductTab.value === 'subscription') {
    sortablePlanPlatformGroups.value = buildPlanPlatformGroups(plans.value)
  }
  showSortDialog.value = true
}

function closeSortDialog() {
  if (sortSubmitting.value) return
  resetSortDialog()
}

function resetSortDialog() {
  showSortDialog.value = false
  sortableBalanceProducts.value = []
  sortablePlanPlatformGroups.value = []
}

async function saveSortOrder() {
  sortSubmitting.value = true
  try {
    if (sortingProductTab.value === 'balance') {
      const updates = sortableBalanceProducts.value.map((product, index) => ({ id: product.id, sort_order: index * 10 }))
      await adminPaymentAPI.updateBalanceProductSortOrder(updates)
      appStore.showSuccess(t('payment.admin.sortOrderUpdated'))
      resetSortDialog()
      await loadBalanceProducts()
      return
    }

    const orderedPlans = sortablePlanPlatformGroups.value.flatMap(group => group.plans)
    const updates = orderedPlans.map((plan, index) => ({ id: plan.id, sort_order: index * 10 }))
    await adminPaymentAPI.updatePlanSortOrder(updates)
    appStore.showSuccess(t('payment.admin.sortOrderUpdated'))
    resetSortDialog()
    await loadPlans()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('payment.admin.failedToUpdateSortOrder')))
  } finally {
    sortSubmitting.value = false
  }
}

function openPlanEdit(plan: SubscriptionPlan | null) {
  editingPlan.value = plan
  showPlanDialog.value = true
}

function openBalanceProductEdit(product: BalanceProduct | null) {
  editingBalanceProduct.value = product
  showBalanceProductDialog.value = true
}


/** Quick toggle for_sale from the list */
async function toggleForSale(plan: SubscriptionPlan) {
  try {
    await adminPaymentAPI.updatePlan(plan.id, { for_sale: !plan.for_sale })
    plans.value = plans.value.map(item =>
      item.id === plan.id ? { ...item, for_sale: !plan.for_sale } : item,
    )
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  }
}

async function toggleBalanceProductForSale(product: BalanceProduct) {
  try {
    await adminPaymentAPI.updateBalanceProduct(product.id, { for_sale: !product.for_sale })
    balanceProducts.value = balanceProducts.value.map(item =>
      item.id === product.id ? { ...item, for_sale: !product.for_sale } : item,
    )
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  }
}

function confirmDeletePlan(plan: SubscriptionPlan) { deletingPlanId.value = plan.id; showDeletePlanDialog.value = true }
async function handleDeletePlan() {
  if (!deletingPlanId.value) return
  try { await adminPaymentAPI.deletePlan(deletingPlanId.value); appStore.showSuccess(t('common.deleted')); showDeletePlanDialog.value = false; loadPlans() }
  catch (err: unknown) { appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error'))) }
}

function confirmDeleteBalanceProduct(product: BalanceProduct) {
  deletingBalanceProductId.value = product.id
  showDeleteBalanceProductDialog.value = true
}

async function handleDeleteBalanceProduct() {
  if (!deletingBalanceProductId.value) return
  try {
    await adminPaymentAPI.deleteBalanceProduct(deletingBalanceProductId.value)
    appStore.showSuccess(t('common.deleted'))
    showDeleteBalanceProductDialog.value = false
    loadBalanceProducts()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  }
}

// ==================== Lifecycle ====================

onMounted(() => {
  loadGroups()
  loadBalanceProducts()
  loadPlans()
})
</script>
