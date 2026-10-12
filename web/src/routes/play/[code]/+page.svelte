<script lang="ts">
  // /play/<code>: every mode's room. A first-time visitor (no stored name)
  // who opens an invite link is asked for a name first, since the other
  // player sees it. A room this browser created or played before needs none:
  // the Session (and the dev room helper) remember a room the moment it is
  // made, before the page moves here. Neither does a device link
  // (/play/<code>#claim=...): the room already knows its seat's name. Then
  // Room joins the room over the app's live connection and shows it.
  import { untrack } from 'svelte';
  import { page } from '$app/state';
  import NamePrompt from '$lib/components/screens/NamePrompt.svelte';
  import { t } from '$lib/i18n';
  import { liveGame, savedRooms } from '$lib/session/app.svelte';
  import { parseClaimFragment } from '$lib/session/claim';
  import { loadName, saveName } from '$lib/session/identity';
  import { normalizeCode } from '$lib/session/session.svelte';
  import { needsName, suggestName } from '$lib/ui/online';
  import Room from './Room.svelte';

  const code = $derived(normalizeCode(page.params.code ?? ''));
  let named = $state(loadName() !== '');
  // The saved list is read when the code (or the name) changes, never
  // tracked: a room is dropped from it when its game ends, and that must
  // not swap the finished game for the prompt.
  const claim = $derived(parseClaimFragment(page.url.hash));
  const ask = $derived(
    claim === null &&
      needsName(
        named,
        code,
        untrack(() => savedRooms().list)
      )
  );
  const suggestion = suggestName(t('online.name.suggestions'));

  function submit(name: string): void {
    const saved = saveName(name);
    named = true;
    // The live connection, when it is already up, says hello again under it.
    liveGame().session.setName(saved);
  }
</script>

{#if ask}
  <NamePrompt {suggestion} {code} onsubmit={submit} />
{:else}
  <Room />
{/if}
