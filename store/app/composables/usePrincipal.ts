import { computed, watch } from 'vue'
import type { Principal } from '../../shared/principal'

export function usePrincipal() {
  const session = useFetch<Principal>('/api/session')
  const pending = session.pending
  const error = session.error
  const data = session.data

  const principal = computed<Principal | null>(() => {
    if (error.value || pending.value) {
      return null
    }
    return data.value ?? null
  })

  let loadPromise: Promise<void> | null = null

  function ensureLoaded(): Promise<void> {
    if (loadPromise === null) {
      loadPromise = new Promise<void>((resolve) => {
        const stop = watch(
          () => pending.value || (data.value === null && error.value === null),
          (notSettled) => {
            if (!notSettled) {
              resolve()
              stop?.()
            }
          },
          { immediate: true },
        )
      })
    }
    return loadPromise
  }

  return { principal, ensureLoaded }
}
