import { findRouteEntry } from '~/config/routes'
import { usePrincipal } from '~/composables/usePrincipal'

export default defineNuxtRouteMiddleware(async (to) => {
  const entry = findRouteEntry(to.path)
  const permission = entry?.requiredPermission

  if (permission === undefined) {
    return
  }

  const { principal, ensureLoaded } = usePrincipal()
  await ensureLoaded()

  if (principal.value === null) {
    return navigateTo('/')
  }

  if (!principal.value.permissions.includes(permission)) {
    throw createError({
      statusCode: 403,
      statusMessage: 'Permission required',
    })
  }
})
