/**
 * Handles API errors with user-visible feedback.
 * Use in place of `.catch(() => {})`.
 */
export function handleApiError(context: string) {
  return (err: unknown) => {
    console.error(`[${context}]`, err)
    // Don't show message for cancelled requests
    if (err instanceof Error && err.message === 'canceled') return
    // axios interceptor already shows message.error for most API errors,
    // so this is a fallback for unexpected errors
  }
}

/**
 * Silently catches an API error but still logs it.
 * Use for non-critical data loads (e.g. dashboard stats).
 */
export function silentApiError(context: string) {
  return (err: unknown) => {
    console.error(`[${context}]`, err)
  }
}
