import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { CUES } from './cues';
import { SOUND_FILES } from './sounds.gen';

const repo = new URL('../../../../', import.meta.url);

describe('approved sound inventory', () => {
  it('names exactly the player cues with the actual shipped content hashes', () => {
    expect(Object.keys(SOUND_FILES).sort()).toEqual([...CUES].sort());
    let bytes = 0;
    for (const cue of CUES) {
      const entry = SOUND_FILES[cue];
      const match = /^\/sfx\/([a-zA-Z]+)\.m4a\?v=([0-9a-f]{12})$/.exec(entry.url);
      expect(match, cue).not.toBeNull();
      expect(match![1]).toBe(cue);
      const body = readFileSync(new URL(`web/static/sfx/${cue}.m4a`, repo));
      expect(createHash('sha256').update(body).digest('hex').slice(0, 12)).toBe(match![2]);
      bytes += body.length;
    }
    expect(bytes).toBeLessThan(160_000);
  });

  it('separates archive checksums from member checksums', () => {
    const sources = JSON.parse(
      readFileSync(new URL('art/originals/sound/sources.json', repo), 'utf8')
    );
    const archives = new Map<string, string>();
    for (const cue of CUES) {
      const source = sources[cue];
      if (!source.archiveMember) continue;
      expect(source.memberSha256).toMatch(/^[0-9a-f]{64}$/);
      expect(source.downloadSha256).not.toBe(source.memberSha256);
      const previous = archives.get(source.download);
      if (previous) expect(source.downloadSha256).toBe(previous);
      archives.set(source.download, source.downloadSha256);
    }
  });

  it('records every source as CC0 and preserves the owner-selected turn swap', () => {
    const sources = JSON.parse(
      readFileSync(new URL('art/originals/sound/sources.json', repo), 'utf8')
    );
    expect(Object.keys(sources).sort()).toEqual([...CUES].sort());
    for (const cue of CUES) {
      expect(sources[cue].license).toBe('CC0-1.0');
      expect(readFileSync(new URL(sources[cue].original, repo)).length).toBeGreaterThan(0);
    }
    expect(sources.yourTurn.selection).toBe('extraTurn-03');
    expect(sources.extraTurn.selection).toBe('yourTurn-00');
  });
});
