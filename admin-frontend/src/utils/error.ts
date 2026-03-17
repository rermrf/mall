export function handleApiError(context: string) {
  return (err: unknown) => {
    console.error(`[${context}]`, err)
    if (err instanceof Error && err.message === 'canceled') return
  }
}

export function silentApiError(context: string) {
  return (err: unknown) => {
    console.error(`[${context}]`, err)
  }
}
