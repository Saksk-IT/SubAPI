import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AdminDataDashboardView from '../AdminDataDashboardView.vue'

const { getDailyMetrics, showError } = vi.hoisted(() => ({
  getDailyMetrics: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    dashboard: {
      getDailyMetrics
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError
  })
}))

vi.mock('vue-chartjs', () => ({
  Line: {
    name: 'Line',
    props: ['data', 'options'],
    template: '<div data-test="daily-metrics-chart"></div>'
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string>) => {
        if (params?.start && params?.end) {
          return `${key}:${params.start}-${params.end}`
        }
        return key
      }
    })
  }
})

const mountView = () => mount(AdminDataDashboardView, {
  global: {
    stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      LoadingSpinner: true,
      Icon: true,
      DateRangePicker: {
        props: ['startDate', 'endDate'],
        emits: ['update:startDate', 'update:endDate', 'change'],
        template: `
          <button
            data-test="date-range"
            @click="$emit('update:startDate', '2026-04-01'); $emit('update:endDate', '2026-04-02'); $emit('change', { startDate: '2026-04-01', endDate: '2026-04-02', preset: null })"
          >
            date range
          </button>
        `
      }
    }
  }
})

describe('AdminDataDashboardView', () => {
  beforeEach(() => {
    getDailyMetrics.mockReset()
    showError.mockReset()
    getDailyMetrics.mockResolvedValue({
      start_date: '2026-03-01',
      end_date: '2026-03-03',
      series: [
        { date: '2026-03-01', total_tokens: 100, new_users: 2, active_users: 5 },
        { date: '2026-03-02', total_tokens: 0, new_users: 0, active_users: 0 },
        { date: '2026-03-03', total_tokens: 300, new_users: 1, active_users: 4 }
      ],
      totals: {
        total_tokens: 400,
        new_users: 3,
        active_users: 9
      }
    })
  })

  it('loads daily metrics on mount', async () => {
    mountView()
    await flushPromises()

    expect(getDailyMetrics).toHaveBeenCalledTimes(1)
    expect(getDailyMetrics).toHaveBeenCalledWith(expect.objectContaining({
      start_date: expect.any(String),
      end_date: expect.any(String)
    }))
  })

  it('reloads daily metrics when date range changes', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="date-range"]').trigger('click')
    await flushPromises()

    expect(getDailyMetrics).toHaveBeenCalledTimes(2)
    expect(getDailyMetrics).toHaveBeenLastCalledWith({
      start_date: '2026-04-01',
      end_date: '2026-04-02'
    })
  })

  it('shows empty state when all values are zero', async () => {
    getDailyMetrics.mockResolvedValueOnce({
      start_date: '2026-03-01',
      end_date: '2026-03-02',
      series: [
        { date: '2026-03-01', total_tokens: 0, new_users: 0, active_users: 0 },
        { date: '2026-03-02', total_tokens: 0, new_users: 0, active_users: 0 }
      ],
      totals: {
        total_tokens: 0,
        new_users: 0,
        active_users: 0
      }
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('admin.dataDashboard.noData')
    expect(wrapper.find('[data-test="daily-metrics-chart"]').exists()).toBe(false)
  })

  it('maps chart data to token, new-user, and active-user datasets', async () => {
    const wrapper = mountView()
    await flushPromises()

    const line = wrapper.findComponent({ name: 'Line' })
    expect(line.exists()).toBe(true)
    const data = line.props('data') as { datasets: Array<{ label: string; data: number[]; yAxisID: string }> }

    expect(data.datasets).toHaveLength(3)
    expect(data.datasets.map((dataset) => dataset.label)).toEqual([
      'admin.dataDashboard.tokenTrend',
      'admin.dataDashboard.newUsersTrend',
      'admin.dataDashboard.activeUsersTrend'
    ])
    expect(data.datasets[0].data).toEqual([100, 0, 300])
    expect(data.datasets[1].data).toEqual([2, 0, 1])
    expect(data.datasets[2].data).toEqual([5, 0, 4])
    expect(data.datasets[0].yAxisID).toBe('yTokens')
    expect(data.datasets[1].yAxisID).toBe('yUsers')
    expect(data.datasets[2].yAxisID).toBe('yUsers')
  })
})
