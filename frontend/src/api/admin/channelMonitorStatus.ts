/**
 * Admin channel status API endpoints.
 * Read-only status views for administrators; limited by monitor enabled state
 * only, not by user_visible.
 */

import { apiClient } from '../client'
import type {
  UserMonitorDetail,
  UserMonitorListResponse,
} from '../channelMonitor'

export type {
  MonitorTimelinePoint,
  UserMonitorDetail,
  UserMonitorExtraModel,
  UserMonitorListResponse,
  UserMonitorModelDetail,
  UserMonitorView,
} from '../channelMonitor'

export async function list(options?: { signal?: AbortSignal }): Promise<UserMonitorListResponse> {
  const { data } = await apiClient.get<UserMonitorListResponse>('/admin/channel-monitor-status', {
    signal: options?.signal,
  })
  return data
}

export async function status(id: number): Promise<UserMonitorDetail> {
  const { data } = await apiClient.get<UserMonitorDetail>(`/admin/channel-monitor-status/${id}`)
  return data
}

export const channelMonitorStatusAPI = {
  list,
  status,
}

export default channelMonitorStatusAPI
