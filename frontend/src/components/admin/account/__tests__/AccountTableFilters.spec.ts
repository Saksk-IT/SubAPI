import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountTableFilters from '../AccountTableFilters.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountTableFilters', () => {
  it('includes paused in the status filter options', () => {
    const selectOptions: Array<Array<{ value: string; label: string }>> = []

    mount(AccountTableFilters, {
      props: {
        searchQuery: '',
        filters: {
          platform: '',
          type: '',
          plan_type: '',
          status: '',
          privacy_mode: '',
          group: ''
        },
        groups: []
      },
      global: {
        stubs: {
          SearchInput: true,
          Select: {
            props: ['options'],
            setup(props) {
              selectOptions.push(props.options)
              return {}
            },
            template: '<div />'
          }
        }
      }
    })

    const statusOptions = selectOptions.find((options) =>
      options.some((option) => option.value === 'paused')
    )

    expect(statusOptions).toBeDefined()
    expect(statusOptions).toContainEqual({ value: 'paused', label: 'admin.accounts.status.paused' })
  })
})
