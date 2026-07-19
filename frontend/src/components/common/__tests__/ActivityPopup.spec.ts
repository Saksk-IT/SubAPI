import { afterEach, describe, expect, it } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter } from 'vue-router'
import ActivityPopup from '@/components/common/ActivityPopup.vue'
import { useActivityStore } from '@/stores/activities'

let wrapper: VueWrapper | null = null

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  document.body.innerHTML = ''
})

describe('ActivityPopup', () => {
  it('shows unseen activities together and opens the activity center', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/activities', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    await router.isReady()

    const activityStore = useActivityStore()
    activityStore.activities = [{
      id: 'first_recharge',
      type: 'first_recharge',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      first_recharge: {
        enabled: true,
        eligible: true,
        completed: false,
        popup_dismissed: false,
        eligibility_scope: 'all_users',
        purchase_mode: 'product_link',
        product_url: 'https://shop.example.test/first-recharge',
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
        offers: [],
      },
    }]

    wrapper = mount(ActivityPopup, {
      global: {
        plugins: [
          pinia,
          router,
          createI18n({
            legacy: false,
            locale: 'zh',
            messages: { zh: {} },
            missingWarn: false,
            fallbackWarn: false,
          }),
        ],
        stubs: { Icon: true },
      },
    })
    await flushPromises()

    expect(document.querySelector('[data-testid="activity-popup"]')).not.toBeNull()
    expect(document.querySelector('[data-testid="activity-popup-item-first_recharge"]')).not.toBeNull()

    const viewAll = document.querySelector<HTMLButtonElement>('[data-testid="activity-popup-view-all"]')
    expect(viewAll).not.toBeNull()
    viewAll?.click()
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/activities')
  })
})
