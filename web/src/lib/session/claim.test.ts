import { describe, expect, it } from 'vitest';
import { claimFragment, deviceLink, isValidClaim, parseClaimFragment } from './claim';

const CLAIM = 'q3Vb8Kx0Zr7TnW2mYc5JhA';

describe('isValidClaim', () => {
  it('takes 22 base64url characters', () => {
    expect(isValidClaim(CLAIM)).toBe(true);
    expect(isValidClaim('AAAAAAAAAAAAAAAAAAAA-_')).toBe(true);
  });
  it('refuses anything else', () => {
    for (const bad of [
      '',
      CLAIM.slice(1),
      CLAIM + 'A',
      CLAIM.slice(1) + '+',
      CLAIM.slice(1) + '/',
      CLAIM.slice(1) + '=',
      CLAIM.slice(1) + ' ',
      null,
      undefined,
      22
    ]) {
      expect(isValidClaim(bad)).toBe(false);
    }
  });
});

describe('parseClaimFragment', () => {
  it('reads the claim of a device link', () => {
    expect(parseClaimFragment(`#claim=${CLAIM}`)).toBe(CLAIM);
    expect(parseClaimFragment(`claim=${CLAIM}`)).toBe(CLAIM);
    expect(parseClaimFragment(`#x=1&claim=${CLAIM}&y=2`)).toBe(CLAIM);
  });
  it('gives null for no claim or a malformed one', () => {
    expect(parseClaimFragment('')).toBeNull();
    expect(parseClaimFragment('#')).toBeNull();
    expect(parseClaimFragment('#other=1')).toBeNull();
    expect(parseClaimFragment('#claim=')).toBeNull();
    expect(parseClaimFragment('#claim=short')).toBeNull();
    expect(parseClaimFragment(`#claim=${CLAIM}x`)).toBeNull();
  });
});

describe('deviceLink', () => {
  it('puts the claim in the fragment of the invite link', () => {
    const link = deviceLink('https://example.test/play/K7QX2M', CLAIM);
    expect(link).toBe(`https://example.test/play/K7QX2M#claim=${CLAIM}`);
    expect(parseClaimFragment(new URL(link).hash)).toBe(CLAIM);
  });
  it('replaces a fragment already there', () => {
    expect(deviceLink('https://example.test/play/K7QX2M#claim=old', CLAIM)).toBe(
      `https://example.test/play/K7QX2M#claim=${CLAIM}`
    );
  });
  it('builds the bare fragment', () => {
    expect(claimFragment(CLAIM)).toBe(`#claim=${CLAIM}`);
  });
});
