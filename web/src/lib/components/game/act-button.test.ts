import { describe, expect, it } from 'vitest';
import { actButtonState } from './act-button';

const base = { label: 'Build for 3', canAct: true, connected: true, confirm: false };

describe('actButtonState: one rule for every act button', () => {
  it('a local room sends on the first tap', () => {
    expect(actButtonState(base)).toEqual({ label: 'Build for 3', disabled: false, confirm: false });
  });

  it('an online room (or Confirm moves: Always) arms first', () => {
    expect(actButtonState({ ...base, confirm: true })).toEqual({
      label: 'Build for 3',
      disabled: false,
      confirm: true
    });
  });

  it('while the connection is down: disabled, waiting for it', () => {
    for (const confirm of [false, true]) {
      expect(actButtonState({ ...base, confirm, connected: false })).toEqual({
        label: 'Waiting for connection',
        disabled: true,
        confirm
      });
    }
    // Even if the caller still thinks input is open.
    expect(actButtonState({ ...base, canAct: true, connected: false }).disabled).toBe(true);
  });

  it('input closed (a request in flight, not this seat) disables it, label kept', () => {
    expect(actButtonState({ ...base, canAct: false })).toEqual({
      label: 'Build for 3',
      disabled: true,
      confirm: false
    });
  });

  it('a commit button before its pick is disabled', () => {
    expect(actButtonState({ ...base, ready: false }).disabled).toBe(true);
    expect(actButtonState({ ...base, ready: true }).disabled).toBe(false);
  });
});
