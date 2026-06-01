import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { firstRechargeAPI } from '@/api/activities'
import type { FirstRechargeStatus } from '@/types/payment'

const THROTTLE_MS = 5 * 60 * 1000

export const useFirstRechargeStore = defineStore('firstRecharge', () => {
  const status = ref<FirstRechargeStatus | null>(null)
  const loading = ref(false)
  const loaded = ref(false)
  const lastFetchTime = ref(0)

  const available = computed(() => {
    const current = status.value
    return !!(
      current?.enabled
      && current.eligible
      && !current.completed
      && current.offers?.length > 0
    )
  })

  async function fetchStatus(force = false): Promise<FirstRechargeStatus | null> {
    const now = Date.now()
    if (!force && loaded.value && now - lastFetchTime.value < THROTTLE_MS) {
      return status.value
    }
    if (loading.value) {
      return status.value
    }

    loading.value = true
    try {
      const response = await firstRechargeAPI.getStatus()
      status.value = response.data
      loaded.value = true
      lastFetchTime.value = now
      return status.value
    } catch (error: unknown) {
      lastFetchTime.value = 0
      console.error('[firstRecharge] Failed to fetch status:', error)
      return null
    } finally {
      loading.value = false
    }
  }

  async function dismissPopup(): Promise<void> {
    try {
      await firstRechargeAPI.dismissPopup()
      status.value = status.value
        ? { ...status.value, popup_dismissed: true }
        : status.value
    } catch (error: unknown) {
      console.error('[firstRecharge] Failed to dismiss popup:', error)
      throw error
    }
  }

  function clear(): void {
    status.value = null
    loading.value = false
    loaded.value = false
    lastFetchTime.value = 0
  }

  return {
    status,
    loading,
    loaded,
    available,
    fetchStatus,
    dismissPopup,
    clear,
  }
})
