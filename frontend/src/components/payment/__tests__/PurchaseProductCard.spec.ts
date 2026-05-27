import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { describe, expect, it } from 'vitest'
import PurchaseProductCard from '../PurchaseProductCard.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  fallbackWarn: false,
  missingWarn: false,
  messages: {
    'zh-CN': {
      payment: {
        methods: {
          alipay: '支付宝',
          wxpay: '微信支付',
        },
        product: {
          autoApply: '购买后默认直接生效',
          detail: '详情',
        },
      },
    },
  },
})

function mountCard() {
  return mount(PurchaseProductCard, {
    props: {
      product: {
        id: 1,
        name: '旗舰套餐',
        description: '适合高频调用',
        price: 88,
        tags: ['火爆热卖', '八五折扣'],
        features: [],
      },
      metrics: [
        { label: '支付价格', value: '¥88.00' },
        { label: '获得额度', value: '$120.00' },
      ],
      methods: [
        { type: 'alipay', fee_rate: 0, available: true },
        { type: 'wxpay', fee_rate: 0, available: true },
      ],
      currency: 'CNY',
      locale: 'zh-CN',
    },
    global: { plugins: [i18n] },
  })
}

describe('PurchaseProductCard', () => {
  it('renders product tags as prominent sale badges', () => {
    const wrapper = mountCard()

    const tag = wrapper.get('[data-testid="product-tag"]')
    expect(tag.text()).toBe('火爆热卖')
    expect(tag.classes().join(' ')).toContain('from-orange-500')
  })

  it('keeps payment method icons visible on colored buttons', () => {
    const wrapper = mountCard()

    const iconShells = wrapper.findAll('[data-testid="payment-method-icon-shell"]')
    expect(iconShells).toHaveLength(2)
    expect(iconShells.every(shell => shell.classes().includes('bg-white'))).toBe(true)
  })
})
