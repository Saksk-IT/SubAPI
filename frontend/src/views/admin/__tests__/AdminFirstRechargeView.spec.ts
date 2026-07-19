import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import AdminFirstRechargeView from '@/views/admin/activities/AdminFirstRechargeView.vue'

const mocks = vi.hoisted(() => ({
  internalPaymentEnabled: true,
  configEnabled: false,
  getFirstRecharge: vi.fn(),
  listFirstRechargeUsers: vi.fn(),
  getDashboard: vi.fn(),
  getOrders: vi.fn(),
}))

vi.mock('@/api/admin/activities', () => {
  const api = {
    getFirstRecharge: mocks.getFirstRecharge,
    listFirstRechargeUsers: mocks.listFirstRechargeUsers,
    lookupFirstRechargeUsers: vi.fn(),
    addFirstRechargeUser: vi.fn(),
    removeFirstRechargeUser: vi.fn(),
    updateFirstRecharge: vi.fn(),
  }
  return { adminActivitiesAPI: api, default: api }
})

vi.mock('@/api/admin/payment', () => {
  const api = {
    getDashboard: mocks.getDashboard,
    getOrders: mocks.getOrders,
    getOrder: vi.fn(),
  }
  return { adminPaymentAPI: api, default: api }
})

function mountView() {
  return mount(AdminFirstRechargeView, {
    global: {
      plugins: [
        createPinia(),
        createI18n({
          legacy: false,
          locale: 'zh',
          messages: { zh: {} },
          missingWarn: false,
          fallbackWarn: false,
        }),
      ],
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        FirstRechargeDashboard: { template: '<div data-testid="first-recharge-dashboard" />' },
        FirstRechargeOrderList: { template: '<div data-testid="first-recharge-orders" />' },
        AdminOrderDetail: true,
        Icon: true,
        LoadingSpinner: true,
        Pagination: true,
        Select: true,
      },
    },
  })
}

describe('AdminFirstRechargeView entry modes', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.internalPaymentEnabled = true
    mocks.configEnabled = false
    mocks.getFirstRecharge.mockImplementation(async () => ({
      data: {
        config: {
          enabled: mocks.configEnabled,
          eligibility_scope: 'all_users',
          purchase_mode: 'internal_payment',
          product_url: '',
        },
        offers: [],
        internal_payment_enabled: mocks.internalPaymentEnabled,
      },
    }))
    mocks.listFirstRechargeUsers.mockResolvedValue({
      data: { items: [], total: 0, page: 1, page_size: 10 },
    })
    mocks.getDashboard.mockResolvedValue({ data: {} })
    mocks.getOrders.mockResolvedValue({
      data: { items: [], total: 0, page: 1, page_size: 20 },
    })
  })

  it('uses two mutually exclusive switches and only shows internal-payment components for that mode', async () => {
    const wrapper = mountView()
    await flushPromises()

    const productLinkSwitch = wrapper.get('[data-testid="product-link-switch"]')
    const internalPaymentSwitch = wrapper.get('[data-testid="internal-payment-switch"]')
    expect(productLinkSwitch.attributes('aria-checked')).toBe('false')
    expect(internalPaymentSwitch.attributes('aria-checked')).toBe('false')
    expect(wrapper.find('[data-testid="first-recharge-dashboard"]').exists()).toBe(false)

    await productLinkSwitch.trigger('click')
    expect(productLinkSwitch.attributes('aria-checked')).toBe('true')
    expect(wrapper.find('input[type="url"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="first-recharge-dashboard"]').exists()).toBe(false)

    await internalPaymentSwitch.trigger('click')
    expect(productLinkSwitch.attributes('aria-checked')).toBe('false')
    expect(internalPaymentSwitch.attributes('aria-checked')).toBe('true')
    expect(wrapper.find('input[type="url"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="first-recharge-dashboard"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="first-recharge-orders"]').exists()).toBe(true)
  })

  it('disables the internal-payment switch when system payment is disabled', async () => {
    mocks.internalPaymentEnabled = false
    mocks.configEnabled = true
    const wrapper = mountView()
    await flushPromises()

    const internalPaymentSwitch = wrapper.get('[data-testid="internal-payment-switch"]')
    expect(internalPaymentSwitch.attributes('disabled')).toBeDefined()
    await internalPaymentSwitch.trigger('click')
    expect(internalPaymentSwitch.attributes('aria-checked')).toBe('false')
    expect(wrapper.find('[data-testid="first-recharge-dashboard"]').exists()).toBe(false)
    expect(mocks.getDashboard).not.toHaveBeenCalled()
    expect(mocks.getOrders).not.toHaveBeenCalled()
  })
})
