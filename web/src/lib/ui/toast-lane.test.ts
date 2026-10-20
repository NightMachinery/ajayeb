import { describe, expect, it } from 'vitest';
import { clear, clearSpan, fitPanel, overlaps, type Box, type Lane } from './toast-lane';

const box = (left: number, top: number, w: number, h: number): Box => ({
  left,
  top,
  right: left + w, // physical-ok: viewport coordinates
  bottom: top + h
});

describe('toast lanes', () => {
  it('counts a shared area as an overlap, never a shared edge', () => {
    expect(overlaps(box(0, 0, 10, 10), box(5, 5, 10, 10))).toBe(true);
    expect(overlaps(box(0, 0, 10, 10), box(10, 0, 10, 10))).toBe(false);
    expect(overlaps(box(0, 0, 10, 10), box(0, 10.4, 10, 10))).toBe(false);
  });
  it('is clear of every obstacle or not at all', () => {
    const cards = [box(0, 50, 40, 60), box(50, 50, 40, 60)];
    expect(clear(box(0, 0, 90, 48), cards)).toBe(true);
    expect(clear(box(0, 0, 90, 52), cards)).toBe(false);
  });
  it('lets an empty list sit anywhere', () => {
    expect(clear(box(0, 60, 90, 0), [box(0, 50, 40, 60)])).toBe(true);
  });
});

const lane = (
  name: Lane['name'],
  b: Box,
  edge: Lane['edge'],
  width = b.right - b.left // physical-ok: viewport coordinates
): Lane => ({ name, box: { ...b, width: b.right - b.left }, edge, width }); // physical-ok

describe('summary lanes', () => {
  // A board like the finding's at 844x390: the structure from 49 down,
  // with Press takeable in its middle and cards to take along its foot,
  // and the opponent's column beside it, its strip at the top.
  const structure = box(230, 49, 344, 341);
  const press = box(424, 161, 74, 114);
  const foot = [box(240, 300, 74, 90), box(330, 300, 74, 90), box(480, 300, 74, 90)];
  const away = box(574, 49, 270, 341);
  const awayStrip = box(574, 49, 270, 150);
  const lanes = [
    lane('head', structure, 'top', 384),
    lane('foot', structure, 'bottom', 384),
    lane('away', away, 'bottom')
  ];

  it('measures the clear height from the edge to the nearest card in its span', () => {
    expect(clearSpan(lanes[0]!, 234, 570, [press])).toEqual({ top: 53, bottom: 157 });
    expect(clearSpan(lanes[0]!, 234, 400, [press])).toEqual({ top: 53, bottom: 386 });
    expect(clearSpan(lanes[1]!, 234, 570, foot)).toEqual({ top: 394, bottom: 386 });
  });
  it('never takes the head when a card to take lies there, but the free column foot', () => {
    const fit = fitPanel(lanes, [press, ...foot, awayStrip], () => 157);
    expect(fit?.lane).toBe('away');
    expect(fit && fit.top).toBe(386 - 157);
    expect(
      fit &&
        clear({ ...fit, right: fit.left + fit.width, bottom: fit.top + fit.height }, [
          press,
          ...foot,
          awayStrip
        ])
    ).toBe(true); // physical-ok
  });
  it('takes the first lane where it fits whole', () => {
    expect(fitPanel(lanes, [press, ...foot], () => 100)?.lane).toBe('head');
  });
  it('scrolls in the roomiest lane when none fits whole, and gives up below the minimum', () => {
    const fit = fitPanel(lanes, [press, ...foot, awayStrip], () => 400);
    expect(fit?.lane).toBe('away');
    expect(fit?.height).toBe(386 - 203);
    expect(fitPanel(lanes, [box(0, 49, 844, 300), ...foot], () => 400)).toBeNull();
  });
  it('takes a lane shorter than MIN_PANEL that still shows the title and a line', () => {
    const two = [lanes[0]!, lanes[1]!];
    // The head has 104 px above Press; the foot has none.
    expect(fitPanel(two, [press, ...foot], () => 259)).toBeNull();
    const fit = fitPanel(
      two,
      [press, ...foot],
      () => 259,
      () => 86
    );
    expect(fit).toMatchObject({ lane: 'head', top: 53, height: 104 });
    expect(fitPanel(two, [press, ...foot], () => 259, 105)).toBeNull();
    // A title that wraps tall asks for no more than MIN_PANEL.
    expect(fitPanel(lanes, [press, ...foot, awayStrip], () => 400, 300)?.lane).toBe('away');
  });
  it('skips a lane too narrow for the panel', () => {
    const narrow = [lane('away', box(700, 49, 140, 341), 'bottom')];
    expect(fitPanel(narrow, [], () => 100)).toBeNull();
  });
});
