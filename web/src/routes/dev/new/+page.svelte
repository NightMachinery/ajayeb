<script lang="ts">
  // /dev/new?mode=hotseat|ai|online&level=easy|normal|hard&seat=0|1|random&seed=&scenario=
  //
  // Creates a room and opens it at /play/<code>. Without seed and scenario
  // it creates the room over the live connection, like the real app will.
  // With either, it asks the server's development endpoint (POST
  // /api/dev/rooms), which only a server started with --dev --dev-endpoints
  // answers; the room then opens at that seed or scenario.
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { page } from '$app/state';
  import { t } from '$lib/i18n';
  import { liveGame } from '$lib/session/app.svelte';
  import { createDevRoom, DevRoomError } from '$lib/session/dev-rooms';
  import { SessionError } from '$lib/session/session.svelte';
  import type { BotLevel, Mode, SeatChoice } from '$lib/wire';

  const MODES: readonly Mode[] = ['hotseat', 'ai', 'online'];
  const LEVELS: readonly BotLevel[] = ['easy', 'normal', 'hard'];

  const params = page.url.searchParams;
  const modeParam = params.get('mode') ?? 'hotseat';
  const mode = MODES.find((m) => m === modeParam);
  const levelParam = params.get('level') ?? 'easy';
  const level = LEVELS.find((l) => l === levelParam) ?? 'easy';
  const seatParam = params.get('seat');
  const seat: SeatChoice = seatParam === '1' ? 1 : seatParam === 'random' ? 'random' : 0;
  const seed = params.get('seed') || undefined;
  const scenario = params.get('scenario') || undefined;

  let failure = $state<string | null>(null);

  async function start(m: Mode): Promise<void> {
    try {
      let code: string;
      if (seed !== undefined || scenario !== undefined) {
        const room = await createDevRoom({
          mode: m,
          ...(seed !== undefined ? { seed } : {}),
          ...(scenario !== undefined ? { scenario } : {}),
          ...(m === 'ai' ? { bot: level, seat: seat === 1 ? (1 as const) : (0 as const) } : {})
        });
        code = room.code;
      } else {
        code = await liveGame().session.create({
          mode: m,
          seat,
          ...(m === 'ai' ? { bot: level } : {})
        });
      }
      await goto(resolve('/play/[code]', { code }), { replaceState: true });
    } catch (err) {
      failure = err instanceof SessionError || err instanceof DevRoomError ? err.code : String(err);
    }
  }

  // Started from the script, not an effect: it runs once, when the page opens.
  if (mode) void start(mode);
</script>

<main class="flex h-full flex-col items-center justify-center gap-3 px-4 text-center">
  <h1 class="font-display text-2xl text-fg">{t('dev.new.title')}</h1>
  {#if !mode}
    <p class="text-fg" role="alert">{t('dev.new.badMode')}</p>
  {:else if failure}
    <p class="text-fg" role="alert" data-failure={failure}>
      {t('dev.new.failed', { code: failure })}
    </p>
  {:else}
    <p class="text-fg-muted" role="status">{t('dev.new.creating', { mode })}</p>
  {/if}
</main>
