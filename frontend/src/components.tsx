import { createEffect, createSignal, For, JSX, onCleanup, onMount, Show } from 'solid-js';
import { ChevronRight, Film, House, LoaderCircle, Puzzle, Search, Settings, Tv } from 'lucide-solid';
import type { MetaPreview } from './api';
import { t, typeName } from './i18n';
import { focusEl, onBack } from './nav';
import { replaceRoot, route, type Route } from './router';

/** Rail is the left navigation; it expands while focused. */
export function Rail() {
  const items: { icon: typeof House; label: () => string; to: Route; key: string }[] = [
    { icon: Search, label: () => t('nav.search'), to: { name: 'search' }, key: 'search' },
    { icon: House, label: () => t('nav.home'), to: { name: 'home', type: '' }, key: 'home:' },
    { icon: Film, label: () => t('nav.movies'), to: { name: 'home', type: 'movie' }, key: 'home:movie' },
    { icon: Tv, label: () => t('nav.series'), to: { name: 'home', type: 'series' }, key: 'home:series' },
    { icon: Puzzle, label: () => t('nav.addons'), to: { name: 'addons' }, key: 'addons' },
    { icon: Settings, label: () => t('nav.settings'), to: { name: 'settings' }, key: 'settings' },
  ];
  const current = () => {
    const r = route();
    return r.name === 'home' ? `home:${r.type}` : r.name;
  };
  return (
    <nav class="rail" data-nav-group>
      <div class="brand">LOUNGE</div>
      <For each={items}>
        {(item) => (
          <button
            class="rail-item"
            classList={{ active: current() === item.key }}
            data-nav
            data-nav-entry={current() === item.key ? '' : undefined}
            onClick={() => current() !== item.key && replaceRoot(item.to)}
          >
            <item.icon />
            <span>{item.label()}</span>
          </button>
        )}
      </For>
    </nav>
  );
}

/** Backdrop crossfades between background images. */
export function Backdrop(props: { src?: string; full?: boolean }) {
  const [a, setA] = createSignal<string>();
  const [b, setB] = createSignal<string>();
  const [front, setFront] = createSignal<'a' | 'b'>('a');
  createEffect(() => {
    const src = props.src;
    if (!src || src === (front() === 'a' ? a() : b())) return;
    const img = new Image();
    img.onload = () => {
      if (props.src !== src) return;
      if (front() === 'a') {
        setB(src);
        setFront('b');
      } else {
        setA(src);
        setFront('a');
      }
    };
    img.src = src;
  });
  return (
    <div class="backdrop" classList={{ full: props.full }}>
      <Show when={a()}>
        <img src={a()} classList={{ shown: front() === 'a' }} alt="" />
      </Show>
      <Show when={b()}>
        <img src={b()} classList={{ shown: front() === 'b' }} alt="" />
      </Show>
    </div>
  );
}

export function MetaLine(props: { item: Partial<MetaPreview> }) {
  return (
    <div class="meta-line">
      <Show when={props.item.type}>
        <span>{typeName(props.item.type!)}</span>
      </Show>
      <Show when={props.item.genres?.length}>
        <span>{props.item.genres!.slice(0, 3).join(' · ')}</span>
      </Show>
      <Show when={props.item.releaseInfo}>
        <span>{props.item.releaseInfo}</span>
      </Show>
      <Show when={props.item.runtime}>
        <span>{props.item.runtime}</span>
      </Show>
      <Show when={props.item.imdbRating}>
        <span class="imdb">
          <b>IMDb</b>
          {props.item.imdbRating}
        </span>
      </Show>
    </div>
  );
}

export function Card(props: {
  item: MetaPreview;
  shape?: 'poster' | 'landscape' | 'square';
  progress?: number;
  caption?: string;
  subcaption?: string;
  onFocus?: () => void;
  onSelect: () => void;
  default?: boolean;
}) {
  const [failed, setFailed] = createSignal(false);
  const shape = () => props.shape ?? props.item.posterShape ?? 'poster';
  const image = () => (shape() === 'landscape' ? props.item.background || props.item.poster : props.item.poster);
  return (
    <button
      class="card"
      classList={{ landscape: shape() === 'landscape', square: shape() === 'square' }}
      data-nav
      data-nav-default={props.default ? '' : undefined}
      onFocus={() => props.onFocus?.()}
      onClick={() => props.onSelect()}
      aria-label={props.item.name}
    >
      <Show when={image() && !failed()} fallback={<div class="fallback">{props.item.name}</div>}>
        <img src={image()} loading="lazy" alt="" onError={() => setFailed(true)} />
      </Show>
      <Show when={props.caption}>
        <div class="caption">
          {props.caption}
          <Show when={props.subcaption}>
            <small>{props.subcaption}</small>
          </Show>
        </div>
      </Show>
      <Show when={props.progress !== undefined}>
        <div class="progress">
          <i style={{ width: `${Math.min(100, (props.progress ?? 0) * 100)}%` }} />
        </div>
      </Show>
    </button>
  );
}

export function MoreCard(props: { onSelect: () => void }) {
  return (
    <button class="card more" data-nav onClick={() => props.onSelect()}>
      <ChevronRight />
      {t('home.viewAll')}
    </button>
  );
}

export function Spinner() {
  return <LoaderCircle class="spin" />;
}

/** Panel is a right-side dialog that traps focus until closed. */
export function Panel(props: { title: string; subtitle?: string; onClose: () => void; children: JSX.Element }) {
  let ref!: HTMLDivElement;
  const previous = document.activeElement as HTMLElement | null;
  // Focus the preferred option now, or as soon as async content renders it.
  const focusInside = () => {
    if (ref.contains(document.activeElement)) return true;
    const target =
      ref.querySelector<HTMLElement>('[data-nav-default]:not(:disabled)') ?? ref.querySelector<HTMLElement>('[data-nav]:not(:disabled)');
    if (target) focusEl(target);
    return Boolean(target);
  };
  let observer: MutationObserver | undefined;
  onCleanup(() => observer?.disconnect());
  onMount(() => {
    queueMicrotask(() => {
      if (focusInside()) return;
      observer = new MutationObserver(() => focusInside() && observer?.disconnect());
      observer.observe(ref, { childList: true, subtree: true });
    });
  });
  onCleanup(
    onBack(() => {
      props.onClose();
      return true;
    }),
  );
  onCleanup(() => {
    if (previous?.isConnected) focusEl(previous);
  });
  return (
    <>
      <div class="panel-shade" onClick={() => props.onClose()} />
      <div class="panel" data-nav-scope ref={ref}>
        <header>
          <h2>{props.title}</h2>
          <Show when={props.subtitle}>
            <p>{props.subtitle}</p>
          </Show>
        </header>
        <div class="list" data-nav-pane data-nav-top="0.3">
          {props.children}
        </div>
      </div>
    </>
  );
}

const [toastText, setToastText] = createSignal('');
let toastTimer: number | undefined;
export function toast(text: string) {
  setToastText(text);
  clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => setToastText(''), 3500);
}
export function Toast() {
  return (
    <Show when={toastText()}>
      <div class="toast" role="status">
        {toastText()}
      </div>
    </Show>
  );
}

export function errorText(err: unknown): string {
  if (err instanceof Error) return err.message;
  if (typeof err === 'string') return err;
  try {
    const o = err as { message?: string };
    return o?.message ?? JSON.stringify(err);
  } catch {
    return String(err);
  }
}
