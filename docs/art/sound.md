# Sound preparation and playback

The approved set contains 17 CC0 cues. Sources, creator credits, exact download
URLs, download checksums, archive member paths and member checksums, and the date each CC0 statement was checked are in
[the source manifest](../../art/originals/sound/sources.json) and
[the license record](../legal.md#sounds-cc0-10). The manifest distinguishes
Kenney/OpenGameArt pack originals from Freesound public HQ MP3 previews.
`art/originals/sound/<cue>.flac` preserves the decoded source audio with no
extra lossy compression, before trimming or level changes.

## Regeneration

Run from the repository root, with ffmpeg installed:

```sh
python3 tools/heavy.py run --kind build --wait 3600 -- python3 tools/sound/process.py
```

The script has no Python package dependencies. It decodes the approved originals
at 48 kHz in mono, trims silence 45 dB below each clip's peak with a 1 ms lead,
and applies the listening shortlist's preparation: at most 1.5 seconds normally,
3 seconds for fanfare and defeat, and 2 seconds for ageStart. If a long source's
loudest 50 ms is beyond the cut, the window starts shortly before it with a
10 ms fade in. Tails have an 8 ms fade, or 60 ms when cut. The loudest 300 ms
window is matched to -20 dBFS RMS, with a -1 dBFS peak ceiling before encoding.
The selected drum retains the shortlist's actual 1.5 second prepared window;
the shortlist's prose suggested a shorter first stroke, which was not its
prepared file. The turn cues use the explicitly swapped selections.

The outputs are mono AAC-LC in `.m4a` at 64 kbps, under `web/static/sfx/`.
The script prints each encoded file's duration, peak, loudest 300 ms RMS and
byte count. AAC can slightly change peaks and durations, so these measurements
come from decoding the output, not from the input. It also generates
`web/src/lib/fx/sounds.gen.ts`, with a URL containing the first 12 hex digits
of each output's SHA-256. Never edit a shipped file or its hash by hand.
AAC-LC avoids the extra format and fallback set that older Safari versions
would need for Opus.

## Runtime

`sfx.ts` creates or resumes its AudioContext only from `unlockAudio`, called
inside a pointer or key gesture while Sound is on. It then fetches and decodes
the set once, with two requests in flight at most. Before unlock, playback and
preloading do nothing. A loaded cue plays from its cached AudioBuffer; an
unloaded or failed cue immediately uses its synth fallback. Loading never
replays a sound that has already passed. Quiet, Medium and Loud keep their
existing master gains. Slide has a 0.65 gain and the frequent online yourTurn
cue a 0.35 gain relative to the normalized files. Other cues have unity gain.
The hashed URL metadata is loaded only after unlock too. At most three cues overlap; an older voice is cut at the actual overlap time, so scheduling a later cue
does not silence earlier playback. Long files stay connected until their own end.

Audio is excluded from service-worker installation, as are other non-shell
static assets. Hashed sound requests enter a separate 24-entry runtime LRU
cache on first fetch and survive shell updates. No audio bytes are part of
Home's initial JavaScript or its install precache. Event-to-cue wiring,
90 ms spacing without animation, animation timing and haptics remain the
caller's responsibility.

The gallery sound board has separate file and synth buttons for every cue,
plus the existing busy-move sequence and volume steps. A file-only audition reports
when its file is unavailable, rather than silently playing the synth. `sound-files.spec.ts`
checks lazy loading and decodes every shipped file in the test browser.


## macOS WebKit coverage

`make install-webkit` and `make e2e-webkit` add opt-in headless WebKit coverage
for gesture unlock, lazy metadata/AAC loading and native decoding/playback,
explicit missing-file synth fallback, simulated hide/show with real context
suspension and gesture recovery, and the Sound/Animation controls. The report
records whether AAC buffers or synthesized sources actually started. It does
not claim physical speaker audibility or an iOS interruption test. The same
native audio assertions can run in Chromium; that verifies test behavior but
does not replace a blocked WebKit run.

The manual Safari/device check is **deferred-do-no-ask-user-again**. iOS silent
switch behavior, call interruptions and home-screen app mode remain untested
here. See [the WebKit scope and commands](../testing.md#opt-in-webkit-smoke-on-macos).
