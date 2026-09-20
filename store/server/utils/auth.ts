import type { Principal } from '../../shared/principal'

export type SessionVerifier = (
  event: any,
  sessionToken: string,
) => Principal | null | Promise<Principal | null>

let registeredVerifier: SessionVerifier | null = null

export function registerSessionVerifier(verifier: SessionVerifier): void {
  registeredVerifier = verifier
}

export async function verifySessionToken(
  event: any,
  sessionToken: string,
): Promise<Principal | null> {
  if (registeredVerifier === null) {
    return null
  }
  return registeredVerifier(event, sessionToken)
}
