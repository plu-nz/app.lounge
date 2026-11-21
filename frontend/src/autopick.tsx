import { createEffect, createSignal, JSX, on, onCleanup, Show } from 'solid-js';
import { content, sourcesApi, type Stream, type StreamGroup } from './api';
import { Spinner } from './components';
import { t } from './i18n';
import { onBack } from './nav';
import { settings } from './settings';
import { findStream, SourceRows } from './streams';

export type Picked = { stream: Stream; group: StreamGroup };

/** autoMode reports whether sources are picked without asking. */
export const autoMode = () => settings().sourceMode === 'auto';

/**
 * autoPick loads a video's sources and returns the one to play: the last
 * source used when it is offered again, else the best one under the user's
 * rules. It returns undefined when nothing qualifies, so the caller shows
 * the list.
 */
export async function autoPick(type: string, videoId: string, lastKey?: string): Promise<Picked | undefined> {
  if (!autoMode()) return undefined;
  const groups = await content.streams(type, videoId);
  if (lastKey) {
    const found = findStream(groups, { key: lastKey });
    if (found) return found;
  }
  const ref = await sourcesApi.pick(groups);
  const group = ref ? groups[ref.group] : undefined;
  const stream = group?.streams?.[ref!.stream];
  return group && stream ? { stream, group } : undefined;
}

/**
 * AutoSourceRows shows SourceRows, except in automatic mode, where it first
 * plays the best source. Back cancels the pick and shows the list; the list
 * also shows when no source passes the rules or the pick fails.
 */
export function AutoSourceRows(props: {
  type: string;
  videoId: string;
  label: JSX.Element;
  selectedKey?: string;
  active: boolean;
  /** disabled leaves the pick to the caller (for example a resume). */
  disabled?: boolean;
  onPick: (s: Stream, g: StreamGroup) => void;
  onClose?: () => void;
}) {
  // 'played' keeps the list unmounted while the player is open, so the
  // sources are not fetched a second time behind it.
  const [state, setState] = createSignal<'picking' | 'list' | 'played'>(autoMode() && !props.disabled ? 'picking' : 'list');
  let alive = true;
  onCleanup(() => (alive = false));

  if (state() === 'picking') {
    autoPick(props.type, props.videoId, props.selectedKey)
      .then((found) => {
        if (!alive || state() !== 'picking') return;
        if (found && props.active) {
          setState('played');
          props.onPick(found.stream, found.group);
        } else setState('list');
      })
      .catch(() => alive && state() === 'picking' && setState('list'));
  }

  onCleanup(
    onBack(() => {
      if (!props.active || state() !== 'picking') return false;
      setState('list');
      return true;
    }),
  );

  // Coming back from the player shows the list again.
  createEffect(
    on(
      () => props.active,
      (active) => active && state() === 'played' && setState('list'),
      { defer: true },
    ),
  );

  return (
    <Show when={state() !== 'played'}>
      <Show
        when={state() === 'list'}
        fallback={
          <section class="sources">
            <h2>{props.label}</h2>
            <div class="notice" style={{ display: 'flex', 'align-items': 'center', gap: '1rem' }}>
              <Spinner />
              {t('streams.picking')}
            </div>
          </section>
        }
      >
        <SourceRows
          type={props.type}
          videoId={props.videoId}
          label={props.label}
          selectedKey={props.selectedKey}
          onPick={props.onPick}
          onClose={props.onClose}
        />
      </Show>
    </Show>
  );
}
