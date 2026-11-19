import { createResource, For, Show } from 'solid-js';
import { ExternalLink, Play } from 'lucide-solid';
import { content, type Stream, type StreamGroup, type PlayRequest } from './api';
import { Panel, Spinner } from './components';
import { t } from './i18n';

/** streamKey identifies a source across sessions for resume. */
export function streamKey(s: Stream): string {
  if (s.infoHash) return `bt:${s.infoHash.toLowerCase()}:${s.fileIdx ?? ''}`;
  return s.url ?? s.externalUrl ?? s.ytId ?? '';
}

export function playable(s: Stream): boolean {
  return Boolean(s.url || s.infoHash);
}

export function quality(s: Stream): string | undefined {
  const text = `${s.name ?? ''} ${s.title ?? ''} ${s.description ?? ''} ${s.behaviorHints?.filename ?? ''}`;
  if (/2160p|\b4k\b|uhd/i.test(text)) return '4K';
  if (/1080p/i.test(text)) return '1080p';
  if (/720p/i.test(text)) return '720p';
  if (/480p|\bsd\b/i.test(text)) return '480p';
  return undefined;
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

export function StreamPicker(props: {
  type: string;
  videoId: string;
  title: string;
  subtitle?: string;
  selectedKey?: string;
  onPick: (s: Stream, g: StreamGroup) => void;
  onClose: () => void;
}) {
  const [groups] = createResource(() => content.streams(props.type, props.videoId));
  const total = () => (groups() ?? []).reduce((n, g) => n + (g.streams?.length ?? 0), 0);
  let first = true;
  const isDefault = (s: Stream) => {
    if (props.selectedKey) return streamKey(s) === props.selectedKey;
    if (first && playable(s)) {
      first = false;
      return true;
    }
    return false;
  };
  return (
    <Panel title={t('streams.title')} subtitle={props.subtitle ?? props.title} onClose={props.onClose}>
      <Show when={!groups.loading} fallback={<LoadingNote />}>
        <Show when={total() > 0} fallback={<p class="notice">{t('streams.none')}</p>}>
          <For each={groups()}>
            {(g) => (
              <Show when={g.streams?.length}>
                <div class="group-title">{g.addonName}</div>
                <For each={g.streams!}>
                  {(s) => (
                    <button
                      class="option"
                      classList={{ selected: props.selectedKey === streamKey(s) }}
                      data-nav
                      data-nav-default={isDefault(s) ? '' : undefined}
                      disabled={!playable(s)}
                      onClick={() => playable(s) && props.onPick(s, g)}
                    >
                      {playable(s) ? <Play /> : <ExternalLink />}
                      <div class="main">
                        <div class="title">{(s.name ?? g.addonName).replace(/\n/g, ' · ')}</div>
                        <div class="sub">{playable(s) ? (s.description ?? s.title ?? '') : t('streams.unsupported')}</div>
                      </div>
                      <Show when={quality(s)}>
                        <span class="badge">{quality(s)}</span>
                      </Show>
                    </button>
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

function LoadingNote() {
  return (
    <div class="notice" style={{ display: 'flex', 'align-items': 'center', gap: '1rem' }}>
      <Spinner />
      {t('streams.loading')}
    </div>
  );
}
