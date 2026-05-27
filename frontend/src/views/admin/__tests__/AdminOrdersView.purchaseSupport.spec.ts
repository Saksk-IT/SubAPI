import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

import AdminOrdersView from '../orders/AdminOrdersView.vue'

const {
  getOrders,
  getConfig,
  updateConfig,
  getSettings,
  updateSettings,
  showError,
  showSuccess,
  fetchPublicSettings,
} = vi.hoisted(() => ({
  getOrders: vi.fn(),
  getConfig: vi.fn(),
  updateConfig: vi.fn(),
  getSettings: vi.fn(),
  updateSettings: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  fetchPublicSettings: vi.fn(),
}))

vi.mock('@/api/admin/payment', () => {
  const adminPaymentAPI = {
    getOrders,
    getConfig,
    updateConfig,
    getOrder: vi.fn(),
    cancelOrder: vi.fn(),
    retryRecharge: vi.fn(),
    refundOrder: vi.fn(),
  }
  return {
    adminPaymentAPI,
    default: adminPaymentAPI,
  }
})

vi.mock('@/api/admin/settings', () => {
  const settingsAPI = {
    getSettings,
    updateSettings,
  }
  return {
    settingsAPI,
    default: settingsAPI,
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    fetchPublicSettings,
  }),
}))

vi.mock('@/utils/apiError', () => ({
  extractI18nErrorMessage: (_err: unknown, _t: unknown, _scope: string, fallback: string) => fallback,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const SelectStub = defineComponent({
  props: {
    modelValue: {
      type: [String, Number, Boolean, null],
      default: '',
    },
    options: {
      type: Array,
      default: () => [],
    },
  },
  template: '<div data-test="select-stub" />',
})
const OrderTableStub = { template: '<div data-test="order-table"></div>' }
const PaginationStub = { template: '<div data-test="pagination"></div>' }
const BaseDialogStub = { template: '<div><slot /><slot name="footer" /></div>' }
const AdminRefundDialogStub = { template: '<div data-test="refund-dialog"></div>' }

function mountView() {
  return mount(AdminOrdersView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        Select: SelectStub,
        OrderTable: OrderTableStub,
        Pagination: PaginationStub,
        BaseDialog: BaseDialogStub,
        AdminRefundDialog: AdminRefundDialogStub,
        OrderStatusBadge: true,
        Icon: true,
      },
    },
  })
}

describe('admin AdminOrdersView purchase support config', () => {
  beforeEach(() => {
    getOrders.mockReset().mockResolvedValue({
      data: {
        items: [],
        total: 0,
      },
    })
    getConfig.mockReset().mockResolvedValue({
      data: {
        help_image_url: 'https://example.test/old-qr.png',
        help_text: '旧购买说明',
      },
    })
    updateConfig.mockReset().mockResolvedValue({})
    getSettings.mockReset().mockResolvedValue({
      contact_info: '旧客服联系方式',
    })
    updateSettings.mockReset().mockResolvedValue({})
    showError.mockReset()
    showSuccess.mockReset()
    fetchPublicSettings.mockReset().mockResolvedValue(null)
  })

  it('loads and saves purchase page support details from order management', async () => {
    const wrapper = mountView()
    await flushPromises()

    const panel = wrapper.get('[data-test="purchase-support-config"]')
    const textareas = panel.findAll('textarea')
    const imageInput = panel.get('input[type="url"]')

    expect(textareas[0].element.value).toBe('旧客服联系方式')
    expect(imageInput.element.value).toBe('https://example.test/old-qr.png')
    expect(textareas[1].element.value).toBe('旧购买说明')

    await textareas[0].setValue('  微信: subapi-support  ')
    await imageInput.setValue('  https://example.test/new-qr.png  ')
    await textareas[1].setValue('  付款前请确认套餐和到账规则  ')
    const buttons = panel.findAll('button')
    await buttons[buttons.length - 1].trigger('click')
    await flushPromises()

    expect(updateSettings).toHaveBeenCalledWith({
      contact_info: '微信: subapi-support',
    })
    expect(updateConfig).toHaveBeenCalledWith({
      help_image_url: 'https://example.test/new-qr.png',
      help_text: '付款前请确认套餐和到账规则',
    })
    expect(fetchPublicSettings).toHaveBeenCalledWith(true)
    expect(showSuccess).toHaveBeenCalledWith('payment.admin.supportConfigSaved')
    expect(showError).not.toHaveBeenCalled()
  })
})
