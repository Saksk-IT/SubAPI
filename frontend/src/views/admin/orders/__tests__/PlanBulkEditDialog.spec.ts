import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  bulkUpdatePlans: vi.fn(),
}))

const appStoreMocks = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin/payment', () => ({
  adminPaymentAPI: {
    bulkUpdatePlans: apiMocks.bulkUpdatePlans,
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

import PlanBulkEditDialog from '../PlanBulkEditDialog.vue'

function mountDialog() {
  return mount(PlanBulkEditDialog, {
    props: {
      show: true,
      planIds: [1, 2],
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

describe('PlanBulkEditDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMocks.bulkUpdatePlans.mockResolvedValue({ data: { updated: 2 } })
  })

  it('只提交已勾选的批量修改字段', async () => {
    const wrapper = mountDialog()
    const checkboxes = wrapper.findAll('input[type="checkbox"]')

    await checkboxes[0].setValue(true)
    await checkboxes[2].setValue(true)
    const multiplierInput = wrapper.find('input[type="number"]')
    await multiplierInput.setValue('0.25')
    const textareas = wrapper.findAll('textarea')
    await textareas[1].setValue('特性 A\n特性 B')

    await wrapper.get('#plan-bulk-edit-form').trigger('submit.prevent')
    await flushPromises()

    expect(apiMocks.bulkUpdatePlans).toHaveBeenCalledWith({
      plan_ids: [1, 2],
      fields: {
        price_multiplier: 0.25,
        features: '特性 A\n特性 B',
      },
    })
  })

  it('未勾选任何字段时不提交请求', async () => {
    const wrapper = mountDialog()

    await wrapper.get('#plan-bulk-edit-form').trigger('submit.prevent')
    await flushPromises()

    expect(apiMocks.bulkUpdatePlans).not.toHaveBeenCalled()
    expect(appStoreMocks.showError).toHaveBeenCalledWith('payment.admin.bulkEditPlansNoFields')
  })
})
