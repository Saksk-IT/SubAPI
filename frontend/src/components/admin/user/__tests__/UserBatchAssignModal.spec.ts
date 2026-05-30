import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  batchAssign: vi.fn(),
}))

const appStoreMocks = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      batchAssign: apiMocks.batchAssign,
    },
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
      t: (key: string, params?: Record<string, string | number>) => {
        if (!params) return key
        return key.replace(/\{(\w+)\}/g, (_, name) => String(params[name] ?? ''))
      },
    }),
  }
})

import UserBatchAssignModal from '../UserBatchAssignModal.vue'

function mountModal() {
  return mount(UserBatchAssignModal, {
    props: {
      show: true,
      groups: [],
    },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
        GroupBadge: true,
        GroupOptionItem: true,
        Icon: true,
        Select: true,
      },
    },
  })
}

async function switchToRuleMode(wrapper: ReturnType<typeof mountModal>) {
  const ruleButton = wrapper.findAll('button').find((button) =>
    button.text().includes('admin.users.batchAssign.ruleBalance')
  )
  expect(ruleButton).toBeTruthy()
  await ruleButton!.trigger('click')
}

describe('UserBatchAssignModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMocks.batchAssign.mockResolvedValue({
      target_count: 2,
      success_count: 2,
      failed_count: 0,
      balance_affected_count: 2,
      subscription_assigned: 0,
      subscription_extended: 0,
    })
  })

  it('提交规则调整 payload', async () => {
    const wrapper = mountModal()
    await switchToRuleMode(wrapper)

    const inputs = wrapper.findAll('input[type="number"]')
    expect(inputs).toHaveLength(3)
    await inputs[0].setValue('0')
    await inputs[1].setValue('100')
    await inputs[2].setValue('1.5')

    await wrapper.get('#batch-assign-form').trigger('submit.prevent')
    await flushPromises()

    expect(apiMocks.batchAssign).toHaveBeenCalledTimes(1)
    expect(apiMocks.batchAssign).toHaveBeenCalledWith({
      all: true,
      balance: {
        operation: 'rule',
        rules: [
          { min_balance: 0, max_balance: 100, multiplier: 1.5 },
        ],
        notes: '',
      },
    })
    expect(appStoreMocks.showSuccess).toHaveBeenCalled()
  })

  it('添加规则并阻止重叠区间提交', async () => {
    const wrapper = mountModal()
    await switchToRuleMode(wrapper)

    const addButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.users.batchAssign.addRule')
    )
    expect(addButton).toBeTruthy()
    await addButton!.trigger('click')

    const inputs = wrapper.findAll('input[type="number"]')
    expect(inputs).toHaveLength(6)
    await inputs[3].setValue('50')
    await inputs[4].setValue('150')
    await inputs[5].setValue('1.2')

    expect(wrapper.text()).toContain('admin.users.batchAssign.ruleOverlap')
    const submitButton = wrapper.find('button[type="submit"]')
    expect(submitButton.attributes('disabled')).toBeDefined()

    await wrapper.get('#batch-assign-form').trigger('submit.prevent')
    await flushPromises()

    expect(apiMocks.batchAssign).not.toHaveBeenCalled()
  })
})
