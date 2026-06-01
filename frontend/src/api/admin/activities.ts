import { apiClient } from '../client'
import type {
  FirstRechargeAdminConfig,
  FirstRechargeEligibilityScope,
  FirstRechargeOfferInput,
  FirstRechargeSpecifiedUser,
} from '@/types/payment'
import type { BasePaginationResponse } from '@/types'

export interface UpdateFirstRechargeConfigRequest {
  enabled: boolean
  eligibility_scope: FirstRechargeEligibilityScope
  offers: FirstRechargeOfferInput[]
}

export interface FirstRechargeUserLookupItem {
  id: number
  email: string
  username: string
}

export const adminActivitiesAPI = {
  getFirstRecharge() {
    return apiClient.get<FirstRechargeAdminConfig>('/admin/activities/first-recharge')
  },

  updateFirstRecharge(data: UpdateFirstRechargeConfigRequest) {
    return apiClient.put<FirstRechargeAdminConfig>('/admin/activities/first-recharge', data)
  },

  listFirstRechargeUsers(params?: { page?: number; page_size?: number; search?: string }) {
    return apiClient.get<BasePaginationResponse<FirstRechargeSpecifiedUser>>(
      '/admin/activities/first-recharge/users',
      { params },
    )
  },

  lookupFirstRechargeUsers(q: string) {
    return apiClient.get<FirstRechargeUserLookupItem[]>(
      '/admin/activities/first-recharge/users/lookup',
      { params: { q } },
    )
  },

  addFirstRechargeUser(userId: number) {
    return apiClient.post<{ user_id: number }>('/admin/activities/first-recharge/users', {
      user_id: userId,
    })
  },

  removeFirstRechargeUser(userId: number) {
    return apiClient.delete<{ user_id: number }>(`/admin/activities/first-recharge/users/${userId}`)
  },
}

export default adminActivitiesAPI
