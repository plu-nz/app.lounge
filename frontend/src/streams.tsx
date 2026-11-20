import { createResource, For, JSX, onCleanup, onMount, Show } from 'solid-js';
import { HardDrive, Server, Users } from 'lucide-solid';
import { content, type Stream, type StreamGroup, type PlayRequest } from './api';
import { Panel, Spinner } from './components';
import { t } from './i18n';
import { focusEl, onBack } from './nav';

/** streamKey identifies a source across sessions for resume. */
export function streamKey(s: Stream): string {
  if (s.infoHash) return `bt:${s.infoHash.toLowerCase()}:${s.fileIdx ?? ''}`;
  return s.url ?? s.externalUrl ?? s.ytId ?? '';
}

export function playable(s: Stream): boolean {
  return Boolean(s.url || s.infoHash);
}

function allText(s: Stream): string {
  return `${s.name ?? ''} ${s.title ?? ''} ${s.description ?? ''} ${s.behaviorHints?.filename ?? ''}`;
}

export function quality(s: Stream): string | undefined {
  const text = allText(s);
  if (/2160p|\b4k\b|\buhd\b/i.test(text)) return '4K';
  if (/1440p/i.test(text)) return '1440p';
  if (/1080p/i.test(text)) return '1080p';
  if (/720p/i.test(text)) return '720p';
  if (/480p|\bsd\b/i.test(text)) return '480p';
  return undefined;
}

/** SourceInfo is a stream's text reduced to what fits on a TV card. */
export type SourceInfo = {
  title: string;
  file?: string;
  size?: string;
  seeders?: string;
  origin?: string;
  quality?: string;
  tags: string[];
};

// Emoji, flags, and other pictographs that addons use as decoration.
const decoration = /[\p{Extended_Pictographic}\p{Regional_Indicator}\u{FE0F}\u{200D}]/gu;

function clean(line: string): string {
  return line.replace(decoration, ' ').replace(/\s{2,}/g, ' ').trim();
}

export function sourceInfo(s: Stream, addonName: string): SourceInfo {
  const text = allText(s);
  const lines = `${s.description ?? s.title ?? ''}`.split('\n').map((l) => l.trim()).filter(Boolean);
  const size = text.match(/💾\s*([\d.,]+\s*[KMGT]i?B)/i)?.[1] ?? text.match(/\b(\d+(?:[.,]\d+)?\s*[GM]i?B)\b/)?.[1];
  const seeders = text.match(/👤\s*(\d+)/)?.[1];
  const origin = text.match(/⚙️?\s*([^\n💾👤]+)/u)?.[1]?.trim();
  // The file name is the first line that is not one of the stat lines.
  const file =
    s.behaviorHints?.filename ??
    lines.map(clean).find((l) => l && !/^[\d.,]+\s*[KMGT]i?B\b/i.test(l) && !/[💾👤⚙]/u.test(l) && l.length > 3);
  const tags: string[] = [];
  const add = (re: RegExp, tag: string) => re.test(text) && !tags.includes(tag) && tags.push(tag);
  add(/\bDV\b|dolby.?vision/i, 'DV');
  add(/HDR10\+/i, 'HDR10+');
  if (!tags.includes('HDR10+')) add(/\bHDR(10)?\b/i, 'HDR');
  add(/\bREMUX\b/i, 'REMUX');
  add(/atmos/i, 'Atmos');
  add(/\bIMAX\b/i, 'IMAX');
  const title = clean((s.name ?? addonName).replace(/\n/g, ' · '));
  return { title, file: file ? clean(file) : undefined, size, seeders, origin: origin ? clean(origin) : undefined, quality: quality(s), tags };
}

export function toRequest(s: Stream): PlayRequest {
  return {
    url: s.url,
    infoHash: s.infoHash,
    fileIdx: s.fileIdx,
    filename: s.behaviorHints?.filename,
    sources: s.sources,
    headers: s.behaviorHints?.proxyHeaders?.request,
    subtitles: s.subtitles,
  };
}

/** findStream returns the stream matching a key or binge group. */
export function findStream(groups: StreamGroup[], match: { key?: string; bingeGroup?: string; addonId?: string }) {
  for (const g of groups) {
    for (const s of g.streams ?? []) {
      if (!playable(s)) continue;
      if (match.key && streamKey(s) === match.key) return { stream: s, group: g };
    }
  }
  if (match.bingeGroup) {
    for (const g of groups) {
      const s = (g.streams ?? []).find((s) => playable(s) && s.behaviorHints?.bingeGroup === match.bingeGroup);
      if (s) return { stream: s, group: g };
    }
  }
  if (match.addonId) {
    const g = groups.find((g) => g.addonId === match.addonId);
    const s = g?.streams?.find(playable);
    if (g && s) return { stream: s, group: g };
  }
  return undefined;
}

function SourceMeta(props: { info: SourceInfo }) {
  return (
    <div class="source-meta">
      <Show when={props.info.size}>
        <span>
          <HardDrive />
          {props.info.size}
        </span>
      </Show>
      <Show when={props.info.seeders}>
        <span>
          <Users />
          {props.info.seeders}
        </span>
      </Show>
      <Show when={props.info.origin}>
        <span class="origin">
          <Server />
          {props.info.origin}
        </span>
      </Show>
    </div>
  );
}

function SourceCard(props: { stream: Stream; group: StreamGroup; selected: boolean; default: boolean; onPick: () => void }) {
  const info = sourceInfo(props.stream, props.group.addonName);
  return (
    <button
      class="source"
      classList={{ selected: props.selected }}
      data-nav
      data-nav-default={props.default ? '' : undefined}
      disabled={!playable(props.stream)}
      onClick={() => playable(props.stream) && props.onPick()}
    >
      <div class="source-badges">
        <Show when={info.quality}>
          <span class="badge strong">{info.quality}</span>
        </Show>
        <For each={info.tags}>{(tag) => <span class="badge">{tag}</span>}</For>
      </div>
      <div class="source-title">{info.title}</div>
      <Show when={playable(props.stream)} fallback={<div class="source-file">{t('streams.unsupported')}</div>}>
        <div class="source-file">{info.file ?? ''}</div>
      </Show>
      <SourceMeta info={info} />
    </button>
  );
}

/** useStreams loads the sources and picks the card that gets focus first. */
function useStreams(type: () => string, videoId: () => string, selectedKey: () => string | undefined) {
  const [groups] = createResource(() => [type(), videoId()] as const, ([ty, id]) => content.streams(ty, id));
  const total = () => (groups() ?? []).reduce((n, g) => n + (g.streams?.length ?? 0), 0);
  const defaultKey = () => {
    const all = (groups() ?? []).flatMap((g) => g.streams ?? []).filter(playable);
    const key = selectedKey();
    if (key && all.some((s) => streamKey(s) === key)) return key;
    return all[0] ? streamKey(all[0]) : undefined;
  };
  return { groups, total, defaultKey };
}

function Loading() {
  return (
    <div class="notice" style={{ display: 'flex', 'align-items': 'center', gap: '1rem' }}>
      <Spinner />
      {t('streams.loading')}
    </div>
  );
}

/** focusWhenReady focuses the default card once async content renders. */
function focusWhenReady(root: () => HTMLElement | undefined) {
  let observer: MutationObserver | undefined;
  const attempt = () => {
    const el = root();
    if (!el) return false;
    if (el.contains(document.activeElement)) return true;
    const target = el.querySelector<HTMLElement>('[data-nav-default]:not(:disabled)') ?? el.querySelector<HTMLElement>('[data-nav]:not(:disabled)');
    if (target) focusEl(target);
    return Boolean(target);
  };
  onMount(() =>
    queueMicrotask(() => {
      if (attempt() || !root()) return;
      observer = new MutationObserver(() => attempt() && observer?.disconnect());
      observer.observe(root()!, { childList: true, subtree: true });
    }),
  );
  onCleanup(() => observer?.disconnect());
}

/**
 * SourceRows shows sources inline as horizontal rows, one per addon.
 */
export function SourceRows(props: {
  type: string;
  videoId: string;
  label: JSX.Element;
  selectedKey?: string;
  onPick: (s: Stream, g: StreamGroup) => void;
  /** onClose makes Back close the rows instead of leaving the screen. */
  onClose?: () => void;
}) {
  let ref: HTMLElement | undefined;
  const { groups, total, defaultKey } = useStreams(() => props.type, () => props.videoId, () => props.selectedKey);
  focusWhenReady(() => ref);
  if (props.onClose) {
    const close = props.onClose;
    onCleanup(
      onBack(() => {
        close();
        return true;
      }),
    );
  }
  return (
    <section class="sources" ref={ref} data-nav-block>
      <h2>{props.label}</h2>
      <Show when={!groups.loading} fallback={<Loading />}>
        <Show when={total() > 0} fallback={<p class="notice">{t('streams.none')}</p>}>
          <For each={groups()}>
            {(g) => (
              <Show when={g.streams?.length}>
                <div class="source-group" data-nav-block>
                  <div class="group-title">{g.addonName}</div>
                  <div class="source-track" data-nav-row data-nav-group>
                    <For each={g.streams!}>
                      {(s) => (
                        <SourceCard
                          stream={s}
                          group={g}
                          selected={props.selectedKey === streamKey(s)}
                          default={defaultKey() === streamKey(s)}
                          onPick={() => props.onPick(s, g)}
                        />
                      )}
                    </For>
                  </div>
                </div>
              </Show>
            )}
          </For>
        </Show>
      </Show>
    </section>
  );
}

/** StreamPicker is the side-panel variant used over the player. */
export function StreamPicker(props: {
  type: string;
  videoId: string;
  title: string;
  subtitle?: string;
  selectedKey?: string;
  onPick: (s: Stream, g: StreamGroup) => void;
  onClose: () => void;
}) {
  const { groups, total, defaultKey } = useStreams(() => props.type, () => props.videoId, () => props.selectedKey);
  return (
    <Panel title={t('streams.title')} subtitle={props.subtitle ?? props.title} onClose={props.onClose}>
      <Show when={!groups.loading} fallback={<Loading />}>
        <Show when={total() > 0} fallback={<p class="notice">{t('streams.none')}</p>}>
          <For each={groups()}>
            {(g) => (
              <Show when={g.streams?.length}>
                <div class="group-title">{g.addonName}</div>
                <For each={g.streams!}>
                  {(s) => (
                    <SourceCard
                      stream={s}
                      group={g}
                      selected={props.selectedKey === streamKey(s)}
                      default={defaultKey() === streamKey(s)}
                      onPick={() => props.onPick(s, g)}
                    />
                  )}
                </For>
              </Show>
            )}
          </For>
        </Show>
      </Show>
    </Panel>
  );
}
