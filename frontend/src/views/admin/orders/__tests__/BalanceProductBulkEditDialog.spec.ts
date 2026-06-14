import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  bulkUpdateBalanceProducts: vi.fn(),
}))

const appStoreMocks = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin/payment', () => ({
  adminPaymentAPI: {
    bulkUpdateBalanceProducts: apiMocks.bulkUpdateBalanceProducts,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMocks,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => params?.count != null ? `${key}:${params.count}` : key,
    }),
  }
})

import BalanceProductBulkEditDialog from '../BalanceProductBulkEditDialog.vue'

function mountDialog() {
  return mount(BalanceProductBulkEditDialog, {
    props: {
      show: true,
      productIds: [11, 22],
    },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
      },
    },
  })
}

describe('BalanceProductBulkEditDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMocks.bulkUpdateBalanceProducts.mockResolvedValue({ data: { updated: 2 } })
  })

  it('只提交已勾选的批量修改字段', async () => {
    const wrapper = mountDialog()
    const checkboxes = wrapper.findAll('input[type="checkbox"]')

    await checkboxes[0].setValue(true)
    await checkboxes[2].setValue(true)
    const textareas = wrapper.findAll('textarea')
    await textareas[0].setValue('新版描述')
    await textareas[2].setValue('标签 A\n标签 B')

    await wrapper.get('#balance-product-bulk-edit-form').trigger('submit.prevent')
    await flushPromises()

    expect(apiMocks.bulkUpdateBalanceProducts).toHaveBeenCalledWith({
      product_ids: [11, 22],
      fields: {
        description: '新版描述',
        tags: '标签 A\n标签 B',
      },
    })
  })

  it('未勾选任何字段时不提交请求', async () => {
    const wrapper = mountDialog()

    await wrapper.get('#balance-product-bulk-edit-form').trigger('submit.prevent')
    await flushPromises()

    expect(apiMocks.bulkUpdateBalanceProducts).not.toHaveBeenCalled()
    expect(appStoreMocks.showError).toHaveBeenCalledWith('payment.admin.bulkEditBalanceProductsNoFields')
  })
})
