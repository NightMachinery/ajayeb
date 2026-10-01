import { describe, expect, it } from 'vitest';
import { parseSafeParam } from './safe-area';

describe('parseSafeParam', () => {
  it('reads top and bottom', () => {
    expect(parseSafeParam('47,34')).toEqual({ top: 47, right: 0, bottom: 34, left: 0 });
  });

  it('reads all four sides in CSS order', () => {
    expect(parseSafeParam('0, 47, 21, 47')).toEqual({ top: 0, right: 47, bottom: 21, left: 47 });
  });

  it('rejects missing and malformed values', () => {
    expect(parseSafeParam(null)).toBeNull();
    expect(parseSafeParam('')).toBeNull();
    expect(parseSafeParam('47')).toBeNull();
    expect(parseSafeParam('1,2,3')).toBeNull();
    expect(parseSafeParam('47,-3')).toBeNull();
    expect(parseSafeParam('47,1e3')).toBeNull();
    expect(parseSafeParam('47,34px')).toBeNull();
  });
});
