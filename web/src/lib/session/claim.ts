// Device links (docs/protocol.md, "Device links"): /play/<code>#claim=<claim>
// moves one seat of one room to another device. The claim is a room-scoped
// capability the server derives from the seat's user (package ident); it
// is not the identity token and works in no other room. It sits in the
// fragment, so it never reaches the server in an HTTP request or a log, and
// it stays in the address bar, so a reload keeps the same seat.
//
// Pure helpers: parsing the fragment and building the link.

/** The fragment parameter that carries a claim. */
export const CLAIM_PARAM = 'claim';

/** The length of a claim: 22 base64url characters. */
export const CLAIM_LENGTH = 22;

/** Whether a value has the shape of a claim: 22 base64url characters. */
export function isValidClaim(claim: unknown): claim is string {
  return typeof claim === 'string' && /^[A-Za-z0-9_-]{22}$/.test(claim);
}

/**
 * The claim in a URL fragment ("#claim=..."; other parameters are allowed
 * around it), or null when there is none or it is not shaped like one.
 */
export function parseClaimFragment(hash: string): string | null {
  const body = hash.startsWith('#') ? hash.slice(1) : hash;
  if (!body) return null;
  const claim = new URLSearchParams(body).get(CLAIM_PARAM);
  return isValidClaim(claim) ? claim : null;
}

/** The fragment that carries a claim, with its "#". */
export function claimFragment(claim: string): string {
  return `#${CLAIM_PARAM}=${claim}`;
}

/** A seat's device link: the room's invite link with the claim in the fragment. */
export function deviceLink(invite: string, claim: string): string {
  const url = new URL(invite);
  url.hash = `${CLAIM_PARAM}=${claim}`;
  return url.href;
}
