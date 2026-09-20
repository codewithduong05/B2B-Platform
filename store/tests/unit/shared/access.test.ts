import { describe, it, expect } from 'vitest'
import { canAccess, landingPathFor } from '#shared/access'
import type { Principal } from '#shared/principal'
import type { RouteEntry } from '#shared/routes'

const routes: readonly RouteEntry[] = [
  { route: '/', title: 'Store', group: 'public' },
  { route: '/health', title: 'Health', group: 'admin', requiredPermission: 'admin:health' },
  { route: '/settings', title: 'Settings', group: 'admin', requiredPermission: 'admin:settings' },
]

describe('canAccess', () => {
  it('returns true when principal has the required permission', () => {
    const principal: Principal = { id: 'u1', permissions: ['admin:health'] }
    expect(canAccess('admin:health', principal)).toBe(true)
  })

  it('returns false when principal lacks the required permission', () => {
    const principal: Principal = { id: 'u2', permissions: ['store:read'] }
    expect(canAccess('admin:health', principal)).toBe(false)
  })

  it('returns false when principal has empty permissions', () => {
    const principal: Principal = { id: 'u3', permissions: [] }
    expect(canAccess('admin:health', principal)).toBe(false)
  })

  it('returns true when permission is undefined', () => {
    const principal: Principal = { id: 'u4', permissions: [] }
    expect(canAccess(undefined, principal)).toBe(true)
  })
})

describe('landingPathFor', () => {
  it('returns first route without requiredPermission', () => {
    const principal: Principal = { id: 'u1', permissions: ['admin:health'] }
    expect(landingPathFor(principal, routes)).toBe('/')
  })

  it('returns first accessible route when first route requires permission', () => {
    const restrictedRoutes: readonly RouteEntry[] = [
      { route: '/admin', title: 'Admin', group: 'admin', requiredPermission: 'admin:access' },
      { route: '/public', title: 'Public', group: 'public' },
    ]
    const principal: Principal = { id: 'u2', permissions: [] }
    expect(landingPathFor(principal, restrictedRoutes)).toBe('/public')
  })

  it('falls back to / when no routes are accessible', () => {
    const restrictedRoutes: readonly RouteEntry[] = [
      { route: '/admin', title: 'Admin', group: 'admin', requiredPermission: 'admin:access' },
    ]
    const principal: Principal = { id: 'u3', permissions: [] }
    expect(landingPathFor(principal, restrictedRoutes)).toBe('/')
  })

  it('returns first accessible route matching principal permissions', () => {
    const principal: Principal = { id: 'u4', permissions: ['admin:health'] }
    expect(landingPathFor(principal, routes)).toBe('/')
  })
})
