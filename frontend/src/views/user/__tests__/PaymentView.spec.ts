import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import PaymentView from '../PaymentView.vue'
import AmountInput from '@/components/payment/AmountInput.vue'
import PurchaseProductCard from '@/components/payment/PurchaseProductCard.vue'
import { PAYMENT_RECOVERY_STORAGE_KEY } from '@/components/payment/paymentFlow'
import { formatPaymentAmount } from '@/components/payment/currency'
import type { CheckoutInfoResponse, MethodLimit, SubscriptionPlan } from '@/types/payment'

const routeState = vi.hoisted(() => ({
  path: '/purchase',
  query: {} as Record<string, unknown>,
}))

const routerReplace = vi.hoisted(() => vi.fn())
const routerPush = vi.hoisted(() => vi.fn())
const routerResolve = vi.hoisted(() => vi.fn(() => ({ href: '/payment/stripe?mock=1' })))
const createOrder = vi.hoisted(() => vi.fn())
const refreshUser = vi.hoisted(() => vi.fn())
const fetchActiveSubscriptions = vi.hoisted(() => vi.fn().mockResolvedValue(undefined))
const fetchFirstRechargeStatus = vi.hoisted(() => vi.fn().mockResolvedValue(null))
const showError = vi.hoisted(() => vi.fn())
const showInfo = vi.hoisted(() => vi.fn())
const showWarning = vi.hoisted(() => vi.fn())
const getCheckoutInfo = vi.hoisted(() => vi.fn())
const bridgeInvoke = vi.hoisted(() => vi.fn())
const appStoreState = vi.hoisted(() => ({
  contactInfo: 'QQ: 123456789',
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => routeState,
    useRouter: () => ({
      replace: routerReplace,
      push: routerPush,
      resolve: routerResolve,
    }),
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: {
      username: 'demo-user',
      balance: 0,
    },
    refreshUser,
  }),
}))

vi.mock('@/stores/payment', () => ({
  usePaymentStore: () => ({
    createOrder,
  }),
}))

vi.mock('@/stores/subscriptions', () => ({
  useSubscriptionStore: () => ({
    activeSubscriptions: [],
    fetchActiveSubscriptions,
  }),
}))

vi.mock('@/stores/firstRecharge', () => ({
  useFirstRechargeStore: () => ({
    status: null,
    available: false,
    fetchStatus: fetchFirstRechargeStatus,
    clear: vi.fn(),
  }),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    contactInfo: appStoreState.contactInfo,
    showError,
    showInfo,
    showWarning,
  }),
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getCheckoutInfo,
  },
}))

vi.mock('@/utils/device', () => ({
  isMobileDevice: () => true,
}))

function methodLimitFixture(overrides: Partial<MethodLimit> = {}): MethodLimit {
  return {
    daily_limit: 0,
    daily_used: 0,
    daily_remaining: 0,
    single_min: 0,
    single_max: 0,
    fee_rate: 0,
    available: true,
    ...overrides,
  }
}

function checkoutInfoFixture(overrides: Partial<CheckoutInfoResponse> = {}) {
  const data: CheckoutInfoResponse = {
    methods: {
      wxpay: methodLimitFixture(),
    },
    global_min: 0,
    global_max: 0,
    balance_products: [],
    plans: [],
    balance_disabled: false,
    balance_recharge_multiplier: 1,
    subscription_usd_to_cny_rate: 0,
    recharge_fee_rate: 0,
    help_text: '',
    help_image_url: '',
    stripe_publishable_key: '',
  }

  return {
    data: { ...data, ...overrides },
  }
}

function checkoutInfoWithBalanceProductsFixture() {
  return {
    data: {
      ...checkoutInfoFixture().data,
      methods: {
        alipay: methodLimitFixture(),
      },
      balance_products: [
        {
          id: 9,
          name: 'Light Day Pass',
          description: 'Today only',
          price: 6.9,
          amount: 69,
          original_price: 9.9,
          tags: ['Debug'],
          features: ['Light tasks'],
          product_name: '',
          purchase_limit: 0,
          sort_order: 1,
          for_sale: true,
        },
      ],
    },
  }
}

function checkoutInfoWithPlansFixture(options: {
  checkout?: Partial<CheckoutInfoResponse>
  method?: Partial<MethodLimit>
  plan?: Partial<SubscriptionPlan>
} = {}) {
  const base = checkoutInfoFixture(options.checkout).data
  const plan: SubscriptionPlan = {
    id: 7,
    group_id: 3,
    name: 'Starter',
    description: '',
    price: 128,
    original_price: 0,
    validity_days: 30,
    validity_unit: 'day',
    rate_multiplier: 1,
    daily_limit_usd: null,
    weekly_limit_usd: null,
    monthly_limit_usd: null,
    total_quota: 840,
    daily_quota: 120,
    features: [],
    group_platform: 'openai',
    sort_order: 1,
    for_sale: true,
    group_name: 'OpenAI',
    ...options.plan,
  }

  return {
    data: {
      ...base,
      methods: {
        ...base.methods,
        wxpay: {
          ...(base.methods.wxpay ?? methodLimitFixture()),
          ...options.method,
        },
      },
      plans: [plan],
    },
  }
}

function checkoutInfoWithPlatformPlansFixture() {
  return {
    data: {
      ...checkoutInfoFixture().data,
      methods: {
        wxpay: {
          daily_limit: 0,
          daily_used: 0,
          daily_remaining: 0,
          single_min: 0,
          single_max: 0,
          fee_rate: 0,
          available: true,
        },
      },
      plans: [
        {
          id: 21,
          group_id: 301,
          name: 'OpenAI Weekly',
          description: '',
          price: 68,
          original_price: 0,
          validity_days: 7,
          validity_unit: 'days',
          rate_multiplier: 1,
          daily_limit_usd: null,
          weekly_limit_usd: null,
          monthly_limit_usd: null,
          total_quota: 350,
          daily_quota: 50,
          features: [],
          group_platform: 'openai',
          sort_order: 30,
          for_sale: true,
          group_name: 'OpenAI',
        },
        {
          id: 22,
          group_id: 302,
          name: 'Claude Team',
          description: '',
          price: 188,
          original_price: 0,
          validity_days: 30,
          validity_unit: 'days',
          rate_multiplier: 1,
          daily_limit_usd: null,
          weekly_limit_usd: null,
          monthly_limit_usd: null,
          total_quota: 1200,
          daily_quota: 80,
          features: [],
          group_platform: 'anthropic',
          sort_order: 40,
          for_sale: true,
          group_name: 'Anthropic',
        },
        {
          id: 23,
          group_id: 303,
          name: 'Gemini Pro',
          description: '',
          price: 128,
          original_price: 0,
          validity_days: 30,
          validity_unit: 'days',
          rate_multiplier: 1,
          daily_limit_usd: null,
          weekly_limit_usd: null,
          monthly_limit_usd: null,
          total_quota: 900,
          daily_quota: 70,
          features: [],
          group_platform: 'gemini',
          sort_order: 10,
          for_sale: true,
          group_name: 'Gemini',
        },
        {
          id: 24,
          group_id: 301,
          name: 'OpenAI Monthly',
          description: '',
          price: 168,
          original_price: 0,
          validity_days: 30,
          validity_unit: 'days',
          rate_multiplier: 1,
          daily_limit_usd: null,
          weekly_limit_usd: null,
          monthly_limit_usd: null,
          total_quota: 1200,
          daily_quota: 90,
          features: [],
          group_platform: 'openai',
          sort_order: 20,
          for_sale: true,
          group_name: 'OpenAI',
        },
      ],
    },
  }
}

function checkoutInfoWithSupportFixture() {
  return {
    data: {
      ...checkoutInfoWithBalanceProductsFixture().data,
      help_text: '充值前可咨询客服确认套餐',
      help_image_url: 'https://example.test/support-qr.png',
    },
  }
}

function jsapiOrderFixture(resumeToken: string) {
  return {
    order_id: 123,
    amount: 88,
    pay_amount: 88,
    fee_rate: 0,
    expires_at: '2099-01-01T00:10:00.000Z',
    payment_type: 'wxpay',
    out_trade_no: 'sub2_jsapi_123',
    result_type: 'jsapi_ready' as const,
    resume_token: resumeToken,
    jsapi: {
      appId: 'wx123',
      timeStamp: '1712345678',
      nonceStr: 'nonce',
      package: 'prepay_id=wx123',
      signType: 'RSA',
      paySign: 'signed',
    },
  }
}

function oauthOrderFixture() {
  return {
    order_id: 456,
    amount: 128,
    pay_amount: 128,
    fee_rate: 0,
    expires_at: '2099-01-01T00:10:00.000Z',
    payment_type: 'wxpay',
    result_type: 'oauth_required' as const,
    oauth: {
      authorize_url: '/api/v1/auth/oauth/wechat/payment/start?payment_type=wxpay&redirect=%2Fpurchase%3Ffrom%3Dwechat',
      appid: 'wx123',
      scope: 'snsapi_base',
      redirect_url: '/auth/wechat/payment/callback',
    },
  }
}

async function mountSubscriptionConfirm(options: Parameters<typeof checkoutInfoWithPlansFixture>[0] = {}) {
  vi.useRealTimers()
  routeState.path = '/purchase'
  routeState.query = {
    tab: 'subscription',
    group: '3',
  }
  routerReplace.mockReset().mockResolvedValue(undefined)
  routerPush.mockReset().mockResolvedValue(undefined)
  routerResolve.mockClear()
  createOrder.mockReset()
  refreshUser.mockReset()
  fetchActiveSubscriptions.mockReset().mockResolvedValue(undefined)
  showError.mockReset()
  showInfo.mockReset()
  showWarning.mockReset()
  getCheckoutInfo.mockReset().mockResolvedValue(checkoutInfoWithPlansFixture(options))
  bridgeInvoke.mockReset()
  window.localStorage.clear()
  ;(window as Window & { WeixinJSBridge?: { invoke: typeof bridgeInvoke } }).WeixinJSBridge = undefined

  const wrapper = shallowMount(PaymentView, {
    global: {
      stubs: {
        AppLayout: {
          template: '<div><slot /></div>',
        },
        Teleport: true,
        Transition: false,
      },
    },
  })
  await flushPromises()
  await flushPromises()
  return wrapper
}

describe('PaymentView subscription confirmation amounts', () => {
  it('shows converted CNY pay amount using the subscription rate, not the balance multiplier', async () => {
    const wrapper = await mountSubscriptionConfirm({
      checkout: {
        balance_recharge_multiplier: 0.14,
        subscription_usd_to_cny_rate: 7.15,
      },
      method: {
        currency: 'CNY',
      },
      plan: {
        price: 9.99,
        original_price: 12.99,
      },
    })

    const text = wrapper.text()
    const convertedPrice = formatPaymentAmount(71.43, 'CNY')
    const convertedOriginalPrice = formatPaymentAmount(92.88, 'CNY')

    expect(text).toContain(convertedPrice)
    expect(text).toContain(convertedOriginalPrice)
    expect(text).not.toContain(formatPaymentAmount(9.99, 'CNY'))
    // 换算必须使用订阅汇率（×7.15），而不是余额倍率（÷0.14 = 71.36）
    expect(text).not.toContain(formatPaymentAmount(71.36, 'CNY'))
    expect(wrapper.findAll('button').some(button => button.text().includes(convertedPrice))).toBe(true)
  })

  it('keeps plan price when the subscription rate is not configured or payment currency is not CNY', async () => {
    // opt-in 回归锁：即使余额倍率已配置，未配置订阅汇率时 CNY 订阅仍按 price 直付
    const cnyWrapper = await mountSubscriptionConfirm({
      checkout: {
        balance_recharge_multiplier: 0.14,
        subscription_usd_to_cny_rate: 0,
      },
      method: {
        currency: 'CNY',
      },
      plan: {
        price: 7.99,
      },
    })

    expect(cnyWrapper.text()).toContain(formatPaymentAmount(7.99, 'CNY'))
    expect(cnyWrapper.text()).not.toContain(formatPaymentAmount(57.07, 'CNY'))
    expect(cnyWrapper.text()).not.toContain(formatPaymentAmount(57.13, 'CNY'))

    const usdWrapper = await mountSubscriptionConfirm({
      checkout: {
        subscription_usd_to_cny_rate: 7.15,
      },
      method: {
        currency: 'USD',
      },
      plan: {
        price: 7.99,
        original_price: 9.99,
      },
    })

    expect(usdWrapper.text()).toContain(formatPaymentAmount(7.99, 'USD'))
    expect(usdWrapper.text()).toContain(formatPaymentAmount(9.99, 'USD'))
  })

  it('adds fee rate after CNY rate conversion to match backend pay_amount', async () => {
    const wrapper = await mountSubscriptionConfirm({
      checkout: {
        subscription_usd_to_cny_rate: 7.15,
        recharge_fee_rate: 2.5,
      },
      method: {
        currency: 'CNY',
      },
      plan: {
        price: 9.99,
      },
    })

    const text = wrapper.text()
    const convertedPrice = formatPaymentAmount(71.43, 'CNY')
    const fee = formatPaymentAmount(1.79, 'CNY')
    const total = formatPaymentAmount(73.22, 'CNY')

    expect(text).toContain(convertedPrice)
    expect(text).toContain(fee)
    expect(text).toContain(total)
    expect(wrapper.findAll('button').some(button => button.text().includes(total))).toBe(true)
  })
})

describe('PaymentView payment recovery', () => {
  beforeEach(() => {
    vi.useRealTimers()
    routeState.path = '/purchase'
    routeState.query = {}
    routerReplace.mockReset().mockResolvedValue(undefined)
    routerPush.mockReset().mockResolvedValue(undefined)
    routerResolve.mockClear()
    createOrder.mockReset()
    refreshUser.mockReset()
    fetchActiveSubscriptions.mockReset().mockResolvedValue(undefined)
    showError.mockReset()
    showInfo.mockReset()
    showWarning.mockReset()
    bridgeInvoke.mockReset()
    window.localStorage.clear()
    ;(window as Window & { WeixinJSBridge?: { invoke: typeof bridgeInvoke } }).WeixinJSBridge = undefined
  })

  it('restores a custom EasyPay method as the selected payment method', async () => {
    getCheckoutInfo.mockResolvedValue(checkoutInfoFixture({
      methods: {
        wxpay: checkoutInfoFixture().data.methods.wxpay,
        ldc: {
          daily_limit: 0,
          daily_used: 0,
          daily_remaining: 0,
          single_min: 0,
          single_max: 0,
          fee_rate: 0,
          available: true,
          display_name: 'LDC Pay',
        },
      },
    }))
    window.localStorage.setItem(PAYMENT_RECOVERY_STORAGE_KEY, JSON.stringify({
      orderId: 888,
      amount: 66,
      qrCode: 'ldc-qr',
      expiresAt: '2099-01-01T00:10:00.000Z',
      paymentType: 'ldc',
      payUrl: 'https://pay.example.com/ldc',
      outTradeNo: 'sub2_ldc_888',
      clientSecret: '',
      intentId: '',
      currency: '',
      countryCode: '',
      paymentEnv: '',
      payAmount: 66,
      orderType: 'balance',
      paymentMode: 'popup',
      resumeToken: '',
      createdAt: Date.now(),
    }))

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: {
            template: '<div><slot /></div>',
          },
          PaymentStatusPanel: {
            template: '<button data-test="payment-done" @click="$emit(\'done\')" />',
          },
          PaymentMethodSelector: {
            props: ['selected'],
            template: '<div data-test="method-selector">{{ selected }}</div>',
          },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()
    await wrapper.find('[data-test="payment-done"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-test="method-selector"]').text()).toBe('ldc')
  })
})

describe('PaymentView WeChat JSAPI flow', () => {
  beforeEach(() => {
    routeState.path = '/purchase'
    routeState.query = {
      wechat_resume: '1',
      wechat_resume_token: 'resume-token-123',
    }
    routerReplace.mockReset().mockResolvedValue(undefined)
    routerPush.mockReset().mockResolvedValue(undefined)
    routerResolve.mockClear()
    createOrder.mockReset()
    refreshUser.mockReset()
    fetchActiveSubscriptions.mockReset().mockResolvedValue(undefined)
    fetchFirstRechargeStatus.mockReset().mockResolvedValue(null)
    showError.mockReset()
    showInfo.mockReset()
    showWarning.mockReset()
    getCheckoutInfo.mockReset().mockResolvedValue(checkoutInfoFixture())
    bridgeInvoke.mockReset()
    appStoreState.contactInfo = 'QQ: 123456789'
    window.localStorage.clear()
    ;(window as Window & { WeixinJSBridge?: { invoke: typeof bridgeInvoke } }).WeixinJSBridge = {
      invoke: bridgeInvoke,
    }
  })

  it('resets payment state and redirects to /payment/result after JSAPI reports success', async () => {
    createOrder.mockResolvedValue(jsapiOrderFixture('resume-token-123'))
    bridgeInvoke.mockImplementation((_action, _payload, callback) => {
      callback({ err_msg: 'get_brand_wcpay_request:ok' })
    })

    shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(routerReplace).toHaveBeenCalledWith({ path: '/purchase', query: {} })
    expect(routerPush).toHaveBeenCalledWith({
      path: '/payment/result',
      query: {
        order_id: '123',
        out_trade_no: 'sub2_jsapi_123',
        resume_token: 'resume-token-123',
      },
    })
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)).toBeNull()
  })

  it('resets payment state when JSAPI reports cancellation', async () => {
    createOrder.mockResolvedValue(jsapiOrderFixture('resume-token-cancel'))
    bridgeInvoke.mockImplementation((_action, _payload, callback) => {
      callback({ err_msg: 'get_brand_wcpay_request:cancel' })
    })

    shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(showInfo).toHaveBeenCalledWith('payment.qr.cancelled')
    expect(routerPush).not.toHaveBeenCalled()
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)).toBeNull()
  })

  it('clears stale recovery state when JSAPI never becomes available', async () => {
    vi.useFakeTimers()
    createOrder.mockResolvedValue(jsapiOrderFixture('resume-token-missing-bridge'))
    ;(window as Window & { WeixinJSBridge?: { invoke: typeof bridgeInvoke } }).WeixinJSBridge = undefined

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })

    await flushPromises()
    await vi.advanceTimersByTimeAsync(4000)
    await flushPromises()
    await flushPromises()

    expect(showError).toHaveBeenCalledWith(
      'payment.errors.wechatJsapiUnavailable payment.errors.wechatOpenInWeChatHint',
    )
    expect(routerPush).not.toHaveBeenCalled()
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)).toBeNull()
    expect(wrapper.html()).not.toContain('payment-status-panel-stub')
  })

  it('clears a stale recovery snapshot before handling wechat resume callback params', async () => {
    createOrder.mockRejectedValueOnce(new Error('resume failed'))
    window.localStorage.setItem(PAYMENT_RECOVERY_STORAGE_KEY, JSON.stringify({
      orderId: 999,
      amount: 66,
      qrCode: 'stale-qr',
      expiresAt: '2099-01-01T00:10:00.000Z',
      paymentType: 'alipay',
      payUrl: 'https://pay.example.com/stale',
      outTradeNo: 'stale-out-trade-no',
      clientSecret: '',
      intentId: '',
      currency: '',
      countryCode: '',
      paymentEnv: '',
      payAmount: 66,
      orderType: 'balance',
      paymentMode: 'popup',
      resumeToken: '',
      createdAt: Date.UTC(2099, 0, 1, 0, 0, 0),
    }))

    shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(createOrder).toHaveBeenCalledWith(expect.objectContaining({
      wechat_resume_token: 'resume-token-123',
    }))
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)).toBeNull()
  })

  it('keeps subscription resume context for token-only WeChat callbacks', async () => {
    routeState.query = {
      wechat_resume: '1',
      wechat_resume_token: 'resume-subscription-7',
      payment_type: 'wxpay_direct',
      order_type: 'subscription',
      plan_id: '7',
    }
    getCheckoutInfo.mockResolvedValue(checkoutInfoWithPlansFixture())
    createOrder.mockResolvedValue(oauthOrderFixture())

    const originalLocation = window.location
    const locationState = {
      href: 'http://localhost/purchase',
      origin: 'http://localhost',
    }
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: locationState,
    })

    shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(routerReplace).toHaveBeenCalledWith({ path: '/purchase', query: {} })
    expect(createOrder).toHaveBeenCalledWith(expect.objectContaining({
      payment_type: 'wxpay',
      order_type: 'subscription',
      plan_id: 7,
      wechat_resume_token: 'resume-subscription-7',
    }))
    expect(locationState.href).toContain('/api/v1/auth/oauth/wechat/payment/start?')
    expect(new URL(locationState.href, 'http://localhost').searchParams.get('redirect')).toBe(
      '/purchase?from=wechat&payment_type=wxpay&order_type=subscription&plan_id=7',
    )

    Object.defineProperty(window, 'location', {
      configurable: true,
      value: originalLocation,
    })
  })

  it('falls back to QR flow when mobile WeChat payment is unavailable', async () => {
    routeState.query = {
      wechat_resume: '1',
      wechat_resume_token: 'resume-token-h5',
      payment_type: 'wxpay_direct',
    }
    createOrder
      .mockRejectedValueOnce({ reason: 'WECHAT_H5_NOT_AUTHORIZED' })
      .mockResolvedValueOnce({
        order_id: 778,
        amount: 88,
        pay_amount: 88,
        fee_rate: 0,
        expires_at: '2099-01-01T00:10:00.000Z',
        payment_type: 'wxpay',
        qr_code: 'weixin://wxpay/bizpayurl?pr=fallback-native',
        out_trade_no: 'sub2_qr_778',
      })

    shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(createOrder).toHaveBeenNthCalledWith(1, expect.objectContaining({
      payment_type: 'wxpay',
      is_mobile: true,
      wechat_resume_token: 'resume-token-h5',
    }))
    expect(createOrder).toHaveBeenNthCalledWith(2, expect.objectContaining({
      payment_type: 'wxpay',
      is_mobile: false,
      payment_source: 'hosted_redirect',
    }))
    expect(showWarning).toHaveBeenCalledWith('payment.errors.mobilePaymentFallbackToQr')
    expect(showError).not.toHaveBeenCalled()
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)).toContain('weixin://wxpay/bizpayurl?pr=fallback-native')
  })

  it('creates a balance order from a product card with the selected payment method and product id', async () => {
    routeState.query = {}
    getCheckoutInfo.mockResolvedValue(checkoutInfoWithBalanceProductsFixture())
    createOrder.mockResolvedValue({
      order_id: 901,
      amount: 69,
      pay_amount: 6.9,
      fee_rate: 0,
      expires_at: '2099-01-01T00:10:00.000Z',
      payment_type: 'alipay',
      qr_code: 'https://pay.example.test/qr',
      out_trade_no: 'sub2_balance_901',
    })

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    await wrapper.findComponent(PurchaseProductCard).vm.$emit('pay', 'alipay')
    await flushPromises()

    expect(createOrder).toHaveBeenCalledWith(expect.objectContaining({
      amount: 6.9,
      payment_type: 'alipay',
      order_type: 'balance',
      balance_product_id: 9,
    }))
  })

  it('creates a custom balance recharge order without a product id', async () => {
    routeState.query = {}
    getCheckoutInfo.mockResolvedValue(checkoutInfoWithBalanceProductsFixture())
    createOrder.mockResolvedValue({
      order_id: 903,
      amount: 100,
      pay_amount: 100,
      fee_rate: 0,
      expires_at: '2099-01-01T00:10:00.000Z',
      payment_type: 'alipay',
      qr_code: 'https://pay.example.test/custom-qr',
      out_trade_no: 'sub2_custom_903',
    })

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    const html = wrapper.html()
    expect(html.indexOf('purchase-product-card-stub')).toBeGreaterThanOrEqual(0)
    expect(html.indexOf('amount-input-stub')).toBeGreaterThan(html.indexOf('purchase-product-card-stub'))

    await wrapper.findComponent(AmountInput).vm.$emit('update:modelValue', 100)
    await flushPromises()

    const customButtons = wrapper.findAll('button').filter(button => button.text().includes('payment.createOrder'))
    expect(customButtons.length).toBeGreaterThan(0)
    await customButtons[0].trigger('click')
    await flushPromises()

    expect(createOrder).toHaveBeenCalledWith(expect.objectContaining({
      amount: 100,
      payment_type: 'alipay',
      order_type: 'balance',
    }))
    expect(createOrder.mock.calls[0][0]).not.toHaveProperty('balance_product_id')
  })

  it('shows product exchange rate and received credits in USD', async () => {
    routeState.query = {}
    getCheckoutInfo.mockResolvedValue(checkoutInfoWithBalanceProductsFixture())

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    const card = wrapper.findComponent(PurchaseProductCard)
    const metrics = card.props('metrics') as { label: string; value: string }[]
    const heroMetrics = card.props('heroMetrics') as { label: string; value: string }[]
    const priceRows = card.props('priceRows') as { label: string; value: string }[]
    const exchangeRate = metrics.find(item => item.label === 'payment.product.exchangeRate')?.value || ''
    const validity = metrics.find(item => item.label === 'payment.product.validity')?.value || ''
    const balanceAmount = heroMetrics.find(item => item.label === 'payment.product.balanceAmount')?.value || ''

    expect(card.props('currency')).toBe('CNY')
    expect(exchangeRate).toBe('1¥:10$')
    expect(validity).toBe('payment.product.permanent')
    expect(metrics.some(item => item.label === 'payment.product.balanceAmount')).toBe(false)
    expect(balanceAmount).toBe('$69')
    expect(priceRows.map(item => item.label)).toEqual([
      'payment.product.originalPrice',
      'payment.product.payPrice',
    ])
  })

  it('shows customer purchase guidance and administrator support details', async () => {
    routeState.query = {}
    appStoreState.contactInfo = '微信: subapi-support'
    getCheckoutInfo.mockResolvedValue(checkoutInfoWithSupportFixture())

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(wrapper.text()).toContain('payment.purchaseGuide.title')
    expect(wrapper.text()).toContain('payment.purchaseGuide.billingTitle')
    expect(wrapper.text()).toContain('微信: subapi-support')
    expect(wrapper.text()).toContain('充值前可咨询客服确认套餐')
    expect(wrapper.find('img[src="https://example.test/support-qr.png"]').exists()).toBe(true)
  })

  it('shows administrator purchase notes below the custom recharge amount area', async () => {
    routeState.query = {}
    getCheckoutInfo.mockResolvedValue(checkoutInfoWithSupportFixture())

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    const helpText = wrapper.get('[data-testid="payment-recharge-help-text"]')
    expect(helpText.text()).toContain('payment.purchaseGuide.helpTitle')
    expect(helpText.text()).toContain('充值前可咨询客服确认套餐')
    expect(wrapper.html().indexOf('amount-input-stub')).toBeLessThan(wrapper.html().indexOf('payment-recharge-help-text'))
    expect(wrapper.html().indexOf('payment-recharge-help-text')).toBeLessThan(wrapper.html().indexOf('payment-method-selector-stub'))
  })

  it('creates a subscription order from a product card with the selected payment method and plan id', async () => {
    routeState.query = {
      tab: 'subscription',
    }
    getCheckoutInfo.mockResolvedValue(checkoutInfoWithPlansFixture())
    createOrder.mockResolvedValue({
      order_id: 902,
      amount: 128,
      pay_amount: 128,
      fee_rate: 0,
      expires_at: '2099-01-01T00:10:00.000Z',
      payment_type: 'wxpay',
      qr_code: 'weixin://wxpay/bizpayurl?pr=plan',
      out_trade_no: 'sub2_plan_902',
    })

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    await wrapper.findComponent(PurchaseProductCard).vm.$emit('pay', 'wxpay')
    await flushPromises()

    expect(createOrder).toHaveBeenCalledWith(expect.objectContaining({
      amount: 128,
      payment_type: 'wxpay',
      order_type: 'subscription',
      plan_id: 7,
    }))
  })

  it('groups subscription products by platform based on existing plan sort order and keeps product cards payable', async () => {
    routeState.query = {
      tab: 'subscription',
    }
    getCheckoutInfo.mockResolvedValue(checkoutInfoWithPlatformPlansFixture())
    createOrder.mockResolvedValue({
      order_id: 904,
      amount: 128,
      pay_amount: 128,
      fee_rate: 0,
      expires_at: '2099-01-01T00:10:00.000Z',
      payment_type: 'wxpay',
      qr_code: 'weixin://wxpay/bizpayurl?pr=gemini-plan',
      out_trade_no: 'sub2_platform_plan_904',
    })

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    const sections = wrapper.findAll('[data-testid="subscription-platform-section"]')
    expect(sections.map(section => section.text())).toEqual([
      expect.stringContaining('Gemini'),
      expect.stringContaining('OpenAI'),
      expect.stringContaining('Anthropic'),
    ])

    const cards = wrapper.findAllComponents(PurchaseProductCard)
    expect(cards.map(card => (card.props('product') as { name: string }).name)).toEqual([
      'Gemini Pro',
      'OpenAI Monthly',
      'OpenAI Weekly',
      'Claude Team',
    ])

    await cards[0].vm.$emit('pay', 'wxpay')
    await flushPromises()

    expect(createOrder).toHaveBeenCalledWith(expect.objectContaining({
      amount: 128,
      payment_type: 'wxpay',
      order_type: 'subscription',
      plan_id: 23,
    }))
  })

  it('uses five-column product grids and displays stored subscription quota clearly', async () => {
    routeState.query = {
      tab: 'subscription',
    }
    getCheckoutInfo.mockResolvedValue(checkoutInfoWithPlansFixture())

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    const card = wrapper.findComponent(PurchaseProductCard)
    const metrics = card.props('metrics') as { label: string; value: string }[]
    const heroMetrics = card.props('heroMetrics') as { label: string; value: string }[]
    const priceRows = card.props('priceRows') as { label: string; value: string }[]
    const totalQuota = metrics.find(item => item.label === 'payment.product.totalQuota')?.value || ''
    const dailyQuota = heroMetrics.find(item => item.label === 'payment.product.dailyQuota')?.value || ''

    expect(wrapper.html()).toContain('lg:grid-cols-5')
    expect(totalQuota).toBe('$840')
    expect(metrics.some(item => item.label === 'payment.product.dailyQuota')).toBe(false)
    expect(dailyQuota).toBe('$120')
    expect(priceRows.map(item => item.label)).toEqual(['payment.product.payPrice'])
  })

  it('shows only active subscription quota periods on product cards', async () => {
    routeState.query = {
      tab: 'subscription',
    }
    const checkout = checkoutInfoWithPlansFixture()
    checkout.data.plans[0] = {
      ...checkout.data.plans[0],
      validity_days: 4,
      validity_unit: 'weeks',
      daily_limit_usd: 10,
      daily_quota: null,
      weekly_limit_usd: 50,
      monthly_limit_usd: 0,
      total_quota: null,
    }
    getCheckoutInfo.mockResolvedValue(checkout)

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    const card = wrapper.findComponent(PurchaseProductCard)
    const metrics = card.props('metrics') as { label: string; value: string }[]
    const heroMetrics = card.props('heroMetrics') as { label: string; value: string }[]
    const labels = heroMetrics.map(item => item.label)

    expect(labels).toContain('payment.product.dailyQuota')
    expect(labels).toContain('payment.product.weeklyQuota')
    expect(labels).not.toContain('payment.product.monthlyQuota')
    expect(metrics.map(item => item.label)).not.toContain('payment.product.dailyQuota')
    expect(metrics.find(item => item.label === 'payment.product.totalQuota')?.value).toBe('$200')
  })

  it('calculates subscription total quota from the selected validity unit when no stored quota exists', async () => {
    routeState.query = {
      tab: 'subscription',
    }
    const checkout = checkoutInfoWithPlansFixture()
    checkout.data.plans[0] = {
      ...checkout.data.plans[0],
      validity_days: 4,
      validity_unit: 'weeks',
      total_quota: null,
      weekly_limit_usd: 140,
    }
    getCheckoutInfo.mockResolvedValue(checkout)

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    const card = wrapper.findComponent(PurchaseProductCard)
    const metrics = card.props('metrics') as { label: string; value: string }[]
    const totalQuota = metrics.find(item => item.label === 'payment.product.totalQuota')?.value || ''

    expect(totalQuota).toBe('$560')
  })
})
