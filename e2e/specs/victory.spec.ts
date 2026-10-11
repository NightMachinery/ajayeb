// Every way a game ends, each from a named scenario (server/internal/engine/
// scenarios_named.go) played through the real UI: the scenario opens a
// hot-seat room just before its trigger, the spec plays the trigger's action
// keys with the clicks a player makes, and the result screen must name the
// victory and the winner the scenario's Expect describes and show the
// server's own score sheet (window.__ajayeb), not numbers computed here.

import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import {
  currentView,
  expectResultScreen,
  openHotseatRoom,
  playKey,
  type ExpectedResult
} from '../fixtures/scenario';
import type { Flow } from '../fixtures/driver';

interface Case {
  scenario: string;
  title: string;
  /** The trigger, with the flow to play each key by when not the default. */
  trigger: readonly (string | [string, Flow])[];
  expected: ExpectedResult;
}

const CASES: readonly Case[] = [
  {
    scenario: 'military-supremacy',
    title: 'military supremacy: two shields reach the capital',
    trigger: ['construct:walls'],
    expected: { victory: 'military', winner: 0 }
  },
  {
    scenario: 'military-via-wonder',
    title: "military supremacy: a wonder's shield ends the game before its destroy choice",
    trigger: [['wonder:statue-of-zeus:rostrum', 'wonderFirst']],
    expected: { victory: 'military', winner: 0 }
  },
  {
    scenario: 'science-supremacy',
    title: 'scientific supremacy: a sixth distinct symbol from a card',
    trigger: ['construct:university'],
    expected: { victory: 'science', winner: 0 }
  },
  {
    scenario: 'law-supremacy',
    title: "scientific supremacy: Law's scales as the sixth symbol",
    trigger: ['construct:academy', 'token:law'],
    expected: { victory: 'science', winner: 0 }
  },
  {
    scenario: 'civilian-end',
    title: 'civilian victory: the last Age III card ends the game',
    trigger: ['discard:academy'],
    expected: { victory: 'civilian', winner: 0 }
  },
  {
    scenario: 'tie-blue',
    title: 'civilian victory: a tie on points broken by blue cards',
    trigger: ['discard:academy'],
    expected: { victory: 'civilian', winner: 0, tieBreak: true }
  },
  {
    scenario: 'draw',
    title: 'shared victory: equal points and equal blue points',
    trigger: ['discard:academy'],
    expected: { victory: 'shared', winner: null }
  }
];

test.describe('victory', { tag: ['@mobile'] }, () => {
  for (const c of CASES) {
    test(`${c.title} (${c.scenario})`, async ({ page }) => {
      const problems = watchHygiene(page);
      const driver = await openHotseatRoom(page, c.scenario);

      for (const step of c.trigger) {
        const [key, flow] = typeof step === 'string' ? [step, undefined] : step;
        const before = await currentView(page);
        expect(before.phase, `the game is still on before ${key}`).not.toBe('gameOver');
        await playKey(driver, key, flow);
      }

      const end = await driver.nextTurn();
      expect(end.phase).toBe('gameOver');
      const result = await expectResultScreen(page, c.expected);
      const view = await currentView(page);

      switch (c.scenario) {
        case 'military-supremacy':
        case 'military-via-wonder':
          // The pawn stands in seat 1's capital, and the destroy choice the
          // Statue of Zeus would open never did.
          expect(view.military.position).toBeGreaterThanOrEqual(9);
          expect(view.pending).toBeUndefined();
          break;
        case 'law-supremacy':
          expect(view.players[0].tokens).toContain('law');
          break;
        case 'tie-blue':
          expect(result.scores[0].total).toBe(result.scores[1].total);
          expect(result.scores[0].blue).toBeGreaterThan(result.scores[1].blue);
          break;
        case 'draw':
          expect(result.scores[0].total).toBe(result.scores[1].total);
          expect(result.scores[0].blue).toBe(result.scores[1].blue);
          break;
        case 'civilian-end':
          expect(result.scores[0].total).toBeGreaterThan(result.scores[1].total);
          break;
      }

      // The turn bar says the game is over, and no move is offered.
      await expect(driver.visible('[data-turn]')).toHaveAttribute('data-turn', 'gameOver');
      expect(view.choices).toEqual([]);
      expect(problems, 'console errors, page errors or failed requests').toEqual([]);
    });
  }
});
