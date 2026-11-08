// Keep game rules and their lazy overlay inside Haftgane's entry graph.
import { tooltip, type TooltipContent } from '$lib/ui/tooltip.svelte';
import { openHaftganeRules } from '../rules/rules-overlay.svelte';
import type { HgRuleSectionId } from '../rules/sections';

export function hgTooltip(
  content: Omit<TooltipContent, 'rulesId' | 'onrules'> & {
    rulesId: HgRuleSectionId;
  }
) {
  return tooltip({ ...content, onrules: openHaftganeRules });
}
