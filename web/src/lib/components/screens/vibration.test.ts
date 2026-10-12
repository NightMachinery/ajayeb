import { describe, expect, it } from 'vitest';
import { vibrationShown } from './vibration';

const vibrate = () => true;

describe('the Vibration switch', () => {
  it('shows on a touch device that can vibrate (an Android phone)', () => {
    expect(vibrationShown({ vibrate, maxTouchPoints: 5 })).toBe(true);
  });

  it('hides on desktop Chrome, which has vibrate but no touch', () => {
    expect(vibrationShown({ vibrate, maxTouchPoints: 0 })).toBe(false);
  });

  it('hides where the browser cannot vibrate (iOS) or there is no navigator', () => {
    expect(vibrationShown({ maxTouchPoints: 5 })).toBe(false);
    expect(vibrationShown(undefined)).toBe(false);
  });
});
