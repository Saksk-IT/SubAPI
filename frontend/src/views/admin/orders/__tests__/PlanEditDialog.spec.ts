import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  createPlan: vi.fn(),
  updatePlan: vi.fn(),
  getRateSchedule: vi.fn(),
  bulkUpdatePlans: vi.fn(),
}))

const appStoreMocks = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
  showWarning: vi.fn(),
}))

vi.mock('@/api/admin/payment', () => ({
  adminPaymentAPI: {
    createPlan: apiMocks.createPlan,
    updatePlan: apiMocks.updatePlan,
    bulkUpdatePlans: apiMocks.bulkUpdatePlans,
  },
}))

vi.mock('@/api/admin/groups', () => ({
  getRateSchedule: apiMocks.getRateSchedule,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMocks,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (key === 'payment.admin.subscriptionCnyPayPreview') return `preview ${params?.amount}`
        if (key === 'payment.admin.subscriptionCnyPayPreviewWithFee') return `fee ${params?.feeRate} ${params?.total}`
        return key
      },
    }),
  }
})

import PlanEditDialog from '../PlanEditDialog.vue'
import type { AdminGroup } from '@/types'

const subscriptionGroup: AdminGroup = {
  id: 12,
  name: '订阅 120 刀',
  description: null,
  platform: 'openai',
  rate_multiplier: 2.25,
  is_exclusive: false,
  status: 'active',
  subscription_type: 'subscription',
  daily_limit_usd: 120,
  weekly_limit_usd: null,
  monthly_limit_usd: null,
  allow_image_generation: false,
  image_rate_independent: false,
  image_rate_multiplier: 1,
  image_price_1k: null,
  image_price_2k: null,
  image_price_4k: null,
  claude_code_only: false,
  fallback_group_id: null,
  fallback_group_id_on_invalid_request: null,
  require_oauth_only: false,
  require_privacy_set: false,
  created_at: '',
  updated_at: '',
  model_routing: null,
  model_routing_enabled: false,
  mcp_xml_inject: false,
}

function mountDialog(paymentConfig: Record<string, unknown> | null = null) {
  return mount(PlanEditDialog, {
    props: {
      show: false,
      plan: null,
      groups: [subscriptionGroup],
      paymentConfig,
    },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
        Select: {
          props: ['modelValue', 'options', 'disabled'],
          emits: ['update:modelValue'],
          template: `
            <select
              :disabled="disabled"
              :value="modelValue ?? ''"
              @change="$emit('update:modelValue', Number($event.target.value))"
            >
              <option value=""></option>
              <option v-for="option in options" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          `,
        },
        GroupBadge: {
          props: ['name', 'rateMultiplier'],
          template: '<span>{{ name }} {{ rateMultiplier }}x</span>',
        },
        Icon: true,
      },
    },
  })
}

function mountEditDialog(plan: Record<string, unknown>, paymentConfig: Record<string, unknown> | null = null) {
  return mount(PlanEditDialog, {
    props: {
      show: false,
      plan: {
        id: 99,
        group_id: 12,
        name: '旧套餐',
        description: '旧描述',
        price: 336,
        price_multiplier: 0.2,
        original_price: 0,
        validity_days: 7,
        validity_unit: 'days',
        features: ['旧特性'],
        tags: [],
        for_sale: true,
        sort_order: 0,
        ...plan,
      },
      groups: [subscriptionGroup],
      paymentConfig,
    },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
        Select: {
          props: ['modelValue', 'options', 'disabled'],
          emits: ['update:modelValue'],
          template: `
            <select
              :disabled="disabled"
              :value="modelValue ?? ''"
              @change="$emit('update:modelValue', Number($event.target.value))"
            >
              <option value=""></option>
              <option v-for="option in options" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          `,
        },
        GroupBadge: {
          props: ['name', 'rateMultiplier'],
          template: '<span>{{ name }} {{ rateMultiplier }}x</span>',
        },
        Icon: true,
      },
    },
  })
}

describe('PlanEditDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMocks.getRateSchedule.mockResolvedValue({
      enabled: true,
      active: true,
      start_time: '00:00',
      end_time: '23:59',
      percent: 90,
      timezone: 'Asia/Shanghai',
      original_rates: { '12': 2.5 },
    })
    apiMocks.createPlan.mockResolvedValue({ data: { id: 1 } })
  })

  it('按分组未优惠实际倍率、周期数和套餐倍率自动计算价格', async () => {
    const wrapper = mountDialog()
    await wrapper.setProps({ show: true })
    await flushPromises()

    await wrapper.find('select').setValue('12')
    await flushPromises()

    const inputs = wrapper.findAll('input')
    await inputs[0].setValue('周卡')
    await inputs[1].setValue('7')
    await inputs[2].setValue('0.14')
    await wrapper.find('textarea').setValue('0.14 倍率')
    await flushPromises()

    const priceInput = wrapper.findAll('input[type="text"]')[1]
    expect(priceInput.element.value).toBe('47.04')
    expect(wrapper.text()).toContain('2.5x')

    await wrapper.get('#plan-form').trigger('submit.prevent')
    await flushPromises()

    expect(apiMocks.createPlan).toHaveBeenCalledTimes(1)
    expect(apiMocks.createPlan).toHaveBeenCalledWith(expect.objectContaining({
      group_id: 12,
      price: 47.04,
      price_multiplier: 0.14,
      validity_days: 7,
      validity_unit: 'days',
      total_quota: 840,
      daily_quota: 120,
    }))
  })

  it('允许管理员填写任意精度的套餐倍率', async () => {
    const wrapper = mountDialog()
    await wrapper.setProps({ show: true })
    await flushPromises()

    await wrapper.find('select').setValue('12')
    await flushPromises()

    const inputs = wrapper.findAll('input')
    const multiplierInput = inputs[2]
    expect(multiplierInput.attributes('step')).toBe('any')
    expect(multiplierInput.attributes('min')).toBe('0')

    await inputs[0].setValue('精确倍率周卡')
    await inputs[1].setValue('7')
    await multiplierInput.setValue('0.123')
    await wrapper.find('textarea').setValue('0.123 倍率')
    await flushPromises()

    const priceInput = wrapper.findAll('input[type="text"]')[1]
    expect(priceInput.element.value).toBe('41.33')

    await wrapper.get('#plan-form').trigger('submit.prevent')
    await flushPromises()

    expect(apiMocks.createPlan).toHaveBeenCalledTimes(1)
    expect(apiMocks.createPlan).toHaveBeenCalledWith(expect.objectContaining({
      group_id: 12,
      price: 41.33,
      price_multiplier: 0.123,
      validity_days: 7,
    }))
  })

  it('基于自动计算价格展示订阅人民币实扣预览', async () => {
    const wrapper = mountDialog({
      subscription_usd_to_cny_rate: 7.15,
      recharge_fee_rate: 2.5,
    })
    await wrapper.setProps({ show: true })
    await flushPromises()

    await wrapper.find('select').setValue('12')
    await flushPromises()

    const inputs = wrapper.findAll('input')
    await inputs[0].setValue('周卡')
    await inputs[1].setValue('7')
    await inputs[2].setValue('0.14')
    await flushPromises()

    expect(wrapper.text()).toContain('preview')
    expect(wrapper.text()).toContain('336.34')
    expect(wrapper.text()).toContain('fee 2.5')
    expect(wrapper.text()).toContain('344.75')
  })

  it('未配置订阅人民币汇率时隐藏实扣预览', async () => {
    const wrapper = mountDialog({
      subscription_usd_to_cny_rate: 0,
      recharge_fee_rate: 2.5,
    })
    await wrapper.setProps({ show: true })
    await flushPromises()

    await wrapper.find('select').setValue('12')
    await flushPromises()

    const inputs = wrapper.findAll('input')
    await inputs[0].setValue('周卡')
    await inputs[1].setValue('7')
    await inputs[2].setValue('0.14')
    await flushPromises()

    expect(wrapper.text()).not.toContain('preview')
    expect(wrapper.text()).not.toContain('336.34')
  })

  it('编辑时优先使用持久化套餐倍率而不是从价格反推', async () => {
    apiMocks.updatePlan.mockResolvedValue({ data: { id: 99 } })
    const wrapper = mountEditDialog({ price: 1, price_multiplier: 0.2 })

    await wrapper.setProps({ show: true })
    await flushPromises()

    const inputs = wrapper.findAll('input')
    expect((inputs[2].element as HTMLInputElement).value).toBe('0.2')
    const priceInput = wrapper.findAll('input[type="text"]')[1]
    expect(priceInput.element.value).toBe('67.20')

    await wrapper.get('#plan-form').trigger('submit.prevent')
    await flushPromises()

    expect(apiMocks.updatePlan).toHaveBeenCalledWith(99, expect.objectContaining({
      price: 67.2,
      price_multiplier: 0.2,
    }))
  })
})
