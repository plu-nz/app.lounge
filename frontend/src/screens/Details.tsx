import { createEffect, createMemo, createResource, createSignal, For, on, onCleanup, onMount, Show } from 'solid-js';
import { CircleCheck, X } from 'lucide-solid';
import { content, library, type Meta, type MetaPreview, type Stream, type StreamGroup, type Video } from '../api';
import { Backdrop, MetaLine, Spinner, errorText, toast } from '../components';
import { t } from '../i18n';
import { focusEl, focusFirst, glide, onBack } from '../nav';
import { push } from '../router';
import { AutoSourceRows, autoPick } from '../autopick';
import { findStream, streamKey, toRequest } from '../streams';

type Pick = { video: Video | null; videoId: string; start: number };

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
  // The episode whose sources are open. Series pick sources in two steps:
  // first the episode, then the source.
  const [chosen, setChosen] = createSignal<Pick>();
  let root: HTMLDivElement | undefined;
  let body: HTMLDivElement | undefined;

  const item = (): Partial<Meta> => ({ ...props.preview, ...(meta() && !('error' in meta()!) ? meta() : {}) });
  const videos = createMemo(() => sortVideos(item().videos));
  const isSeries = () => props.type === 'series' || (videos().length > 0 && props.type !== 'movie');
  const seasons = createMemo(() => [...new Set(videos().map((v) => v.season ?? 0))]);
  const seasonVideos = createMemo(() => videos().filter((v) => (v.season ?? 0) === season()));

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

  /** resumeAt is where a video starts: its saved position unless it was finished. */
  const resumeAt = (videoId: string) => {
    const e = entry();
    return e && e.videoId === videoId && e.position > 30 && e.position < e.duration * 0.92 ? e.position : 0;
  };

  /** next is what plays by default: the resume point or the first episode. */
  const next = createMemo((): Pick | undefined => {
    if (!isSeries()) return { video: null, videoId: props.id, start: resumeAt(props.id) };
    if (!videos().length) return undefined;
    const e = entry();
    const i = e ? videos().findIndex((v) => v.id === e.videoId) : -1;
    if (i >= 0) {
      const done = e!.position >= e!.duration * 0.92;
      const v = done ? videos()[i + 1] : videos()[i];
      if (v) return { video: v, videoId: v.id, start: done ? 0 : e!.position };
    }
    const v = videos().find((v) => (v.season ?? 0) > 0) ?? videos()[0];
    return { video: v, videoId: v.id, start: 0 };
  });

  // Movies load their sources as soon as the title opens. The rows reload
  // only when the resume point changes.
  const movieSources = createMemo(() => (props.type === 'movie' || (!meta.loading && !isSeries()) ? next() : undefined), undefined, {
    equals: (a, b) => a?.videoId === b?.videoId && a?.start === b?.start,
  });

  const play = (pick: Pick, s: Stream, g: StreamGroup) => {
    const m = item();
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

  /** resuming is true for the video that continue watching reopens; its source is chosen below. */
  const resuming = (videoId: string) => Boolean(props.autoplay) && videoId === (entry()?.videoId ?? (isSeries() ? undefined : props.id));

  // Resume from continue watching: reuse the last source when it is offered
  // again, else (in automatic mode) pick the best one.
  const [autoplaying, setAutoplaying] = createSignal(false);
  createEffect(
    on([next, entry, () => meta.loading || entry.loading], ([pick, e, loading]) => {
      if (!props.autoplay || loading || !pick || autoplaying()) return;
      setAutoplaying(true);
      if (!e?.streamKey) {
        if (e) void autoPick(props.type, pick.videoId).then((found) => found && props.active && play(pick, found.stream, found.group)).catch(() => undefined);
        return;
      }
      content
        .streams(props.type, pick.videoId)
        .then(async (groups) => {
          const found = findStream(groups, { key: e.streamKey });
          if (found && props.active) return play(pick, found.stream, found.group);
          const best = await autoPick(props.type, pick.videoId);
          if (best && props.active) play(pick, best.stream, best.group);
        })
        .catch(() => undefined);
    }),
  );

  /** focusEpisode shows an episode's season and focuses its card. */
  const focusEpisode = (videoId?: string) => {
    const v = videos().find((v) => v.id === videoId);
    if (v && (v.season ?? 0) !== season()) setSeason(v.season ?? 0);
    queueMicrotask(() => {
      const cards = [...(root?.querySelectorAll<HTMLElement>('.episode') ?? [])];
      const el = cards.find((c) => c.dataset.id === videoId) ?? cards[0];
      if (el) focusEl(el);
    });
  };

  const choose = (v: Video) => {
    if (chosen()?.videoId === v.id) {
      const card = root?.querySelector<HTMLElement>('.episode-sources .source:not(:disabled)');
      if (card) focusEl(card);
      return;
    }
    setChosen({ video: v, videoId: v.id, start: resumeAt(v.id) });
  };

  const closeSources = () => {
    const id = chosen()?.videoId;
    setChosen(undefined);
    focusEpisode(id);
    if (body) glide(body, 'y', 0);
  };

  // Back closes the open sources before it leaves the screen.
  onCleanup(
    onBack(() => {
      if (!props.active || !chosen()) return false;
      closeSources();
      return true;
    }),
  );

  // Series open on the episode to watch next; movies on their sources.
  let placed = false;
  createEffect(() => {
    if (placed || !props.active || !isSeries() || !seasonVideos().length) return;
    placed = true;
    focusEpisode(next()?.videoId);
  });
  onMount(() => queueMicrotask(() => !placed && focusFirst(root?.querySelector('.details-head') ?? document)));

  // Coming back from the player: refresh progress, close the sources, and
  // focus the episode that now comes next.
  createEffect(
    on(
      () => props.active,
      (active, was) => {
        if (!active || was !== false) return;
        loadWatched();
        const refreshed = Promise.resolve(refetchEntry());
        if (!isSeries()) return;
        setChosen(undefined);
        if (body) glide(body, 'y', 0);
        void refreshed.then(() => focusEpisode(next()?.videoId));
      },
    ),
  );

  const clearProgress = async () => {
    await library.remove(props.id);
    refetchEntry();
    toast(t('details.clearProgress'));
  };

  const progress = (v: Video) => {
    const e = entry();
    return e && e.videoId === v.id && e.duration > 0 && e.position < e.duration * 0.92 ? e.position / e.duration : 0;
  };

  return (
    <div class="screen details" classList={{ series: isSeries() }} ref={root}>
      <Backdrop src={item().background} full />
      <div class="details-body" ref={body} data-nav-pane data-nav-top="0.2">
        <div class="details-head" data-nav-block>
          <Show when={item().logo} fallback={<h1>{item().name}</h1>}>
            <img class="logo" src={item().logo} alt={item().name} />
          </Show>
          <MetaLine item={item()} />
          <Show when={item().description}>
            <p class="description">{item().description}</p>
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
          <Show when={meta.loading || entry()}>
            <div class="btn-row">
              <Show when={meta.loading}>
                <Spinner />
              </Show>
              <Show when={entry()}>
                <button class="btn" data-nav onClick={clearProgress}>
                  <X />
                  {t('details.clearProgress')}
                </button>
              </Show>
            </div>
          </Show>
          <Show when={movieSources()} keyed>
            {(pick) => (
              <AutoSourceRows
                type={props.type}
                videoId={pick.videoId}
                label={sourcesLabel(pick)}
                selectedKey={entry()?.videoId === pick.videoId ? entry()?.streamKey : undefined}
                onPick={(s, g) => play(pick, s, g)}
                active={props.active}
                disabled={resuming(pick.videoId)}
              />
            )}
          </Show>
        </div>

        <Show when={isSeries() && videos().length}>
          <div class="seasons" data-nav-row data-nav-group data-nav-block data-nav-top="nearest">
            <For each={seasons()}>
              {(s) => (
                <button
                  class="chip"
                  classList={{ selected: season() === s }}
                  data-nav
                  data-nav-entry={season() === s ? '' : undefined}
                  onClick={() => setSeason(s)}
                  onFocusIn={() => {
                    if (season() === s) return;
                    setSeason(s);
                    setChosen(undefined);
                  }}
                >
                  {s === 0 ? t('details.specials') : t('details.season', { n: s })}
                </button>
              )}
            </For>
          </div>
          <div class="episode-track" data-nav-row data-nav-group data-nav-block data-nav-top="nearest">
            <For each={seasonVideos()}>
              {(v) => (
                <button
                  class="episode"
                  classList={{ chosen: chosen()?.videoId === v.id }}
                  data-id={v.id}
                  data-nav
                  data-nav-entry={chosen()?.videoId === v.id ? '' : undefined}
                  onClick={() => choose(v)}
                >
                  <div class="thumb">
                    <Show when={v.thumbnail}>
                      <img src={v.thumbnail} loading="lazy" alt="" />
                    </Show>
                    <Show when={watched().has(v.id)}>
                      <span class="watched">
                        <CircleCheck aria-label={t('details.watched')} />
                      </span>
                    </Show>
                    <Show when={progress(v)}>
                      {(p) => (
                        <div class="progress">
                          <i style={{ width: `${p() * 100}%` }} />
                        </div>
                      )}
                    </Show>
                  </div>
                  <div class="episode-label">{episodeLabel(v)}</div>
                  <h3>{v.name ?? v.title ?? t('details.episode', { n: v.episode ?? v.number ?? 0 })}</h3>
                </button>
              )}
            </For>
          </div>
          <Show when={chosen()} keyed>
            {(pick) => (
              <div class="episode-sources" data-nav-top="0.5">
                <AutoSourceRows
                  type={props.type}
                  videoId={pick.videoId}
                  label={sourcesLabel(pick)}
                  selectedKey={entry()?.videoId === pick.videoId ? entry()?.streamKey : undefined}
                  onPick={(s, g) => play(pick, s, g)}
                  active={props.active}
                  disabled={resuming(pick.videoId)}
                />
              </div>
            )}
          </Show>
        </Show>
      </div>
    </div>
  );
}
