import { apiClient } from './client'
import type { FirstRechargeStatus, UserActivity } from '@/types/payment'

export const activitiesAPI = {
  list() {
    return apiClient.get<UserActivity[]>('/activities')
  },

  markViewed(activityId: string) {
    return apiClient.post<{ message: string }>(`/activities/${encodeURIComponent(activityId)}/view`)
  },
}

export const firstRechargeAPI = {
  getStatus() {
    return apiClient.get<FirstRechargeStatus>('/activities/first-recharge/status')
  },

  dismissPopup() {
    return apiClient.post<{ message: string }>('/activities/first-recharge/dismiss-popup')
  },
}

export default firstRechargeAPI
