// What the admin menus and the spectator panel read from a frame: the
// people watching, who may be seated, and who plays a former player's seat.
import { describe, expect, it } from 'vitest';
import { tableFrame } from '../session/test-frames';
import { formerOf, isAdmin, isTemporary, occupant, roleMap, seatable, watchers } from './admin';

describe('admin reads', () => {
  it('lists the watchers with their roles, and seats only the connected ones', () => {
    const f = tableFrame({}, 0);
    const away = {
      ...f,
      humans: [
        ...f.humans,
        {
          userId: 'x',
          name: 'Wen',
          nameNo: 2,
          connected: false,
          creator: false,
          admin: true,
          temporary: false
        }
      ]
    };
    expect(watchers(away).map((p) => [p.userId, p.name, p.role, p.connected])).toEqual([
      ['w', 'Wen', null, true],
      ['x', 'Wen 2', 'admin', false]
    ]);
    expect(seatable(away).map((p) => p.userId)).toEqual(['w']);
  });

  it('names the human in a seat, and none for a bot', () => {
    const f = tableFrame({}, 0);
    expect(occupant(f, 0)).toMatchObject({ userId: 'u0', role: 'creator' });
    expect(occupant(f, 2)).toBeNull();
  });

  it('gives the seat powers to an admin or the temporary admin only', () => {
    expect(isAdmin(tableFrame({}, 0))).toBe(true);
    expect(isAdmin(tableFrame({}, 1))).toBe(false);
    const temp = tableFrame({}, 1);
    const asTemp = {
      ...temp,
      humans: temp.humans.map((h) => (h.userId === 'u1' ? { ...h, temporary: true } : h))
    };
    expect(isAdmin(asTemp)).toBe(true);
    expect(isTemporary(asTemp)).toBe(true);
    expect(roleMap(asTemp)['w']).toEqual({ promote: false, demote: false });
    expect(roleMap(tableFrame({}, 0))['w']).toEqual({ promote: true, demote: false });
  });

  it('says who plays a former player seat, from memory or an admin frame', () => {
    // Seat 2 is a bot's; seat 1 is Bo's.
    const watching = tableFrame({}, null);
    expect(formerOf(watching, null)).toBeNull();
    expect(formerOf(watching, 2)).toEqual({ seat: 2, by: 'bot', name: '' });
    expect(formerOf(watching, 1)).toEqual({ seat: 1, by: 'human', name: 'Bo' });
    const admin = { ...watching, you: { userId: 'w', previousSeats: [{ userId: 'w', seat: 2 }] } };
    expect(formerOf(admin, null)).toEqual({ seat: 2, by: 'bot', name: '' });
    // A seated viewer is no former player.
    expect(formerOf(tableFrame({}, 0), 1)).toBeNull();
  });
});
