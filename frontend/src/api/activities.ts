import { apiClient } from './client'
import type { FirstRechargeStatus } from '@/types/payment'

export const firstRechargeAPI = {
  getStatus() {
    return apiClient.get<FirstRechargeStatus>('/activities/first-recharge/status')
  },

  dismissPopup() {
    return apiClient.post<{ message: string }>('/activities/first-recharge/dismiss-popup')
  },
}

export default firstRechargeAPI
