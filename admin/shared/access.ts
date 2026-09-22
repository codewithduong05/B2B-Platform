import type { Principal } from './principal'
import type { RouteEntry } from './routes'

export function canAccess(permission: string | undefined, principal: Principal): boolean {
  if (permission === undefined) {
    return true
  }
  return principal.permissions.includes(permission)
}

export function landingPathFor(principal: Principal, routes: readonly RouteEntry[]): string {
  for (const entry of routes) {
    if (canAccess(entry.requiredPermission, principal)) {
      return entry.route
    }
  }
  return '/'
}
