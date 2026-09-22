import { findRouteEntry as sharedFindRouteEntry, type RouteEntry } from '#shared/routes'

export const routeRegistry: RouteEntry[] = [
  { route: '/', title: 'Store', group: 'Storefront' },
  { route: '/health', title: 'Health', group: 'Service status' },
]

export function findRouteEntry(route: string): RouteEntry | undefined {
  return sharedFindRouteEntry(route, routeRegistry)
}
