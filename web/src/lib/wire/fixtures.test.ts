import { describe, expect, it } from 'vitest';
import { fixtures, protocolFixtures } from './fixtures.gen';
import type { ServerMessage } from './index';
import type { View } from './view.gen';

// fixtures.gen.ts already proves fixtures satisfies ServerMessage[] and
// protocolFixtures satisfies View[] at the type level (make check-web /
// svelte-check), the moment a real recorded server shape no longer fits the
// generated TypeScript. These are runtime sanity checks on top of that: that
// the generator actually ran (non-empty arrays) and that nothing in a real
// frame or view is a JSON null, which a hand-written example could miss but
// a nil Go slice marshaled without an explicit [] would produce.

function findNull(value: unknown, path: string): string | undefined {
  if (value === null) return path || '(root)';
  if (Array.isArray(value)) {
    for (let i = 0; i < value.length; i++) {
      const found = findNull(value[i], `${path}[${i}]`);
      if (found) return found;
    }
    return undefined;
  }
  if (typeof value === 'object') {
    for (const [key, v] of Object.entries(value as Record<string, unknown>)) {
      const found = findNull(v, path ? `${path}.${key}` : key);
      if (found) return found;
    }
  }
  return undefined;
}

describe('fixtures', () => {
  it('is non-empty', () => {
    expect(fixtures.length).toBeGreaterThan(0);
  });

  it('has no JSON null anywhere', () => {
    fixtures.forEach((frame: ServerMessage, i) => {
      const found = findNull(frame, '');
      expect(found, `fixtures[${i}] (${frame.t}) has a null at ${found}`).toBeUndefined();
    });
  });

  it('covers every recorded frame type', () => {
    const types = new Set(fixtures.map((f: ServerMessage) => f.t));
    expect(types).toEqual(new Set(['welcome', 'created', 'state', 'reject', 'error']));
  });
});

describe('protocolFixtures', () => {
  it('is non-empty', () => {
    expect(protocolFixtures.length).toBeGreaterThan(0);
  });

  it('has no JSON null anywhere', () => {
    protocolFixtures.forEach((view: View, i) => {
      const found = findNull(view, '');
      expect(found, `protocolFixtures[${i}] has a null at ${found}`).toBeUndefined();
    });
  });
});
