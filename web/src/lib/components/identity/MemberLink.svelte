<script lang="ts">
  import { t } from '$lib/lobby/i18n/messages';
  import { formatNumber } from '$lib/i18n';
  import type { TableSession } from '$lib/games/haftgane/session/session.svelte';
  import type { PlatformHuman } from '$lib/wire/platform.gen';
  import DeviceLinkControl from './DeviceLinkControl.svelte';
  interface Props {
    session: TableSession;
    human: PlatformHuman;
  }
  let { session, human }: Props = $props();
  const frame = $derived(session.table ?? session.lobby);
  const own = $derived(human.userId === frame?.you.userId);
  const seat = $derived(
    frame?.seats.find((s) => s.kind === 'human' && s.userId === human.userId)?.seat ?? human.userId
  );
</script>

{#if human.linkable && (own || session.admin)}
  {#key human.userId}
    <DeviceLinkControl
      code={session.code}
      {seat}
      userId={human.userId}
      name={human.nameNo > 1 ? `${human.name} ${formatNumber(human.nameNo)}` : human.name}
      {own}
      description={own ? t('lobby.deviceIntro') : t('lobby.deviceIntroOther', { name: human.name })}
      resetHint={t('lobby.deviceResetHint')}
      request={(reset) => session.requestUserLink(human.userId, reset)}
    />
  {/key}
{/if}
