import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => params?.count != null ? `${key}:${params.count}` : key,
    }),
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
  }),
}))

vi.mock('@/api/admin', () => ({
  default: {
    groups: {
      getAll: vi.fn().mockResolvedValue([]),
    },
  },
}))

vi.mock('@/api/admin/payment', () => ({
  adminPaymentAPI: {
    getPlans: vi.fn().mockResolvedValue({ data: [] }),
    getBalanceProducts: vi.fn().mockResolvedValue({ data: [] }),
  },
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: { template: '<div><slot /></div>' },
}))

vi.mock('@/components/common/DataTable.vue', () => ({
  default: {
    name: 'DataTable',
    props: ['columns', 'data', 'loading', 'stickyFirstColumn'],
    template: '<div class="data-table-stub" />',
  },
}))

vi.mock('@/components/common/BaseDialog.vue', () => ({
  default: {
    props: ['show'],
    template: '<div v-if="show"><slot /><slot name="footer" /></div>',
  },
}))

vi.mock('@/components/common/ConfirmDialog.vue', () => ({
  default: {
    props: ['show'],
    template: '<div v-if="show"><slot /></div>',
  },
}))

vi.mock('@/components/common/GroupBadge.vue', () => ({
  default: {
    props: ['name'],
    template: '<span>{{ name }}</span>',
  },
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    template: '<span />',
  },
}))

vi.mock('vue-draggable-plus', () => ({
  VueDraggable: { template: '<div><slot /></div>' },
}))

vi.mock('../PlanEditDialog.vue', () => ({ default: { template: '<div />' } }))
vi.mock('../PlanBulkEditDialog.vue', () => ({ default: { template: '<div />' } }))
vi.mock('../BalanceProductEditDialog.vue', () => ({ default: { template: '<div />' } }))
vi.mock('../BalanceProductBulkEditDialog.vue', () => ({ default: { template: '<div />' } }))

import AdminPaymentPlansView from '../AdminPaymentPlansView.vue'

describe('AdminPaymentPlansView', () => {
  it('把订阅商品名称列放在选择列之后', () => {
    const wrapper = mount(AdminPaymentPlansView, {
      global: {
        stubs: {
          RouterLink: true,
        },
      },
    })

    const planColumns = (wrapper.vm as any).planColumns
    expect(planColumns[0].key).toBe('select')
    expect(planColumns[1].key).toBe('name')
    expect(planColumns[2].key).toBe('id')
    expect(planColumns[1].class).toContain('w-64')
    expect(planColumns[2].class).toContain('w-16')
    expect(wrapper.findComponent({ name: 'DataTable' }).props('stickyFirstColumn')).toBe(false)
  })

  it('充值商品表格也有选择列', () => {
    const wrapper = mount(AdminPaymentPlansView, {
      global: {
        stubs: {
          RouterLink: true,
        },
      },
    })

    const balanceColumns = (wrapper.vm as any).balanceProductColumns
    expect(balanceColumns[0].key).toBe('select')
    expect(balanceColumns[1].key).toBe('name')
    expect(balanceColumns[1].class).toContain('w-56')
    expect(balanceColumns[2].class).toContain('w-16')
    expect(wrapper.findComponent({ name: 'DataTable' }).props('stickyFirstColumn')).toBe(false)
  })
})
