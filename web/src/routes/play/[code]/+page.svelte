<script lang="ts">
  // /play/<code>: every mode's room. A first-time visitor (no stored name)
  // who opens an invite link is asked for a name first, since the other
  // player sees it; a room this browser has played before needs none. Then
  // Room joins the room over the app's live connection and shows it.
  import { page } from '$app/state';
  import NamePrompt from '$lib/components/screens/NamePrompt.svelte';
  import { t } from '$lib/i18n';
  import { liveGame, savedRooms } from '$lib/session/app.svelte';
  import { loadName, saveName } from '$lib/session/identity';
  import { normalizeCode } from '$lib/session/session.svelte';
  import { suggestName } from '$lib/ui/online';
  import Room from './Room.svelte';

  const code = $derived(normalizeCode(page.params.code ?? ''));
  const known = $derived(savedRooms().list.some((r) => r.code === code));
  let named = $state(loadName() !== '');
  const suggestion = suggestName(t('online.name.suggestions'));

  function submit(name: string): void {
    const saved = saveName(name);
    named = true;
    // The live connection, when it is already up, says hello again under it.
    liveGame().session.setName(saved);
  }
</script>

{#if named || known}
  <Room />
{:else}
  <NamePrompt {suggestion} {code} onsubmit={submit} />
{/if}
