import { findRouteEntry as sharedFindRouteEntry, type RouteEntry } from '../../shared/routes'

export const routeRegistry: RouteEntry[] = [
  { route: '/dashboard', title: 'Dashboard', group: 'Overview' },
  { route: '/products', title: 'Products', group: 'Catalog', requiredPermission: 'catalog.view' },
  { route: '/orders', title: 'Orders', group: 'Commerce', requiredPermission: 'orders.view' },
  { route: '/customers', title: 'Customers', group: 'Customers', requiredPermission: 'users.view' },
  { route: '/inventory', title: 'Inventory', group: 'Inventory', requiredPermission: 'inventory.view' },
  { route: '/settings', title: 'Settings', group: 'Platform', requiredPermission: 'settings.view' },
]

export function findRouteEntry(route: string): RouteEntry | undefined {
  return sharedFindRouteEntry(route, routeRegistry)
}
