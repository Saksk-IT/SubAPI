import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { activitiesAPI } from '@/api/activities'
import type { UserActivity } from '@/types/payment'

const THROTTLE_MS = 5 * 60 * 1000

export const useActivityStore = defineStore('activities', () => {
  const activities = ref<UserActivity[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  const lastFetchTime = ref(0)
  const sessionDismissedIds = ref<Set<string>>(new Set())

  const unviewedActivities = computed(() =>
    activities.value.filter((activity) => !activity.viewed_at)
  )

  const popupActivities = computed(() =>
    unviewedActivities.value.filter((activity) => !sessionDismissedIds.value.has(activity.id))
  )

  async function fetchActivities(force = false): Promise<UserActivity[]> {
    const now = Date.now()
    if (!force && loaded.value && now - lastFetchTime.value < THROTTLE_MS) {
      return activities.value
    }
    if (loading.value) return activities.value

    loading.value = true
    try {
      const response = await activitiesAPI.list()
      activities.value = response.data || []
      loaded.value = true
      lastFetchTime.value = now
      return activities.value
    } catch (error: unknown) {
      lastFetchTime.value = 0
      console.error('[activities] Failed to fetch activities:', error)
      return activities.value
    } finally {
      loading.value = false
    }
  }

  async function markViewed(activityId: string): Promise<void> {
    const activity = activities.value.find((item) => item.id === activityId)
    if (!activity || activity.viewed_at) return

    await activitiesAPI.markViewed(activityId)
    activity.viewed_at = new Date().toISOString()
  }

  async function markAllViewed(): Promise<void> {
    const ids = unviewedActivities.value.map((activity) => activity.id)
    await Promise.all(ids.map(markViewed))
  }

  function dismissPopupForSession(): void {
    sessionDismissedIds.value = new Set([
      ...sessionDismissedIds.value,
      ...popupActivities.value.map((activity) => activity.id),
    ])
  }

  function reset(): void {
    activities.value = []
    loading.value = false
    loaded.value = false
    lastFetchTime.value = 0
    sessionDismissedIds.value = new Set()
  }

  return {
    activities,
    loading,
    loaded,
    unviewedActivities,
    popupActivities,
    fetchActivities,
    markViewed,
    markAllViewed,
    dismissPopupForSession,
    reset,
  }
})
