import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const {
  listAccounts,
  listWithEtag,
  getById,
  getBatchTodayStats,
  deleteAccount,
  batchRefresh,
  bulkUpdate,
  getAllProxies,
  getAllGroups
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  listWithEtag: vi.fn(),
  getById: vi.fn(),
  getBatchTodayStats: vi.fn(),
  deleteAccount: vi.fn(),
  batchRefresh: vi.fn(),
  bulkUpdate: vi.fn(),
  getAllProxies: vi.fn(),
  getAllGroups: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listWithEtag,
      getById,
      getBatchTodayStats,
      delete: deleteAccount,
      batchClearError: vi.fn(),
      batchRefresh,
      bulkUpdate,
      toggleSchedulable: vi.fn()
    },
    proxies: {
      getAll: getAllProxies
    },
    groups: {
      getAll: getAllGroups
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    token: 'test-token'
  })
}))

vi.mock('@/services/nativeDialog', () => ({
  nativeConfirm: vi.fn(() => Promise.resolve(true))
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

Object.defineProperty(globalThis, 'localStorage', {
  value: {
    getItem: vi.fn(() => null),
    setItem: vi.fn(),
    removeItem: vi.fn(),
    clear: vi.fn()
  },
  configurable: true
})

const DataTableStub = {
  props: ['columns', 'data'],
  template: `
    <div data-test="data-table">
      <span v-for="column in columns" :key="column.key" data-test="column-key">{{ column.key }}</span>
      <div v-for="row in data" :key="row.id">
        <slot name="cell-created_at" :value="row.created_at" :row="row" />
      </div>
    </div>
  `
}

const AccountBulkActionsBarStub = {
  props: ['selectedIds'],
  emits: ['edit-filtered'],
  template: '<button data-test="edit-filtered" @click="$emit(\'edit-filtered\')">edit filtered</button>'
}

const BulkEditAccountModalStub = {
  props: ['show', 'target'],
  template: '<div data-test="bulk-edit-modal" :data-show="String(show)" :data-target-mode="target?.mode ?? \'\'"></div>'
}

describe('admin AccountsView bulk edit scope', () => {
  beforeEach(() => {
    localStorage.clear()

    listAccounts.mockReset()
    listWithEtag.mockReset()
    getById.mockReset()
    getBatchTodayStats.mockReset()
    deleteAccount.mockReset()
    batchRefresh.mockReset()
    bulkUpdate.mockReset()
    getAllProxies.mockReset()
    getAllGroups.mockReset()

    listAccounts.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 0
    })
    listWithEtag.mockResolvedValue({
      notModified: true,
      etag: null,
      data: null
    })
    getById.mockRejectedValue(new Error('not found'))
    getBatchTodayStats.mockResolvedValue({ stats: {} })
    deleteAccount.mockResolvedValue({ message: 'ok' })
    batchRefresh.mockResolvedValue({ total: 0, success: 0, failed: 0, results: [] })
    bulkUpdate.mockResolvedValue({ success: 0, failed: 0, success_ids: [], failed_ids: [], results: [] })
    getAllProxies.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])
  })

  it('opens bulk edit in filtered-results mode from the bulk actions dropdown', async () => {
    const { default: AccountsView } = await import('../AccountsView.vue')
    const wrapper = mount(AccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          ConfirmDialog: true,
          AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
          AccountTableFilters: { template: '<div></div>' },
          AccountBulkActionsBar: AccountBulkActionsBarStub,
          AccountActionMenu: true,
          ImportDataModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          SyncFromCrsModal: true,
          TempUnschedStatusModal: true,
          ErrorPassthroughRulesModal: true,
          TLSFingerprintProfilesModal: true,
          CreateAccountModal: true,
          EditAccountModal: true,
          BulkEditAccountModal: BulkEditAccountModalStub,
          BatchRefreshResultModal: true,
          PlatformTypeBadge: true,
          AccountCapacityCell: true,
          AccountStatusIndicator: true,
          AccountTodayStatsCell: true,
          AccountGroupsCell: true,
          AccountUsageCell: true,
          Icon: true
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-test="edit-filtered"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="bulk-edit-modal"]').attributes('data-show')).toBe('true')
    expect(wrapper.get('[data-test="bulk-edit-modal"]').attributes('data-target-mode')).toBe('filtered')
  })

  it('keeps filtered bulk edit OpenAI-specific options hidden when only a partial unfiltered preview is known', async () => {
    listAccounts.mockResolvedValueOnce({
      items: [
        {
          id: 1,
          name: 'openai-preview',
          platform: 'openai',
          type: 'oauth',
          status: 'active',
          credentials: {},
          extra: {},
          groups: []
        }
      ],
      total: 101,
      page: 1,
      page_size: 100,
      pages: 2
    })

    const { default: AccountsView } = await import('../AccountsView.vue')
    const wrapper = mount(AccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          ConfirmDialog: true,
          AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
          AccountTableFilters: { template: '<div></div>' },
          AccountBulkActionsBar: AccountBulkActionsBarStub,
          AccountActionMenu: true,
          ImportDataModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          SyncFromCrsModal: true,
          TempUnschedStatusModal: true,
          ErrorPassthroughRulesModal: true,
          TLSFingerprintProfilesModal: true,
          CreateAccountModal: true,
          EditAccountModal: true,
          BulkEditAccountModal: {
            props: ['show', 'target'],
            template: '<div data-test="bulk-edit-modal" :data-platforms="target?.selectedPlatforms?.join(\',\') ?? \'\'"></div>'
          },
          BatchRefreshResultModal: true,
          PlatformTypeBadge: true,
          AccountCapacityCell: true,
          AccountStatusIndicator: true,
          AccountTodayStatsCell: true,
          AccountGroupsCell: true,
          AccountUsageCell: true,
          Icon: true
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-test="edit-filtered"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="bulk-edit-modal"]').attributes('data-platforms')).toBe('')
  })

  it('renders the created_at column by default', async () => {
    listAccounts.mockResolvedValue({
      items: [
        {
          id: 1,
          name: 'test-account',
          platform: 'anthropic',
          type: 'oauth',
          status: 'active',
          schedulable: true,
          created_at: '2026-03-07T10:00:00Z',
          updated_at: '2026-03-07T10:00:00Z'
        }
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })

    const { default: AccountsView } = await import('../AccountsView.vue')
    const wrapper = mount(AccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          ConfirmDialog: true,
          AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
          AccountTableFilters: { template: '<div></div>' },
          AccountBulkActionsBar: AccountBulkActionsBarStub,
          AccountActionMenu: true,
          ImportDataModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          SyncFromCrsModal: true,
          TempUnschedStatusModal: true,
          ErrorPassthroughRulesModal: true,
          TLSFingerprintProfilesModal: true,
          CreateAccountModal: true,
          EditAccountModal: true,
          BulkEditAccountModal: BulkEditAccountModalStub,
          BatchRefreshResultModal: true,
          PlatformTypeBadge: true,
          AccountCapacityCell: true,
          AccountStatusIndicator: true,
          AccountTodayStatsCell: true,
          AccountGroupsCell: true,
          AccountUsageCell: true,
          Icon: true
        }
      }
    })

    await flushPromises()

    const columnKeys = wrapper.findAll('[data-test="column-key"]').map(node => node.text())
    expect(columnKeys).toContain('created_at')
    const columns = wrapper.getComponent(DataTableStub).props('columns') as Array<{ key: string; label: string; sortable: boolean }>
    expect(columns.find(column => column.key === 'created_at')).toMatchObject({
      label: 'admin.accounts.columns.createdAt',
      sortable: true
    })
  })

  it('resolves selected bulk edit metadata by selected ids instead of current page only', async () => {
    listAccounts.mockResolvedValue({
      items: [
        {
          id: 1,
          name: 'openai-current-page',
          platform: 'openai',
          type: 'oauth',
          status: 'active',
          credentials: {},
          extra: {},
          groups: []
        }
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    getById.mockResolvedValue({
      id: 2,
      name: 'anthropic-other-page',
      platform: 'anthropic',
      type: 'apikey',
      status: 'active',
      credentials: {},
      extra: {},
      groups: []
    })

    const { default: AccountsView } = await import('../AccountsView.vue')
    const wrapper = mount(AccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: {
            props: ['columns', 'data'],
            emits: ['sort'],
            template: `
              <div data-test="data-table">
                <button data-test="select-current" @click="$attrs.onChange?.({ target: { checked: true } })">select current</button>
              </div>
            `
          },
          Pagination: true,
          ConfirmDialog: true,
          AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
          AccountTableFilters: { template: '<div></div>' },
          AccountBulkActionsBar: {
            props: ['selectedIds'],
            emits: ['edit-selected'],
            template: '<button data-test="edit-selected" @click="$emit(\'edit-selected\')">edit selected</button>'
          },
          AccountActionMenu: true,
          ImportDataModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          SyncFromCrsModal: true,
          TempUnschedStatusModal: true,
          ErrorPassthroughRulesModal: true,
          TLSFingerprintProfilesModal: true,
          CreateAccountModal: true,
          EditAccountModal: true,
          BulkEditAccountModal: {
            props: ['show', 'target'],
            template: `
              <div
                data-test="bulk-edit-modal"
                :data-platforms="target?.selectedPlatforms?.join(',') ?? ''"
                :data-types="target?.selectedTypes?.join(',') ?? ''"
              ></div>
            `
          },
          BatchRefreshResultModal: true,
          PlatformTypeBadge: true,
          AccountCapacityCell: true,
          AccountStatusIndicator: true,
          AccountTodayStatsCell: true,
          AccountGroupsCell: true,
          AccountUsageCell: true,
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).setSelectedIds([1, 2])
    await wrapper.get('[data-test="edit-selected"]').trigger('click')
    await flushPromises()

    expect(getById).toHaveBeenCalledWith(2)
    expect(wrapper.get('[data-test="bulk-edit-modal"]').attributes('data-platforms')).toBe('openai,anthropic')
    expect(wrapper.get('[data-test="bulk-edit-modal"]').attributes('data-types')).toBe('oauth,apikey')
  })

  it('opens a batch refresh result modal with success and failed accounts', async () => {
    listAccounts.mockResolvedValue({
      items: [
        {
          id: 1,
          name: 'refresh-ok',
          platform: 'openai',
          type: 'oauth',
          status: 'active',
          credentials: {},
          extra: {},
          groups: []
        },
        {
          id: 2,
          name: 'refresh-failed',
          platform: 'openai',
          type: 'oauth',
          status: 'active',
          credentials: {},
          extra: {},
          groups: []
        }
      ],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })
    batchRefresh.mockResolvedValue({
      total: 2,
      success: 1,
      failed: 1,
      success_ids: [1],
      failed_ids: [2],
      results: [
        { account_id: 1, name: 'refresh-ok', platform: 'openai', type: 'oauth', success: true },
        { account_id: 2, name: 'refresh-failed', platform: 'openai', type: 'oauth', success: false, error: 'invalid_grant' }
      ]
    })
    const { default: AccountsView } = await import('../AccountsView.vue')
    const wrapper = mount(AccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          ConfirmDialog: true,
          AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
          AccountTableFilters: { template: '<div></div>' },
          AccountBulkActionsBar: {
            props: ['selectedIds'],
            emits: ['refresh-token'],
            template: '<button data-test="refresh-token" @click="$emit(\'refresh-token\')">refresh token</button>'
          },
          AccountActionMenu: true,
          ImportDataModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          SyncFromCrsModal: true,
          TempUnschedStatusModal: true,
          ErrorPassthroughRulesModal: true,
          TLSFingerprintProfilesModal: true,
          CreateAccountModal: true,
          EditAccountModal: true,
          BulkEditAccountModal: BulkEditAccountModalStub,
          BatchRefreshResultModal: {
            props: ['show', 'summary', 'successAccounts', 'failedAccounts'],
            template: `
              <div
                data-test="batch-refresh-result"
                :data-show="String(show)"
                :data-success="successAccounts.map((account) => account.name).join(',')"
                :data-failed="failedAccounts.map((account) => account.name).join(',')"
                :data-failed-count="String(summary?.failed ?? '')"
              ></div>
            `
          },
          PlatformTypeBadge: true,
          AccountCapacityCell: true,
          AccountStatusIndicator: true,
          AccountTodayStatsCell: true,
          AccountGroupsCell: true,
          AccountUsageCell: true,
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).setSelectedIds([1, 2])
    await wrapper.get('[data-test="refresh-token"]').trigger('click')
    await flushPromises()

    expect(batchRefresh).toHaveBeenCalledWith([1, 2])
    expect(wrapper.get('[data-test="batch-refresh-result"]').attributes('data-show')).toBe('true')
    expect(wrapper.get('[data-test="batch-refresh-result"]').attributes('data-success')).toBe('refresh-ok')
    expect(wrapper.get('[data-test="batch-refresh-result"]').attributes('data-failed')).toBe('refresh-failed')
    expect(wrapper.get('[data-test="batch-refresh-result"]').attributes('data-failed-count')).toBe('1')
  })
})
