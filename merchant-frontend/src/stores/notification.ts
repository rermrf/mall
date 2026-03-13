import { create } from 'zustand'
import { getUnreadCount } from '@/api/notification'

interface NotificationState {
  unreadCount: number
  fetchUnreadCount: () => Promise<void>
}

export const useNotificationStore = create<NotificationState>((set) => ({
  unreadCount: 0,
  fetchUnreadCount: async () => {
    try {
      const count = await getUnreadCount()
      set({ unreadCount: count ?? 0 })
    } catch {
      // ignore
    }
  },
}))
