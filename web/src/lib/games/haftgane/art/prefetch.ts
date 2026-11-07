import { art } from '$lib/art/available-haftgane.gen';
import { artPrefetchPlanFor, type ConnectionInfo } from '$lib/art/prefetch';
import { prefetchCatalog } from './catalog';

// The only frame input is the public Age. The plan cannot inspect a hand,
// committed pick, guild draw, notebook, discard pile or tucked identity.
export function haftganePrefetchPlan(age: number, connection?: ConnectionInfo) {
  return artPrefetchPlanFor(prefetchCatalog(age), age, { connection, index: art });
}
