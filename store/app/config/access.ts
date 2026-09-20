import { canAccess, landingPathFor } from '../../shared/access'
import type { Principal } from '../../shared/principal'
import { routeRegistry } from '~/config/routes'

export function isRouteAccessible(route: string, principal: Principal): boolean {
  const entry = routeRegistry.find((item) => item.route === route)
  return canAccess(entry?.requiredPermission, principal)
}

export function resolveLandingPath(principal: Principal): string {
  return landingPathFor(principal, routeRegistry)
}
