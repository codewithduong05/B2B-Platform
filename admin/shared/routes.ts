export interface RouteEntry {
  route: string
  title: string
  group: string
  requiredPermission?: string
}

export function findRouteEntry(
  route: string,
  routes: readonly RouteEntry[],
): RouteEntry | undefined {
  return routes.find((entry) => entry.route === route)
}
