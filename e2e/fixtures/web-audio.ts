// Observe real WebAudio calls without replacing decode, resume or playback.
import type { Page } from '@playwright/test';

export interface AudioSnapshot {
  available: boolean;
  states: string[];
  decoded: { duration: number; channels: number }[];
  decodeErrors: number;
  fileStarts: number;
  synthStarts: number;
}

declare global {
  interface Window {
    __webAudioProbe?: {
      snapshot(): AudioSnapshot;
      suspend(): Promise<void>;
    };
  }
}

export async function observeWebAudio(page: Page): Promise<void> {
  await page.addInitScript(() => {
    const Native = window.AudioContext;
    const contexts: AudioContext[] = [];
    const decoded: { duration: number; channels: number }[] = [];
    const fileBuffers = new WeakSet<AudioBuffer>();
    let decodeErrors = 0;
    let fileStarts = 0;
    let synthStarts = 0;
    window.__webAudioProbe = {
      snapshot: () => ({
        available: typeof Native === 'function',
        states: contexts.map((c) => c.state),
        decoded: [...decoded],
        decodeErrors,
        fileStarts,
        synthStarts
      }),
      suspend: async () => {
        for (const context of contexts) await context.suspend();
      }
    };
    if (!Native) return;
    window.AudioContext = new Proxy(Native, {
      construct(target, args) {
        const context = Reflect.construct(target, args) as AudioContext;
        contexts.push(context);
        const decode = context.decodeAudioData.bind(context);
        context.decodeAudioData = (data: ArrayBuffer) => {
          let result: Promise<AudioBuffer>;
          try {
            result = decode(data);
          } catch (error) {
            decodeErrors++;
            throw error;
          }
          return result.then(
            (buffer) => {
              fileBuffers.add(buffer);
              decoded.push({ duration: buffer.duration, channels: buffer.numberOfChannels });
              return buffer;
            },
            (error: unknown) => {
              decodeErrors++;
              throw error;
            }
          );
        };
        const createSource = context.createBufferSource.bind(context);
        context.createBufferSource = () => {
          const source = createSource();
          const start = source.start.bind(source);
          source.start = (...values: Parameters<typeof start>) => {
            start(...values);
            if (source.buffer && fileBuffers.has(source.buffer)) fileStarts++;
            else synthStarts++;
          };
          return source;
        };
        const createOscillator = context.createOscillator.bind(context);
        context.createOscillator = () => {
          const oscillator = createOscillator();
          const start = oscillator.start.bind(oscillator);
          oscillator.start = (...values: Parameters<typeof start>) => {
            start(...values);
            synthStarts++;
          };
          return oscillator;
        };
        return context;
      }
    });
  });
}

export async function audioSnapshot(page: Page): Promise<AudioSnapshot> {
  return page.evaluate(() => {
    if (!window.__webAudioProbe) throw new Error('WebAudio observer did not install');
    return window.__webAudioProbe.snapshot();
  });
}

// Headless pages stay active. This explicitly simulates the app's visibility
// event and suspends the real context; it does not emulate an iOS interruption.
export async function audioVisibility(page: Page, state: 'hidden' | 'visible'): Promise<void> {
  await page.evaluate(async (visibility) => {
    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      get: () => visibility
    });
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      get: () => visibility === 'hidden'
    });
    document.dispatchEvent(new Event('visibilitychange'));
    if (visibility === 'hidden') await window.__webAudioProbe?.suspend();
  }, state);
}
