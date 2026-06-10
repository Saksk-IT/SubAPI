<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ t('admin.dataDashboard.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.dataDashboard.description') }}
          </p>
        </div>
        <button
          type="button"
          class="btn btn-secondary w-full justify-center md:w-auto"
          :disabled="loading"
          @click="loadDailyMetrics"
        >
          <Icon name="refresh" size="sm" class="mr-2" :class="loading ? 'animate-spin' : ''" />
          {{ t('admin.dataDashboard.refresh') }}
        </button>
      </div>

      <div class="card p-4">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.dataDashboard.timeRange') }}
            </p>
            <p v-if="metrics" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.dataDashboard.rangeSummary', { start: metrics.start_date, end: metrics.end_date }) }}
            </p>
          </div>
          <DateRangePicker
            v-model:start-date="startDate"
            v-model:end-date="endDate"
            @change="onDateRangeChange"
          />
        </div>
      </div>

      <div v-if="loading && !metrics" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <template v-else>
        <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
          <div class="card p-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-amber-100 p-2 dark:bg-amber-900/30">
                <Icon name="database" size="md" class="text-amber-600 dark:text-amber-400" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dataDashboard.totalTokens') }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ formatTokens(totals.total_tokens) }}
                </p>
              </div>
            </div>
          </div>

          <div class="card p-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-emerald-100 p-2 dark:bg-emerald-900/30">
                <Icon name="userPlus" size="md" class="text-emerald-600 dark:text-emerald-400" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dataDashboard.newUsers') }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ formatNumber(totals.new_users) }}
                </p>
              </div>
            </div>
          </div>

          <div class="card p-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-blue-100 p-2 dark:bg-blue-900/30">
                <Icon name="users" size="md" class="text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t('admin.dataDashboard.activeUsers') }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ formatNumber(totals.active_users) }}
                </p>
              </div>
            </div>
          </div>
        </div>

        <div class="card p-4">
          <div class="mb-4 flex flex-col gap-1">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.dataDashboard.trendTitle') }}
            </h2>
            <p v-if="metrics" class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.dataDashboard.rangeSummary', { start: metrics.start_date, end: metrics.end_date }) }}
            </p>
          </div>

          <div v-if="loading" class="flex h-80 items-center justify-center">
            <LoadingSpinner />
          </div>
          <div v-else-if="hasTrendData && chartData" class="h-80">
            <Line :data="chartData" :options="chartOptions" />
          </div>
          <div v-else class="flex h-80 flex-col items-center justify-center text-center">
            <Icon name="inbox" size="xl" class="mb-3 text-gray-400 dark:text-gray-500" />
            <p class="text-sm font-medium text-gray-600 dark:text-gray-300">
              {{ t('admin.dataDashboard.noData') }}
            </p>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.dataDashboard.noDataHint') }}
            </p>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'
import { adminAPI } from '@/api/admin'
import type { DailyMetricsResponse } from '@/api/admin/dashboard'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
)

const { t } = useI18n()
const appStore = useAppStore()

const formatLocalDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const getDefaultRange = (): { start: string; end: string } => {
  const end = new Date()
  const start = new Date()
  start.setDate(start.getDate() - 29)
  return {
    start: formatLocalDate(start),
    end: formatLocalDate(end)
  }
}

const defaultRange = getDefaultRange()
const startDate = ref(defaultRange.start)
const endDate = ref(defaultRange.end)
const metrics = ref<DailyMetricsResponse | null>(null)
const loading = ref(false)
let loadSeq = 0

const emptyTotals = {
  total_tokens: 0,
  new_users: 0,
  active_users: 0
}

const totals = computed(() => metrics.value?.totals ?? emptyTotals)

const hasTrendData = computed(() => {
  const series = metrics.value?.series ?? []
  return series.some((point) => (
    point.total_tokens > 0 ||
    point.new_users > 0 ||
    point.active_users > 0
  ))
})

const isDarkMode = computed(() => document.documentElement.classList.contains('dark'))

const chartColors = computed(() => ({
  text: isDarkMode.value ? '#e5e7eb' : '#374151',
  grid: isDarkMode.value ? '#374151' : '#e5e7eb',
  tokens: '#f59e0b',
  newUsers: '#10b981',
  activeUsers: '#3b82f6'
}))

const chartData = computed(() => {
  const series = metrics.value?.series ?? []
  if (!series.length) return null

  return {
    labels: series.map((point) => point.date),
    datasets: [
      {
        label: t('admin.dataDashboard.tokenTrend'),
        data: series.map((point) => point.total_tokens),
        borderColor: chartColors.value.tokens,
        backgroundColor: `${chartColors.value.tokens}20`,
        fill: false,
        tension: 0.3,
        yAxisID: 'yTokens'
      },
      {
        label: t('admin.dataDashboard.newUsersTrend'),
        data: series.map((point) => point.new_users),
        borderColor: chartColors.value.newUsers,
        backgroundColor: `${chartColors.value.newUsers}20`,
        fill: false,
        tension: 0.3,
        yAxisID: 'yUsers'
      },
      {
        label: t('admin.dataDashboard.activeUsersTrend'),
        data: series.map((point) => point.active_users),
        borderColor: chartColors.value.activeUsers,
        backgroundColor: `${chartColors.value.activeUsers}20`,
        fill: false,
        tension: 0.3,
        yAxisID: 'yUsers'
      }
    ]
  }
})

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 15,
        font: {
          size: 11
        }
      }
    },
    tooltip: {
      callbacks: {
        label: (context: any) => {
          const rawValue = Number(context.raw ?? 0)
          const value = context.dataset.yAxisID === 'yTokens'
            ? formatTokens(rawValue)
            : formatNumber(rawValue)
          return `${context.dataset.label}: ${value}`
        }
      }
    }
  },
  scales: {
    x: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        }
      }
    },
    yTokens: {
      type: 'linear' as const,
      position: 'left' as const,
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.tokens,
        font: {
          size: 10
        },
        callback: (value: string | number) => formatTokens(Number(value))
      }
    },
    yUsers: {
      type: 'linear' as const,
      position: 'right' as const,
      grid: {
        drawOnChartArea: false
      },
      ticks: {
        color: chartColors.value.activeUsers,
        font: {
          size: 10
        },
        callback: (value: string | number) => formatNumber(Number(value))
      }
    }
  }
}))

const formatTokens = (value: number): string => {
  if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(2)}B`
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(2)}K`
  return value.toLocaleString()
}

const formatNumber = (value: number): string => value.toLocaleString()

const loadDailyMetrics = async () => {
  const currentSeq = ++loadSeq
  loading.value = true
  try {
    const response = await adminAPI.dashboard.getDailyMetrics({
      start_date: startDate.value,
      end_date: endDate.value
    })
    if (currentSeq !== loadSeq) return
    metrics.value = {
      ...response,
      series: response.series ?? [],
      totals: response.totals ?? emptyTotals
    }
  } catch (error) {
    if (currentSeq !== loadSeq) return
    appStore.showError(t('admin.dataDashboard.failedToLoad'))
    console.error('Error loading data dashboard metrics:', error)
  } finally {
    if (currentSeq === loadSeq) {
      loading.value = false
    }
  }
}

const onDateRangeChange = (range: { startDate: string; endDate: string; preset: string | null }) => {
  startDate.value = range.startDate
  endDate.value = range.endDate
  void loadDailyMetrics()
}

onMounted(() => {
  void loadDailyMetrics()
})
</script>
