// The Session and the Presenter of a page, shared with its components
// through Svelte context. A page that plays a room calls setGameContext
// once; components below it call getSession() and getPresenter().

import { createContext } from 'svelte';
import type { Presenter } from './presenter.svelte';
import type { Session } from './session.svelte';

export interface GameContext {
  session: Session;
  presenter: Presenter;
}

const [getGameContext, setContext] = createContext<GameContext>();

export { getGameContext };

export function setGameContext(ctx: GameContext): GameContext {
  return setContext(ctx);
}

export function getSession(): Session {
  return getGameContext().session;
}

export function getPresenter(): Presenter {
  return getGameContext().presenter;
}
