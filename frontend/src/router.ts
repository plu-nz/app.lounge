import { createSignal } from 'solid-js';
import type { MetaPreview, PlayRequest, Video } from './api';

export type Route =
  | { name: 'home'; type: '' | 'movie' | 'series' }
  | { name: 'search' }
  | { name: 'addons' }
  | { name: 'settings' }
  | { name: 'login' }
  | { name: 'grid'; addonId: string; type: string; catalogId: string; title: string; genres?: string[] }
  | { name: 'details'; type: string; id: string; preview?: MetaPreview; autoplay?: boolean }
  | { name: 'player'; play: PlayContext };

/** PlayContext is everything the player needs to start and track progress. */
export type PlayContext = {
  request: PlayRequest;
  meta: { id: string; type: string; name: string; poster?: string; background?: string; logo?: string };
  video: Video | null;
  videos: Video[];
  streamKey: string;
  bingeGroup?: string;
  addonId: string;
};

const [stack, setStack] = createSignal<Route[]>([{ name: 'home', type: '' }]);

/** focusMemory keeps the focused element's selector per stack depth. */
const focusMemory = new Map<number, HTMLElement>();

export const stackRoutes = stack;
export const route = () => stack()[stack().length - 1];
export const depth = () => stack().length;

export function push(r: Route) {
  focusMemory.set(stack().length, document.activeElement as HTMLElement);
  setStack([...stack(), r]);
}

/** replaceRoot switches top-level sections from the sidebar. */
export function replaceRoot(r: Route) {
  focusMemory.clear();
  setStack([r]);
}

export function replaceTop(r: Route) {
  setStack([...stack().slice(0, -1), r]);
}

export function pop(): boolean {
  if (stack().length <= 1) return false;
  setStack(stack().slice(0, -1));
  return true;
}

/** restoredFocus returns the element focused before the current route was pushed. */
export function restoredFocus(): HTMLElement | undefined {
  const el = focusMemory.get(stack().length);
  focusMemory.delete(stack().length);
  return el?.isConnected ? el : undefined;
}
