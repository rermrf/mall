type PendingRequest = {
  resolve: (token: string) => void
  reject: (error: unknown) => void
}

export function createRefreshFlow() {
  let refreshing = false
  let pendingRequests: PendingRequest[] = []

  return {
    isRefreshing() {
      return refreshing
    },
    begin() {
      refreshing = true
    },
    waitForToken() {
      return new Promise<string>((resolve, reject) => {
        pendingRequests.push({ resolve, reject })
      })
    },
    succeed(token: string) {
      pendingRequests.forEach(({ resolve }) => resolve(token))
      pendingRequests = []
      refreshing = false
    },
    fail(error: unknown) {
      pendingRequests.forEach(({ reject }) => reject(error))
      pendingRequests = []
      refreshing = false
    },
    buildRefreshHeaders(refreshToken: string) {
      return {
        'X-Refresh-Token': refreshToken,
      }
    },
  }
}
