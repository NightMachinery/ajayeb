<script lang="ts">
  // /dev/table?seats=3..7
  //
  // Creates a shared lobby over a v2 connection (on a server started with
  // -haftgane-tables) and opens it at /play/<code>, where the Haftgane
  // entry's development screen shows it. Until the New table flow of
  // M18.home-picker-and-lobby, this is how a browser starts a table.
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { page } from '$app/state';
  import { ht } from '$lib/games/haftgane/i18n/messages.svelte';

  const seatsParam = Number(page.url.searchParams.get('seats') ?? '3');
  const seats = Number.isInteger(seatsParam) && seatsParam >= 3 && seatsParam <= 7 ? seatsParam : 0;

  let failure = $state<string | null>(null);

  async function start(count: number): Promise<void> {
    try {
      const { createLobby } = await import('$lib/games/haftgane/session/create');
      const code = await createLobby(count);
      await goto(resolve('/play/[code]', { code }), { replaceState: true });
    } catch (err) {
      failure = err instanceof Error ? err.message : String(err);
    }
  }

  // Started from the script, not an effect: it runs once, when the page opens.
  if (seats) void start(seats);
</script>

<main class="flex h-full flex-col items-center justify-center gap-3 px-4 text-center">
  <h1 class="font-display text-2xl text-fg">{ht('hg.table.dev.create.title')}</h1>
  {#if !seats}
    <p class="text-fg" role="alert">{ht('hg.table.dev.create.badSeats')}</p>
  {:else if failure}
    <p class="text-fg" role="alert" data-failure={failure}>
      {ht('hg.table.dev.create.failed', { code: failure })}
    </p>
  {:else}
    <p class="text-fg-muted" role="status">
      {ht('hg.table.dev.create.creating', { count: seats })}
    </p>
  {/if}
</main>
