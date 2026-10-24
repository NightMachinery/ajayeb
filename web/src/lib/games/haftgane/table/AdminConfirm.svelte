<script lang="ts">
  // The confirmation of an admin action, a role change or a resignation:
  // what will change, in one or two sentences, then Confirm and Back. It
  // sends nothing itself; `onconfirm` does, and the server decides.
  import Button from '$lib/components/shell/Button.svelte';
  import { ht } from '../i18n/messages.svelte';

  interface Props {
    /** What will change, already translated. */
    text: string;
    /** The request cannot go out now (another is in flight, or offline). */
    disabled?: boolean;
    onconfirm: () => void;
    onback: () => void;
  }

  let { text, disabled = false, onconfirm, onback }: Props = $props();
</script>

<div class="confirm" data-admin-confirm role="group" aria-label={ht('hg.admin.confirm.ok')}>
  <p dir="auto">{text}</p>
  <div class="actions">
    <Button variant="secondary" onclick={onback} data-act="admin-back">
      {ht('hg.admin.confirm.back')}
    </Button>
    <Button {disabled} onclick={onconfirm} data-act="admin-confirm">
      {ht('hg.admin.confirm.ok')}
    </Button>
  </div>
</div>

<style>
  .confirm {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    padding: 0.7rem;
    border: 1px solid var(--rule-strong);
    border-radius: var(--shape-panel);
    background: var(--panel-sunken);
  }
  p {
    margin: 0;
    line-height: 1.4;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 0.5rem;
  }
  .actions :global(button) {
    min-height: 44px;
  }
</style>
