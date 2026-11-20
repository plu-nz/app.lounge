import { createEffect, createMemo, createResource, createSignal, For, on, onMount, Show } from 'solid-js';
import { CircleCheck, List, X } from 'lucide-solid';
import { content, library, type Meta, type MetaPreview, type Stream, type StreamGroup, type Video } from '../api';
import { Backdrop, MetaLine, Spinner, errorText, toast } from '../components';
import { t } from '../i18n';
import { focusEl, focusFirst } from '../nav';
import { push } from '../router';
import { findStream, SourceRows, streamKey, toRequest } from '../streams';

type Pick = { video: Video | null; videoId: string; start: number; label: string };

export function sortVideos(videos: Video[] = []): Video[] {
  return [...videos].sort((a, b) => {
    const sa = a.season ?? 0;
    const sb = b.season ?? 0;
    if (sa !== sb) return (sa === 0 ? 1e6 : sa) - (sb === 0 ? 1e6 : sb);
    return (a.episode ?? a.number ?? 0) - (b.episode ?? b.number ?? 0);
  });
}

export function episodeLabel(v: Video): string {
  return v.season !== undefined || v.episode !== undefined ? `S${v.season ?? 0} E${v.episode ?? v.number ?? 0}` : '';
}

function clock(seconds: number): string {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const sec = Math.floor(seconds % 60);
  return h ? `${h}:${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}` : `${m}:${String(sec).padStart(2, '0')}`;
}

/** sourcesLabel names what the source rows will play. */
function sourcesLabel(pick: Pick): string {
  const parts = [pick.video ? `${episodeLabel(pick.video)} ${pick.video.name ?? pick.video.title ?? ''}`.trim() : t('streams.title')];
  if (pick.start > 0) parts.push(`${t('details.resume')} ${clock(pick.start)}`);
  return parts.join(' · ');
}

export function Details(props: { type: string; id: string; preview?: MetaPreview; autoplay?: boolean; active: boolean }) {
  const [meta] = createResource(() => content.meta(props.type, props.id).catch((err) => ({ error: errorText(err) }) as Meta & { error: string }));
  const [entry, { refetch: refetchEntry }] = createResource(() => library.get(props.id));
  const [watched, setWatched] = createSignal<Set<string>>(new Set());
  const [season, setSeason] = createSignal<number>();
  // An episode the user picked; otherwise sources are for next().
  const [chosen, setChosen] = createSignal<Pick>();

  const item = (): Partial<Meta> => ({ ...props.preview, ...(meta() && !('error' in meta()!) ? meta() : {}) });
  const videos = createMemo(() => sortVideos(item().videos));
  const isSeries = () => videos().length > 0 && props.type !== 'movie';
  const seasons = createMemo(() => [...new Set(videos().map((v) => v.season ?? 0))]);
  const seasonVideos = createMemo(() => videos().filter((v) => (v.season ?? 0) === season()));

  createEffect(on(() => props.active, (active, was) => active && was === false && (refetchEntry(), loadWatched())));

  const loadWatched = async () => {
    const ids = videos().map((v) => v.id);
    if (ids.length) setWatched(new Set(await library.watched(ids)));
  };

  // Choose the starting season: the resumed episode's, else the first regular one.
  createEffect(() => {
    if (!videos().length || season() !== undefined) return;
    const e = entry();
    const resumed = e && videos().find((v) => v.id === e.videoId);
    setSeason(resumed?.season ?? seasons().find((s) => s > 0) ?? seasons()[0]);
    loadWatched();
  });

  /** next is what plays by default: the resume point or the first episode. */
  const next = createMemo((): Pick | undefined => {
    const e = entry();
    if (!isSeries()) {
      const resume = e && e.position > 30 && e.position < e.duration * 0.92;
      return { video: null, videoId: props.id, start: resume ? e!.position : 0, label: resume ? t('details.resume') : t('details.play') };
    }
    if (!videos().length) return undefined;
    if (e) {
      const i = videos().findIndex((v) => v.id === e.videoId);
      if (i >= 0) {
        const done = e.position >= e.duration * 0.92;
        const v = done ? videos()[i + 1] : videos()[i];
        if (v) {
          const vars = { s: v.season ?? 0, e: v.episode ?? v.number ?? 0 };
          return done
            ? { video: v, videoId: v.id, start: 0, label: t('details.playEpisode', vars) }
            : { video: v, videoId: v.id, start: e.position, label: t('details.resumeEpisode', vars) };
        }
      }
    }
    const v = videos().find((v) => (v.season ?? 0) > 0) ?? videos()[0];
    return { video: v, videoId: v.id, start: 0, label: t('details.playEpisode', { s: v.season ?? 0, e: v.episode ?? v.number ?? 0 }) };
  });

  // Sources load as soon as the title opens; the rows only reload when the
  // video or the resume point changes.
  const shown = createMemo(() => chosen() ?? next(), undefined, {
    equals: (a, b) => a?.videoId === b?.videoId && a?.start === b?.start,
  });

  const play = (pick: Pick, s: Stream, g: StreamGroup) => {
    const m = item();
    setChosen(undefined);
    push({
      name: 'player',
      play: {
        request: { ...toRequest(s), start: pick.start, title: pick.video ? `${m.name} · ${episodeLabel(pick.video)}` : m.name },
        meta: { id: props.id, type: props.type, name: m.name ?? '', poster: m.poster, background: m.background, logo: m.logo },
        video: pick.video,
        videos: videos(),
        streamKey: streamKey(s),
        bingeGroup: s.behaviorHints?.bingeGroup,
        addonId: g.addonId,
      },
    });
  };

  // Resume from continue watching: reuse the last source when it is offered again.
  const [autoplaying, setAutoplaying] = createSignal(false);
  createEffect(
    on([next, entry, () => meta.loading || entry.loading], ([pick, e, loading]) => {
      if (!props.autoplay || loading || !pick || autoplaying()) return;
      setAutoplaying(true);
      if (!e?.streamKey) return;
      content
        .streams(props.type, pick.videoId)
        .then((groups) => {
          const found = findStream(groups, { key: e.streamKey });
          if (found && props.active) play(pick, found.stream, found.group);
        })
        .catch(() => undefined);
    }),
  );

  // The source rows take focus once they load; until then focus the buttons.
  onMount(() => queueMicrotask(() => focusFirst(document.querySelector('.details-head') ?? document)));

  const clearProgress = async () => {
    await library.remove(props.id);
    refetchEntry();
    toast(t('details.clearProgress'));
  };

  const openEpisodes = () => {
    const el =
      document.querySelector<HTMLElement>('.episode.current') ??
      document.querySelector<HTMLElement>('.seasons .chip.selected');
    if (el) focusEl(el);
  };

  return (
    <div class="screen details">
      <Backdrop src={item().background} full />
      <div class="details-body" data-nav-pane data-nav-top="0.2">
        <div class="details-head" data-nav-block>
          <Show when={item().logo} fallback={<h1>{item().name}</h1>}>
            <img class="logo" src={item().logo} alt={item().name} />
          </Show>
          <MetaLine item={item()} />
          <Show when={item().description}>
            <p class="description" style={{ '-webkit-line-clamp': 5 }}>
              {item().description}
            </p>
          </Show>
          <Show when={item().cast?.length || item().director?.length}>
            <div class="people">
              <Show when={item().director?.length}>
                <div>
                  <b>{t('details.director')}:</b> {item().director!.join(', ')}
                </div>
              </Show>
              <Show when={item().cast?.length}>
                <div>
                  <b>{t('details.cast')}:</b> {item().cast!.slice(0, 6).join(', ')}
                </div>
              </Show>
            </div>
          </Show>
          <Show when={meta() && 'error' in meta()!}>
            <p class="error">{(meta() as unknown as { error: string }).error}</p>
          </Show>
          <div class="btn-row">
            <Show when={meta.loading}>
              <Spinner />
            </Show>
            <Show when={isSeries()}>
              <button class="btn" data-nav onClick={openEpisodes}>
                <List />
                {t('details.episodes')}
              </button>
            </Show>
            <Show when={entry()}>
              <button class="btn" data-nav onClick={clearProgress}>
                <X />
                {t('details.clearProgress')}
              </button>
            </Show>
          </div>
          <Show when={shown()} keyed>
            {(pick) => (
              <SourceRows
                type={props.type}
                videoId={pick.videoId}
                label={sourcesLabel(pick)}
                selectedKey={entry()?.videoId === pick.videoId ? entry()?.streamKey : undefined}
                onPick={(s, g) => play(pick, s, g)}
              />
            )}
          </Show>
        </div>

        <Show when={isSeries()}>
          <div class="seasons" data-nav-row data-nav-group data-nav-block>
            <For each={seasons()}>
              {(s) => (
                <button class="chip" classList={{ selected: season() === s }} data-nav onClick={() => setSeason(s)} onFocus={() => setSeason(s)}>
                  {s === 0 ? t('details.specials') : t('details.season', { n: s })}
                </button>
              )}
            </For>
          </div>
          <div class="episodes" data-nav-group>
            <For each={seasonVideos()}>
              {(v) => (
                <button
                  class="episode"
                  classList={{ current: entry()?.videoId === v.id }}
                  data-nav
                  data-nav-block
                  onClick={() =>
                    setChosen({
                      video: v,
                      videoId: v.id,
                      start: entry()?.videoId === v.id && entry()!.position < entry()!.duration * 0.92 ? entry()!.position : 0,
                      label: episodeLabel(v),
                    })
                  }
                >
                  <div class="thumb">
                    <Show when={v.thumbnail}>
                      <img src={v.thumbnail} loading="lazy" alt="" />
                    </Show>
                  </div>
                  <div>
                    <h3>
                      {(v.episode ?? v.number) !== undefined ? `${v.episode ?? v.number}. ` : ''}
                      {v.name ?? v.title ?? t('details.episode', { n: v.episode ?? 0 })}
                      <Show when={watched().has(v.id)}>
                        <CircleCheck aria-label={t('details.watched')} />
                      </Show>
                    </h3>
                    <Show when={v.overview}>
                      <p>{v.overview}</p>
                    </Show>
                  </div>
                </button>
              )}
            </For>
          </div>
        </Show>
      </div>

    </div>
  );
}
