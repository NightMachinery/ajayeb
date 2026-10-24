<script lang="ts">
  // The role tag (phone-layout.md, "Terms"): a crown and "creator",
  // "admin" or "temporary admin" beside a name, on a panel, the strip, a
  // spectator row or a seat menu. `compact` keeps only the crown, for a
  // far chip. The temporary admin's crown is drawn hollow, and its tooltip
  // says the role lasts until an admin returns. The frame's roles decide
  // which tag a person carries (model.roleOf).
  import { tooltip } from '$lib/ui/tooltip.svelte';
  import { ht } from '../i18n/messages.svelte';
  import HgGlyph from './HgGlyph.svelte';
  import type { Role } from './model';

  interface Props {
    role: Role;
    /** Whose role it is, for the tooltip. */
    name: string;
    compact?: boolean;
  }

  let { role, name, compact = false }: Props = $props();
  const label = $derived(ht(`hg.admin.role.${role}`));
  const tip = $derived(ht(`hg.admin.role.${role}.tip`, { name }));
</script>

<span
  class="tag {role}"
  class:compact
  data-role-tag={role}
  role="img"
  aria-label={label}
  {@attach tooltip({ text: tip })}
>
  <HgGlyph name="hint-crown" />
  {#if !compact}<span class="label">{label}</span>{/if}
</span>

<style>
  .tag {
    display: inline-flex;
    flex-shrink: 0;
    align-items: center;
    gap: 0.2rem;
    padding-block: 0.05rem;
    padding-inline: 0.3rem 0.4rem;
    border: 1px solid currentColor;
    border-radius: 999px;
    color: var(--fg-muted);
    font-size: 0.7rem;
    font-weight: 600;
    line-height: 1.2;
    white-space: nowrap;
  }
  .compact {
    padding: 0.1rem;
    border: 0;
    font-size: 0.85rem;
  }
  .creator {
    color: var(--gold-deep);
  }
  .admin {
    color: var(--lapis);
  }
  .temporary {
    border-style: dashed;
  }
  .temporary :global(svg) {
    opacity: 0.55;
  }
</style>
